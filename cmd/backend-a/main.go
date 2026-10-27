package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from backend-a! You requested: %s\n", r.URL.Path)
	})

	log.Println("Backend A listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
