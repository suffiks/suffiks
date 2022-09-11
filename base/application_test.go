package base

// func TestApplication_GetSpec(t *testing.T) {
// 	tests := map[string]struct {
// 		job  *Application
// 		want []byte
// 	}{
// 		"nil job": {
// 			job:  nil,
// 			want: nil,
// 		},
// 		"empty job": {
// 			job:  &Application{},
// 			want: nil,
// 		},
// 		"job with spec": {
// 			job: &Application{
// 				Spec: runtime.RawExtension{
// 					Raw: []byte(`{"foo":"bar"}`),
// 				},
// 			},
// 			want: []byte(`{"foo":"bar"}`),
// 		},
// 	}
// 	for name, tt := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			if got := tt.job.GetSpec(); !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("Application.GetSpec() = %v, want %v", string(got), string(tt.want))
// 			}
// 		})
// 	}
// }

// func TestApplication_WellKnownSpec(t *testing.T) {
// 	tests := map[string]struct {
// 		job     *Application
// 		want    suffiksv1.ApplicationSpec
// 		wantErr bool
// 	}{
// 		"nil job": {
// 			job:     nil,
// 			want:    suffiksv1.ApplicationSpec{},
// 			wantErr: false,
// 		},
// 		"empty job": {
// 			job:     &Application{},
// 			want:    suffiksv1.ApplicationSpec{},
// 			wantErr: true,
// 		},
// 		"job with spec": {
// 			job: &Application{
// 				Spec: runtime.RawExtension{
// 					Raw: []byte(`{"image": "hello-world", "port": 1337, "command": ["sleep"], "env": [{"name": "FOO", "value": "bar"}], "envFrom": [{"configMap": "cm"}]}`),
// 				},
// 			},
// 			want: suffiksv1.ApplicationSpec{
// 				Image:   "hello-world",
// 				Port:    1337,
// 				Command: []string{"sleep"},
// 				Env:     suffiksv1.EnvVars{{Name: "FOO", Value: "bar"}},
// 				EnvFrom: []suffiksv1.EnvFrom{{ConfigMap: "cm"}},
// 			},
// 			wantErr: false,
// 		},
// 	}
// 	for name, tt := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			got, err := tt.job.WellKnownSpec()
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("Application.WellKnownSpec() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if !cmp.Equal(tt.want, got) {
// 				t.Errorf(cmp.Diff(tt.want, got))
// 			}
// 		})
// 	}
// }

// func TestApplication_Hash(t *testing.T) {
// 	tests := map[string]struct {
// 		job     *Application
// 		want    string
// 		wantErr bool
// 	}{
// 		"nil job": {
// 			job:     nil,
// 			want:    "",
// 			wantErr: true,
// 		},
// 		"empty job": {
// 			job:     &Application{},
// 			want:    "cbf29ce484222325",
// 			wantErr: false,
// 		},
// 		"job with spec": {
// 			job: &Application{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      "name",
// 					Namespace: "namespace",
// 				},
// 				Spec: runtime.RawExtension{
// 					Raw: []byte(`{"image": "hello-world", "port": 1337, "command": ["sleep"], "env": [{"name": "FOO", "value": "bar"}], "envFrom": [{"configMap": "cm"}]}`),
// 				},
// 			},
// 			want: "f15934685ba44026",
// 		},
// 	}
// 	for name, tt := range tests {
// 		t.Run(name, func(t *testing.T) {
// 			got, err := tt.job.Hash()
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("Application.Hash() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if got != tt.want {
// 				t.Errorf("Application.Hash() = %q, want %q", got, tt.want)
// 			}
// 		})
// 	}
// }
