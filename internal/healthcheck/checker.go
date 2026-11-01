package healthcheck

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/abhishek8841/beacon/internal/balancer"
)

type Checker struct {
	backends []*balancer.Backend
	client   *http.Client
	interval time.Duration
}

func NewChecker(backends []*balancer.Backend, interval time.Duration) *Checker {
	return &Checker{
		backends: backends,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
		interval: interval,
	}
}

func (c *Checker) checkBackend(backend *balancer.Backend) {
	healthURL := *backend.URL
	healthURL.Path = "/health"
	resp, err := c.client.Get(healthURL.String())
	if err != nil {
		backend.SetHealthy(false)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	backend.SetHealthy(http.StatusOK == resp.StatusCode)
}

func (c *Checker) Start(ctx context.Context) {
	for _, backend := range c.backends {
		// TODO
		// this for loop check is sequential 
		// address this later
		c.checkBackend(backend)
	}
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, backend := range c.backends {
				c.checkBackend(backend)
			}
		}
	}
}

// health checker needs access to the Backend structs that RoundRobin created
// so that if checker updates the health status
// round robin also can see the update...