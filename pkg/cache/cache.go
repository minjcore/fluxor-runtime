package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// Cache provides a generic interface for caching operations
// This interface can be implemented by various backends (memory, Redis, etc.)
type Cache interface {
	// Get retrieves a value by key
	// Returns error if key not found or expired
	Get(ctx context.Context, key string) ([]byte, error)

	// Set stores a value by key with optional TTL
	// TTL of 0 means no expiration
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a value by key
	Delete(ctx context.Context, key string) error

	// Clear removes all cached values
	Clear(ctx context.Context) error

	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)

	// GetTTL returns the remaining TTL for a key
	// Returns 0 if key doesn't exist or has no expiration
	GetTTL(ctx context.Context, key string) (time.Duration, error)
}

// Stats provides cache statistics
type Stats struct {
	// Hits is the number of successful cache hits
	Hits int64

	// Misses is the number of cache misses
	Misses int64

	// Size is the current number of entries in the cache
	Size int64

	// MemoryUsage is the approximate memory usage in bytes (if available)
	MemoryUsage int64
}

// CacheWithStats extends Cache with statistics
type CacheWithStats interface {
	Cache
	// Stats returns cache statistics
	Stats() Stats
}

// ValidateKey validates a cache key
// Fail-fast: Returns error if key is invalid
func ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("fail-fast: cache key cannot be empty")
	}
	if len(key) > 250 {
		return fmt.Errorf("fail-fast: cache key too long (max 250 characters), got %d", len(key))
	}
	return nil
}

// ValidateTTL validates a TTL value
// Fail-fast: Returns error if TTL is invalid
func ValidateTTL(ttl time.Duration) error {
	if ttl < 0 {
		return fmt.Errorf("fail-fast: cache TTL cannot be negative: %v", ttl)
	}
	return nil
}

// ValidateContext validates a context
// Fail-fast: Panics if context is nil
func ValidateContext(ctx context.Context) {
	failfast.NotNil(ctx, "context")
}
