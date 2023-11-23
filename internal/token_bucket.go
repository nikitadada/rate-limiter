package internal

import (
	"sync"
	"time"
)

type TokenBucket struct {
	cap            int
	insertInterval time.Duration
	mu             sync.RWMutex
	// TODO реализовать через atomic
	size       int
	lastTakeAt time.Time
}

func NewTokenBucket(cap int, insertInterval time.Duration) *TokenBucket {
	return &TokenBucket{
		cap:            cap,
		insertInterval: insertInterval,
		size:           cap,
	}
}

func (t *TokenBucket) InsertInterval() time.Duration {
	return t.insertInterval
}

func (t *TokenBucket) Inc() {
	t.mu.Lock()
	t.size++
	t.mu.Unlock()
}

func (t *TokenBucket) Add(delta int) {
	t.mu.Lock()
	t.size += delta
	if t.size > t.cap {
		t.size = t.cap
	}
	t.mu.Unlock()
}

func (t *TokenBucket) Dec() {
	t.mu.Lock()
	t.size--
	t.mu.Unlock()

	t.lastTakeAt = time.Now()
}

func (t *TokenBucket) LastTakeAt() time.Time {
	return t.lastTakeAt
}

func (t *TokenBucket) AllowTake() bool {
	t.mu.RLock()
	curSize := t.size
	t.mu.RUnlock()

	if curSize > 0 {
		t.Dec()
		return true
	}

	return false
}

func (t *TokenBucket) IsFull() bool {
	t.mu.RLock()
	if t.size == t.cap {
		return true
	}
	t.mu.RUnlock()

	return false
}
