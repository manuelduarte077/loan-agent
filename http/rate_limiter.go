package http

import (
	"sync"
	"time"
)

type clientBucket struct {
	tokens     int
	lastRefill time.Time
}

type RateLimiter struct {
	mu        sync.Mutex
	capacity  int
	refillDur time.Duration
	clients   map[string]*clientBucket
}

func NewRateLimiter(capacity int, refillDur time.Duration) *RateLimiter {
	return &RateLimiter{
		capacity:  capacity,
		refillDur: refillDur,
		clients:   make(map[string]*clientBucket),
	}
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
