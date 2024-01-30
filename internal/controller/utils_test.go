package controller

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMergeMaps(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		maps []map[string]string
		want map[string]string
	}{
		"empty": {
			maps: []map[string]string{},
			want: map[string]string{},
		},
		"one": {
			maps: []map[string]string{
				{
					"foo": "bar",
				},
			},
			want: map[string]string{
				"foo": "bar",
			},
		},
		"two": {
			maps: []map[string]string{
				{
					"foo": "bar",
				},
				{
					"bar": "baz",
				},
			},
			want: map[string]string{
				"foo": "bar",
				"bar": "baz",
			},
		},
		"overwrite": {
			maps: []map[string]string{
				{
					"foo": "bar",
				},
				{
					"foo": "baz",
				},
			},
			want: map[string]string{
				"foo": "baz",
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := mergeMaps(tc.maps...)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("-want +got\n%s", diff)
			}
		})
	}
}
