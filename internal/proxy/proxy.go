package proxy

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/abhishek8841/beacon/internal/balancer"
	"github.com/abhishek8841/beacon/internal/circuitbreaker"
)

func NewTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     50,
		IdleConnTimeout:     90 * time.Second,
	}
}

func newModifyResponse() func(r *http.Response) error {
	return func(r *http.Response) error {
		target := r.Request.Context().Value(BackendKey).(*balancer.Backend)
		permit := r.Request.Context().Value(PermitKey).(*circuitbreaker.Permit)

		success := r.StatusCode < 500

		target.RecordResult(success, permit)

		return nil
	}
}

func newErrorHandler() func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		target := r.Context().Value(BackendKey).(*balancer.Backend)
		permit := r.Context().Value(PermitKey).(*circuitbreaker.Permit)

		if r.Context().Err() != nil {
			// Here if the client is in Half Open Condition and not recording result means the server is forever in underRequestMode and never gets another retry
			target.ClearPermit(permit)
			log.Printf("request cancelled: %v", r.Context().Err())
			return
		}

		target.RecordResult(false, permit)

		log.Printf("proxy error: %v", err)
		http.Error(w, "Beacon: backend unavailable", http.StatusBadGateway)
	}
}

func newRewrite() func(pr *httputil.ProxyRequest) {
	return func(pr *httputil.ProxyRequest) {
		target := pr.In.Context().Value(BackendKey).(*balancer.Backend)
		pr.SetURL(target.URL)
	}
}

func NewProxyHandler(transport *http.Transport) *httputil.ReverseProxy {
	proxyHandler := &httputil.ReverseProxy{
		Rewrite:        newRewrite(),
		ErrorHandler:   newErrorHandler(),
		Transport:      transport,
		ModifyResponse: newModifyResponse(),
	}
	return proxyHandler
}

// The Transport is being passed instead of making a new transport inside the newProxyHandler
// This is because the transport is its own entity

//  The Following is the old Director config i.e. Director is now replaced by Rewrite ->

// Director: func(r *http.Request) {
// 	target, _ := url.Parse(rr.Next())
// 	// dont change the r.URL.path!!!
// 	r.URL.Scheme = target.Scheme
// 	r.URL.Host = target.Host
// 	r.Host = target.Host
// },

//  The Following is the old proxy config i.e. non changable static config ->

// target, err := url.Parse("http://localhost:8081")
// if err != nil {
// 	log.Fatal(err)
// }

// proxy := httputil.NewSingleHostReverseProxy(target)

// proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
// 	log.Printf("proxy error: %v", err)
// 	http.Error(w, "Beacon: backend unavailable", http.StatusBadGateway)
// }
