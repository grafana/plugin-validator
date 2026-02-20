package storage

import (
	"sync"
	"time"
)

type CacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
}

type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]CacheEntry
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		entries: make(map[string]CacheEntry),
	}
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	return entry.Value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (c *MemoryCache) Evict(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}
