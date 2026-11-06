package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/abhishek8841/beacon/internal/balancer"
	"github.com/abhishek8841/beacon/internal/healthcheck"
	"github.com/abhishek8841/beacon/internal/proxy"
)

// Beacon
//   |
//   v
// RoundRobin
//   |
//   v
// Backend
//   |
//   v
// Circuit Breaker
//   |
//   v
// ReverseProxy

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
	defer stop()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		checker.Start(ctx)
	}()

	transport := proxy.NewTransport()

	proxyHandler := proxy.NewProxyHandler(transport)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		target, permit, err := rr.Next()
		if err != nil {
			http.Error(w, "Beacon: no available backends", http.StatusServiceUnavailable)
			return
		}
		ctx := context.WithValue(r.Context(), proxy.BackendKey, target)
		ctx = context.WithValue(ctx, proxy.PermitKey, permit)
		r = r.WithContext(ctx)

		proxyHandler.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second, // be careful with
		WriteTimeout:      30 * time.Second, // these both
		IdleTimeout:       60 * time.Second,
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

	transport.CloseIdleConnections()

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
