package controller

import (
	"context"

	"github.com/suffiks/suffiks/internal/extension"
	"github.com/suffiks/suffiks/internal/tracing"
)

type traceWrapper[V Object] struct {
	Reconciler[V]
}

func (t *traceWrapper[V]) CreateOrUpdate(ctx context.Context, obj V, changeset *extension.Changeset) error {
	ctx, span := tracing.Start(ctx, "CreateOrUpdate")
	defer span.End()
	return t.Reconciler.CreateOrUpdate(ctx, obj, changeset)
}

func (t *traceWrapper[V]) Delete(ctx context.Context, obj V) error {
	ctx, span := tracing.Start(ctx, "Delete")
	defer span.End()
	return t.Reconciler.Delete(ctx, obj)
}

func (t *traceWrapper[V]) UpdateStatus(ctx context.Context, obj V, extensions []string) (bool, error) {
	ctx, span := tracing.Start(ctx, "UpdateStatus")
	defer span.End()
	return t.Reconciler.UpdateStatus(ctx, obj, extensions)
}

func (t *traceWrapper[V]) IsModified(ctx context.Context, obj V) (bool, error) {
	ctx, span := tracing.Start(ctx, "IsModified")
	defer span.End()
	return t.Reconciler.IsModified(ctx, obj)
}
