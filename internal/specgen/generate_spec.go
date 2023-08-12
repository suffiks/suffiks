package specgen

import (
	"encoding/json"
	"fmt"
	"io"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

type AlreadyDefinedError struct {
	Name string
}

func (a *AlreadyDefinedError) Error() string {
	return fmt.Sprintf("%v already defined", a.Name)
}

func (a *AlreadyDefinedError) Is(target error) bool {
	_, ok := target.(*AlreadyDefinedError)
	return ok
}

type Generator struct {
	schema *apiextv1.JSONSchemaProps
	crd    apiextv1.CustomResourceDefinition
}

func FromYAML(r io.Reader) (*Generator, error) {
	in := &apiextv1.CustomResourceDefinition{}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	for i := 0; i < 2; i++ {
		in = &apiextv1.CustomResourceDefinition{}
		if err := yaml.Unmarshal(b, in); err != nil {
			return nil, err
		}
		if len(in.Spec.Versions) > 0 {
			break
		}
	}

	return &Generator{
		schema: in.Spec.Versions[0].Schema.OpenAPIV3Schema,
		crd:    *in,
	}, nil
}

func (g *Generator) Kind() string {
	return g.crd.Spec.Names.Kind
}

func (g *Generator) Render() ([]byte, error) {
	return json.MarshalIndent(g.schema, "", "  ")
}

func (g *Generator) Schema() *apiextv1.JSONSchemaProps {
	return g.schema.DeepCopy()
}

func (g *Generator) Add(ext json.RawMessage) error {
	in := &apiextv1.JSONSchemaProps{}
	if err := json.Unmarshal(ext, in); err != nil {
		return fmt.Errorf("Generator.Add: json umarshal error: %w", err)
	}

	spec := g.schema.Properties["spec"]
	if err := g.add(&spec, in.Properties, ""); err != nil {
		return err
	}
	g.schema.Properties["spec"] = spec
	return nil
}

func (g *Generator) add(schema *apiextv1.JSONSchemaProps, properties map[string]apiextv1.JSONSchemaProps, path ...string) error {
	if properties == nil {
		return nil
	}

	for name, val := range properties {
		if _, ok := schema.Properties[name]; ok {
			if val.Type == "object" {
				inner := schema.Properties[name]
				if err := g.add(&inner, val.Properties, append(path, name)...); err != nil {
					return err
				}

				schema.Properties[name] = inner
				continue
			}

			return &AlreadyDefinedError{Name: name}
		}

		if schema.Properties == nil {
			schema.Properties = make(map[string]apiextv1.JSONSchemaProps)
		}
		schema.Properties[name] = val
	}

	return nil
}

func (g *Generator) Remove(ext json.RawMessage) error {
	in := &apiextv1.JSONSchemaProps{}
	if err := json.Unmarshal(ext, in); err != nil {
		return err
	}

	spec := g.schema.Properties["spec"]
	if err := g.remove(&spec, in.Properties, ""); err != nil {
		return err
	}
	g.schema.Properties["spec"] = spec
	return nil
}

func (g *Generator) remove(schema *apiextv1.JSONSchemaProps, properties map[string]apiextv1.JSONSchemaProps, path ...string) error {
	if properties == nil {
		return nil
	}

	for name, val := range properties {
		if _, ok := schema.Properties[name]; !ok {
			continue
		}

		if val.Type == "object" {
			inner := schema.Properties[name]
			if err := g.remove(&inner, val.Properties, append(path, name)...); err != nil {
				return err
			}
			continue
		}

		delete(schema.Properties, name)
	}

	for name, p := range schema.Properties {
		if p.Type == "object" && len(p.Properties) == 0 && len(path) > 1 {
			delete(schema.Properties, name)
		}
	}

	return nil
}
