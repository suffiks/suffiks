package wasi

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/suffiks/suffiks"
	"github.com/suffiks/suffiks/internal/controller"
	"github.com/suffiks/suffiks/internal/extension"
	"github.com/suffiks/suffiks/internal/extension/oci"
	suffiksv1 "github.com/suffiks/suffiks/pkg/api/suffiks/v1"
	"github.com/suffiks/suffiks/pkg/client/clientset/versioned/scheme"
	"github.com/urfave/cli/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
	"sigs.k8s.io/yaml"
)

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
