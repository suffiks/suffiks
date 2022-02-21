package main

import (
	"context"
	"log"

	"github.com/suffiks/suffiks/extension"
	"github.com/suffiks/suffiks/extension/example/controller"
)

func main() {
	if err := extension.Serve[*controller.Ingresses](context.Background(), ":3000", &controller.IngressExtension{}); err != nil {
		log.Fatal(err)
	}
}
