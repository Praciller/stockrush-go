package ratelimit

import (
	"sync"
	"time"
)

const maxClients = 10000

type bucket struct {
	tokens float64
	last   time.Time
}

type Limiter struct {
	mu      sync.Mutex
	perSec  float64
	burst   float64
	buckets map[string]bucket
	now     func() time.Time
}

func New(requestsPerSecond, burst int) *Limiter {
	return &Limiter{perSec: float64(requestsPerSecond), burst: float64(burst), buckets: make(map[string]bucket), now: time.Now}
}

func (l *Limiter) Allow(client string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, ok := l.buckets[client]
	if !ok {
		if len(l.buckets) >= maxClients {
			l.removeIdle(now.Add(-10 * time.Minute))
			if len(l.buckets) >= maxClients {
				return false
			}
		}
		b = bucket{tokens: l.burst, last: now}
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.perSec)
	b.last = now
	if b.tokens < 1 {
		l.buckets[client] = b
		return false
	}
	b.tokens--
	l.buckets[client] = b
	return true
}

func (l *Limiter) removeIdle(before time.Time) {
	for client, b := range l.buckets {
		if b.last.Before(before) {
			delete(l.buckets, client)
		}
	}
}
