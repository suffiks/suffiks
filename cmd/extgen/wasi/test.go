package wasi

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/suffiks/suffiks"
	"github.com/suffiks/suffiks/extension/protogen"
	"github.com/suffiks/suffiks/internal/controller"
	"github.com/suffiks/suffiks/internal/extension"
	"github.com/suffiks/suffiks/internal/extension/oci"
	suffiksv1 "github.com/suffiks/suffiks/pkg/api/suffiks/v1"
	"github.com/suffiks/suffiks/pkg/client/clientset/versioned/scheme"
	"github.com/urfave/cli/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	"sigs.k8s.io/yaml"
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

		if !cmp.Equal(lookupObj, &lookup, cmpopts.EquateEmpty()) {
			printError(t.Name, "diff -want +got:\n%s", cmp.Diff(lookupObj, &lookup, cmpopts.EquateEmpty()))
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

func testCmd() *cli.Command {
	return &cli.Command{
		Name:        "test",
		Description: "Test the extension by running it in a local environment.",
		ArgsUsage:   "A single argument is required: path to the wasi file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:       "tests",
				Usage:      "Path to tests directory",
				TakesFile:  true,
				Value:      "./tests",
				HasBeenSet: true,
				Required:   true,
			},
			&cli.StringFlag{
				Name:       "ext",
				Usage:      "Path extension CRD",
				TakesFile:  true,
				Value:      "ext.yaml",
				HasBeenSet: true,
				Required:   true,
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Verbose output",
			},
		},
		Action: func(c *cli.Context) error {
			log.SetFlags(0)

			ctx := c.Context
			if c.Bool("verbose") {
				ctx = context.WithValue(c.Context, ctxKey("verbose"), true)
			}
			wasi := loader(c.Args().First())

			var extObj suffiksv1.Extension
			b, err := os.ReadFile(c.String("ext"))
			if err != nil {
				return fmt.Errorf("failed to read extension file: %w", err)
			}
			if err := yaml.Unmarshal(b, &extObj); err != nil {
				return fmt.Errorf("failed to unmarshal extension file: %w", err)
			}

			return filepath.WalkDir(c.String("tests"), func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.IsDir() || filepath.Ext(path) != ".yaml" {
					return nil
				}
				log.Println("Running", path)
				if err := testFile(ctx, wasi, extObj, path); err != nil {
					return fmt.Errorf("%v: %w", path, err)
				}
				return nil
			})
		},
	}
}

func testFile(ctx context.Context, wasi extension.WASILoader, extObj suffiksv1.Extension, testPath string) error {
	b, err := os.ReadFile(testPath)
	if err != nil {
		return fmt.Errorf("failed to read test file: %w", err)
	}

	var t tests
	if err := yaml.Unmarshal(b, &t); err != nil {
		return fmt.Errorf("failed to unmarshal test file: %w", err)
	}

	var objs []runtime.Object
	if len(t.Config) > 0 {
		switch {
		case extObj.Spec.Controller.WASI.ConfigMap == nil:
			return fmt.Errorf("config not supported when no config map is defined in the extension CRD")
		case extObj.Spec.Controller.WASI.ConfigMap.Name == "":
			return fmt.Errorf("config not supported when no config map name is defined in the extension CRD")
		case extObj.Spec.Controller.WASI.ConfigMap.Namespace == "":
			return fmt.Errorf("config not supported when no config map namespace is defined in the extension CRD")
		}

		cm := corev1.ConfigMap{}
		cm.SetName(extObj.Spec.Controller.WASI.ConfigMap.Name)
		cm.SetNamespace(extObj.Spec.Controller.WASI.ConfigMap.Namespace)
		cm.Data = t.Config
		objs = append(objs, &cm)

		verboseLog(ctx, testPath, "Created config %q in %q", cm.GetName(), cm.GetNamespace())
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	client := fake.NewSimpleDynamicClient(scheme, objs...)
	mgr, err := extension.NewExtensionManager(
		context.Background(),
		suffiks.CRDFiles,
		client,
		extension.WithWASILoader(wasi),
	)
	if err != nil {
		return fmt.Errorf("failed to create extension manager: %w", err)
	}

	if err := mgr.Add(extObj); err != nil {
		return fmt.Errorf("failed to add extension: %w", err)
	}

	ctrl := controller.NewExtensionController(mgr)

	for _, test := range t.Tests {
		if !test.Test(ctx, ctrl, client) {
			return fmt.Errorf("test failed: %s", test.Name)
		}
	}
	return nil
}

func loader(path string) extension.WASILoader {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return func(ctx context.Context, image, tag string) (map[string][]byte, error) {
		return map[string][]byte{
			oci.MediaTypeWASI: b,
		}, nil
	}
}

func newObject(u *unstructured.Unstructured) controller.Object {
	if u == nil {
		return nil
	}

	var obj controller.Object
	switch u.GetKind() {
	case "Application":
		obj = &suffiksv1.Application{}
	case "Work":
		obj = &suffiksv1.Work{}
	default:
		panic(fmt.Errorf("unknown kind: %s", u.GetKind()))
	}

	sc := scheme.Scheme
	if err := sc.Convert(u, obj, nil); err != nil {
		panic(err)
	}

	if obj.GetObjectKind().GroupVersionKind().Kind == "" {
		gvk := schema.FromAPIVersionAndKind(u.GetAPIVersion(), u.GetKind())
		obj.GetObjectKind().SetGroupVersionKind(gvk)
	}

	fmt.Println(obj.GetObjectKind().GroupVersionKind().GroupVersion())
	return obj
}
