package retry

import (
	"math/rand/v2"
	"time"
)

func (p *Policy) ceiling(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	d := p.BaseDelay
	for i := 0; i < attempt; i++ {
		d *= 2
		if d >= p.MaxDelay || d <= 0 {
			return p.MaxDelay
		}
	}
	if d > p.MaxDelay {
		return p.MaxDelay
	}
	return d
}

func (p *Policy) Backoff(attempt int) time.Duration {
	ceiling := p.ceiling(attempt)
	if ceiling <= 0 {
		return 0
	}
	// Int64N(n) returns [0, n) ie 0,1,2,3,...n-1 so +1 makes n reachable
	return time.Duration(rand.Int64N(int64(ceiling) + 1))
}
