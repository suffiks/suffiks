package main

import (
	"context"
	"log"

	"github.com/suffiks/suffiks/extension"
	"github.com/suffiks/suffiks/extension/example/controller"
)

func main() {
	config := extension.ConfigSpec{}
	if err := extension.Serve[*controller.Ingresses](context.Background(), config, &controller.IngressExtension{}); err != nil {
		log.Fatal(err)
	}
}
