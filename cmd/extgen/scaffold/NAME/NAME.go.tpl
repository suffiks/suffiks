package {{.Name}}

import (
	"context"
	"fmt"

	"github.com/suffiks/suffiks/extension"
	{{- if .Kubernetes}}
	"k8s.io/client-go/kubernetes"
	{{end}}
)


{{if .Kubernetes}}
// You can define required RBAC roles in this file. For example:
// //+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
{{end}}

type Extension struct {
	{{if .Kubernetes }}
	Client *kubernetes.Clientset
	{{end}}
}

func ({{.Receiver}} *Extension) Sync(ctx context.Context, owner extension.Owner, obj *{{.GoName}}, rw *extension.ResponseWriter) error {
	fmt.Println("Start syncing", owner.Name(), "in", owner.Namespace())

	rw.AddEnv(string(obj.{{.GoName}}.ExtraEnv.Name), obj.{{.GoName}}.ExtraEnv.Value)

	return nil
}

func ({{.Receiver}} *Extension) Delete(ctx context.Context, owner extension.Owner, obj *{{.GoName}}) error {
	fmt.Println("Start delete", owner.Name(), "in", owner.Namespace())

	return nil
}

{{if .Validation }}
func ({{.Receiver}} *Extension) Validate(ctx context.Context, typ extension.ValidationType, owner extension.Owner, newObj, oldObj *{{.GoName}}) ([]extension.ValidationErrors, error) {
	if typ == extension.ValidationDelete {
		return nil, nil
	}

	return nil, nil
}
{{end}}

{{if .Defaulting }}
func ({{.Receiver}} *Extension) Default(ctx context.Context, owner extension.Owner, obj *{{.GoName}}) (*{{.GoName}}, error) {
	fmt.Println("Start defaulting", owner.Name(), "in", owner.Namespace())
	return obj, nil
}
{{end}}
