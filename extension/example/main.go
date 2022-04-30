package main

import (
	"context"
	"embed"
	"log"

	"github.com/suffiks/suffiks/extension"
	"github.com/suffiks/suffiks/extension/example/controller"
)

//go:embed docs/*.md
var docFiles embed.FS

func main() {
	config := extension.ConfigSpec{}
	if err := extension.Serve[*controller.Ingresses](context.Background(), config, &controller.IngressExtension{}, docFiles); err != nil {
		log.Fatal(err)
	}
}
