package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abhishek8841/beacon/internal/balancer"
)

func TestProxyForwardsRequest(t *testing.T) {
	type gotStruct struct {
		gotPath   string
		gotMethod string
		gotQuery  string
	}

	gotChan := make(chan gotStruct, 1)

	transport := NewTransport()
	defer transport.CloseIdleConnections()
	proxyHandler := NewProxyHandler(transport)
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotChan <- gotStruct{
			gotPath:   r.URL.Path,
			gotMethod: r.Method,
			gotQuery:  r.URL.Query().Get("id"),
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from backend"))
	}))
	defer upstreamServer.Close()
	rr, err := balancer.NewRoundRobin([]string{upstreamServer.URL})
	if err != nil {
		t.Fatalf("balancer.NewRoundRobin() error :%v", err)
	}

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target, permit, err := rr.Next()
		if err != nil {
			http.Error(w, "Beacon: no available backends", http.StatusServiceUnavailable)
			return
		}
		ctx := context.WithValue(r.Context(), BackendKey, target)
		ctx = context.WithValue(ctx, PermitKey, permit)
		r = r.WithContext(ctx)

		proxyHandler.ServeHTTP(w, r)
	}))
	defer proxyServer.Close()
	req, err := http.NewRequest(http.MethodGet, proxyServer.URL+"/test?id=10", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do() error: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error: %v", err)
	}
	if string(body) != "hello from backend" {
		t.Errorf("body = %q, want %q", body, "hello from backend")
	}
	res := <-gotChan
	if res.gotPath != "/test" || res.gotMethod != http.MethodGet || res.gotQuery != "10" {
		t.Fatalf("The method or the path or the query recieved were invalid = %s & %s & %s", res.gotMethod, res.gotPath, res.gotQuery)
	}
}

