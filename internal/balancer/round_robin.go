package balancer

import (
	"errors"
	"sync/atomic"
)

var ErrNoBackends = errors.New("no backends configured")

type RoundRobin struct {
	backends []string
	counter  atomic.Uint64
}

func NewRoundRobin(backends []string) (*RoundRobin, error) {
	if len(backends) == 0 {
		return nil, ErrNoBackends
	}
	copied := append([]string(nil), backends...)
	return &RoundRobin{
		backends: copied,
	}, nil
}

func (rr *RoundRobin) Next() string {
	index := rr.counter.Add(1) - 1
	return rr.backends[index%uint64(len(rr.backends))]
}
