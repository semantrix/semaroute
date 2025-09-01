package cache

import (
	"context"
	"time"
)

// CacheClient defines the interface for caching operations.
type CacheClient interface {
	// Get retrieves a value from the cache.
	Get(ctx context.Context, key string) (interface{}, bool, error)
	
	// Set stores a value in the cache with an optional TTL.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	
	// Delete removes a value from the cache.
	Delete(ctx context.Context, key string) error
	
	// Exists checks if a key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)
	
	// Clear removes all values from the cache.
	Clear(ctx context.Context) error
	
	// Close closes the cache client and releases resources.
	Close() error
}

// CacheConfig holds configuration for the cache.
type CacheConfig struct {
	Type        string        `mapstructure:"type"`        // memory, redis, etc.
	TTL         time.Duration `mapstructure:"ttl"`         // default TTL
	MaxSize     int           `mapstructure:"max_size"`    // maximum number of items
	MaxMemory   int64         `mapstructure:"max_memory"`  // maximum memory usage in bytes
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
}

// MemoryCache implements an in-memory cache client.
type MemoryCache struct {
	config CacheConfig
	data   map[string]*cacheItem
	// In production, this would use a proper LRU cache implementation
}

// cacheItem represents a cached item with metadata.
type cacheItem struct {
	Value      interface{}
	ExpiresAt  time.Time
	CreatedAt  time.Time
	AccessCount int64
}

// NewMemoryCache creates a new in-memory cache instance.
func NewMemoryCache(config CacheConfig) *MemoryCache {
	return &MemoryCache{
		config: config,
		data:   make(map[string]*cacheItem),
	}
}

// Get retrieves a value from the memory cache.
func (c *MemoryCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	item, exists := c.data[key]
	if !exists {
		return nil, false, nil
	}

	// Check if item has expired
	if time.Now().After(item.ExpiresAt) {
		delete(c.data, key)
		return nil, false, nil
	}

	// Update access count and return value
	item.AccessCount++
	return item.Value, true, nil
}

// Set stores a value in the memory cache.
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.config.TTL
	}

	item := &cacheItem{
		Value:      value,
		ExpiresAt:  time.Now().Add(ttl),
		CreatedAt:  time.Now(),
		AccessCount: 0,
	}

	c.data[key] = item

	// Simple cleanup: remove expired items if we're over the limit
	if len(c.data) > c.config.MaxSize {
		c.cleanup()
	}

	return nil
}

// Delete removes a value from the memory cache.
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	delete(c.data, key)
	return nil
}

// Exists checks if a key exists in the memory cache.
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	item, exists := c.data[key]
	if !exists {
		return false, nil
	}

	// Check if item has expired
	if time.Now().After(item.ExpiresAt) {
		delete(c.data, key)
		return false, nil
	}

	return true, nil
}

// Clear removes all values from the memory cache.
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.data = make(map[string]*cacheItem)
	return nil
}

// Close closes the memory cache.
func (c *MemoryCache) Close() error {
	c.data = nil
	return nil
}

// cleanup removes expired items from the cache.
func (c *MemoryCache) cleanup() {
	now := time.Now()
	for key, item := range c.data {
		if now.After(item.ExpiresAt) {
			delete(c.data, key)
		}
	}
}

// GetStats returns cache statistics.
func (c *MemoryCache) GetStats() map[string]interface{} {
	now := time.Now()
	expired := 0
	totalSize := 0

	for _, item := range c.data {
		if now.After(item.ExpiresAt) {
			expired++
		}
		totalSize++
	}

	return map[string]interface{}{
		"total_items":    len(c.data),
		"expired_items":  expired,
		"active_items":   totalSize - expired,
		"max_size":       c.config.MaxSize,
		"cleanup_needed": expired > 0,
	}
}
