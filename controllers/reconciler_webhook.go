package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/suffiks/suffiks/base"
	"github.com/suffiks/suffiks/extension/protogen"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ admission.CustomValidator = &ReconcilerWrapper[*base.Application]{}

func (r *ReconcilerWrapper[V]) validate(ctx context.Context, typ protogen.ValidationType, newObj, oldObj runtime.Object) error {
	log := log.FromContext(ctx)

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

	if err := r.CRDController.Validate(ctx, typ, newV, oldV); err != nil {
		if ferr, ok := err.(base.FieldErrsWrapper); ok {
			return apierrors.NewInvalid(
				v.GetObjectKind().GroupVersionKind().GroupKind(),
				v.GetName(),
				field.ErrorList(ferr),
			)
		}

		log.Error(err, "extension validation error")
		return apierrors.NewInternalError(err)
	}

	return nil
}

func (r *ReconcilerWrapper[V]) Default(ctx context.Context, obj runtime.Object) error {
	v := obj.(V)

	defaults, err := r.CRDController.Default(ctx, v)
	if err != nil {
		return fmt.Errorf("Default crdmanager: %w", err)
	}

	changeset := &base.Changeset{}
	for _, d := range defaults {
		if err := changeset.AddMergePatch(d.GetSpec()); err != nil {
			return err
		}
	}

	return r.applyChangeset(v, changeset)
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

func (r *ReconcilerWrapper[V]) applyChangeset(v V, changeset *base.Changeset) error {
	v.SetLabels(mergeMaps(v.GetLabels(), changeset.Labels))
	v.SetAnnotations(mergeMaps(v.GetAnnotations(), changeset.Annotations))

	if len(changeset.MergePatch) > 0 {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("applyChangeset unmarshal: %w", err)
		}
		out, err := jsonpatch.MergePatch(b, changeset.MergePatch)
		if err != nil {
			return fmt.Errorf("applyChangeset mergePatch: %w", err)
		}
		err = json.Unmarshal(out, v)
		if err != nil {
			return fmt.Errorf("applyChangeset unmarshal: %w", err)
		}
	}
	return nil
}
