package balancer

import (
	"net/url"
	"sync/atomic"
)

type Backend struct {
	URL     *url.URL
	healthy atomic.Bool
}

func NewBackend(url *url.URL) *Backend {
	b := &Backend{
		URL: url,
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
