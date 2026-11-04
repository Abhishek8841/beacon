package balancer

import (
	"net/url"
	"sync/atomic"

	"github.com/abhishek8841/beacon/internal/circuitbreaker"
)

const (
	FailureThreshold = 3
	SuccessThreshold = 2
)

type Backend struct {
	URL     *url.URL
	healthy atomic.Bool

	consecutiveFailures  int
	consecutiveSuccesses int

	breaker *circuitbreaker.Breaker
}

func NewBackend(url *url.URL) *Backend {
	b := &Backend{
		URL:     url,
		breaker: circuitbreaker.NewBreaker(),
	}
	b.healthy.Store(true)
	return b
}

func (b *Backend) IsHealthy() bool {
	return b.healthy.Load()
}

func (b *Backend) SetHealthy(healthy bool) {
	b.healthy.Store(healthy)
}

func (b *Backend) RecordHealthCheck(success bool) {
	if success {
		b.consecutiveFailures = 0
		b.consecutiveSuccesses++

		if b.consecutiveSuccesses >= SuccessThreshold {
			b.healthy.Store(true)
		}
		return
	}

	b.consecutiveSuccesses = 0
	b.consecutiveFailures++

	if b.consecutiveFailures >= FailureThreshold {
		b.healthy.Store(false)
	}
}

func (b *Backend) RecordResult(success bool, permit *circuitbreaker.Permit) {
	b.breaker.RecordResult(success, permit)
}
