// Package ratelimit provides a small fixed-window limiter for the one endpoint
// that accepts writes without authentication.
package ratelimit

import (
	"sync"
	"time"
)

type entry struct {
	count int
	reset time.Time
}

// Limiter allows n requests per key per window. It holds counters in memory:
// a single API process serves this site, so a shared store would be
// infrastructure without a purpose.
type Limiter struct {
	mu      sync.Mutex
	entries map[string]*entry
	limit   int
	window  time.Duration
}

func New(limit int, window time.Duration) *Limiter {
	l := &Limiter{
		entries: make(map[string]*entry),
		limit:   limit,
		window:  window,
	}
	go l.cleanupLoop()
	return l
}

// Allow reports whether the key may proceed, and how long until it may retry.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	e, ok := l.entries[key]
	if !ok || now.After(e.reset) {
		l.entries[key] = &entry{count: 1, reset: now.Add(l.window)}
		return true, 0
	}

	if e.count >= l.limit {
		return false, time.Until(e.reset)
	}

	e.count++
	return true, 0
}

// cleanupLoop drops expired windows so the map cannot grow without bound.
func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for k, e := range l.entries {
			if now.After(e.reset) {
				delete(l.entries, k)
			}
		}
		l.mu.Unlock()
	}
}
