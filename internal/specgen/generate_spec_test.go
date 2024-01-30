package specgen_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/suffiks/suffiks/internal/specgen"
	"sigs.k8s.io/yaml"
)

func TestGenerate(t *testing.T) {
	bases := []string{
		"../../config/crd/bases/suffiks.com_applications.yaml",
	}

	for _, base := range bases {
		t.Run(filepath.Base(base), func(t *testing.T) {
			f, err := os.OpenFile(base, os.O_RDONLY, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			gen, err := specgen.FromYAML(f)
			if err != nil {
				t.Fatal(err)
			}

			pathParts := []string{".", "testdata", "generate", gen.Kind()}

			files, err := os.ReadDir(filepath.Join(pathParts...))
			if err != nil {
				t.Fatal(err)
			}

			hasError := false
			for _, file := range files {
				ok := t.Run(file.Name(), func(t *testing.T) {
					if hasError {
						t.Skip("Skipping due to previous error")
					}
					switch op(file.Name()) {
					case "add":
						metadata, jrm, err := parseFile(filepath.Join(append(pathParts, file.Name())...))
						if err != nil {
							t.Fatal(err)
						}
						if err := gen.Add(metadata["name"], jrm); err != nil {
							t.Fatal(err)
						}
					case "remove":
						metadata, _, err := parseFile(filepath.Join(append(pathParts, file.Name())...))
						if err != nil {
							t.Fatal(err)
						}
						if err := gen.Remove(metadata["name"]); err != nil {
							t.Fatal(err)
						}
					case "check":
						b, err := gen.Render()
						if err != nil {
							t.Fatal(err)
						}

						if os.Getenv("SUFFIKS_UPDATE_TESTDATA") == "1" {
							err := os.WriteFile(filepath.Join(append(pathParts, file.Name())...), b, 0o644)
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
							t.Fatal("+expected, -actual\n" + cmp.Diff(string(fb), string(b)))
						}
					}
				})
				if !ok {
					hasError = true
				}
			}
		})
	}
}

func parseFile(path string) (metadata map[string]string, spec json.RawMessage, err error) {
	body := struct {
		Metadata map[string]string `json:"metadata"`
		Spec     json.RawMessage   `json:"spec"`
	}{}
	switch filepath.Ext(path) {
	case ".json":
		f, err := os.OpenFile(path, os.O_RDONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&body); err != nil {
			return nil, nil, err
		}
	case ".yaml":
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		if err := yaml.Unmarshal(b, &body); err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, fmt.Errorf("File must be either .json or .yaml")
	}

	if body.Metadata == nil {
		if body.Metadata["name"] == "" {
			return nil, nil, fmt.Errorf("Metadata must contain a name")
		}
	}

	return body.Metadata, body.Spec, nil
}

func op(s string) string {
	name := filepath.Base(s)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	parts := strings.Split(name, "_")
	switch parts[len(parts)-1] {
	case "add", "remove", "check":
		return parts[len(parts)-1]
	default:
		panic("Filename must end with one of '_add', '_remove', or '_check'")
	}
}
