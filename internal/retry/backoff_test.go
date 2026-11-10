package retry

import (
	"testing"
	"time"
)

func TestCeilingGrowsAndCaps(t *testing.T) {
	p := Policy{MaxAttempts: 5, BaseDelay: 50 * time.Millisecond, MaxDelay: 400 * time.Millisecond}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 50 * time.Millisecond},
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 400 * time.Millisecond},   // capped
		{100, 400 * time.Millisecond}, // no overflow
		{-1, 50 * time.Millisecond},   // negative treated as 0
	}
	for _, tt := range tests {
		if got := p.ceiling(tt.attempt); got != tt.want {
			t.Errorf("ceiling(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestBackoffStaysWithinBounds(t *testing.T) {
	p := DefaultPolicy()
	for attempt := 0; attempt < 10; attempt++ {
		limit := p.ceiling(attempt)
		for i := 0; i < 1000; i++ {
			got := p.Backoff(attempt)
			if got < 0 || got > limit {
				t.Fatalf("Backoff(%d) = %v, want within [0, %v]", attempt, got, limit)
			}
		}
	}
}
