package controllers

import (
	"context"
	"fmt"

	suffiksv1 "github.com/suffiks/suffiks/apis/suffiks/v1"
	"github.com/suffiks/suffiks/base/tracing"
	"github.com/suffiks/suffiks/extension/protogen"
	controller "github.com/suffiks/suffiks/internal/controllers"
	"github.com/suffiks/suffiks/internal/extension"
	"go.opentelemetry.io/otel/attribute"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logr "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.CustomValidator = &ReconcilerWrapper[*suffiksv1.Application]{}

func (r *ReconcilerWrapper[V]) validate(ctx context.Context, typ protogen.ValidationType, newObj, oldObj runtime.Object) error {
	kind := r.Child.NewObject().GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		kind = fmt.Sprintf("%T", r.Child.NewObject())
	}

	ctx, span := tracing.Start(ctx, "Validate."+kind)
	defer span.End()
	span.SetAttributes(attribute.String("type", string(typ)))

	log := logr.FromContext(ctx).WithValues("trace_id", span.SpanContext().TraceID().String())
	ctx = logr.IntoContext(ctx, log)

	var (
		newV  V
		oldV  V
		v     V
		isSet bool
	)
	if newObj != nil {
		newV = newObj.(V)
		v = newV
		isSet = true
	}

	if oldObj != nil {
		oldV = oldObj.(V)
		if !isSet {
			v = oldV
		}
	}

	span.SetAttributes(attribute.String("name", v.GetName()), attribute.String("namespace", v.GetNamespace()))

	if err := r.CRDController.Validate(ctx, typ, newV, oldV); err != nil {
		if ferr, ok := err.(controller.FieldErrsWrapper); ok {
			return apierrors.NewInvalid(
				v.GetObjectKind().GroupVersionKind().GroupKind(),
				v.GetName(),
				field.ErrorList(ferr),
			)
		}

		log.Error(err, "extension validation error")
		span.RecordError(err)
		return apierrors.NewInternalError(err)
	}

	return nil
}

func (r *ReconcilerWrapper[V]) Default(ctx context.Context, obj runtime.Object) error {
	v := obj.(V)
	kind := r.Child.NewObject().GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		kind = fmt.Sprintf("%T", r.Child.NewObject())
	}

	ctx, span := tracing.Start(ctx, "Validate."+kind)
	defer span.End()
	span.SetAttributes(attribute.String("type", "default"))
	span.SetAttributes(attribute.String("name", v.GetName()), attribute.String("namespace", v.GetNamespace()))

	log := logr.FromContext(ctx).WithValues("trace_id", span.SpanContext().TraceID().String())
	ctx = logr.IntoContext(ctx, log)

	if defaulter, ok := r.Child.(ReconcilerDefault[V]); ok {
		if err := defaulter.Default(ctx, v); err != nil {
			return err
		}
	}

	defaults, err := r.CRDController.Default(ctx, v)
	if err != nil {
		return fmt.Errorf("Default crdmanager: %w", err)
	}

	changeset := &extension.Changeset{}
	for _, d := range defaults {
		if err := changeset.AddMergePatch(d.GetSpec()); err != nil {
			return err
		}
	}

	return changeset.Apply(v)
}

func (r *ReconcilerWrapper[V]) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return r.validate(ctx, protogen.ValidationType_CREATE, obj, nil)
}

func (r *ReconcilerWrapper[V]) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	return r.validate(ctx, protogen.ValidationType_UPDATE, newObj, oldObj)
}

func (r *ReconcilerWrapper[V]) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return r.validate(ctx, protogen.ValidationType_DELETE, nil, obj)
}

func (r *ReconcilerWrapper[V]) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r.Child.NewObject()).
		WithDefaulter(r).
		WithValidator(r).
		// TODO(thokra): Currently, the webhook builder doesn't expose the webhook ContextFunc.
		// This requires a change in controller-runtime.
		// WithContextFunc(func(ctx context.Context, r *http.Request) context.Context {
		// 	return otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		// }).
		Complete()
}
