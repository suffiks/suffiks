package controller

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/suffiks/suffiks/internal/extension"
	"github.com/suffiks/suffiks/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logr "sigs.k8s.io/controller-runtime/pkg/log"
)

type Reconciler[V Object] interface {
	NewObject() V
	CreateOrUpdate(ctx context.Context, obj V, changeset *extension.Changeset) error
	Delete(ctx context.Context, obj V) error
	UpdateStatus(ctx context.Context, obj V, extensions []string) (changes bool, err error)
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

	Child         traceWrapper[V] // Reconciler[V]
	CRDController *ExtensionController
}

func New[V Object](client client.Client, child Reconciler[V], crdController *ExtensionController) *ReconcilerWrapper[V] {
	return &ReconcilerWrapper[V]{
		Client: client,
		Child: traceWrapper[V]{
			Reconciler: child,
		},
		CRDController: crdController,
	}
}

func (r *ReconcilerWrapper[V]) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, span := tracing.Start(ctx, "Reconcile")
	defer span.End()

	kind := r.Child.NewObject().GetObjectKind().GroupVersionKind().Kind
	if kind == "" {
		parts := strings.Split(fmt.Sprintf("%T", r.Child.NewObject()), ".")
		kind = parts[len(parts)-1]
	}

	span.SetAttributes(attribute.String("name", req.Name), attribute.String("namespace", req.Namespace), attribute.String("kind", kind))

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
		span.AddEvent("add finalizer")
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
		changes, err := r.Child.UpdateStatus(ctx, v, r.Child.Extensions(v))
		if err != nil {
			return r.handleError(ctx, err, "unable to update child status on non-modified object")
		}

		if changes {
			err = r.Status().Update(ctx, v)
			return r.handleError(ctx, err, "unable to update status on non-modified object")
		}
		return ctrl.Result{}, nil
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

	changes, err := r.Child.UpdateStatus(ctx, v, result.Extensions.Slice())
	if err != nil {
		return r.handleError(ctx, err, "unable to update child status")
	}

	if changes {
		err = r.Status().Update(ctx, v)
		return r.handleError(ctx, err, "unable to update status")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReconcilerWrapper[V]) SetupWithManager(mgr ctrl.Manager) (err error) {
	bldr := ctrl.NewControllerManagedBy(mgr).
		Watches(r.Child.NewObject(), &handler.EnqueueRequestForObject{}).
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

	var opts []trace.EventOption
	_, fn, l, ok := runtime.Caller(1)
	if ok {
		opts = append(opts, trace.WithAttributes(
			attribute.String("location", fn+":"+strconv.Itoa(l)),
		), trace.WithStackTrace(true))
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, opts...)

	for _, f := range f {
		err = f(err)
		if err == nil {
			return ctrl.Result{}, nil
		}
	}

	log.Error(err, msg)
	return ctrl.Result{}, err
}
