package config

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// Cache is an alias for cache.Cache for backward compatibility
// Use cache.Cache directly for new code
type Cache = cache.Cache

// MemoryCache is an alias for cache.MemoryCache for backward compatibility
// Use cache.NewMemoryCache() for new code
type MemoryCache = cache.MemoryCache

// RedisCache is an alias for cache.RedisCache for backward compatibility
// Use cache.NewRedisCache() for new code
type RedisCache = cache.RedisCache

// RedisClient is an alias for cache.RedisClient for backward compatibility
// Use cache.RedisClient for new code
type RedisClient = cache.RedisClient

// MultiCache is an alias for cache.MultiCache for backward compatibility
// Use cache.NewMultiCache() for new code
type MultiCache = cache.MultiCache

// NewMemoryCache creates a new in-memory cache (backward compatibility)
// Use cache.NewMemoryCache() for new code
func NewMemoryCache() *MemoryCache {
	return cache.NewMemoryCache()
}

// NewRedisCache creates a new Redis cache (backward compatibility)
// Use cache.NewRedisCache() for new code
func NewRedisCache(client RedisClient, prefix string) *RedisCache {
	return cache.NewRedisCache(client, prefix)
}

// NewMultiCache creates a new multi-cache (backward compatibility)
// Use cache.NewMultiCache() for new code
func NewMultiCache(caches ...Cache) *MultiCache {
	return cache.NewMultiCache(caches...)
}

// LoadWithCache loads configuration with optional caching
// cache: Optional cache (can be nil to disable caching)
// cacheKey: Key to use for caching (defaults to file path)
// ttl: Time-to-live for cached config (0 = no expiration)
// Fail-fast: Validates inputs before processing
func LoadWithCache(path string, target interface{}, cache Cache, cacheKey string, ttl time.Duration) error {
	// Fail-fast: path cannot be empty
	if path == "" {
		return fmt.Errorf("fail-fast: config file path cannot be empty")
	}

	// Fail-fast: target cannot be nil
	failfast.NotNil(target, "target")

	// Fail-fast: if cache is provided, cacheKey cannot be empty
	if cache != nil && cacheKey == "" {
		return fmt.Errorf("fail-fast: cacheKey cannot be empty when cache is provided")
	}

	// Fail-fast: ttl cannot be negative
	if ttl < 0 {
		return fmt.Errorf("fail-fast: cache TTL cannot be negative: %v", ttl)
	}

	ctx := context.Background()

	// If cache is provided, try to load from cache first
	if cache != nil {
		if cacheKey == "" {
			cacheKey = path
		}

		cachedData, err := cache.Get(ctx, cacheKey)
		if err == nil {
			// Found in cache, unmarshal
			if err := json.Unmarshal(cachedData, target); err == nil {
				return nil
			}
			// Cache data corrupted, continue to load from file
		}
	}

	// Load from file
	if err := LoadProperties(path, target); err != nil {
		return err
	}

	// Store in cache if available
	if cache != nil {
		data, err := json.Marshal(target)
		if err == nil {
			_ = cache.Set(ctx, cacheKey, data, ttl)
		}
	}

	return nil
}

// LoadWithCacheAndEnv loads configuration with cache and environment variable overrides
// Fail-fast: Validates inputs before processing
func LoadWithCacheAndEnv(path string, prefix string, target interface{}, cache Cache, cacheKey string, ttl time.Duration) error {
	// Fail-fast: prefix cannot be empty
	if prefix == "" {
		return fmt.Errorf("fail-fast: environment variable prefix cannot be empty")
	}

	// Load with cache (validates path, target, cache, cacheKey, ttl)
	if err := LoadWithCache(path, target, cache, cacheKey, ttl); err != nil {
		return err
	}

	// Apply environment variable overrides
	return ApplyEnvOverrides(prefix, target)
}
