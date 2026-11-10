package retry

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Policy struct {
	// MaxAttempts is the total number fo attempts including the first attempt -> i.e. MaxAttempts = 1 means never retry
	MaxAttempts int

	// BaseDelay is the backoff ceil before the first retry
	BaseDelay time.Duration

	// MaxDelay is the max limit of backoff ceil no matter on which attempt we are
	MaxDelay time.Duration
}

var ErrInvalidPolicy = errors.New("retry: invalid policy")

func DefaultPolicy() *Policy {
	return &Policy{
		MaxAttempts: 3,
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}
}

func (p *Policy) Validate() error {
	if p.MaxAttempts < 1 {
		return fmt.Errorf("%w: MaxAttempts must be at least 1, got %d", ErrInvalidPolicy, p.MaxAttempts)
	}
	if p.BaseDelay <= 0 {
		return fmt.Errorf("%w: BaseDelay must be positive, got %v", ErrInvalidPolicy, p.BaseDelay)
	}
	if p.MaxDelay < p.BaseDelay {
		return fmt.Errorf("%w: MaxDelay (%v) must be >= BaseDelay (%v)", ErrInvalidPolicy, p.MaxDelay, p.BaseDelay)
	}
	return nil
}

// only Read Only methods qualify in the below due to IDEMPOTENCY
func IsRetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func IsRetryableStatus(code int) bool {
	switch code {
	case http.StatusBadGateway, // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout:     // 504
		return true
	default:
		return false
	}
}

// 500 is not retryable as it is usually a real bug