package pokecache

import (
	"sync"
	"time"
)

type cacheEntry struct {
	createdAt time.Time
	val       []byte
}

type Cache struct {
	cacheData map[string]cacheEntry
	mu        sync.Mutex
}

func (c *Cache) Add(key string, val []byte) {
	c.mu.Lock()
	c.cacheData[key] = cacheEntry{createdAt: time.Now(),
		val: val,
	}
	c.mu.Unlock()
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	index, ok := c.cacheData[key]
	if ok {
		return index.val, true
	} else {
		return []byte{}, false
	}
}

func (c *Cache) reapLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		for key, entry := range c.cacheData {
			if time.Since(entry.createdAt) > interval {
				delete(c.cacheData, key)
			}
		}
		c.mu.Unlock()
	}
}

func NewCache(interval time.Duration) *Cache {
	var cache Cache
	cache.cacheData = make(map[string]cacheEntry)
	go cache.reapLoop(interval)
	return &cache
}
