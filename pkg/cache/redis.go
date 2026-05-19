package cache

import (
	"context"
	"fmt"
	"time"
)

// RedisClient interface for Redis operations
// Applications can provide their own Redis client implementation
// This allows the cache package to work with any Redis client library
type RedisClient interface {
	// Get retrieves a value by key
	Get(ctx context.Context, key string) (string, error)

	// Set stores a value by key with expiration
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// Del removes one or more keys
	Del(ctx context.Context, keys ...string) error

	// FlushDB removes all keys from the current database
	FlushDB(ctx context.Context) error

	// Exists checks if one or more keys exist
	Exists(ctx context.Context, keys ...string) (int64, error)

	// TTL returns the remaining TTL for a key
	TTL(ctx context.Context, key string) (time.Duration, error)
}

// RedisCache is a Redis implementation of Cache
// This requires the redis client to be provided
type RedisCache struct {
	client RedisClient
	prefix string
}

// NewRedisCache creates a new Redis cache
// client: Redis client implementation (can be nil if Redis is not available)
// prefix: Key prefix for all cache keys (e.g., "cache:")
func NewRedisCache(client RedisClient, prefix string) *RedisCache {
	if prefix == "" {
		prefix = "cache:"
	}
	return &RedisCache{
		client: client,
		prefix: prefix,
	}
}

// IsAvailable returns whether Redis cache is available
func (rc *RedisCache) IsAvailable() bool {
	return rc.client != nil
}

// Get retrieves a value from Redis cache
// Fail-fast: Validates inputs and availability before processing
func (rc *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return nil, err
	}

	if !rc.IsAvailable() {
		return nil, fmt.Errorf("fail-fast: redis cache not available")
	}

	fullKey := rc.prefix + key
	value, err := rc.client.Get(ctx, fullKey)
	if err != nil {
		return nil, fmt.Errorf("fail-fast: redis get failed for key %s: %w", fullKey, err)
	}
	return []byte(value), nil
}

// Set stores a value in Redis cache
// Fail-fast: Validates inputs and availability before processing
func (rc *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
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

	if !rc.IsAvailable() {
		return fmt.Errorf("fail-fast: redis cache not available")
	}

	fullKey := rc.prefix + key
	if err := rc.client.Set(ctx, fullKey, string(value), ttl); err != nil {
		return fmt.Errorf("fail-fast: redis set failed for key %s: %w", fullKey, err)
	}
	return nil
}

// Delete removes a value from Redis cache
// Fail-fast: Validates inputs and availability before processing
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return err
	}

	if !rc.IsAvailable() {
		return fmt.Errorf("fail-fast: redis cache not available")
	}

	fullKey := rc.prefix + key
	if err := rc.client.Del(ctx, fullKey); err != nil {
		return fmt.Errorf("fail-fast: redis delete failed for key %s: %w", fullKey, err)
	}
	return nil
}

// Clear removes all values from Redis cache
// Fail-fast: Validates context and availability before processing
func (rc *RedisCache) Clear(ctx context.Context) error {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	if !rc.IsAvailable() {
		return fmt.Errorf("fail-fast: redis cache not available")
	}

	if err := rc.client.FlushDB(ctx); err != nil {
		return fmt.Errorf("fail-fast: redis flush failed: %w", err)
	}
	return nil
}

// Exists checks if a key exists in Redis cache
// Fail-fast: Validates inputs and availability before processing
func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return false, err
	}

	if !rc.IsAvailable() {
		return false, fmt.Errorf("fail-fast: redis cache not available")
	}

	fullKey := rc.prefix + key
	count, err := rc.client.Exists(ctx, fullKey)
	if err != nil {
		return false, fmt.Errorf("fail-fast: redis exists failed for key %s: %w", fullKey, err)
	}
	return count > 0, nil
}

// GetTTL returns the remaining TTL for a key
// Fail-fast: Validates inputs and availability before processing
func (rc *RedisCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	// Fail-fast: Validate context
	ValidateContext(ctx)

	// Fail-fast: Validate key
	if err := ValidateKey(key); err != nil {
		return 0, err
	}

	if !rc.IsAvailable() {
		return 0, fmt.Errorf("fail-fast: redis cache not available")
	}

	fullKey := rc.prefix + key
	ttl, err := rc.client.TTL(ctx, fullKey)
	if err != nil {
		return 0, fmt.Errorf("fail-fast: redis TTL failed for key %s: %w", fullKey, err)
	}
	return ttl, nil
}
