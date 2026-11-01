package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abhishek8841/beacon/internal/balancer"
)

func TestCheckBackendHealthy(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	backend := balancer.NewBackend(target)
	backend.SetHealthy(false)

	checker := NewChecker(
		[]*balancer.Backend{backend},
		5*time.Second,
	)

	checker.checkBackend(backend)

	if !backend.IsHealthy() {
		t.Fatal("expected backend to be healthy")
	}
}

// 1. Backend returns HTTP 500 → unhealthy.
func TestBackendUnhealthy(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			w.WriteHeader(http.StatusBadGateway)
		}),
	)
	defer server.Close()

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	backend := balancer.NewBackend(target)

	checker := NewChecker(
		[]*balancer.Backend{backend},
		5*time.Second,
	)

	checker.checkBackend(backend)

	if backend.IsHealthy() {
		t.Fatal("expected backend to be unhealthy")
	}

}

// 2. Backend is unavailable → unhealthy.
func TestBackendUnavailable(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			w.WriteHeader(http.StatusBadGateway)
		}),
	)

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	server.Close()

	backend := balancer.NewBackend(target)

	checker := NewChecker(
		[]*balancer.Backend{backend},
		5*time.Second,
	)

	checker.checkBackend(backend)

	if backend.IsHealthy() {
		t.Fatal("expected backend to be unhealthy")
	}

}

// 3. Backend recovers from unhealthy to healthy.
func TestBackendRecovers(t *testing.T) {
	var status atomic.Int32
	status.Store(http.StatusInternalServerError)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(status.Load()))
	}))
	defer server.Close()

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	backend := balancer.NewBackend(target)

	checker := NewChecker(
		[]*balancer.Backend{backend},
		5*time.Second,
	)

	checker.checkBackend(backend)

	if backend.IsHealthy() {
		t.Fatal("expected backend to be unhealthy")
	}

	// Simulate backend recovery.
	status.Store(http.StatusOK)

	checker.checkBackend(backend)

	if !backend.IsHealthy() {
		t.Fatal("expected backend to recover")
	}
}
