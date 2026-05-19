package cache

import (
	"context"
	"fmt"
	"time"
)

// MultiCache combines multiple caches (e.g., memory + Redis)
// Reads from all caches in order, writes to all caches
// Useful for implementing cache hierarchies (L1: memory, L2: Redis)
type MultiCache struct {
	caches []Cache
}

// NewMultiCache creates a new multi-cache that uses multiple cache backends
// Fail-fast: Validates that at least one cache is provided
func NewMultiCache(caches ...Cache) *MultiCache {
	// Fail-fast: must have at least one cache
	if len(caches) == 0 {
		panic(fmt.Errorf("fail-fast: MultiCache must have at least one cache"))
	}

	// Fail-fast: no cache can be nil
	for i, cache := range caches {
		if cache == nil {
			panic(fmt.Errorf("fail-fast: cache at index %d cannot be nil", i))
		}
	}

	return &MultiCache{
		caches: caches,
	}
}

// Get retrieves from the first available cache
// If found, propagates the value to other caches
// Fail-fast: Validates inputs before processing
func (mc *MultiCache) Get(ctx context.Context, key string) ([]byte, error) {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return nil, err
	}

	var lastErr error
	for i, cache := range mc.caches {
		// Fail-fast: cache cannot be nil (shouldn't happen after construction, but check anyway)
		if cache == nil {
			return nil, fmt.Errorf("fail-fast: cache at index %d is nil", i)
		}

		value, err := cache.Get(ctx, key)
		if err == nil {
			// Found in this cache, propagate to other caches asynchronously
			go mc.propagateToOthers(ctx, key, value, cache)
			return value, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// Set stores in all caches
// Fail-fast: Validates inputs before processing
func (mc *MultiCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
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

	var lastErr error
	for i, cache := range mc.caches {
		// Fail-fast: cache cannot be nil
		if cache == nil {
			return fmt.Errorf("fail-fast: cache at index %d is nil", i)
		}

		if err := cache.Set(ctx, key, value, ttl); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Delete removes from all caches
// Fail-fast: Validates inputs before processing
func (mc *MultiCache) Delete(ctx context.Context, key string) error {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return err
	}

	var lastErr error
	for i, cache := range mc.caches {
		// Fail-fast: cache cannot be nil
		if cache == nil {
			return fmt.Errorf("fail-fast: cache at index %d is nil", i)
		}

		if err := cache.Delete(ctx, key); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Clear clears all caches
// Fail-fast: Validates context before processing
func (mc *MultiCache) Clear(ctx context.Context) error {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	var lastErr error
	for i, cache := range mc.caches {
		// Fail-fast: cache cannot be nil
		if cache == nil {
			return fmt.Errorf("fail-fast: cache at index %d is nil", i)
		}

		if err := cache.Clear(ctx); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Exists checks if a key exists in any cache
// Returns true if key exists in at least one cache
// Fail-fast: Validates inputs before processing
func (mc *MultiCache) Exists(ctx context.Context, key string) (bool, error) {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return false, err
	}

	for i, cache := range mc.caches {
		// Fail-fast: cache cannot be nil
		if cache == nil {
			return false, fmt.Errorf("fail-fast: cache at index %d is nil", i)
		}

		exists, err := cache.Exists(ctx, key)
		if err == nil && exists {
			return true, nil
		}
	}
	return false, nil
}

// GetTTL returns the remaining TTL for a key from the first cache that has it
// Fail-fast: Validates inputs before processing
func (mc *MultiCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return 0, err
	}

	var lastErr error
	for i, cache := range mc.caches {
		// Fail-fast: cache cannot be nil
		if cache == nil {
			return 0, fmt.Errorf("fail-fast: cache at index %d is nil", i)
		}

		ttl, err := cache.GetTTL(ctx, key)
		if err == nil {
			return ttl, nil
		}
		lastErr = err
	}
	return 0, lastErr
}

// propagateToOthers propagates a cache hit to other caches
// This is called asynchronously to avoid blocking the Get operation
func (mc *MultiCache) propagateToOthers(ctx context.Context, key string, value []byte, source Cache) {
	for _, cache := range mc.caches {
		if cache == source {
			continue
		}
		// Use background context for propagation to avoid cancellation
		_ = cache.Set(context.Background(), key, value, 0) // No TTL for propagation
	}
}
