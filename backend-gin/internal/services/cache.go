package services

import (
	"sync"
	"time"
)

type Cache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

type cacheItem struct {
	value    any
	expireAt time.Time
}

func NewCache() *Cache {
	return &Cache{items: map[string]cacheItem{}}
}

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
	if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
		c.Delete(key)
		return nil, false
	}
	return item.value, true
}

func (c *Cache) Set(key string, value any, ttl time.Duration) {
	if c == nil {
		return
	}
	item := cacheItem{value: value}
	if ttl > 0 {
		item.expireAt = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.items[key] = item
	c.mu.Unlock()
}

func (c *Cache) Delete(key string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

// Take atomically returns and removes a non-expired cache entry.
func (c *Cache) Take(key string) (any, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	if !ok {
		return nil, false
	}
	delete(c.items, key)
	if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
		return nil, false
	}
	return item.value, true
}

func (c *Cache) InvalidatePrefix(prefix string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.items, key)
		}
	}
	c.mu.Unlock()
}

func (c *Cache) GetOrSet(key string, ttl time.Duration, loader func() (any, error)) (any, error) {
	if value, ok := c.Get(key); ok {
		return value, nil
	}
	value, err := loader()
	if err != nil {
		return nil, err
	}
	c.Set(key, value, ttl)
	return value, nil
}
