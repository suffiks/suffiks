package v1

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestApplication_GetSpec(t *testing.T) {
	tests := map[string]struct {
		app  *Application
		want string
	}{
		"nil": {
			app:  nil,
			want: "",
		},

		"empty": {
			app:  &Application{},
			want: `{"image":""}`,
		},

		"well-known": {
			app: &Application{
				Spec: ApplicationSpec{
					Image: "foo",
				},
			},
			want: `{"image":"foo"}`,
		},

		"custom": {
			app: &Application{
				Spec: ApplicationSpec{
					Image: "foo",
					Rest: unstructured.Unstructured{
						Object: map[string]interface{}{
							"foo": "bar",
						},
					},
				},
			},
			want: `{"image":"foo","foo":"bar"}`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := string(tt.app.GetSpec())
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("-want +got\n%s", diff)
			}
		})
	}
}

func TestApplication_WellKnownSpec(t *testing.T) {
	tests := map[string]struct {
		app  *Application
		want *ApplicationSpec
	}{
		"nil": {
			app:  nil,
			want: &ApplicationSpec{},
		},

		"empty": {
			app:  &Application{},
			want: &ApplicationSpec{},
		},

		"well-known": {
			app: &Application{
				Spec: ApplicationSpec{
					Image: "foo",
				},
			},
			want: &ApplicationSpec{
				Image: "foo",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.app.WellKnownSpec()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.want, &got); diff != "" {
				t.Errorf("-want +got\n%s", diff)
			}
		})
	}
}

func TestApplication_Hash(t *testing.T) {
	tests := map[string]struct {
		app  *Application
		want string
		err  error
	}{
		"nil": {
			app:  nil,
			want: "",
			err:  fmt.Errorf("unable to hash nil application"),
		},

		"empty": {
			app:  &Application{},
			want: "cbf29ce484222325",
		},

		"well-known": {
			app: &Application{
				Spec: ApplicationSpec{
					Image: "foo",
				},
			},
			want: "9b6d14b6248dee5b",
		},

		"custom": {
			app: &Application{
				Spec: ApplicationSpec{
					Image: "foo",
					Rest: unstructured.Unstructured{
						Object: map[string]interface{}{
							"foo": "bar",
						},
					},
				},
			},
			want: "f99e84f3d5a5a676",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.app.Hash()
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("-want +got\n%s", diff)
			}
		})
	}
}

func TestApplicationSpec_UnmarshalJSON(t *testing.T) {
	tests := map[string]struct {
		data []byte
		want ApplicationSpec
	}{
		"empty": {
			data: []byte(`{}`),
			want: ApplicationSpec{},
		},

		"simple": {
			data: []byte(`{"image":"foo"}`),
			want: ApplicationSpec{
				Image: "foo",
			},
		},

		"custom": {
			data: []byte(`{"image":"foo","foo":"bar"}`),
			want: ApplicationSpec{
				Image: "foo",
				Rest: unstructured.Unstructured{
					Object: map[string]interface{}{
						"foo": "bar",
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var got ApplicationSpec
			if err := got.UnmarshalJSON(tt.data); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("-want +got\n%s", diff)
			}
		})
	}
}

func TestApplicationSpec_MarshalJSON(t *testing.T) {
	tests := map[string]struct {
		spec ApplicationSpec
		want []byte
	}{
		"empty": {
			spec: ApplicationSpec{},
			want: []byte(`{"image":""}`),
		},

		"simple": {
			spec: ApplicationSpec{
				Image: "foo",
			},
			want: []byte(`{"image":"foo"}`),
		},

		"custom": {
			spec: ApplicationSpec{
				Image: "foo",
				Rest: unstructured.Unstructured{
					Object: map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			want: []byte(`{"image":"foo","foo":"bar"}`),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.spec.MarshalJSON()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(string(tt.want), string(got)); diff != "" {
				t.Errorf("-want +got\n%s", diff)
			}
		})
	}
}
