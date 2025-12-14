package http

import (
	"sync"
	"time"
)

const (
	bucketCleanupThreshold = 1 * time.Hour
	cleanupInterval        = 30 * time.Minute
)

type clientBucket struct {
	tokens     int
	lastRefill time.Time
}

type RateLimiter struct {
	mu          sync.Mutex
	capacity    int
	refillDur   time.Duration
	clients     map[string]*clientBucket
	stopCleanup chan struct{}
}

func NewRateLimiter(capacity int, refillDur time.Duration) *RateLimiter {
	rl := &RateLimiter{
		capacity:    capacity,
		refillDur:   refillDur,
		clients:     make(map[string]*clientBucket),
		stopCleanup: make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (r *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cleanup()
		case <-r.stopCleanup:
			return
		}
	}
}

func (r *RateLimiter) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for ip, bucket := range r.clients {
		if now.Sub(bucket.lastRefill) > bucketCleanupThreshold {
			delete(r.clients, ip)
		}
	}
}

func (r *RateLimiter) Stop() {
	close(r.stopCleanup)
}

func (r *RateLimiter) Allow(ip string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	bucket, exists := r.clients[ip]

	if !exists {
		r.clients[ip] = &clientBucket{
			tokens:     r.capacity - 1,
			lastRefill: now,
		}
		return true
	}

	if now.Sub(bucket.lastRefill) >= r.refillDur {
		bucket.tokens = r.capacity
		bucket.lastRefill = now
	}

	if bucket.tokens <= 0 {
		return false
	}

	bucket.tokens--
	return true
}
