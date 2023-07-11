package wasi

import (
	"context"
	"log"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/suffiks/suffiks/extension/protogen"
	"github.com/suffiks/suffiks/internal/controller"
	"github.com/suffiks/suffiks/internal/extension"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

type ctxKey string

type validateTest struct {
	Resource *unstructured.Unstructured `yaml:"resource"`
	Invalid  bool                       `yaml:"invalid"`
	Old      *unstructured.Unstructured `yaml:"old"`
	// Type is the type of validation to perform. It can be either "create", "update" or "delete". Defaults to "create".
	Type string `yaml:"type"`
}

type defaultingTest struct {
	Resource *unstructured.Unstructured `yaml:"resource"`
	Expected *unstructured.Unstructured `yaml:"expected"`
}

type syncTest struct {
	Resource *unstructured.Unstructured  `yaml:"resource"`
	Expected *unstructured.Unstructured  `yaml:"expected"`
	Lookup   []unstructured.Unstructured `yaml:"lookup"`
}

type deleteTest struct {
	Resource *unstructured.Unstructured  `yaml:"resource"`
	NotFound []unstructured.Unstructured `yaml:"notFound"`
}

type test struct {
	Name       string          `yaml:"name"`
	Validate   *validateTest   `yaml:"validate"`
	Defaulting *defaultingTest `yaml:"defaulting"`
	Sync       *syncTest       `yaml:"sync"`
	Delete     *deleteTest     `yaml:"delete"`
}

func (t test) Test(ctx context.Context, ctrl *controller.ExtensionController, client dynamic.Interface) bool {
	if t.Validate != nil {
		return t.validate(ctx, ctrl)
	}
	if t.Defaulting != nil {
		return t.defaulting(ctx, ctrl)
	}
	if t.Sync != nil {
		return t.sync(ctx, ctrl, client)
	}
	if t.Delete != nil {
		return t.delete(ctx, ctrl, client)
	}

	printLog(t.Name, "No test found")
	return false
}

func (t test) validate(ctx context.Context, ctrl *controller.ExtensionController) bool {
	typ := protogen.ValidationType_CREATE
	switch t.Validate.Type {
	case "create":
	case "update":
		typ = protogen.ValidationType_UPDATE
	case "delete":
		typ = protogen.ValidationType_DELETE
	default:
		printLog(t.Name, "Invalid validation type: %q, expected 'create', 'update', 'delete'", t.Validate.Type)
		return false
	}

	verboseLog(ctx, t.Name, "Validate (%v)", typ)

	var old controller.Object
	newO := newObject(t.Validate.Resource)
	switch t.Validate.Type {
	case "update":
		if t.Validate.Old == nil {
			old = newObject(t.Validate.Resource)
		} else {
			old = newObject(t.Validate.Old)
		}
	case "delete":
		old = newO
		newO = nil
	}

	err := ctrl.Validate(ctx, typ, newO, old)
	if err != nil {
		if t.Validate.Invalid {
			verboseLog(ctx, t.Name, "Expected error: %v", err)
			return true
		}

		printError(t.Name, "Unexpected error: %v", err)
		return false
	}

	if t.Validate.Invalid {
		printError(t.Name, "Expected error, but got none")
		return false
	}

	verboseLog(ctx, t.Name, "Resource is valid")
	return true
}

func (t test) defaulting(ctx context.Context, ctrl *controller.ExtensionController) bool {
	verboseLog(ctx, t.Name, "Defaulting")

	obj := newObject(t.Defaulting.Resource)
	resp, err := ctrl.Default(ctx, obj)
	if err != nil {
		printError(t.Name, "Unexpected error: %v", err)
		return false
	}

	if len(resp) != 1 {
		printError(t.Name, "Expected exactly one object, but got %d", len(resp))
		return false
	}

	changeset := &extension.Changeset{}
	if err := changeset.AddMergePatch(resp[0].GetSpec()); err != nil {
		printError(t.Name, "Failed to add merge patch: %v", err)
		return false
	}

	old := obj.DeepCopyObject()
	if err := changeset.Apply(obj); err != nil {
		printError(t.Name, "Failed to apply changeset: %v", err)
		return false
	}

	if t.Defaulting.Expected != nil {
		old = newObject(t.Defaulting.Expected)
	}

	if !cmp.Equal(obj, old, cmpopts.EquateEmpty()) {
		printError(t.Name, "diff -want +got:\n%s", cmp.Diff(old, obj, cmpopts.EquateEmpty()))
		return false
	}

	verboseLog(ctx, t.Name, "Resource is valid")
	return true
}

func (t test) sync(ctx context.Context, ctrl *controller.ExtensionController, client dynamic.Interface) bool {
	verboseLog(ctx, t.Name, "Sync")

	obj := newObject(t.Sync.Resource)
	resp, err := ctrl.Sync(ctx, obj)
	if err != nil {
		printError(t.Name, "Unexpected error: %v", err)
		return false
	}

	old := obj.DeepCopyObject()
	if err := resp.Changeset.Apply(obj); err != nil {
		printError(t.Name, "Failed to apply changeset: %v", err)
		return false
	}

	if t.Sync.Expected != nil {
		old = newObject(t.Sync.Expected)
	}

	if !cmp.Equal(obj, old, cmpopts.EquateEmpty()) {
		printError(t.Name, "diff -want +got:\n%s", cmp.Diff(old, obj, cmpopts.EquateEmpty()))
		return false
	}

	for _, lookup := range t.Sync.Lookup {
		verboseLog(ctx, t.Name, "Lookup %s", lookup.GetName())

		gvr, _ := meta.UnsafeGuessKindToResource(lookup.GroupVersionKind())

		ns := lookup.GetNamespace()
		if ns == "" {
			ns = obj.GetNamespace()
		}

		lookupObj, err := client.Resource(gvr).Namespace(ns).Get(ctx, lookup.GetName(), metav1.GetOptions{})
		if err != nil {
			printError(t.Name, "Failed to get object: %v", err)
			return false
		}

		if !cmp.Equal(&lookup, lookupObj, cmpopts.EquateEmpty()) {
			printError(t.Name, "diff -want +got:\n%s", cmp.Diff(&lookup, lookupObj, cmpopts.EquateEmpty()))
			return false
		}
	}

	return true
}

func (t test) delete(ctx context.Context, ctrl *controller.ExtensionController, client dynamic.Interface) bool {
	verboseLog(ctx, t.Name, "Delete")

	obj := newObject(t.Delete.Resource)
	err := ctrl.Delete(ctx, obj)
	if err != nil {
		printError(t.Name, "Unexpected error: %v", err)
		return false
	}

	for _, lookup := range t.Delete.NotFound {
		verboseLog(ctx, t.Name, "NotFound %s", lookup.GetName())

		gvr, _ := meta.UnsafeGuessKindToResource(lookup.GroupVersionKind())

		ns := lookup.GetNamespace()
		if ns == "" {
			ns = obj.GetNamespace()
		}

		_, err := client.Resource(gvr).Namespace(ns).Get(ctx, lookup.GetName(), metav1.GetOptions{})
		if !errors.IsNotFound(err) {
			printError(t.Name, "Expected not found: %v", err)
			return false
		}
	}

	return true
}

func printError(name, format string, args ...any) {
	printLog(name, "[ERROR] "+format, args...)
}

func printLog(name, format string, args ...any) {
	log.Printf("[%s] "+format, append([]any{name}, args...)...)
}

func verboseLog(ctx context.Context, name, format string, args ...any) {
	if v, ok := ctx.Value(ctxKey("verbose")).(bool); ok && v {
		printLog(name, format, args...)
	}
}

type tests struct {
	Config map[string]string `yaml:"config"`
	Tests  []test            `yaml:"tests"`
}
