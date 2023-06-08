package controllers

import (
	"context"
	"fmt"
	"runtime"
	"strconv"

	"github.com/suffiks/suffiks/base/tracing"
	controller "github.com/suffiks/suffiks/internal/controllers"
	"github.com/suffiks/suffiks/internal/extension"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logr "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type Reconciler[V Object] interface {
	NewObject() V
	CreateOrUpdate(ctx context.Context, obj V, changeset *extension.Changeset) error
	Delete(ctx context.Context, obj V) error
	UpdateStatus(ctx context.Context, obj V, extensions []string) error
	IsModified(ctx context.Context, obj V) (bool, error)
	Extensions(obj V) []string
	// This might be required for some reconcilers, but not for others.
	// So initially it's not required, but we might either make it optional with
	// a sane default, or make it required.
	// ApplyChangeset(obj V, changeset *base.Changeset) error
	Owns() []client.Object
}

type ReconcilerDefault[V Object] interface {
	Default(ctx context.Context, obj V) error
}

const suffiksFinalizer = "suffiks.suffiks.com/finalizer"

type ReconcilerWrapper[V Object] struct {
	client.Client

	Child         Reconciler[V]
	CRDController *controller.ExtensionController
}

func (r *ReconcilerWrapper[V]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	kind := r.Child.NewObject().GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		kind = fmt.Sprintf("%T", r.Child.NewObject())
	}

	ctx, span := tracing.Start(ctx, "Reconcile."+kind)
	defer span.End()
	span.SetAttributes(attribute.String("name", req.Name), attribute.String("namespace", req.Namespace))

	log := logr.FromContext(ctx).WithValues("trace_id", span.SpanContext().TraceID().String())
	ctx = logr.IntoContext(ctx, log)

	v := r.Child.NewObject()
	if err := r.Get(ctx, req.NamespacedName, v); err != nil {
		return r.handleError(ctx, err, "unable to fetch "+kind, client.IgnoreNotFound)
	}

	if v.GetDeletionTimestamp() != nil {
		if !controllerutil.ContainsFinalizer(v, suffiksFinalizer) {
			return ctrl.Result{}, nil
		}

		if err := r.Child.Delete(ctx, v); err != nil {
			log.Error(err, "unable to delete from child, will try again later")
			return r.handleError(ctx, err, "unable to delete from child")
		}

		if err := r.CRDController.Delete(ctx, v); err != nil {
			log.Error(err, "unable to delete from extensions, will try again later")
			return r.handleError(ctx, err, "unable to delete from extensions")
		}

		span.SetAttributes(attribute.String("action", "delete"))
		// remove our finalizer from the list and update it.
		controllerutil.RemoveFinalizer(v, suffiksFinalizer)

		if err := r.Update(ctx, v); err != nil {
			return r.handleError(ctx, err, "unable to remove finalizer", client.IgnoreNotFound)
		}

		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(v, suffiksFinalizer) {
		span.SetAttributes(attribute.String("action", "add finalizer"))
		controllerutil.AddFinalizer(v, suffiksFinalizer)
		if err := r.Update(ctx, v); err != nil {
			return r.handleError(ctx, err, "unable to add finalizer")
		}
		return ctrl.Result{}, nil
	}

	modified, err := r.Child.IsModified(ctx, v)
	if err != nil {
		return r.handleError(ctx, err, "unable to check if application is modified", client.IgnoreNotFound)
	}
	if !modified {
		if err := r.Child.UpdateStatus(ctx, v, r.Child.Extensions(v)); err != nil {
			return r.handleError(ctx, err, "unable to update child status on non-modified object")
		}

		err = r.Status().Update(ctx, v)
		return r.handleError(ctx, err, "unable to update status on non-modified object")
	}

	result, err := r.CRDController.Sync(ctx, v)
	if err != nil {
		return r.handleError(ctx, err, "unable to sync CRD")
	}

	for _, old := range r.Child.Extensions(v) {
		if !result.Extensions.Contains(old) {
			if err := r.CRDController.DeleteExtension(ctx, v, old); err != nil {
				log.Error(err, "unable to delete single extension, will try again later", "extension", old)
				result.Extensions.Add(old)
			}
		}
	}

	if err := r.Child.CreateOrUpdate(ctx, v, result.Changeset); err != nil {
		return r.handleError(ctx, err, "unable to create or update")
	}

	if err := r.Child.UpdateStatus(ctx, v, result.Extensions.Slice()); err != nil {
		return r.handleError(ctx, err, "unable to update child status")
	}

	err = r.Status().Update(ctx, v)
	return r.handleError(ctx, err, "unable to update status")
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReconcilerWrapper[V]) SetupWithManager(mgr ctrl.Manager) (err error) {
	bldr := ctrl.NewControllerManagedBy(mgr).
		Watches(&source.Kind{Type: r.Child.NewObject()}, &handler.EnqueueRequestForObject{}).
		For(r.Child.NewObject(), builder.OnlyMetadata)

	for _, o := range r.Child.Owns() {
		bldr = bldr.Owns(o)
	}

	return bldr.Complete(r)
}

// handleError will, if the error is not nil:
// - record the error in the span
// - log the error with the msg
// - return the error after modifying it using the optional f functions
func (r *ReconcilerWrapper[V]) handleError(ctx context.Context, err error, msg string, f ...func(err error) error) (ctrl.Result, error) {
	log := logr.FromContext(ctx)
	if err == nil {
		return ctrl.Result{}, nil
	}

	span := trace.SpanFromContext(ctx)
	span.RecordError(err)

	_, fn, l, ok := runtime.Caller(1)
	if ok {
		span.SetAttributes(
			attribute.String("location", fn+":"+strconv.Itoa(l)),
		)
	}

	for _, f := range f {
		err = f(err)
		if err == nil {
			return ctrl.Result{}, nil
		}
	}

	log.Error(err, msg)
	return ctrl.Result{}, err
}
