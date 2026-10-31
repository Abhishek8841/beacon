package main

import (
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/abhishek8841/beacon/internal/balancer"
)

func main() {

	backends := []string{
		"http://localhost:8081",
		"http://localhost:8082",
	}

	rr, err := balancer.NewRoundRobin(backends)
	if err != nil {
		log.Fatal(err)
	}

	proxy := &httputil.ReverseProxy{
		// Director: func(r *http.Request) {
		// 	target, _ := url.Parse(rr.Next())
		// 	// dont change the r.URL.path!!!
		// 	r.URL.Scheme = target.Scheme
		// 	r.URL.Host = target.Host
		// 	r.Host = target.Host
		// },
		Rewrite: func(pr *httputil.ProxyRequest) {
			target := rr.Next()
			pr.SetURL(target)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error: %v", err)
			http.Error(w, "Beacon: backend unavailable", http.StatusBadGateway)
		},
	}

	// target, err := url.Parse("http://localhost:8081")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// proxy := httputil.NewSingleHostReverseProxy(target)

	// proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
	// 	log.Printf("proxy error: %v", err)
	// 	http.Error(w, "Beacon: backend unavailable", http.StatusBadGateway)
	// }

	server := &http.Server{
		Addr:    ":8080",
		Handler: proxy,
	}
	log.Println("Beacon is listening on :8080")
	log.Fatal(server.ListenAndServe())
}
