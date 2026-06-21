package cache

import (
	"sync"
	"time"
)

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

// NewMemoryCache creates a new in-memory cache and starts a background cleanup loop.
func NewMemoryCache(cleanupInterval time.Duration) *MemoryCache {
	c := &MemoryCache{
		items: make(map[string]cacheItem),
	}
	go c.startCleanupLoop(cleanupInterval)
	return c
}

// Get retrieves an item from the cache if it exists and has not expired.
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.items[key]
	if !found {
		return nil, false
	}

	if time.Now().After(item.expiration) {
		return nil, false
	}

	return item.value, true
}

// Set adds or replaces an item in the cache with a specified TTL.
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(ttl),
	}
}

func (c *MemoryCache) startCleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.expiration) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
