package cache

import (
	"errors"
	"sync"
)

// Cache represents a simple thread-safe in-memory key-value store.
type Cache struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewCache creates and returns a new Cache instance.
func NewCache() *Cache {
	return &Cache{
		data: make(map[string]interface{}),
	}
}

// Set inserts or updates the value for a given key.
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// Get retrieves the value for a given key. Returns an error if the key is not found.
func (c *Cache) Get(key string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists := c.data[key]
	if !exists {
		return nil, errors.New("key not found")
	}
	return value, nil
}

// Delete removes a key-value pair from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}
