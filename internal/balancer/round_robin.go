package balancer

import (
	"errors"
	"fmt"
	"net/url"
	"sync/atomic"
)

var ErrNoBackends = errors.New("no backends configured")

type RoundRobin struct {
	backends []*url.URL
	counter  atomic.Uint64
}

func NewRoundRobin(backends []string) (*RoundRobin, error) {
	if len(backends) == 0 {
		return nil, ErrNoBackends
	}

	copied := make([]*url.URL, 0, len(backends))

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
		copied = append(copied, target)

	}

	return &RoundRobin{
		backends: copied,
	}, nil
}

func (rr *RoundRobin) Next() *url.URL {
	index := rr.counter.Add(1) - 1
	return rr.backends[index%uint64(len(rr.backends))]
}
