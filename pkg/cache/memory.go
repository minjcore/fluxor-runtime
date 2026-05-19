package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryCache is an in-memory implementation of Cache
// Thread-safe and suitable for single-instance applications
type MemoryCache struct {
	data  map[string]*cacheEntry
	mu    sync.RWMutex
	stats *memoryStats
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
	createdAt time.Time
}

type memoryStats struct {
	hits   int64
	misses int64
	mu     sync.RWMutex
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		data:  make(map[string]*cacheEntry),
		stats: &memoryStats{},
	}
}

// Get retrieves a value from the cache
// Fail-fast: Validates inputs before processing
func (mc *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		mc.recordMiss()
		return nil, err
	}

	mc.mu.RLock()
	entry, exists := mc.data[key]
	mc.mu.RUnlock()

	if !exists {
		mc.recordMiss()
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// Check if expired
	now := time.Now()
	if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
		// Entry expired, delete it
		mc.mu.Lock()
		delete(mc.data, key)
		mc.mu.Unlock()
		mc.recordMiss()
		return nil, fmt.Errorf("key expired: %s", key)
	}

	// Return a copy of the value
	result := make([]byte, len(entry.value))
	copy(result, entry.value)
	mc.recordHit()
	return result, nil
}

// Set stores a value in the cache
// Fail-fast: Validates inputs before processing
func (mc *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return err
	}

	// Fail-fast: Validate value
	if value == nil {
		return fmt.Errorf("fail-fast: cache value cannot be nil")
	}

	// Fail-fast: Validate TTL
	if err := ValidateTTL(ttl); err != nil {
		return err
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	entry := &cacheEntry{
		value:     make([]byte, len(value)),
		createdAt: time.Now(),
	}
	copy(entry.value, value)

	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}

	mc.data[key] = entry
	return nil
}

// Delete removes a value from the cache
// Fail-fast: Validates inputs before processing
func (mc *MemoryCache) Delete(ctx context.Context, key string) error {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return err
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()
	delete(mc.data, key)
	return nil
}

// Clear removes all values from the cache
// Fail-fast: Validates context
func (mc *MemoryCache) Clear(ctx context.Context) error {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.data = make(map[string]*cacheEntry)
	return nil
}

// Exists checks if a key exists in the cache
// Fail-fast: Validates inputs before processing
func (mc *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return false, err
	}

	mc.mu.RLock()
	defer mc.mu.RUnlock()

	entry, exists := mc.data[key]
	if !exists {
		return false, nil
	}

	// Check if expired
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return false, nil
	}

	return true, nil
}

// GetTTL returns the remaining TTL for a key
// Fail-fast: Validates inputs before processing
func (mc *MemoryCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return 0, err
	}

	mc.mu.RLock()
	defer mc.mu.RUnlock()

	entry, exists := mc.data[key]
	if !exists {
		return 0, fmt.Errorf("key not found: %s", key)
	}

	// Check if expired
	if !entry.expiresAt.IsZero() {
		now := time.Now()
		if now.After(entry.expiresAt) {
			return 0, fmt.Errorf("key expired: %s", key)
		}
		return entry.expiresAt.Sub(now), nil
	}

	// No expiration
	return 0, nil
}

// Stats returns cache statistics
func (mc *MemoryCache) Stats() Stats {
	mc.mu.RLock()
	size := int64(len(mc.data))
	mc.mu.RUnlock()

	mc.stats.mu.RLock()
	hits := mc.stats.hits
	misses := mc.stats.misses
	mc.stats.mu.RUnlock()

	// Calculate approximate memory usage
	mc.mu.RLock()
	memoryUsage := int64(0)
	for _, entry := range mc.data {
		memoryUsage += int64(len(entry.value))
		memoryUsage += 100 // Approximate overhead per entry
	}
	mc.mu.RUnlock()

	return Stats{
		Hits:        hits,
		Misses:      misses,
		Size:        size,
		MemoryUsage: memoryUsage,
	}
}

// recordHit records a cache hit
func (mc *MemoryCache) recordHit() {
	mc.stats.mu.Lock()
	mc.stats.hits++
	mc.stats.mu.Unlock()
}

// recordMiss records a cache miss
func (mc *MemoryCache) recordMiss() {
	mc.stats.mu.Lock()
	mc.stats.misses++
	mc.stats.mu.Unlock()
}
