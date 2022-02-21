package main

import (
	"context"
	"log"
	"os"
	"os/signal"

{{if .Kubernetes}}
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
{{end -}}

	"github.com/suffiks/suffiks/extension"
	"{{ .Repo }}/controllers"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	{{- if .Kubernetes}}
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	{{end -}}

	ext := &controllers.{{.GoName}}Extension{
	{{- if .Kubernetes}}
		Client: client,
	{{end}}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	return extension.Serve[*controllers.{{.GoName}}](ctx, ":4269", ext)
}
