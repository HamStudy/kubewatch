package template

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Cache provides LRU caching for template execution results
type Cache struct {
	entries map[string]*cacheEntry
	order   []string
	maxSize int
	mu      sync.RWMutex
}

type cacheEntry struct {
	result    string
	timestamp time.Time
}

// NewCache creates a new cache with the specified max size
func NewCache(maxSize int) *Cache {
	return &Cache{
		entries: make(map[string]*cacheEntry),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

// Get retrieves a cached result
func (c *Cache) Get(template string, data interface{}) (string, bool) {
	key := c.makeKey(template, data)

	c.mu.RLock()
	defer c.mu.RUnlock()

	if entry, ok := c.entries[key]; ok {
		// Check if cache is still fresh (5 minutes)
		if time.Since(entry.timestamp) < 5*time.Minute {
			return entry.result, true
		}
	}

	return "", false
}

// Set stores a result in the cache
func (c *Cache) Set(template string, data interface{}, result string) {
	key := c.makeKey(template, data)

	c.mu.Lock()
	defer c.mu.Unlock()

	// If at capacity, remove oldest entry
	if len(c.entries) >= c.maxSize && c.maxSize > 0 {
		oldest := c.order[0]
		delete(c.entries, oldest)
		c.order = c.order[1:]
	}

	c.entries[key] = &cacheEntry{
		result:    result,
		timestamp: time.Now(),
	}
	c.order = append(c.order, key)
}

// Clear empties the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
	c.order = make([]string, 0, c.maxSize)
}

// makeKey creates a cache key from template and data
func (c *Cache) makeKey(template string, data interface{}) string {
	dataBytes, _ := json.Marshal(data)
	hash := md5.Sum(append([]byte(template), dataBytes...))
	return fmt.Sprintf("%x", hash)
}
