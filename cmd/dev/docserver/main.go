package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/suffiks/suffiks/docparser"
)

func main() {
	controller := docparser.NewController()
	controller.AddFS("_suffiks", os.DirFS("./docs"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		categories := controller.GetAll()

		w.Header().Add("content-type", "application/json")
		json.NewEncoder(w).Encode(categories)
	})

	fmt.Println("listening on :8084")
	log.Fatal(http.ListenAndServe(":8084", nil))
}
