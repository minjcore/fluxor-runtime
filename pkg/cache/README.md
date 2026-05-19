# Cache Package

A generic, high-performance caching package for the Fluxor framework with fail-fast validation and support for multiple backends.

## Features

- **Generic Interface**: Works with any data type (stores as `[]byte`)
- **Multiple Backends**: Memory, Redis, and Multi-cache (hierarchical)
- **Fail-Fast Validation**: All operations validate inputs before processing
- **Thread-Safe**: All implementations are safe for concurrent use
- **Statistics**: Memory cache includes hit/miss statistics
- **TTL Support**: Time-to-live expiration for cache entries
- **Context Support**: All operations accept `context.Context` for cancellation

## Quick Start

### Basic Usage

```go
import "github.com/fluxorio/fluxor/pkg/cache"

// Create a memory cache
memCache := cache.NewMemoryCache()

ctx := context.Background()

// Set a value with TTL
err := memCache.Set(ctx, "user:123", []byte("John Doe"), 5*time.Minute)
if err != nil {
    log.Fatal(err)
}

// Get a value
value, err := memCache.Get(ctx, "user:123")
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(value)) // "John Doe"

// Check if key exists
exists, err := memCache.Exists(ctx, "user:123")
if err != nil {
    log.Fatal(err)
}

// Get remaining TTL
ttl, err := memCache.GetTTL(ctx, "user:123")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("TTL: %v\n", ttl)

// Delete a key
err = memCache.Delete(ctx, "user:123")

// Clear all keys
err = memCache.Clear(ctx)
```

### Redis Cache

```go
import "github.com/fluxorio/fluxor/pkg/cache"

// Implement cache.RedisClient interface
type MyRedisClient struct {
    // Your Redis client implementation
}

func (c *MyRedisClient) Get(ctx context.Context, key string) (string, error) {
    // Implementation
}

func (c *MyRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
    // Implementation
}

func (c *MyRedisClient) Del(ctx context.Context, keys ...string) error {
    // Implementation
}

func (c *MyRedisClient) FlushDB(ctx context.Context) error {
    // Implementation
}

func (c *MyRedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
    // Implementation
}

func (c *MyRedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
    // Implementation
}

// Create Redis cache
redisClient := &MyRedisClient{}
redisCache := cache.NewRedisCache(redisClient, "app:")

// Use it like memory cache
err := redisCache.Set(ctx, "key", []byte("value"), 10*time.Minute)
```

### Multi-Cache (Hierarchical)

```go
import "github.com/fluxorio/fluxor/pkg/cache"

// Create L1 (memory) and L2 (Redis) caches
memCache := cache.NewMemoryCache()
redisCache := cache.NewRedisCache(redisClient, "app:")

// Combine into multi-cache
multiCache := cache.NewMultiCache(memCache, redisCache)

// Get: Tries L1 first, then L2, propagates to other caches on hit
value, err := multiCache.Get(ctx, "key")

// Set: Writes to all caches
err = multiCache.Set(ctx, "key", []byte("value"), 5*time.Minute)

// Delete: Removes from all caches
err = multiCache.Delete(ctx, "key")
```

### Cache Statistics

```go
// Memory cache supports statistics
memCache := cache.NewMemoryCache()

// ... use cache ...

stats := memCache.Stats()
fmt.Printf("Hits: %d, Misses: %d, Size: %d, Memory: %d bytes\n",
    stats.Hits, stats.Misses, stats.Size, stats.MemoryUsage)
```

## API Reference

### Cache Interface

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Clear(ctx context.Context) error
    Exists(ctx context.Context, key string) (bool, error)
    GetTTL(ctx context.Context, key string) (time.Duration, error)
}
```

### Validation Functions

```go
// ValidateKey validates a cache key
func ValidateKey(key string) error

// ValidateTTL validates a TTL value
func ValidateTTL(ttl time.Duration) error

// ValidateContext validates a context (panics if nil)
func ValidateContext(ctx context.Context)
```

## Fail-Fast Principles

All cache operations follow fail-fast principles:

- **Input Validation**: Keys, values, and TTLs are validated before processing
- **Nil Checks**: Context and cache instances are checked for nil
- **Immediate Errors**: Errors are returned immediately, not deferred
- **Clear Messages**: Error messages include context about what failed

Example:

```go
// This will fail-fast with a clear error
err := memCache.Set(ctx, "", []byte("value"), 5*time.Minute)
// Error: "fail-fast: cache key cannot be empty"

// This will fail-fast
err := memCache.Set(nil, "key", []byte("value"), 5*time.Minute)
// Panic: "fail-fast: context is nil"
```

## Best Practices

1. **Use Context**: Always pass a context for cancellation support
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   value, err := cache.Get(ctx, "key")
   ```

2. **Key Naming**: Use consistent key naming conventions
   ```go
   // Good
   "user:123"
   "session:abc"
   "config:app"
   
   // Avoid
   "user123"  // No namespace
   "user_123" // Inconsistent separator
   ```

3. **TTL Selection**: Choose appropriate TTLs based on data freshness requirements
   ```go
   // Frequently changing data
   cache.Set(ctx, "user:123", data, 1*time.Minute)
   
   // Rarely changing data
   cache.Set(ctx, "config:app", data, 1*time.Hour)
   
   // No expiration
   cache.Set(ctx, "static:data", data, 0)
   ```

4. **Error Handling**: Always check errors
   ```go
   value, err := cache.Get(ctx, "key")
   if err != nil {
       // Handle error (key not found, expired, etc.)
   }
   ```

5. **Multi-Cache for Performance**: Use multi-cache for best performance
   ```go
   // L1: Fast memory cache
   // L2: Distributed Redis cache
   multiCache := cache.NewMultiCache(memCache, redisCache)
   ```

## Thread Safety

All cache implementations are thread-safe and can be used concurrently from multiple goroutines.

## Performance Considerations

- **Memory Cache**: Fastest, but limited to single instance
- **Redis Cache**: Slower, but supports distributed caching
- **Multi-Cache**: Best of both worlds (fast local + distributed)

## Examples

See the `examples/` directory for complete examples of using the cache package.

