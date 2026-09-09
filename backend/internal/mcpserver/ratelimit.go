package mcpserver

import (
	"crypto/sha256"
	"sync"
	"time"
)

type fixedWindowLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[[32]byte]rateBucket
}

type rateBucket struct {
	started time.Time
	count   int
}

func newLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{limit: limit, window: window, buckets: make(map[[32]byte]rateBucket)}
}

func (l *fixedWindowLimiter) allow(token string) bool {
	key := sha256.Sum256([]byte(token))
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.buckets) > 4096 {
		for candidate, bucket := range l.buckets {
			if now.Sub(bucket.started) >= 2*l.window {
				delete(l.buckets, candidate)
			}
		}
	}

	bucket := l.buckets[key]
	if bucket.started.IsZero() || now.Sub(bucket.started) >= l.window {
		l.buckets[key] = rateBucket{started: now, count: 1}
		return true
	}
	if bucket.count >= l.limit {
		return false
	}
	bucket.count++
	l.buckets[key] = bucket
	return true
}
