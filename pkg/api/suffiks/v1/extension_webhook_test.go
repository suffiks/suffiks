package v1

import (
	"context"
	"encoding/json"
	"testing"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func mustJSON(v any) runtime.RawExtension {
	r := runtime.RawExtension{}
	r.Raw, _ = json.Marshal(v)
	return r
}

func TestExtension_ValidateCreate(t *testing.T) {
	tests := map[string]struct {
		ext     *Extension
		wantErr bool
	}{
		"valid grpc": {
			ext: &Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: ExtensionSpec{
					Targets: []Target{"Application", "Work"},
					Controller: ControllerSpec{
						GRPC: &ExtensionGRPCController{
							Namespace: "somenamespace",
							Service:   "servicename",
						},
					},
					Webhooks: ExtensionWebhooks{
						Validation: true,
						Defaulting: true,
					},
					Always: true,
					OpenAPIV3Schema: mustJSON(apiextv1.JSONSchemaProps{
						Type: "object",
					}),
				},
			},
		},
		"multiple props": {
			ext: &Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "multiprops",
					Namespace: "test",
				},
				Spec: ExtensionSpec{
					Targets: []Target{"Application"},
					Controller: ControllerSpec{
						GRPC: &ExtensionGRPCController{
							Service: "servicename",
						},
					},
					OpenAPIV3Schema: mustJSON(apiextv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiextv1.JSONSchemaProps{
							"foo": {
								Type: "string",
							},
							"bar": {
								Type: "string",
							},
						},
					}),
				},
			},
		},

		"valid wasi": {
			ext: &Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: ExtensionSpec{
					Targets: []Target{"Application", "Work"},
					Controller: ControllerSpec{
						WASI: &ExtensionWASIController{
							Image: "somenamespace/somerepo",
							Tag:   "sometag",
						},
					},
					Webhooks: ExtensionWebhooks{
						Validation: true,
						Defaulting: true,
					},
					Always: true,
					OpenAPIV3Schema: mustJSON(apiextv1.JSONSchemaProps{
						Type: "object",
					}),
				},
			},
		},

		"invalid target": {
			ext: &Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "multiprops",
					Namespace: "test",
				},
				Spec: ExtensionSpec{
					Targets: []Target{"Invalid"},
					Controller: ControllerSpec{
						GRPC: &ExtensionGRPCController{
							Service: "servicename",
						},
					},
					OpenAPIV3Schema: mustJSON(apiextv1.JSONSchemaProps{
						Type: "object",
					}),
				},
			},
			wantErr: true,
		},
		"invalid spec (invalid json)": {
			ext: &Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "multiprops",
					Namespace: "test",
				},
				Spec: ExtensionSpec{
					Targets: []Target{"Invalid"},
					Controller: ControllerSpec{
						GRPC: &ExtensionGRPCController{
							Service: "servicename",
						},
					},
					OpenAPIV3Schema: runtime.RawExtension{},
				},
			},
			wantErr: true,
		},
		"invalid spec (not object)": {
			ext: &Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "multiprops",
					Namespace: "test",
				},
				Spec: ExtensionSpec{
					Targets: []Target{"Invalid"},
					Controller: ControllerSpec{
						GRPC: &ExtensionGRPCController{
							Service: "servicename",
						},
					},
					OpenAPIV3Schema: mustJSON(apiextv1.JSONSchemaProps{
						Type: "string",
					}),
				},
			},
			wantErr: true,
		},

		"empty wasi image": {
			ext: &Extension{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nowasiimg",
					Namespace: "test",
				},
				Spec: ExtensionSpec{
					Targets: []Target{"Application"},
					Controller: ControllerSpec{
						WASI: &ExtensionWASIController{},
					},
					OpenAPIV3Schema: mustJSON(apiextv1.JSONSchemaProps{
						Type: "string",
					}),
				},
			},
			wantErr: true,
		},
	}

	opts := []valOpts{
		withOciGetter(func(ctx context.Context, image, tag string) (map[string][]byte, error) {
			return nil, nil
		}),
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if err := tt.ext.validateExtension(opts...); (err != nil) != tt.wantErr {
				t.Errorf("Extension.ValidateCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	if _, err := (&Extension{}).ValidateDelete(); err != nil {
		t.Errorf("Extension.ValidateDelete() error = %v", err)
	}
}
