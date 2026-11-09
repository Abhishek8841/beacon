package retry

import (
	"testing"
	"time"
)

func TestRetryableMethodsAndStatuses(t *testing.T) {
	for _, m := range []string{"GET", "HEAD", "OPTIONS"} {
		if !IsRetryableMethod(m) {
			t.Errorf("%s should be retryable", m)
		}
	}
	for _, m := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		if IsRetryableMethod(m) {
			t.Errorf("%s should not be retryable", m)
		}
	}
	for _, c := range []int{502, 503, 504} {
		if !IsRetryableStatus(c) {
			t.Errorf("%d should be retryable", c)
		}
	}
	for _, c := range []int{200, 400, 404, 429, 500} {
		if IsRetryableStatus(c) {
			t.Errorf("%d should not be retryable", c)
		}
	}
}

func TestValidate(t *testing.T) {
	if err := DefaultPolicy().Validate(); err != nil {
		t.Fatalf("default policy invalid: %v", err)
	}
	bad := []Policy{
		{MaxAttempts: 0, BaseDelay: time.Millisecond, MaxDelay: time.Second},
		{MaxAttempts: 3, BaseDelay: 0, MaxDelay: time.Second},
		{MaxAttempts: 3, BaseDelay: time.Second, MaxDelay: time.Millisecond},
	}
	for i, p := range bad {
		if p.Validate() == nil {
			t.Errorf("case %d: expected validation error", i)
		}
	}
}
