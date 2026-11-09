package circuitbreaker

import (
	"testing"
	"time"
)

func getState(b *Breaker) State {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.state
}

func TestNewBreaker(t *testing.T) {
	breaker := NewBreaker()

	if got := getState(breaker); got != StateClosed {
		t.Fatalf("expected CLOSED, got %v", got)
	}
}
func TestFailureThresholdOpensCircuit(t *testing.T) {
	breaker := NewBreaker()

	for i := 0; i < 5; i++ {
		allowed, permit := breaker.Allow()

		if !allowed {
			t.Fatal("request should be allowed while CLOSED")
		}

		breaker.RecordResult(false, permit)
	}

	if got := getState(breaker); got != StateOpen {
		t.Fatalf("expected OPEN, got %v", got)
	}
}

func TestSuccessResetsFailureCount(t *testing.T) {
	breaker := NewBreaker()

	// Two consecutive failures
	for i := 0; i < 2; i++ {
		allowed, permit := breaker.Allow()
		if !allowed {
			t.Fatal("request should be allowed")
		}

		breaker.RecordResult(false, permit)
	}

	// successful request resets the failure count
	allowed, permit := breaker.Allow()
	if !allowed {
		t.Fatal("request should be allowed")
	}

	breaker.RecordResult(true, permit)

	// Two more failures should not open the circuit
	for i := 0; i < 2; i++ {
		allowed, permit := breaker.Allow()
		if !allowed {
			t.Fatal("request should be allowed")
		}

		breaker.RecordResult(false, permit)
	}

	if got := getState(breaker); got != StateClosed {
		t.Fatalf("expected CLOSED, got %v", got)
	}
}

func TestOpenCircuitRejectsRequests(t *testing.T) {
	breaker := NewBreaker()

	for i := 0; i < 5; i++ {
		_, permit := breaker.Allow()
		breaker.RecordResult(false, permit)
	}

	allowed, permit := breaker.Allow()

	if allowed {
		t.Fatal("request should be rejected while circuit is OPEN")
	}

	if permit != nil {
		t.Fatal("rejected request should not receive a permit")
	}
}

func TestOpenTransitionsToHalfOpen(t *testing.T) {
	breaker := NewBreaker()
	breaker.recoveryTimeout = 50 * time.Millisecond

	for i := 0; i < 5; i++ {
		allowed, permit := breaker.Allow()
		if !allowed {
			t.Fatal("request should be allowed")
		}
		breaker.RecordResult(false, permit)
	}

	time.Sleep(60 * time.Millisecond)

	allowed, permit := breaker.Allow()

	if !allowed {
		t.Fatal("trial request should be allowed after recovery timeout")
	}

	if got := getState(breaker); got != StateHalfOpen {
		t.Fatalf("expected HALF-OPEN, got %v", got)
	}

	if permit == nil || permit.state != StateHalfOpen {
		t.Fatal("expected a HALF-OPEN permit")
	}
}

func TestSuccessfulTrialsCloseCircuit(t *testing.T) {
	breaker := NewBreaker()
	breaker.recoveryTimeout = 50 * time.Millisecond

	for i := 0; i < 5; i++ {
		_, permit := breaker.Allow()
		breaker.RecordResult(false, permit)
	}

	time.Sleep(60 * time.Millisecond)

	for i := 0; i < 2; i++ {
		allowed, permit := breaker.Allow()
		if !allowed {
			t.Fatal("trial request should be allowed")
		}

		breaker.RecordResult(true, permit)
	}

	if got := getState(breaker); got != StateClosed {
		t.Fatalf("expected CLOSED, got %v", got)
	}
}

func TestFailedTrialReopensCircuit(t *testing.T) {
	breaker := NewBreaker()
	breaker.recoveryTimeout = 50 * time.Millisecond

	for i := 0; i < 5; i++ {
		_, permit := breaker.Allow()
		breaker.RecordResult(false, permit)
	}

	time.Sleep(60 * time.Millisecond)

	allowed, permit := breaker.Allow()
	if !allowed {
		t.Fatal("trial request should be allowed")
	}

	breaker.RecordResult(false, permit)

	if got := getState(breaker); got != StateOpen {
		t.Fatalf("expected OPEN after failed trial, got %v", got)
	}
}

func TestStalePermitIgnored(t *testing.T) {
	breaker := NewBreaker()

	// Request A gets a permit in generation 0
	_, stalePermit := breaker.Allow()

	// Five failures open the circuit
	for i := 0; i < 5; i++ {
		_, permit := breaker.Allow()
		breaker.RecordResult(false, permit)
	}

	// Circuit is now in generation 1
	breaker.RecordResult(false, stalePermit)

	if got := getState(breaker); got != StateOpen {
		t.Fatalf("expected OPEN, got %v", got)
	}

	breaker.mu.Lock()
	failures := breaker.failures
	breaker.mu.Unlock()

	if failures != 5 {
		t.Fatalf("expected failure count to remain 5, got %d", failures)
	}
}

func TestClearPermitForHalfOpenState(t *testing.T) {
	breaker := NewBreaker()
	breaker.recoveryTimeout = time.Millisecond * 50

	for range 5 {
		allowed, permit := breaker.Allow()
		if !allowed || permit == nil {
			t.Fatal("request should be allowed while CLOSED")
		}
		breaker.RecordResult(false, permit)
	}
	if got := getState(breaker); got != StateOpen {
		t.Fatalf("expected OPEN, got %v", got)
	}

	time.Sleep(time.Millisecond * 60)
	okay, permit := breaker.Allow()
	if !okay || permit == nil || permit.state != StateHalfOpen {
		t.Fatal("expected a HALF-OPEN trial permit")
	}
	
	if allowed, _ := breaker.Allow(); allowed {
		t.Fatal("second request should be rejected while trial is in flight")
	}

	breaker.ClearPermit(permit)
	if got := getState(breaker); got != StateHalfOpen {
		t.Fatalf("expected HALF-OPEN after ClearPermit, got %v", got)
	}

	// A new trial should now be allowed.
	allowed, next := breaker.Allow()
	if !allowed || next == nil {
		t.Fatal("expected a new trial to be allowed after ClearPermit")
	}
}
