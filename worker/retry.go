package worker

import "time"

// RetryPolicy controls how a queue re-delivers failed tasks. The zero value
// is normalized to DefaultRetryPolicy; set MaxAttempts to 1 to disable
// retries and dead-letter on first failure.
type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultRetryPolicy allows three attempts with one second initial backoff,
// capped at one minute.
var DefaultRetryPolicy = RetryPolicy{
	MaxAttempts:    3,
	InitialBackoff: time.Second,
	MaxBackoff:     time.Minute,
}

// Backoff returns the delay before attempt n+1 after n consecutive failures:
// exponential from InitialBackoff, capped at MaxBackoff. It returns zero for
// n < 1. The receiver is normalized first, so the zero value yields the
// defaults.
func (p RetryPolicy) Backoff(n int) time.Duration {
	p = p.orDefault()
	if n < 1 {
		return 0
	}
	d := p.InitialBackoff
	for i := 1; i < n && d < p.MaxBackoff; i++ {
		if d > p.MaxBackoff/2 {
			d = p.MaxBackoff
			break
		}
		d *= 2
	}
	return d
}

func (p RetryPolicy) orDefault() RetryPolicy {
	if p.MaxAttempts < 1 {
		p = DefaultRetryPolicy
	}
	if p.InitialBackoff <= 0 {
		p.InitialBackoff = DefaultRetryPolicy.InitialBackoff
	}
	if p.MaxBackoff <= 0 {
		p.MaxBackoff = DefaultRetryPolicy.MaxBackoff
	}
	return p
}

func (p *RetryPolicy) resolved() RetryPolicy {
	if p == nil {
		return DefaultRetryPolicy
	}
	return p.orDefault()
}
