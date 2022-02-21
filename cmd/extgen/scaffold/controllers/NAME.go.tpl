package controllers

import (
	"context"
	"fmt"

	"github.com/suffiks/suffiks/extension"
	{{- if .Kubernetes}}
	"k8s.io/client-go/kubernetes"
	{{end}}
)


var _ extension.Extension[*{{.GoName}}] = &{{.GoName}}Extension{}
{{- if .Validation }}
var _ extension.ValidatableExtension[*{{.GoName}}] = &{{.GoName}}Extension{}
{{end}}
{{- if .Defaulting }}
var _ extension.DefaultableExtension[*{{.GoName}}] = &{{.GoName}}Extension{}
{{end}}

{{if .Kubernetes}}
// You can define required RBAC roles in this file. For example:
// //+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
{{end}}

type {{.GoName}}Extension struct {
	{{if .Kubernetes }}
	Client *kubernetes.Clientset
	{{end}}
}

func ({{.Receiver}} *{{.GoName}}Extension) Sync(ctx context.Context, owner extension.Owner, obj *{{.GoName}}, rw *extension.ResponseWriter) error {
	fmt.Println("Start syncing", owner.Name(), "in", owner.Namespace())

	fmt.Printf("is %v super? %v\n", owner.Name(), obj.{{.GoName}}.Super)

	return nil
}

func ({{.Receiver}} *{{.GoName}}Extension) Delete(ctx context.Context, owner extension.Owner, obj *{{.GoName}}) error {
	fmt.Println("Start delete", owner.Name(), "in", owner.Namespace())

	return nil
}

{{if .Validation }}
func ({{.Receiver}} *{{.GoName}}Extension) Validate(ctx context.Context, typ extension.ValidationType, owner extension.Owner, newObj, oldObj *{{.GoName}}) ([]extension.ValidationErrors, error) {
	if typ == extension.ValidationDelete {
		return nil, nil
	}

	return nil, nil
}
{{end}}

{{if .Defaulting }}
func ({{.Receiver}} *{{.GoName}}Extension) Default(ctx context.Context, req extension.Sync, owner extension.Owner, obj *{{.GoName}}) (*{{.GoName}}, error) {
	fmt.Println("Start defaulting", owner.Name(), "in", owner.Namespace())
	return nil
}
{{end}}
