package agent

import (
	"sync"
	"time"
)

// Reconnect handles reconnection logic
type Reconnect struct {
	interval    time.Duration // initial reconnect interval
	maxAttempts int           // max attempts, 0 = infinite
	maxInterval time.Duration // max interval
	attempts    int
	mu          sync.Mutex
}

// NewReconnect creates a new Reconnect handler
func NewReconnect(interval, maxAttempts, maxInterval int) *Reconnect {
	return &Reconnect{
		interval:    time.Duration(interval) * time.Second,
		maxAttempts: maxAttempts,
		maxInterval: time.Duration(maxInterval) * time.Second,
	}
}

// shouldRetry returns true if another attempt should be made
func (r *Reconnect) shouldRetry() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.maxAttempts == 0 {
		return true
	}
	return r.attempts < r.maxAttempts
}

// wait sleeps for the current backoff interval
func (r *Reconnect) wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	interval := r.interval * time.Duration(1<<uint(r.attempts))
	if interval > r.maxInterval {
		interval = r.maxInterval
	}

	time.Sleep(interval)
	r.attempts++
}

// beforeRetry is called before a new connection attempt
func (r *Reconnect) beforeRetry() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Exponential backoff: 5s, 10s, 20s, 40s, ...
	interval := r.interval * time.Duration(1<<uint(r.attempts))
	if interval > r.maxInterval {
		interval = r.maxInterval
	}

	time.Sleep(interval)
	r.attempts++
}

// reset resets the reconnect state
func (r *Reconnect) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attempts = 0
}
