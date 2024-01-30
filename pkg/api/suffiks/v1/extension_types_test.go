package v1

import "testing"

func TestExtensionGRPCController_Target(t *testing.T) {
	tests := map[string]struct {
		c    *ExtensionGRPCController
		want string
	}{
		"empty": {
			c:    &ExtensionGRPCController{},
			want: ".:0",
		},
		"valid": {
			c: &ExtensionGRPCController{
				Namespace: "foo",
				Service:   "bar",
				Port:      1234,
			},
			want: "bar.foo:1234",
		},

		"without port": {
			c: &ExtensionGRPCController{
				Namespace: "foo",
				Service:   "bar",
			},
			want: "bar.foo:0",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.c.Target()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtensionWASIControllerResource_String(t *testing.T) {
	tests := map[string]struct {
		r    ExtensionWASIControllerResource
		want string
	}{
		"empty": {
			r:    ExtensionWASIControllerResource{},
			want: "//",
		},

		"valid": {
			r: ExtensionWASIControllerResource{
				Group:    "foo",
				Version:  "bar",
				Resource: "baz",
			},
			want: "foo/bar/baz",
		},

		"valid with methods": {
			r: ExtensionWASIControllerResource{
				Group:    "foo",
				Version:  "bar",
				Resource: "baz",
				Methods:  []Method{"GET", "POST"},
			},
			want: "foo/bar/baz",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.r.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtensionWASIController_ImageTag(t *testing.T) {
	tests := map[string]struct {
		c    *ExtensionWASIController
		want string
	}{
		"empty": {
			c:    &ExtensionWASIController{},
			want: ":",
		},

		"valid": {
			c: &ExtensionWASIController{
				Image: "foo",
				Tag:   "bar",
			},
			want: "foo:bar",
		},

		"without tag": {
			c: &ExtensionWASIController{
				Image: "foo",
			},
			want: "foo:",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.c.ImageTag()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
