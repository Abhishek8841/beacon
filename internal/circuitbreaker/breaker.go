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

type Permit struct {
	state      State
	generation uint64
}

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

	generation uint64
}

func NewBreaker() *Breaker {
	return &Breaker{
		state:            StateClosed,
		failureThreshold: 5,
		successThreshold: 2,
		recoveryTimeout:  10 * time.Second,
	}
}

func (b *Breaker) Allow() (bool, *Permit) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:
		return true, &Permit{state: StateClosed, generation: b.generation}

	case StateOpen:
		if time.Since(b.openedAt) < b.recoveryTimeout {
			return false, &Permit{state: StateOpen, generation: b.generation}
		}

		b.state = StateHalfOpen
		b.generation++
		b.halfOpenInFlight = true
		return true, &Permit{state: StateHalfOpen, generation: b.generation}

	case StateHalfOpen:
		if b.halfOpenInFlight {
			return false, &Permit{state: StateHalfOpen, generation: b.generation}
		}

		b.halfOpenInFlight = true
		return true, &Permit{state: StateHalfOpen, generation: b.generation}

	default:
		return false, nil
	}
}

func (b *Breaker) RecordResult(success bool, permit *Permit) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch b.state {
	case StateClosed:

		if permit == nil || permit.state != StateClosed || b.generation != permit.generation {
			return
		}

		if success {
			b.failures = 0
			return
		}

		b.failures++

		if b.failures >= b.failureThreshold {
			b.state = StateOpen
			b.generation++
			b.openedAt = time.Now()
			b.successes = 0
		}

	case StateHalfOpen:

		if permit == nil || permit.state != StateHalfOpen || b.generation != permit.generation {
			return
		}

		b.halfOpenInFlight = false

		if !success {
			b.state = StateOpen
			b.generation++
			b.openedAt = time.Now()
			b.successes = 0
			return
		}

		b.successes++

		if b.successes >= b.successThreshold {
			b.state = StateClosed
			b.generation++
			b.failures = 0
			b.successes = 0
		}
	}
}

// IMPORTANT RACE CONDITIONS ->

// A--->
// 1. Request A was allowed while CLOSED.
// 2. Five other requests fail, opening the circuit.
// 3. The recovery timeout expires.
// 4. A trial request moves the circuit to HALF-OPEN.
// 5. Request A finally finishes.

// RecordResult() sees HALF-OPEN and treats Request A's result as a trial result, even though it was admitted while CLOSED!

// TO COUNTER THIS I INTRODUCED PERMITS WHICH TELL WHAT THE STATE WAS OF THE CIRCUIT WHEN THE REQUEST WAS ALLOWED THROUGH

// B--->
// stale permits can interfere with a newer circuit state
// Consider this sequence:
// 1. Request A is admitted in CLOSED.
// 2. Five failures open the circuit.
// 3. Recovery timeout expires.
// 4. Request B is admitted in HALF-OPEN.
// 5. Request B succeeds, and another trial succeeds. Circuit becomes CLOSED.
// 6. Request A finally fails.

// Previous code saw that the circuit is CLOSED and the permit says CLOSED. It increments the failure counter, even though Request A belongs to an older circuit generation.

// TO COUNTER THIS I INTRODUCED GENERATIONS WHICH CHECKS  WHETHER THE GENERATION OF THE IS CIRCUITS ARE SAME OR IS THE REQUEST STALE
