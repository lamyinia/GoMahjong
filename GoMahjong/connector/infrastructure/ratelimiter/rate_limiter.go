package ratelimiter

import (
	"sync"
	"time"
)

type RateLimiter struct {
	rate       float64
	capacity   float64
	tokens     float64
	lastRefill time.Time
	mu         sync.Mutex
}

func NewRateLimiter(rate int, burst int) *RateLimiter {
	return &RateLimiter{
		rate:       float64(rate),
		capacity:   float64(burst * rate),
		tokens:     float64(burst * rate),
		lastRefill: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()

	newTokens := elapsed * rl.rate

	if rl.tokens < rl.capacity {
		rl.tokens = min(rl.capacity, rl.tokens+newTokens)
	}

	rl.lastRefill = now

	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}

	return false
}
