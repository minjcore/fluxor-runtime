# WordPress Client

Go client for WordPress REST API with built-in caching support.

## Features

- ✅ WordPress REST API integration
- ✅ In-memory caching with configurable TTL
- ✅ Thread-safe operations
- ✅ Automatic cache key generation
- ✅ Support for posts, categories, and more
- ✅ **Image proxy with caching**

## Usage

### Basic Usage

```go
import "github.com/fluxorio/fluxor/pkg/wordpress"

// Create client
client := wordpress.NewClient("https://your-site.com")
client.SetTimeout(10 * time.Second)

// Fetch posts
posts, err := client.GetPosts(ctx, &wordpress.GetPostsOptions{
    Page:    1,
    PerPage: 10,
})
```

### With Caching

```go
import (
    "github.com/fluxorio/fluxor/pkg/cache"
    "github.com/fluxorio/fluxor/pkg/wordpress"
)

// Create cache
wpCache := cache.NewMemoryCache()

// Create client with cache
client := wordpress.NewClient("https://your-site.com")
client.SetCache(wpCache)
client.SetCacheTTL(5 * time.Minute) // 5 minutes TTL

// Fetch posts (will be cached)
posts, err := client.GetPosts(ctx, nil)
```

## API Methods

### GetPosts

Fetch posts from WordPress:

```go
opts := &wordpress.GetPostsOptions{
    Page:      1,
    PerPage:   10,
    Search:    "keyword",
    Categories: []int{1, 2},
    Tags:      []int{3},
    OrderBy:   "date",
    Order:     "desc",
}
posts, err := client.GetPosts(ctx, opts)
```

### GetPost

Fetch a single post by ID:

```go
post, err := client.GetPost(ctx, 12345)
```

### GetCategories

Fetch all categories:

```go
categories, err := client.GetCategories(ctx)
```

## Image Proxy

### Basic Usage

```go
import "github.com/fluxorio/fluxor/pkg/wordpress"

// Create image proxy
imageProxy := wordpress.NewImageProxy("https://your-site.com")
imageProxy.SetTimeout(30 * time.Second)

// With caching
imageProxy.SetCache(wpCache)
imageProxy.SetCacheTTL(24 * time.Hour) // 24 hours for images

// Proxy an image
imageData, contentType, err := imageProxy.ProxyImage(ctx, "/wp-content/uploads/2024/01/image.jpg")
```

### Integration with FastHTTP Router

```go
router := server.FastRouter()

// Register image proxy routes
router.GETFast("/wp-images/*", imageProxy.HandleFastHTTP)
router.GETFast("/wp-content/*", imageProxy.HandleFastHTTP)
```

### Image Proxy Features

- ✅ **Proxy images from WordPress** - Serve images through your backend
- ✅ **Image caching** - Cache images for 24 hours (configurable)
- ✅ **Content-type detection** - Automatically detects image type
- ✅ **Cache headers** - Sets proper HTTP cache headers
- ✅ **Full URL support** - Supports both relative paths and full URLs

### Image URL Conversion

Convert WordPress image URLs to proxy URLs:

```go
// WordPress image URL
wpURL := "https://your-site.com/wp-content/uploads/2024/01/image.jpg"

// Convert to proxy URL
proxyURL := imageProxy.GetImageURL(wpURL)
// Returns: "/wp-images/wp-content/uploads/2024/01/image.jpg"
```

## Caching

### Cache Configuration

- **Default TTL**: 5 minutes (posts), 30 minutes (categories)
- **Image TTL**: 24 hours (images)
- **Cache Key**: SHA256 hash of endpoint + options

### Cache Behavior

- First request: Fetches from WordPress API, stores in cache
- Subsequent requests: Returns from cache (if not expired)
- Cache expiration: Automatic cleanup when TTL expires

### Cache Statistics

If using `cache.MemoryCache` with stats:

```go
if cacheWithStats, ok := wpCache.(cache.CacheWithStats); ok {
    stats := cacheWithStats.Stats()
    fmt.Printf("Hits: %d, Misses: %d\n", stats.Hits, stats.Misses)
}
```

## Configuration

### Timeout

```go
client.SetTimeout(30 * time.Second)
imageProxy.SetTimeout(30 * time.Second)
```

### Cache TTL

```go
client.SetCacheTTL(10 * time.Minute)
imageProxy.SetCacheTTL(24 * time.Hour)
```

### Custom Cache Backend

```go
// Use any cache.Cache implementation
client.SetCache(myCustomCache)
imageProxy.SetCache(myCustomCache)
```

## Error Handling

All methods return errors for:
- Network failures
- Invalid responses
- API errors (non-200 status codes)
- JSON parsing errors

## Thread Safety

The client and image proxy are **thread-safe** and can be used concurrently from multiple goroutines.

## Examples

See `cmd/ssr-example/main.go` for a complete integration example with:
- WordPress REST API client
- Image proxy
- Caching
- FastHTTP server integration
