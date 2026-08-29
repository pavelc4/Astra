// Package cache is a bounded in-memory result cache for scrape responses.
//
// Entries carry two horizons: a fresh window (served directly) and a longer
// stale window (served only as last-known-good when a refresh fails — see the
// handler's serve-stale path). The bound caps memory so a flood of unique URLs
// can't grow the map without limit.
package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

type State int

const (
	Miss  State = iota // absent or past the stale window
	Fresh              // within the fresh window — serve directly
	Stale              // fresh window passed, still serveable as last-known-good
)

type entry struct {
	data       any
	err        error
	freshUntil time.Time
	staleUntil time.Time
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]entry
	max   int

	hits      atomic.Uint64
	staleHits atomic.Uint64
	misses    atomic.Uint64
}

// New builds a cache holding at most maxEntries, sweeping expired entries every
// cleanupInterval.
func New(maxEntries int, cleanupInterval time.Duration) *Cache {
	c := &Cache{items: make(map[string]entry), max: maxEntries}
	go c.cleanupLoop(cleanupInterval)
	return c
}

// Get reports whether key is Fresh, Stale or Miss, returning the stored data/err
// for Fresh and Stale.
func (c *Cache) Get(key string) (data any, err error, state State) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		c.misses.Add(1)
		return nil, nil, Miss
	}

	now := time.Now()
	switch {
	case now.Before(e.freshUntil):
		c.hits.Add(1)
		return e.data, e.err, Fresh
	case now.Before(e.staleUntil):
		c.staleHits.Add(1)
		return e.data, e.err, Stale
	default:
		c.misses.Add(1)
		return nil, nil, Miss
	}
}

// SetOK caches a successful result: fresh for freshTTL, then serveable-stale
// until staleTTL.
func (c *Cache) SetOK(key string, data any, freshTTL, staleTTL time.Duration) {
	now := time.Now()
	c.set(key, entry{data: data, freshUntil: now.Add(freshTTL), staleUntil: now.Add(staleTTL)})
}

// SetErr negatively caches a failure. Errors are never served stale, so the
// fresh and stale horizons coincide.
func (c *Cache) SetErr(key string, err error, ttl time.Duration) {
	now := time.Now()
	c.set(key, entry{err: err, freshUntil: now.Add(ttl), staleUntil: now.Add(ttl)})
}

func (c *Cache) set(key string, e entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.items[key]; !exists && len(c.items) >= c.max {
		c.evictLocked()
	}
	c.items[key] = e
}

// evictLocked bounds memory: drop expired entries first, then — if still at
// capacity — the entry closest to death.
//
// ponytail: O(n) scan, and only when full (rare: entries expire on their own via
// cleanupLoop). Swap for a linked-list LRU if Set-at-capacity shows up in a
// profile.
func (c *Cache) evictLocked() {
	now := time.Now()
	for k, v := range c.items {
		if now.After(v.staleUntil) {
			delete(c.items, k)
		}
	}
	if len(c.items) < c.max {
		return
	}
	var oldestKey string
	var oldest time.Time
	for k, v := range c.items {
		if oldestKey == "" || v.staleUntil.Before(oldest) {
			oldestKey, oldest = k, v.staleUntil
		}
	}
	delete(c.items, oldestKey)
}

// Stats returns cumulative fresh hits, stale hits and misses.
func (c *Cache) Stats() (hits, stale, misses uint64) {
	return c.hits.Load(), c.staleHits.Load(), c.misses.Load()
}

func (c *Cache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, v := range c.items {
			if now.After(v.staleUntil) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
