package utils

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
