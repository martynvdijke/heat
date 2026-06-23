package racing

import (
	"sync"
	"time"
)

// CacheItem holds a cached value with expiry.
type CacheItem struct {
	Data      any
	ExpiresAt time.Time
}

// Cache is a simple TTL-based in-memory cache.
type Cache struct {
	mu    sync.RWMutex
	items map[string]*CacheItem
	ttl   time.Duration
}

// NewCache creates a new cache with the given default TTL.
func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]*CacheItem),
		ttl:   ttl,
	}
	go c.cleanup()
	return c
}

// Get retrieves a value from the cache. Returns nil, false if cache is nil.
func (c *Cache) Get(key string) (any, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(item.ExpiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}
	return item.Data, true
}

// Set stores a value in the cache with the default TTL. No-op if cache is nil.
func (c *Cache) Set(key string, data any) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.items[key] = &CacheItem{
		Data:      data,
		ExpiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// Invalidate removes a key from the cache. No-op if cache is nil.
func (c *Cache) Invalidate(key string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// InvalidatePrefix removes all keys with the given prefix. No-op if cache is nil.
func (c *Cache) InvalidatePrefix(prefix string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	for k := range c.items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

// Clear empties the cache. No-op if cache is nil.
func (c *Cache) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.items = make(map[string]*CacheItem)
	c.mu.Unlock()
}

// cleanup periodically removes expired items.
func (c *Cache) cleanup() {
	if c == nil || c.ttl <= 0 {
		return
	}
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, v := range c.items {
			if now.After(v.ExpiresAt) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
