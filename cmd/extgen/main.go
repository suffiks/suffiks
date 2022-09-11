package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	suffiksv1 "github.com/suffiks/suffiks/apis/suffiks/v1"
	"github.com/urfave/cli/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"
	"sigs.k8s.io/controller-tools/pkg/rbac"
	"sigs.k8s.io/yaml"
)

// TODO: Cleanup, this is currently a mess. The goals is to generate Extension CRDs
// using controller-tools and support kubebuilder markers.

//go:embed scaffold/*
var scaffoldFiles embed.FS

func main() {
	app := &cli.App{
		Name:     "extgen",
		Usage:    "Generate extension code",
		Commands: []*cli.Command{newGen, crdGen, rbacGen},
	}

	if err := app.Run(os.Args); err != nil {
		log.Println(err)
	}
}

var crdGen = &cli.Command{
	Name:  "crd",
	Usage: "Generate extension CRD",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "output",
			Usage: "Output directory `./config/crd`",
			Value: "./config/crd",
		},
		&cli.StringSliceFlag{
			Name:     "type",
			Usage:    "Types to generate extensions for",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "source",
			Usage: "Source directory where the types are defined",
			Value: "./controllers",
		},
	},

	Action: func(c *cli.Context) error {
		types := c.StringSlice("type")
		if len(types) == 0 {
			return fmt.Errorf("no types specified")
		}
		packages, err := loader.LoadRoots(c.String("source"))
		if err != nil {
			return err
		}
		registry := &markers.Registry{}
		if err := registry.Define("suffiks:extension", markers.DescribesType, ExtensionMarker{}); err != nil {
			return err
		}
		collector := &markers.Collector{
			Registry: registry,
		}

		typechecker := &loader.TypeChecker{}
		pars := &crd.Parser{
			Collector: collector,
			Checker:   typechecker,
		}

		for _, pkg := range packages {
			pars.NeedPackage(pkg)
		}
		if err := os.MkdirAll(c.String("output"), 0o755); err != nil {
			return err
		}

		for _, typ := range types {
			pars.NeedSchemaFor(crd.TypeIdent{
				Package: packages[0],
				Name:    typ,
			})
		}

		createdCRDs := 0
	OUTER:
		for k := range pars.Schemata {
			found := false
			for _, typ := range types {
				if k.Name == typ {
					found = true
					break
				}
			}
			if !found {
				continue OUTER
			}
			pars.NeedFlattenedSchemaFor(k)
			schema := pars.FlattenedSchemata[k]
			b, err := json.Marshal(schema)
			if err != nil {
				return err
			}
			ext := suffiksv1.Extension{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Extension",
					APIVersion: suffiksv1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: strings.ToLower(k.Name),
				},
				Spec: suffiksv1.ExtensionSpec{
					OpenAPIV3Schema: runtime.RawExtension{
						Raw: b,
					},
				},
			}

			mrks, err := collector.MarkersInPackage(k.Package)
			if err != nil {
				return err
			}

			for v, m := range mrks {
				if typ, ok := v.(*ast.TypeSpec); ok {
					if typ.Name.Name == k.Name {
						mrk := m.Get("suffiks:extension").(ExtensionMarker)
						ext.Spec.Targets = append(ext.Spec.Targets, mrk.Types...)
						ext.Spec.Always = mrk.Always
						if mrk.Defaulting || mrk.Validation {
							ext.Spec.Webhooks = suffiksv1.ExtensionWebhooks{
								Validation: mrk.Validation,
								Defaulting: mrk.Defaulting,
							}
						}
					}
				}
			}

			b, err = yaml.Marshal(ext)
			if err != nil {
				return err
			}
			f, err := os.Create(filepath.Join(c.String("output"), strings.ToLower(k.Name)+".yaml"))
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = f.Write(b)
			if err != nil {
				return err
			}
			createdCRDs++
		}

		if createdCRDs == 0 {
			return fmt.Errorf("no CRDs created")
		}

		return nil
	},
}

var rbacGen = &cli.Command{
	Name:  "rbac",
	Usage: "Generate extension RBAC",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Usage:    "Name of the role",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "output",
			Usage: "Output directory `./config/rbac`",
			Value: "./config/rbac",
		},
		&cli.StringFlag{
			Name:  "source",
			Usage: "Source directory where the types are defined",
			Value: "./controllers",
		},
	},
	Action: func(c *cli.Context) error {
		packages, err := loader.LoadRoots(c.String("source"))
		if err != nil {
			return err
		}
		registry := &markers.Registry{}
		registry.Register(rbac.RuleDefinition)
		collector := &markers.Collector{
			Registry: registry,
		}

		typechecker := &loader.TypeChecker{}

		v, err := rbac.GenerateRoles(&genall.GenerationContext{
			Collector: collector,
			Roots:     packages,
			Checker:   typechecker,
		}, c.String("name"))
		if err != nil {
			return err
		}

		if err := os.MkdirAll(c.String("output"), 0o755); err != nil {
			return err
		}

		if len(v) == 0 {
			return nil
		}

		f, err := os.Create(filepath.Join(c.String("output"), "role.yaml"))
		if err != nil {
			return err
		}
		defer f.Close()

		for _, obj := range v {
			_, err := f.Write([]byte("---\n"))
			if err != nil {
				return err
			}

			b, err := yaml.Marshal(obj)
			if err != nil {
				return err
			}
			_, err = f.Write(b)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

var newGen = &cli.Command{
	Name:  "new",
	Usage: "Scaffold a new extension",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "validation",
			Usage: "Add validation webhook support",
		},
		&cli.BoolFlag{
			Name:  "defaulting",
			Usage: "Add defaulting webhook support",
		},
		&cli.BoolFlag{
			Name:  "kubernetes",
			Usage: "Add kubernetes support",
		},
		&cli.BoolFlag{
			Name:  "always",
			Usage: "Always run the sync of this extension",
		},
		&cli.StringSliceFlag{
			Name:     "target",
			Usage:    "Types to target",
			Required: true,
		},
	},
	Before: func(c *cli.Context) error {
		targets := c.StringSlice("target")
		if len(targets) == 0 {
			return errors.New("target is required")
		}

		for _, target := range targets {
			switch target {
			case "Application", "Work":
			default:
				return errors.New("target must be one of: Application, Work")
			}
		}
		return nil
	},
	ArgsUsage: "First arg is the repo url (Go package path)",
	Action: func(c *cli.Context) error {
		repo := c.Args().Get(0)
		if repo == "" {
			return fmt.Errorf("missing repo url")
		}

		nameParts := strings.Split(repo, "/")
		name := nameParts[len(nameParts)-1]

		data := struct {
			Repo       string
			Name       string
			GoName     string
			Receiver   string
			Validation bool
			Defaulting bool
			Kubernetes bool
			Always     bool
			Targets    string
		}{
			Repo:       repo,
			Name:       name,
			GoName:     strings.Title(name),
			Receiver:   strings.ToLower(name[0:1]),
			Validation: c.Bool("validation"),
			Defaulting: c.Bool("defaulting"),
			Kubernetes: c.Bool("kubernetes"),
			Always:     c.Bool("always"),
			Targets:    strings.Join(c.StringSlice("target"), ";"),
		}

		if err := os.Mkdir(name, 0o755); err != nil {
			return err
		}

		err := fs.WalkDir(scaffoldFiles, "scaffold", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if path == "scaffold" {
				return nil
			}

			createPath := strings.TrimPrefix(path, "scaffold/")
			createPath = strings.ReplaceAll(createPath, "NAME", name)
			if d.IsDir() {
				return os.Mkdir(filepath.Join(name, createPath), 0o755)
			}

			b, err := scaffoldFiles.ReadFile(path)
			if err != nil {
				return err
			}
			tpl, err := template.New("tpl").Parse(string(b))
			if err != nil {
				return fmt.Errorf("%v: %w", path, err)
			}

			fname := strings.TrimSuffix(filepath.Join(name, createPath), ".tpl")
			f, err := os.Create(fname)
			if err != nil {
				return fmt.Errorf("%v: %w", path, err)
			}
			defer f.Close()

			buf := &bytes.Buffer{}
			err = tpl.Execute(buf, data)
			if err != nil {
				return fmt.Errorf("%v: %w", path, err)
			}

			b = buf.Bytes()
			if filepath.Ext(fname) == ".go" {
				b, err = format.Source(b)
				if err != nil {
					return fmt.Errorf("%v: %w", path, err)
				}
			}

			_, err = f.Write(b)
			if err != nil {
				return fmt.Errorf("%v: %w", path, err)
			}
			return nil
		})
		if err != nil {
			return err
		}

		_ = os.Mkdir(filepath.Join(name, "docs"), 0o755)

		fmt.Println(name, "successfully created.")
		fmt.Println()
		fmt.Println("Now, cd into", name, "and run the following commands")
		fmt.Println("  go mod tidy")
		fmt.Println("  make generate")
		return nil
	},
}

type ExtensionMarker struct {
	Types      []suffiksv1.Target `marker:"Targets,optional"`
	Validation bool               `marker:"Validation,optional"`
	Defaulting bool               `marker:"Defaulting,optional"`
	Always     bool               `marker:"Always,optional"`
}
