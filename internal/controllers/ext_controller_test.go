package controller

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"strings"
// 	"testing"

// 	"github.com/google/go-cmp/cmp"
// 	"github.com/google/go-cmp/cmp/cmpopts"
// 	"github.com/prometheus/client_golang/prometheus"
// 	suffiksv1 "github.com/suffiks/suffiks/apis/suffiks/v1"
// 	"github.com/suffiks/suffiks/extension/protogen"
// 	"google.golang.org/grpc"
// 	corev1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/runtime"
// 	"k8s.io/utils/pointer"
// )

// func TestExtensionControllerSync(t *testing.T) {
// 	t.Parallel()

// 	// The test have a single extension available, which manages the
// 	// extension field, with one option `enabled`.
// 	tests := map[string]struct {
// 		appSpec       map[string]any
// 		syncResponses []*protogen.Response
// 		syncErr       error
// 		requestError  error
// 		expectedErr   error
// 		result        *Result
// 	}{
// 		"no spec": {
// 			appSpec: map[string]any{},
// 			result:  &Result{Changeset: &Changeset{}},
// 		},
// 		"no extension": {
// 			appSpec: map[string]any{"someotherfield": "somevalue"},
// 			result:  &Result{Changeset: &Changeset{}},
// 		},
// 		"1 merge patch": {
// 			appSpec: map[string]any{"extension": map[string]any{"enabled": true}},
// 			syncResponses: []*protogen.Response{
// 				{
// 					OFResponse: &protogen.Response_MergePatch{
// 						MergePatch: []byte(`{"field": "value"}`),
// 					},
// 				},
// 			},
// 			result: &Result{
// 				Extensions: lockedList[string]{list: []string{"extension"}},
// 				Changeset: &Changeset{
// 					MergePatch: []byte(`{"field": "value"}`),
// 				},
// 			},
// 		},
// 		"2 merge patch": {
// 			appSpec: map[string]any{"extension": map[string]any{"enabled": true}},
// 			syncResponses: []*protogen.Response{
// 				{
// 					OFResponse: &protogen.Response_MergePatch{
// 						MergePatch: []byte(`{"field": "value"}`),
// 					},
// 				}, {
// 					OFResponse: &protogen.Response_MergePatch{
// 						MergePatch: []byte(`{"field2": "value2"}`),
// 					},
// 				},
// 			},
// 			result: &Result{
// 				Extensions: lockedList[string]{list: []string{"extension"}},
// 				Changeset: &Changeset{
// 					MergePatch: []byte(`{"field":"value","field2":"value2"}`),
// 				},
// 			},
// 		},
// 		"extension enabled": {
// 			appSpec: map[string]any{"extension": map[string]any{"enabled": true}},
// 			syncResponses: []*protogen.Response{
// 				{
// 					OFResponse: &protogen.Response_Env{
// 						Env: &protogen.KeyValue{
// 							Name:  "EXTENSION",
// 							Value: "SOMEVALUE",
// 						},
// 					},
// 				},
// 				{
// 					OFResponse: &protogen.Response_EnvFrom{
// 						EnvFrom: &protogen.EnvFrom{
// 							Name:     "cm",
// 							Optional: true,
// 							Type:     protogen.EnvFromType_CONFIGMAP,
// 						},
// 					},
// 				},
// 				{
// 					OFResponse: &protogen.Response_EnvFrom{
// 						EnvFrom: &protogen.EnvFrom{
// 							Name: "secret",
// 							Type: protogen.EnvFromType_SECRET,
// 						},
// 					},
// 				},
// 				{
// 					OFResponse: &protogen.Response_Label{
// 						Label: &protogen.KeyValue{
// 							Name:  "somelabel",
// 							Value: "labelvalue",
// 						},
// 					},
// 				},
// 				{
// 					OFResponse: &protogen.Response_Annotation{
// 						Annotation: &protogen.KeyValue{
// 							Name:  "someannotation",
// 							Value: "annotationvalue",
// 						},
// 					},
// 				},
// 			},
// 			result: &Result{
// 				Extensions: lockedList[string]{list: []string{"extension"}},
// 				Changeset: &Changeset{
// 					Environment: []corev1.EnvVar{{Name: "EXTENSION", Value: "SOMEVALUE"}},
// 					EnvFrom: []corev1.EnvFromSource{{
// 						ConfigMapRef: &corev1.ConfigMapEnvSource{
// 							LocalObjectReference: corev1.LocalObjectReference{
// 								Name: "cm",
// 							},
// 							Optional: pointer.Bool(true),
// 						},
// 					}, {
// 						SecretRef: &corev1.SecretEnvSource{
// 							LocalObjectReference: corev1.LocalObjectReference{
// 								Name: "secret",
// 							},
// 							Optional: pointer.Bool(false),
// 						},
// 					}},
// 					Labels:      map[string]string{"somelabel": "labelvalue"},
// 					Annotations: map[string]string{"someannotation": "annotationvalue"},
// 				},
// 			},
// 		},
// 	}

// 	for name, tt := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			m := NewExtensionController(mockManager{
// 				extension{
// 					sourceSpec: []string{"extension"},
// 					client: &mockGRPCClient{
// 						responses:     tt.syncResponses,
// 						responseError: tt.syncErr,
// 						requestError:  tt.requestError,
// 					},
// 				},
// 			})

// 			baseObject := &suffiksv1.Application{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      strings.ReplaceAll(name, " ", ""),
// 					Namespace: strings.ReplaceAll(name, " ", ""),
// 				},
// 				Spec: jsonAppSpec(t, tt.appSpec),
// 			}

// 			changeset, err := m.Sync(context.Background(), baseObject)
// 			if err != nil {
// 				t.Fatal(err)
// 			}

// 			if !cmp.Equal(tt.result, changeset, cmpopts.IgnoreUnexported(lockedList[string]{}, Result{}, Changeset{})) {
// 				t.Errorf(cmp.Diff(tt.result, changeset, cmpopts.IgnoreUnexported(lockedList[string]{}, Result{}, Changeset{})))
// 			}
// 		})
// 	}
// }

// func TestExtensionControllerDelete(t *testing.T) {
// 	t.Parallel()

// 	// The test have a single extension available, which manages the
// 	// extension field, with one option `enabled`.
// 	tests := map[string]struct {
// 		appSpec      map[string]any
// 		responseErr  error
// 		requestError error
// 		wantErr      bool
// 		numRequests  int
// 	}{
// 		"no spec": {
// 			appSpec: map[string]any{},
// 		},
// 		"no extension": {
// 			appSpec: map[string]any{"someotherfield": "somevalue"},
// 		},
// 		"extension enabled": {
// 			appSpec:     map[string]any{"extension": map[string]any{"enabled": true}},
// 			numRequests: 1,
// 		},
// 		"extension enabled, ext err": {
// 			appSpec:     map[string]any{"extension": map[string]any{"enabled": true}},
// 			responseErr: fmt.Errorf("some error"),
// 			wantErr:     true,
// 			numRequests: 1,
// 		},
// 	}

// 	for name, tt := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			mockClient := &mockGRPCClient{
// 				responseError: tt.responseErr,
// 				requestError:  tt.requestError,
// 			}

// 			m := NewExtensionController(mockManager{
// 				extension{
// 					sourceSpec: []string{"extension"},
// 					client:     mockClient,
// 				},
// 			})

// 			baseObject := &Application{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      strings.ReplaceAll(name, " ", ""),
// 					Namespace: strings.ReplaceAll(name, " ", ""),
// 				},
// 				Spec: jsonAppSpec(t, tt.appSpec),
// 			}

// 			err := m.Delete(context.Background(), baseObject)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 			if len(mockClient.requests) != tt.numRequests {
// 				t.Errorf("expected %v calls, got %v", tt.numRequests, len(mockClient.requests))
// 			}
// 		})
// 	}
// }

// func TestExtensionControllerDefault(t *testing.T) {
// 	t.Parallel()

// 	// The test have a single extension available, which manages the
// 	// extension field, with one option `enabled`.
// 	tests := map[string]struct {
// 		appSpec      map[string]any
// 		defaulting   bool
// 		responseErr  error
// 		requestError error
// 		wantErr      bool
// 		numRequests  int
// 	}{
// 		"no spec, no defaulting": {
// 			appSpec:    map[string]any{},
// 			defaulting: false,
// 		},
// 		"no spec": {
// 			appSpec:     map[string]any{},
// 			defaulting:  true,
// 			numRequests: 1,
// 		},
// 		"no extension": {
// 			appSpec:     map[string]any{"someotherfield": "somevalue"},
// 			defaulting:  true,
// 			numRequests: 1,
// 		},
// 		"extension enabled": {
// 			appSpec:     map[string]any{"extension": map[string]any{"enabled": true}},
// 			defaulting:  true,
// 			numRequests: 1,
// 		},
// 		"extension enabled, ext err": {
// 			appSpec:     map[string]any{"extension": map[string]any{"enabled": true}},
// 			defaulting:  true,
// 			responseErr: fmt.Errorf("some error"),
// 			wantErr:     true,
// 			numRequests: 1,
// 		},
// 	}

// 	for name, tt := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			mockClient := &mockGRPCClient{
// 				responseError: tt.responseErr,
// 				requestError:  tt.requestError,
// 			}

// 			m := NewExtensionController(mockManager{
// 				extension{
// 					sourceSpec: []string{"extension"},
// 					client:     mockClient,
// 					Extension: suffiksv1.Extension{
// 						Spec: suffiksv1.ExtensionSpec{
// 							Webhooks: suffiksv1.ExtensionWebhooks{
// 								Defaulting: tt.defaulting,
// 							},
// 						},
// 					},
// 				},
// 			})

// 			baseObject := &Application{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      strings.ReplaceAll(name, " ", ""),
// 					Namespace: strings.ReplaceAll(name, " ", ""),
// 				},
// 				Spec: jsonAppSpec(t, tt.appSpec),
// 			}

// 			_, err := m.Default(context.Background(), baseObject)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 			if len(mockClient.requests) != tt.numRequests {
// 				t.Errorf("expected %v calls, got %v", tt.numRequests, len(mockClient.requests))
// 			}
// 		})
// 	}
// }

// func TestExtensionControllerValidate(t *testing.T) {
// 	t.Parallel()

// 	// The test have a single extension available, which manages the
// 	// extension field, with one option `enabled`.
// 	tests := map[string]struct {
// 		appSpec      map[string]any
// 		response     *protogen.ValidationResponse
// 		responseErr  error
// 		requestError error
// 		validation   bool
// 		wantErr      bool
// 		hasRequest   bool
// 	}{
// 		"no spec, no validation": {
// 			appSpec: map[string]any{},
// 		},
// 		"no spec": {
// 			appSpec:    map[string]any{},
// 			validation: true,
// 			response:   &protogen.ValidationResponse{},
// 			hasRequest: true,
// 		},
// 		"no extension": {
// 			appSpec:    map[string]any{"someotherfield": "somevalue"},
// 			validation: true,
// 			response:   &protogen.ValidationResponse{},
// 			hasRequest: true,
// 		},
// 		"extension enabled": {
// 			appSpec:    map[string]any{"extension": map[string]any{"enabled": true}},
// 			validation: true,
// 			response:   &protogen.ValidationResponse{},
// 			hasRequest: true,
// 		},
// 		"extension enabled, ext err": {
// 			appSpec:     map[string]any{"extension": map[string]any{"enabled": true}},
// 			validation:  true,
// 			responseErr: fmt.Errorf("some error"),
// 			wantErr:     true,
// 			hasRequest:  true,
// 		},
// 	}

// 	for name, tt := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			mockClient := &mockGRPCClient{
// 				validationResponse: tt.response,
// 				responseError:      tt.responseErr,
// 				requestError:       tt.requestError,
// 			}

// 			m := NewExtensionController(mockManager{
// 				extension{
// 					sourceSpec: []string{"extension"},
// 					client:     mockClient,
// 					Extension: suffiksv1.Extension{
// 						Spec: suffiksv1.ExtensionSpec{
// 							Webhooks: suffiksv1.ExtensionWebhooks{
// 								Validation: tt.validation,
// 							},
// 						},
// 					},
// 				},
// 			})

// 			baseObject := &Application{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      strings.ReplaceAll(name, " ", ""),
// 					Namespace: strings.ReplaceAll(name, " ", ""),
// 				},
// 				Spec: jsonAppSpec(t, tt.appSpec),
// 			}

// 			err := m.Validate(context.Background(), protogen.ValidationType_CREATE, baseObject, nil)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 			if (mockClient.validationRequest != nil) != tt.hasRequest {
// 				t.Errorf("expected call %v, got call %v", tt.hasRequest, mockClient.validationRequest != nil)
// 			}
// 		})
// 	}
// }

// func TestExtensionControllerRegisterMetrics(t *testing.T) {
// 	t.Parallel()

// 	m := NewExtensionController(nil)
// 	reg := prometheus.NewRegistry()
// 	if err := m.RegisterMetrics(reg); err != nil {
// 		t.Fatal(err)
// 	}
// }

// func TestFieldErrsWrapper(t *testing.T) {
// 	t.Parallel()

// 	few := FieldErrsWrapper{}
// 	if few.Error() != "Field errors" {
// 		t.Error("Implement better test here")
// 	}
// }

// func TestContains(t *testing.T) {
// 	t.Parallel()

// 	type mytype int

// 	testContains(t, "ints found", []int{1, 2, 3, 4}, 3, true)
// 	testContains(t, "ints not found", []int{1, 2, 3, 4}, 10, false)

// 	testContains(t, "strings found", []string{"a", "2", "3", "4"}, "3", true)
// 	testContains(t, "strings not found", []string{"a", "2", "3", "4"}, "z", false)

// 	testContains(t, "custom type found", []mytype{1, 2, 3, 4}, 3, true)
// 	testContains(t, "custom type not found", []mytype{1, 2, 3, 4}, 10, false)
// }

// func testContains[T comparable](t *testing.T, name string, s []T, val T, expected bool) {
// 	t.Run(name, func(t *testing.T) {
// 		if contains(s, val) != expected {
// 			t.Errorf("%v != %v, expected: %v", s, val, expected)
// 		}
// 	})
// }

// type mockManager []extension

// func (m mockManager) ExtensionsFor(kind string) []extension {
// 	return m
// }

// func jsonAppSpec(t *testing.T, spec map[string]any) runtime.RawExtension {
// 	re := runtime.RawExtension{}
// 	var err error
// 	re.Raw, err = json.Marshal(spec)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	return re
// }

// type mockGRPCClient struct {
// 	requests           []*protogen.SyncRequest
// 	responses          []*protogen.Response
// 	validationRequest  *protogen.ValidationRequest
// 	validationResponse *protogen.ValidationResponse
// 	defaultResponse    *protogen.DefaultResponse
// 	responseError      error

// 	requestError error
// }

// func (m *mockGRPCClient) Sync(ctx context.Context, in *protogen.SyncRequest, opts ...grpc.CallOption) (protogen.Extension_SyncClient, error) {
// 	m.requests = append(m.requests, in)
// 	return &mockStreamClient{
// 		responses: m.responses,
// 		err:       m.responseError,
// 	}, m.requestError
// }

// func (m *mockGRPCClient) Delete(ctx context.Context, in *protogen.SyncRequest, opts ...grpc.CallOption) (protogen.Extension_DeleteClient, error) {
// 	m.requests = append(m.requests, in)
// 	return &mockStreamClient{
// 		responses: m.responses,
// 		err:       m.responseError,
// 	}, m.requestError
// }

// func (m *mockGRPCClient) Default(ctx context.Context, in *protogen.SyncRequest, opts ...grpc.CallOption) (*protogen.DefaultResponse, error) {
// 	m.requests = append(m.requests, in)
// 	return m.defaultResponse, m.responseError
// }

// func (m *mockGRPCClient) Validate(ctx context.Context, in *protogen.ValidationRequest, opts ...grpc.CallOption) (*protogen.ValidationResponse, error) {
// 	m.validationRequest = in
// 	return m.validationResponse, m.responseError
// }

// func (m *mockGRPCClient) Documentation(ctx context.Context, in *protogen.DocumentationRequest, opts ...grpc.CallOption) (*protogen.DocumentationResponse, error) {
// 	return nil, nil
// }

// type mockStreamClient struct {
// 	grpc.ClientStream

// 	responses []*protogen.Response
// 	err       error
// }

// func (m *mockStreamClient) Recv() (*protogen.Response, error) {
// 	if m.err != nil {
// 		return nil, m.err
// 	}

// 	if len(m.responses) == 0 {
// 		return nil, io.EOF
// 	}
// 	r := m.responses[0]
// 	m.responses = m.responses[1:]
// 	return r, nil
// }
