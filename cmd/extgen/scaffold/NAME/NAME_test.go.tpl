package {{ .Name }}

import (
	"os"
	"testing"

	v1 "github.com/suffiks/suffiks/api/suffiks/v1"
	"github.com/suffiks/suffiks/base"
	"github.com/suffiks/suffiks/extension"
	"github.com/suffiks/suffiks/extension/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var _ extension.Extension[*{{.GoName}}] = &Extension{}
{{- if .Validation }}
var _ extension.ValidatableExtension[*{{.GoName}}] = &Extension{}
{{- end}}
{{- if .Defaulting }}
var _ extension.DefaultableExtension[*{{.GoName}}] = &Extension{}
{{end}}


func Test{{.GoName}}(t *testing.T) {
	app := &v1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-app",
			Namespace: "mynamespace",
		},
		Spec: testutil.AppSpec(nil, map[string]any{
			"{{.Name}}": map[string]any{
				"extraEnv": map[string]any{
					"name":  "EXTRA_ENV_NAME",
					"value": "EXTRA_ENV_VALUE",
				},
			},
		}),
	}

	tests := []testutil.TestCase{
		testutil.SyncTest{
			Name:   "create application",
			Object: app,
			Changeset: &base.Changeset{
				Environment: []corev1.EnvVar{
					{Name: "EXTRA_ENV_NAME", Value: "EXTRA_ENV_VALUE"},
				},
			},
		},
	}

	f, err := os.OpenFile("../config/crd/{{.Name}}.yaml", os.O_RDONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	it := testutil.NewIntegrationTester(f, newExtension)
	it.Run(t, tests...)
}

func newExtension(client *fake.Clientset) extension.Extension[*{{.GoName}}] {
	return &Extension{}
}
