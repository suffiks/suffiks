package v1

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWork_GetSpec(t *testing.T) {
	tests := map[string]struct {
		work *Work
		want []byte
	}{
		"nil": {
			work: nil,
			want: nil,
		},

		"empty": {
			work: &Work{},
			want: []byte(`{"image":""}`),
		},

		"simple": {
			work: &Work{
				Spec: WorkSpec{
					Image: "foo",
				},
			},
			want: []byte(`{"image":"foo"}`),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.work.GetSpec()
			if !cmp.Equal(tt.want, got) {
				if got == nil && tt.want != nil {
					t.Error("got nil, want non-nil")
				} else {
					t.Errorf("-want +got\n%s", cmp.Diff(string(tt.want), string(got)))
				}
			}
		})
	}
}
