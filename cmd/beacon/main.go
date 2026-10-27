package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	target, err := url.Parse("http://localhost:8081")
	if err != nil {
		log.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %v", err)
		http.Error(w, "Beacon: backend unavailable", http.StatusBadGateway)
	}

	server := http.Server{
		Addr:    ":8080",
		Handler: proxy,
	}
	log.Println("Beacon is listening on :8080")
	log.Fatal(server.ListenAndServe())
}
