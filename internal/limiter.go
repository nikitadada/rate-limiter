package internal

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu                 sync.RWMutex
	currentRequestsMap map[string]int
	timeInterval       time.Duration
	allowRequestsCount int
}

func NewRateLimiter(timeInterval time.Duration, allowRequestsCount int) *RateLimiter {
	limiter := &RateLimiter{
		currentRequestsMap: make(map[string]int),
		timeInterval:       timeInterval,
		allowRequestsCount: allowRequestsCount,
	}
	limiter.reset()

	return limiter
}

func (r *RateLimiter) Allow(ip string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cur := r.currentRequestsMap[ip]

	if cur >= r.allowRequestsCount {
		return false
	}

	return true
}

func (r *RateLimiter) AddRequest(ip string) {
	r.mu.Lock()
	r.currentRequestsMap[ip]++
	r.mu.Unlock()
}

func (r *RateLimiter) reset() {
	go func() {
		ticker := time.NewTicker(r.timeInterval)
		for {
			select {
			case <-ticker.C:
				for key := range r.currentRequestsMap {
					r.mu.Lock()
					r.currentRequestsMap[key] = 0
					r.mu.Unlock()
				}
			}
		}
	}()
}
