package wasi

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/suffiks/suffiks"
	suffiksv1 "github.com/suffiks/suffiks/api/suffiks/v1"
	"github.com/suffiks/suffiks/extension/protogen"
	"github.com/suffiks/suffiks/internal/controller"
	"github.com/suffiks/suffiks/internal/extension"
	"github.com/suffiks/suffiks/internal/extension/oci"
	"github.com/suffiks/suffiks/pkg/client/clientset/versioned/scheme"
	"github.com/urfave/cli/v2"
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
	// Type is the type of validation to perform. It can be either "create" or "update". Defaults to "create".
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

type test struct {
	Name       string          `yaml:"name"`
	Validate   *validateTest   `yaml:"validate"`
	Defaulting *defaultingTest `yaml:"defaulting"`
	Sync       *syncTest       `yaml:"sync"`
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

	printLog(t.Name, "No test found")
	return false
}

func (t test) validate(ctx context.Context, ctrl *controller.ExtensionController) bool {
	typ := protogen.ValidationType_CREATE
	if t.Validate.Type == "update" {
		typ = protogen.ValidationType_UPDATE
	}
	verboseLog(ctx, t.Name, "Validate (%v)", typ)

	var old controller.Object
	if t.Validate.Type == "update" {
		if t.Validate.Old == nil {
			old = newObject(t.Validate.Resource)
		} else {
			old = newObject(t.Validate.Old)
		}
	}

	err := ctrl.Validate(ctx, typ, newObject(t.Validate.Resource), old)
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
	Tests []test `yaml:"tests"`
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
			b, err := os.ReadFile("./tests/test.yaml")
			if err != nil {
				return fmt.Errorf("failed to read test file: %w", err)
			}

			var t tests
			if err := yaml.Unmarshal(b, &t); err != nil {
				return fmt.Errorf("failed to unmarshal test file: %w", err)
			}

			client := fake.NewSimpleDynamicClient(runtime.NewScheme())
			mgr, err := extension.NewExtensionManager(
				context.Background(),
				suffiks.CRDFiles,
				client,
				extension.WithWASILoader(loader(c.Args().First())),
			)
			if err != nil {
				return fmt.Errorf("failed to create extension manager: %w", err)
			}

			var extObj suffiksv1.Extension
			b, err = os.ReadFile(c.String("ext"))
			if err != nil {
				return fmt.Errorf("failed to read extension file: %w", err)
			}
			if err := yaml.Unmarshal(b, &extObj); err != nil {
				return fmt.Errorf("failed to unmarshal extension file: %w", err)
			}

			if err := mgr.Add(extObj); err != nil {
				return fmt.Errorf("failed to add extension: %w", err)
			}

			ctrl := controller.NewExtensionController(mgr)

			ctx := c.Context
			if c.Bool("verbose") {
				ctx = context.WithValue(c.Context, ctxKey("verbose"), true)
			}

			for _, test := range t.Tests {
				if !test.Test(ctx, ctrl, client) {
					return fmt.Errorf("test failed: %s", test.Name)
				}
			}
			return nil
		},
	}
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
