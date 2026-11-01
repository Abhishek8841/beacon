package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/abhishek8841/beacon/internal/balancer"
	"github.com/abhishek8841/beacon/internal/healthcheck"
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

	checker := healthcheck.NewChecker(rr.Backends(), 5*time.Second)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		checker.Start(ctx)
	}()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		target, err := rr.Next()
		if err != nil {
			http.Error(w, "Beacon: no healthy backends available",
				http.StatusServiceUnavailable)
			return
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
				pr.SetURL(target.URL)
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				log.Printf("proxy error: %v", err)
				http.Error(w, "Beacon: backend unavailable", http.StatusBadGateway)
			},
		}
		proxy.ServeHTTP(w, r)
	})

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
		Handler: handler,
	}

	serverErr := make(chan error, 1)

	go func() {
		log.Println("Beacon is listening on :8080")
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Println("Shutdown signal received")

	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server failed: %v", err)
		}
	}
	stop()
	// stop here instead of defer -> this is to prevent infinite waiting of wg.wait() in case err:=<-serverErr

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)

		if closeErr := server.Close(); closeErr != nil {
			log.Printf("Forced server close failed: %v", closeErr)
		}
	}

	wg.Wait()

	log.Println("Beacon shutdown complete")
}

// Graceful shutdown should:
// 1. Stop accepting new HTTP connections.
// 2. Allow existing requests to finish.
// 3. Stop the health checker.
// 4. Release server resources.

// This all is implemeneted by server.shutdown(ctx)

// If no Graceful Shutdown then default go mechanisms shutdown the server abruptly
// But now only ctx gets cancelled rather than abrupt shutdown !!!
