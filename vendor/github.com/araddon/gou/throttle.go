package gou

import (
	"time"
)

type Throttler struct {

	// Limit to this events/per
	maxPer float64
	per    float64
	count  int32

	// Last Event
	last time.Time

	// How many events are allowed left to happen?
	// Starts at limit, decrements down
	allowance float64
}

// new Throttler that will tell you to limit or not based
// on given @max events @per duration
func NewThrottler(max int, per time.Duration) *Throttler {
	return &Throttler{
		maxPer:    float64(max),
		allowance: float64(max),
		count:     int32(0),
		last:      time.Now(),
		per:       per.Seconds(),
	}
}

// Should we limit this because we are above rate?
// Returns a bool of whether to throttle the message, and a count
// of previous log messages throttled since last log message.
func (r *Throttler) ThrottleAdd(ct int32) (bool, int32) {

	if r.maxPer == 0 {
		return false, 0
	}

	// http://stackoverflow.com/questions/667508/whats-a-good-rate-limiting-algorithm
	now := time.Now()
	elapsed := float64(now.Sub(r.last).Nanoseconds()) / 1e9 // seconds
	r.last = now
	r.allowance += elapsed * (r.maxPer / r.per)

	//Infof("maxRate: %v  cur: %v elapsed:%-6.6f  incr: %v", r.maxPer, int(r.allowance), elapsed, elapsed*float64(r.maxPer))
	if r.allowance > r.maxPer {
		r.allowance = r.maxPer
	}

	if r.allowance < 1.0 {
		r.count += ct        // increment throttled log count
		return true, r.count // do throttle/limit
	}

	tmpCount := r.count
	r.count = 0 // reset count

	r.allowance -= 1.0
	return false, tmpCount // dont throttle, return previous throttle count
}

// Should we limit this because we are above rate?
// Returns a bool of whether to throttle the message, and a count
// of previous log messages throttled since last log message.
func (r *Throttler) Throttle() (bool, int32) {
	return r.ThrottleAdd(1)
}

func (r *Throttler) ThrottleCount() int32 {
	return r.count
}
