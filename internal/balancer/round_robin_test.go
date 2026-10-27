package balancer

import (
	"reflect"
	"testing"
)

func TestRoundRobinCylesThroughBackend(t *testing.T) {
	backends := []string{"A", "B", "C"}
	rr, err := NewRoundRobin(backends)
	if err != nil {
		t.Fatalf("NewRoundRobin() error: %v", err)
	}
	var got []string
	for range 6 {
		got = append(got, rr.Next())
	}
	want := []string{"A", "B", "C", "A", "B", "C"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Next() sequence = %v, want = %v", got, want)
	}
}

func TestRoundRobinRejectsEmptyBackends(t *testing.T) {
	_, err := NewRoundRobin(nil)
	if err != ErrNoBackends {
		t.Fatalf("error = %v, want %v", err, ErrNoBackends)
	}
}

func TestRoundRobinCopiesBackends(t *testing.T) {
	backends := []string{"A", "B"}
	rr, err := NewRoundRobin(backends)
	if err != nil {
		t.Fatal(err)
	}
	backends[0] = "CHANGED"
	if got := rr.Next(); got != "A" {
		t.Fatalf("Next() = %q, want %q", got, "A")
	}
}
