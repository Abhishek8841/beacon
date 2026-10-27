package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from backend-b! You requested: %s\n", r.URL.Path)
	})

	log.Println("Backend B listening on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
