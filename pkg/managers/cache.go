package managers

import (
	"errors"

	"github.com/fluxorio/fluxor/pkg/cache"
)

// CreateCache creates a cache based on Managers configuration
func (m *Managers) CreateCache() (cache.Cache, error) {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()

	switch config.CacheType {
	case "memory":
		return m.CreateMemoryCache(), nil
	case "redis":
		// Redis cache creation would require Redis client configuration
		// For now, return error if Redis is requested but not configured
		return nil, errors.New("Redis cache requires Redis client configuration (not yet implemented)")
	default:
		return m.CreateMemoryCache(), nil // Default to memory cache
	}
}

// CreateMemoryCache creates an in-memory cache
func (m *Managers) CreateMemoryCache() cache.Cache {
	return cache.NewMemoryCache()
}

// CreateRedisCache creates a Redis cache (requires Redis client)
// This is a placeholder - actual implementation would require Redis client configuration
func (m *Managers) CreateRedisCache(redisClient interface{}) (cache.Cache, error) {
	// TODO: Implement Redis cache creation when Redis client is available
	return nil, errors.New("Redis cache creation not yet implemented - requires Redis client")
}
