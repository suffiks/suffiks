package extension

import (
	"reflect"
	"testing"
	"unicode"

	"github.com/google/go-cmp/cmp"
	"github.com/suffiks/suffiks/extension/protogen"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestChangeset_Add(t *testing.T) {
	tests := map[string]struct {
		responses []*protogen.Response
		expected  *Changeset
	}{
		"empty": {
			responses: []*protogen.Response{},
			expected:  &Changeset{},
		},
		"env": {
			responses: []*protogen.Response{
				respKeyValue("foo", "bar"),
			},
			expected: &Changeset{
				environment: []v1.EnvVar{
					{
						Name:  "foo",
						Value: "bar",
					},
				},
			},
		},
		"all": {
			responses: []*protogen.Response{
				respKeyValue("foo", "bar"),
				respLabel("suffiks.io/some-label", "some-value"),
				respAnnotation("suffiks.io/some-annotation", "some-value"),
				respEnvFromSecret("secret", false),
				respEnvFromConfigMap("configmap", true),
				respInitContainer(container("init", "init")),
				respContainer(container("main", "main")),
			},
			expected: &Changeset{
				environment: []v1.EnvVar{{Name: "foo", Value: "bar"}},
				labels:      map[string]string{"suffiks.io/some-label": "some-value"},
				annotations: map[string]string{"suffiks.io/some-annotation": "some-value"},
				envFrom: []v1.EnvFromSource{
					{SecretRef: &v1.SecretEnvSource{Optional: ptr.To(false), LocalObjectReference: v1.LocalObjectReference{Name: "secret"}}},
					{ConfigMapRef: &v1.ConfigMapEnvSource{Optional: ptr.To(true), LocalObjectReference: v1.LocalObjectReference{Name: "configmap"}}},
				},
				sidecars:       []v1.Container{{Name: "main", Image: "main"}},
				initContainers: []v1.Container{{Name: "init", Image: "init"}},
			},
		},
		"multiple of all, with possible duplicates": {
			responses: []*protogen.Response{
				respKeyValue("foo", "bar"),
				respKeyValue("foo", "bar"),
				respKeyValue("otherfoo", "otherbar"),
				respLabel("suffiks.io/some-label", "some-value"),
				respLabel("suffiks.io/some-label", "some-value"),
				respLabel("other.suffiks.io/some-label", "some-other"),
				respAnnotation("suffiks.io/some-annotation", "some-value"),
				respAnnotation("suffiks.io/some-annotation", "some-value"),
				respAnnotation("other.suffiks.io/some-annotation", "some-other"),
				respEnvFromSecret("secret", false),
				respEnvFromSecret("secret", false),
				respEnvFromSecret("othersecret", true),
				respEnvFromConfigMap("configmap", true),
				respEnvFromConfigMap("configmap", true),
				respEnvFromConfigMap("otherconfigmap", false),
				respInitContainer(container("init", "init")),
				respInitContainer(container("init", "init")),
				respInitContainer(container("otherinit", "otherinit")),
				respContainer(container("main", "main")),
				respContainer(container("main", "main")),
				respContainer(container("othermain", "othermain")),
			},
			expected: &Changeset{
				environment: []v1.EnvVar{{Name: "foo", Value: "bar"}, {Name: "foo", Value: "bar"}, {Name: "otherfoo", Value: "otherbar"}},
				labels:      map[string]string{"suffiks.io/some-label": "some-value", "other.suffiks.io/some-label": "some-other"},
				annotations: map[string]string{"suffiks.io/some-annotation": "some-value", "other.suffiks.io/some-annotation": "some-other"},
				envFrom: []v1.EnvFromSource{
					{SecretRef: &v1.SecretEnvSource{Optional: ptr.To(false), LocalObjectReference: v1.LocalObjectReference{Name: "secret"}}},
					{SecretRef: &v1.SecretEnvSource{Optional: ptr.To(false), LocalObjectReference: v1.LocalObjectReference{Name: "secret"}}},
					{SecretRef: &v1.SecretEnvSource{Optional: ptr.To(true), LocalObjectReference: v1.LocalObjectReference{Name: "othersecret"}}},
					{ConfigMapRef: &v1.ConfigMapEnvSource{Optional: ptr.To(true), LocalObjectReference: v1.LocalObjectReference{Name: "configmap"}}},
					{ConfigMapRef: &v1.ConfigMapEnvSource{Optional: ptr.To(true), LocalObjectReference: v1.LocalObjectReference{Name: "configmap"}}},
					{ConfigMapRef: &v1.ConfigMapEnvSource{Optional: ptr.To(false), LocalObjectReference: v1.LocalObjectReference{Name: "otherconfigmap"}}},
				},
				sidecars:       []v1.Container{{Name: "main", Image: "main"}, {Name: "main", Image: "main"}, {Name: "othermain", Image: "othermain"}},
				initContainers: []v1.Container{{Name: "init", Image: "init"}, {Name: "init", Image: "init"}, {Name: "otherinit", Image: "otherinit"}},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			c := &Changeset{}
			for _, resp := range tc.responses {
				if err := c.Add(resp); err != nil {
					t.Fatal(err)
				}
			}

			opts := cmp.Options{
				cmp.Exporter(func(t reflect.Type) bool {
					return unicode.IsUpper(rune(t.Name()[0]))
				}),
			}
			if diff := cmp.Diff(tc.expected, c, opts...); diff != "" {
				t.Errorf("unexpected changeset (-want +got):\n%s", diff)
			}
		})
	}
}

func respKeyValue(name, value string) *protogen.Response {
	return &protogen.Response{
		OFResponse: &protogen.Response_Env{
			Env: &protogen.KeyValue{
				Name:  name,
				Value: value,
			},
		},
	}
}

func respLabel(name, value string) *protogen.Response {
	return &protogen.Response{
		OFResponse: &protogen.Response_Label{
			Label: &protogen.KeyValue{
				Name:  name,
				Value: value,
			},
		},
	}
}

func respAnnotation(name, value string) *protogen.Response {
	return &protogen.Response{
		OFResponse: &protogen.Response_Annotation{
			Annotation: &protogen.KeyValue{
				Name:  name,
				Value: value,
			},
		},
	}
}

func respEnvFromSecret(name string, optional bool) *protogen.Response {
	return &protogen.Response{
		OFResponse: &protogen.Response_EnvFrom{
			EnvFrom: &protogen.EnvFrom{
				Name:     name,
				Optional: optional,
				Type:     protogen.EnvFromType_SECRET,
			},
		},
	}
}

func respEnvFromConfigMap(name string, optional bool) *protogen.Response {
	return &protogen.Response{
		OFResponse: &protogen.Response_EnvFrom{
			EnvFrom: &protogen.EnvFrom{
				Name:     name,
				Optional: optional,
				Type:     protogen.EnvFromType_CONFIGMAP,
			},
		},
	}
}

func respInitContainer(c *protogen.Container) *protogen.Response {
	return &protogen.Response{
		OFResponse: &protogen.Response_InitContainer{
			InitContainer: c,
		},
	}
}

func respContainer(c *protogen.Container) *protogen.Response {
	return &protogen.Response{
		OFResponse: &protogen.Response_Container{
			Container: c,
		},
	}
}

func container(name, image string) *protogen.Container {
	return &protogen.Container{
		Name:  name,
		Image: image,
	}
}
