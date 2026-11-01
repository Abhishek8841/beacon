package balancer

import (
	"errors"
	"fmt"
	"net/url"
	"sync/atomic"
)

var ErrNoBackends = errors.New("no backends configured")
var ErrNoHealthyBackend = errors.New("no healthy backend present")

type RoundRobin struct {
	backends []*Backend
	counter  atomic.Uint64
}

func NewRoundRobin(backends []string) (*RoundRobin, error) {
	if len(backends) == 0 {
		return nil, ErrNoBackends
	}

	copied := make([]*Backend, 0, len(backends))

	for _, backend := range backends {
		target, err := url.Parse(backend)
		if err != nil {
			return nil, fmt.Errorf("invalid backend URL: %w", err)
		}
		if target.Scheme != "http" && target.Scheme != "https" {
			return nil, fmt.Errorf("Unsupported backend scheme %q int %q", target.Scheme, backend)
		}
		if target.Host == "" {
			return nil, fmt.Errorf(
				"backend URL has no host: %q",
				backend,
			)
		}
		if target.Hostname() == "" {
			return nil, fmt.Errorf("backend URL has no hostname: %q", backend)
		}
		copied = append(copied, NewBackend(target))
	}

	return &RoundRobin{
		backends: copied,
	}, nil
}

func (rr *RoundRobin) Next() (*Backend, error) {
	for i := 0; i < len(rr.backends); i++ {
		index := rr.counter.Add(1) - 1
		backend := rr.backends[index%uint64(len(rr.backends))]

		if backend.IsHealthy() {
			return backend, nil
		}
	}

	return nil, ErrNoHealthyBackend
}

func (rr *RoundRobin) Backends() []*Backend {
	return rr.backends
}
