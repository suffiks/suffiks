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
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		categories, err := docparser.Parse(os.DirFS("./docs"))
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Add("content-type", "application/json")
		json.NewEncoder(w).Encode(categories)
	})

	fmt.Println("listening on :8084")
	log.Fatal(http.ListenAndServe(":8084", nil))
}
