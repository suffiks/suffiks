package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/suffiks/suffiks/extension/protogen"
	"github.com/suffiks/suffiks/internal/extension"
	"github.com/suffiks/suffiks/internal/tracing"
	suffiksv1 "github.com/suffiks/suffiks/pkg/api/suffiks/v1"
	"go.opentelemetry.io/otel/attribute"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logr "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.CustomValidator = &ReconcilerWrapper[*suffiksv1.Application]{}

type namespaceName interface {
	GetNamespace() string
	GetName() string
}

func (r *ReconcilerWrapper[V]) validate(ctx context.Context, typ protogen.ValidationType, newObj, oldObj runtime.Object) (admission.Warnings, error) {
	kind := r.Child.NewObject().GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		parts := strings.Split(fmt.Sprintf("%T", r.Child.NewObject()), ".")
		kind = parts[len(parts)-1]
	}

	ctx, span := tracing.Start(ctx, "Validate")
	defer span.End()
	span.SetAttributes(attribute.String("type", protogen.ValidationType_name[int32(typ)]), attribute.String("kind", kind))

	if nn, ok := newObj.(namespaceName); ok {
		span.SetAttributes(attribute.String("name", nn.GetName()), attribute.String("namespace", nn.GetNamespace()))
	} else if nn, ok := oldObj.(namespaceName); ok {
		span.SetAttributes(attribute.String("name", nn.GetName()), attribute.String("namespace", nn.GetNamespace()))
	}

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
		if ferr, ok := err.(FieldErrsWrapper); ok {
			return nil, apierrors.NewInvalid(
				v.GetObjectKind().GroupVersionKind().GroupKind(),
				v.GetName(),
				field.ErrorList(ferr),
			)
		}

		log.Error(err, "extension validation error")
		span.RecordError(err)
		return nil, apierrors.NewInternalError(err)
	}

	return nil, nil
}

func (r *ReconcilerWrapper[V]) Default(ctx context.Context, obj runtime.Object) error {
	v := obj.(V)
	kind := r.Child.NewObject().GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		parts := strings.Split(fmt.Sprintf("%T", r.Child.NewObject()), ".")
		kind = parts[len(parts)-1]
	}

	ctx, span := tracing.Start(ctx, "Default")
	defer span.End()
	span.SetAttributes(attribute.String("type", "default"))
	span.SetAttributes(attribute.String("name", v.GetName()), attribute.String("namespace", v.GetNamespace()))
	span.SetAttributes(attribute.String("kind", kind))

	log := logr.FromContext(ctx).WithValues("trace_id", span.SpanContext().TraceID().String())
	ctx = logr.IntoContext(ctx, log)

	if defaulter, ok := r.Child.Reconciler.(ReconcilerDefault[V]); ok {
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

func (r *ReconcilerWrapper[V]) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return r.validate(ctx, protogen.ValidationType_CREATE, obj, nil)
}

func (r *ReconcilerWrapper[V]) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return r.validate(ctx, protogen.ValidationType_UPDATE, newObj, oldObj)
}

func (r *ReconcilerWrapper[V]) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
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
