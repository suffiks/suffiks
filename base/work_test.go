package base

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	suffiksv1 "github.com/suffiks/suffiks/apis/suffiks/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestWork_GetSpec(t *testing.T) {
	tests := map[string]struct {
		job  *Work
		want []byte
	}{
		"nil job": {
			job:  nil,
			want: nil,
		},
		"empty job": {
			job:  &Work{},
			want: nil,
		},
		"job with spec": {
			job: &Work{
				Spec: runtime.RawExtension{
					Raw: []byte(`{"foo":"bar"}`),
				},
			},
			want: []byte(`{"foo":"bar"}`),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := tt.job.GetSpec(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Work.GetSpec() = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}

func TestWork_WellKnownSpec(t *testing.T) {
	tests := map[string]struct {
		job     *Work
		want    suffiksv1.WorkSpec
		wantErr bool
	}{
		"nil job": {
			job:     nil,
			want:    suffiksv1.WorkSpec{},
			wantErr: false,
		},
		"empty job": {
			job:     &Work{},
			want:    suffiksv1.WorkSpec{},
			wantErr: true,
		},
		"job with spec": {
			job: &Work{
				Spec: runtime.RawExtension{
					Raw: []byte(`{"image": "hello-world", "restartPolicy": "Never", "schedule": "* * * * *", "command": ["sleep"], "env": [{"name": "FOO", "value": "bar"}], "envFrom": [{"configMap": "cm"}]}`),
				},
			},
			want: suffiksv1.WorkSpec{
				Image:         "hello-world",
				RestartPolicy: "Never",
				Schedule:      "* * * * *",
				Command:       []string{"sleep"},
				Env:           suffiksv1.EnvVars{{Name: "FOO", Value: "bar"}},
				EnvFrom:       []suffiksv1.EnvFrom{{ConfigMap: "cm"}},
			},
			wantErr: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.job.WellKnownSpec()
			if (err != nil) != tt.wantErr {
				t.Errorf("Work.WellKnownSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !cmp.Equal(tt.want, got) {
				t.Errorf(cmp.Diff(tt.want, got))
			}
		})
	}
}

func TestWork_Hash(t *testing.T) {
	tests := map[string]struct {
		job     *Work
		want    string
		wantErr bool
	}{
		"nil job": {
			job:     nil,
			want:    "",
			wantErr: true,
		},
		"empty job": {
			job:     &Work{},
			want:    "cbf29ce484222325",
			wantErr: false,
		},
		"job with spec": {
			job: &Work{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: runtime.RawExtension{
					Raw: []byte(`{"image": "hello-world", "restartPolicy": "Never", "schedule": "* * * * *", "command": ["sleep"], "env": [{"name": "FOO", "value": "bar"}], "envFrom": [{"configMap": "cm"}]}`),
				},
			},
			want: "b193ad5ede1fafc4",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.job.Hash()
			if (err != nil) != tt.wantErr {
				t.Errorf("Work.Hash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Work.Hash() = %q, want %q", got, tt.want)
			}
		})
	}
}
