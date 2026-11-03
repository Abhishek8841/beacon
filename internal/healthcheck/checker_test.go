package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abhishek8841/beacon/internal/balancer"
)

//		The following test is testing the following params
//	 Two failures do not mark it unhealthy.
//	 Three failures mark it unhealthy.
//	 One success does not recover an unhealthy backend.
//	 Two successes recover it.
func TestFailureBackendUnhealthy(t *testing.T) {
	var status atomic.Int32
	status.Store(http.StatusBadGateway)
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" {
				t.Errorf("invalid path %s", r.URL.Path)
			}
			w.WriteHeader(int(status.Load()))
		}),
	)
	defer server.Close()
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("error encountered while parsing the test server's url")
	}
	backend := balancer.NewBackend(target)
	checker := NewChecker([]*balancer.Backend{backend}, time.Second*5)
	ctx := context.Background()
	checker.checkBackend(backend, ctx)
	checker.checkBackend(backend, ctx)
	if !backend.IsHealthy() {
		t.Fatal("backend is marked as unhealthy after only 2 tries")
	}
	checker.checkBackend(backend, ctx)
	if backend.IsHealthy() {
		t.Fatal("backend is healthy even after three failed tries")
	}
	status.Store(http.StatusOK)
	checker.checkBackend(backend, ctx)
	if backend.IsHealthy() {
		t.Fatal("one successful check marked the backend healthy")
	}
	checker.checkBackend(backend, ctx)
	if !backend.IsHealthy() {
		t.Fatal("backend is marked as unhealthy even after two successful checks")
	}
}

// TODO
// 6. A success resets the consecutive failure count.
// 7. A failure resets the consecutive success count.
