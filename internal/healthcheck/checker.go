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

func (c *Checker) checkBackend(backend *balancer.Backend, ctx context.Context) {

	healthURL := *backend.URL
	healthURL.Path = "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL.String(), nil)
	if err != nil {
		backend.RecordHealthCheck(false)
		return
	}
	resp, err := c.client.Do(req)
	if err != nil {
		backend.RecordHealthCheck(false)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	backend.RecordHealthCheck(resp.StatusCode == http.StatusOK)
}

func (c *Checker) Start(ctx context.Context) {
	for _, backend := range c.backends {
		// TODO
		// this for loop check is sequential
		// address this later
		if ctx.Err() != nil {
			return
		}
		c.checkBackend(backend, ctx)
	}
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, backend := range c.backends {
				if ctx.Err() != nil {
					return
				}
				c.checkBackend(backend, ctx)
			}
		}
	}
}

// health checker needs access to the Backend structs that RoundRobin created
// so that if checker updates the health status
// round robin also can see the update...
