package runtime_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suffiks/suffiks/base/runtime"
	"sigs.k8s.io/yaml"
)

func TestGenerate(t *testing.T) {
	bases := []string{
		"../../config/crd/bases/suffiks.com_applications.yaml",
	}

	for _, base := range bases {
		t.Run(filepath.Base(base), func(t *testing.T) {
			f, err := os.OpenFile(base, os.O_RDONLY, 0644)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			gen, err := runtime.FromYAML(f)
			if err != nil {
				t.Fatal(err)
			}

			pathParts := []string{".", "testdata", "generate", gen.Kind()}

			files, err := os.ReadDir(filepath.Join(pathParts...))
			if err != nil {
				t.Fatal(err)
			}

			for _, file := range files {
				t.Run(file.Name(), func(t *testing.T) {
					switch op(file.Name()) {
					case "add":
						jrm, err := parseFile(filepath.Join(append(pathParts, file.Name())...))
						if err != nil {
							t.Fatal(err)
						}
						if err := gen.Add(jrm); err != nil {
							t.Fatal(err)
						}
					case "remove":
						jrm, err := parseFile(filepath.Join(append(pathParts, file.Name())...))
						if err != nil {
							t.Fatal(err)
						}
						if err := gen.Remove(jrm); err != nil {
							t.Fatal(err)
						}
					case "check":
						b, err := gen.Render()
						if err != nil {
							t.Fatal(err)
						}

						if os.Getenv("SUFFIKS_UPDATE_TESTDATA") == "1" {
							err := os.WriteFile(filepath.Join(append(pathParts, file.Name())...), b, 0644)
							if err != nil {
								t.Fatal(err)
							}
							break
						}

						fb, err := os.ReadFile(filepath.Join(append(pathParts, file.Name())...))
						if err != nil {
							t.Fatal(err)
						}

						fb = bytes.TrimSpace(fb)
						if !cmp.Equal(string(b), string(fb)) {
							t.Fatal(cmp.Diff(string(b), string(fb)))
						}
					}
				})
			}
		})
	}
}

func parseFile(path string) (json.RawMessage, error) {
	var jrm json.RawMessage
	switch filepath.Ext(path) {
	case ".json":
		f, err := os.OpenFile(path, os.O_RDONLY, 0644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&jrm); err != nil {
			return nil, err
		}
	case ".yaml":
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(b, &jrm); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("File must be either .json or .yaml")
	}

	return jrm, nil
}

func op(s string) string {
	name := filepath.Base(s)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	parts := strings.Split(name, "_")
	switch parts[len(parts)-1] {
	case "add", "remove", "check":
		return parts[len(parts)-1]
	default:
		panic("Filename must end with one of '_add', '_remove' or '_check'")
	}
}
