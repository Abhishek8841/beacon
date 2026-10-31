package balancer

import (
	"sync"
	"testing"
)

func TestRoundRobinCylesThroughBackend(t *testing.T) {
	backends := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}
	rr, err := NewRoundRobin(backends)
	if err != nil {
		t.Fatalf("NewRoundRobin() error: %v", err)
	}
	want := []string{
		backends[0],
		backends[1],
		backends[2],
		backends[0],
		backends[1],
		backends[2],
	}

	for i, expected := range want {
		got := rr.Next()

		if got.String() != expected {
			t.Errorf("Next() at call %d = %q, want %q", i, got.String(), expected)
		}
	}
}

func TestRoundRobinRejectsEmptyBackends(t *testing.T) {
	_, err := NewRoundRobin(nil)
	if err != ErrNoBackends {
		t.Fatalf("error = %v, want %v", err, ErrNoBackends)
	}
}

func TestRoundRobinConcurrentAccess(t *testing.T) {
	backends := []string{
		"http://localhost:8081",
		"http://localhost:8082",
	}
	rr, err := NewRoundRobin(backends)
	if err != nil {
		t.Fatalf("error in NewRoundRobin %v", err)
	}
	var wg sync.WaitGroup
	const NumberOfGoroutines = 10
	const NumberOfRequestPerGoroutine = 100
	for range NumberOfGoroutines {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for range NumberOfRequestPerGoroutine {
				if nxt := rr.Next(); nxt == nil {
					t.Error("Next() returned nil")
				}
			}
		}()
	}
	wg.Wait()
	if got, want := rr.counter.Load(), NumberOfGoroutines*NumberOfRequestPerGoroutine; got != uint64(want) {
		t.Fatalf("concurrency test failed -> got = %v && want = %v", got, want)
	}
}
