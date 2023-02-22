package main

import (
	"context"
	"flag"
	"fmt"
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
	"{{ .Repo }}/{{ .Name }}"
)

var configFile string

//go:embed docs/*.md
var docs embed.FS

func init() {
	flag.StringVar(&configFile, "config-file", "", "path to config file")
}

func main() {
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	{{- if .Kubernetes}}
	konfig, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	client, err := kubernetes.NewForConfig(konfig)
	if err != nil {
		return err
	}

	{{end -}}

	config := &{{ .Name }}.Config{}
	if err := extension.ReadConfig(configFile, config); err != nil {
		return err
	}

	docs := &extension.Documentation{
		FS:   docs,
		Root: "docs",
	}

	ext := &{{ .Name }}.Extension{
	{{- if .Kubernetes}}
		Client: client,
	{{end}}
	}

	fmt.Println("Listening on", config.ListenAddress)
	return extension.Serve[*{{ .Name }}.{{.GoName}}](ctx, config, ext, docs)
}
