package circuitbreaker

import (
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

type Breaker struct {
	mu sync.Mutex

	state State

	failures  int
	successes int

	openedAt time.Time

	failureThreshold int
	successThreshold int
	recoveryTimeout  time.Duration

	halfOpenInFlight bool
}

func NewBreaker() *Breaker {
	return &Breaker{
		state:            StateClosed,
		failureThreshold: 5,
		successThreshold: 2,
		recoveryTimeout:  10 * time.Second,
	}
}

func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return true

	case StateOpen:
		if time.Since(b.openedAt) < b.recoveryTimeout {
			return false
		}

		b.state = StateHalfOpen
		b.halfOpenInFlight = true
		return true

	case StateHalfOpen:
		if b.halfOpenInFlight {
			return false
		}

		b.halfOpenInFlight = true
		return true

	default:
		return false
	}
}

func (b *Breaker) RecordResult(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		if success {
			b.failures = 0
			return
		}

		b.failures++

		if b.failures >= b.failureThreshold {
			b.state = StateOpen
			b.openedAt = time.Now()
			b.successes = 0
		}

	case StateHalfOpen:
		b.halfOpenInFlight = false

		if !success {
			b.state = StateOpen
			b.openedAt = time.Now()
			b.successes = 0
			return
		}

		b.successes++

		if b.successes >= b.successThreshold {
			b.state = StateClosed
			b.failures = 0
			b.successes = 0
		}
	}
}



