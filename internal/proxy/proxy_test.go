package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestProxyForwardsPOSTBody(t *testing.T) {
	type serverBodyStruct struct {
		body        string
		contentType string
		err         error
	}
	serverBodyChan := make(chan serverBodyStruct, 1)
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		res, err := io.ReadAll(r.Body)
		serverBodyChan <- serverBodyStruct{
			body:        string(res),
			err:         err,
			contentType: r.Header.Get("Content-Type"),
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from backend"))
	}))
	defer upstreamServer.Close()

	transport := NewTransport()
	defer transport.CloseIdleConnections()
	proxyHandler := NewProxyHandler(transport)
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

	wantBody := "hello from client"
	req, err := http.NewRequest(http.MethodPost, proxyServer.URL, strings.NewReader(wantBody))
	if err != nil {
		t.Fatalf("http.NewRequest() error: %v", err)
	}
	req.Header.Set("Content-Type", "text/plain")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do() error: %v", err)
	}
	defer res.Body.Close()
	rspns, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("io.ReadAll() error: %v", err)
	}
	if string(rspns) != "hello from backend" {
		t.Errorf("body = %q, want %q", string(rspns), "hello from backend")
	}
	output := <-serverBodyChan
	if output.err != nil {
		t.Fatalf("Unable to read upstream body: %v", output.err)
	}
	if output.contentType != "text/plain" {
		t.Errorf("Content-Type = %q, want %q",
			output.contentType, "text/plain")
	}
	if output.body != wantBody {
		t.Fatalf("gotBody != wantBody i.e. wantBody = %s, gotBody = %s", wantBody, output.body)
	}
}
