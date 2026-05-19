package wordpress

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
	"github.com/fluxorio/fluxor/pkg/web"
)

// ImageProxy handles proxying images from WordPress
type ImageProxy struct {
	wpBaseURL string        // WordPress base URL
	httpClient *http.Client // HTTP client for fetching images
	cache      cache.Cache  // Optional cache for images
	cacheTTL   time.Duration // Cache TTL for images (default: 24 hours)
	timeout    time.Duration // Request timeout
}

// NewImageProxy creates a new WordPress image proxy
func NewImageProxy(wpBaseURL string) *ImageProxy {
	// Normalize base URL (remove /wp-json if present)
	baseURL := strings.TrimSuffix(wpBaseURL, "/wp-json")
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &ImageProxy{
		wpBaseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Longer timeout for images
		},
		cacheTTL: 24 * time.Hour, // Cache images for 24 hours
		timeout:  30 * time.Second,
	}
}

// SetCache sets the cache backend for images
func (p *ImageProxy) SetCache(cacheBackend cache.Cache) {
	p.cache = cacheBackend
}

// SetCacheTTL sets the cache TTL for images (default: 24 hours)
func (p *ImageProxy) SetCacheTTL(ttl time.Duration) {
	p.cacheTTL = ttl
}

// SetTimeout sets the request timeout
func (p *ImageProxy) SetTimeout(timeout time.Duration) {
	p.timeout = timeout
	p.httpClient.Timeout = timeout
}

// ProxyImage proxies an image from WordPress
// imagePath should be relative to WordPress root (e.g., "/wp-content/uploads/2024/01/image.jpg")
// Or can be a full URL to a WordPress image
func (p *ImageProxy) ProxyImage(ctx context.Context, imagePath string) ([]byte, string, error) {
	// Check cache first
	if p.cache != nil {
		cacheKey := fmt.Sprintf("wp-img:%s", imagePath)
		if cached, err := p.cache.Get(ctx, cacheKey); err == nil {
			// Determine content type from file extension
			contentType := p.detectContentType(imagePath)
			return cached, contentType, nil
		}
	}

	// Build image URL
	var imageURL string
	if strings.HasPrefix(imagePath, "http://") || strings.HasPrefix(imagePath, "https://") {
		// Full URL provided
		imageURL = imagePath
	} else {
		// Relative path - construct full URL
		if !strings.HasPrefix(imagePath, "/") {
			imagePath = "/" + imagePath
		}
		imageURL = p.wpBaseURL + imagePath
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to mimic browser request
	req.Header.Set("User-Agent", "WordPress-Image-Proxy/1.0")
	req.Header.Set("Accept", "image/*")

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("WordPress image error: status %d", resp.StatusCode)
	}

	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image: %w", err)
	}

	// Detect content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = p.detectContentType(imagePath)
	}

	// Cache the image
	if p.cache != nil {
		cacheKey := fmt.Sprintf("wp-img:%s", imagePath)
		_ = p.cache.Set(ctx, cacheKey, imageData, p.cacheTTL)
	}

	return imageData, contentType, nil
}

// detectContentType detects content type from file extension
func (p *ImageProxy) detectContentType(imagePath string) string {
	ext := strings.ToLower(filepath.Ext(imagePath))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	default:
		return "image/jpeg" // Default
	}
}

// HandleFastHTTP handles FastHTTP request for image proxy
// Expected path format: /wp-images/* or /wp-content/*
func (p *ImageProxy) HandleFastHTTP(ctx *web.FastRequestContext) error {
	// Get image path from request
	path := string(ctx.Path())

	// Remove leading slash and route prefix if present
	imagePath := strings.TrimPrefix(path, "/wp-images")
	imagePath = strings.TrimPrefix(imagePath, "/wp-content")
	if imagePath == "" || imagePath == "/" {
		ctx.Error("Image path required", 400)
		return fmt.Errorf("image path required")
	}

	// If path starts with /wp-content, include it
	if !strings.HasPrefix(imagePath, "/wp-content") && !strings.HasPrefix(path, "/wp-images") {
		// Assume it's a WordPress content path
		if strings.HasPrefix(path, "/wp-content") {
			imagePath = path
		} else {
			// Add /wp-content prefix
			imagePath = "/wp-content" + imagePath
		}
	}

	// Get image from WordPress
	imageData, contentType, err := p.ProxyImage(ctx.Context(), imagePath)
	if err != nil {
		ctx.Error(fmt.Sprintf("Failed to fetch image: %v", err), 502)
		return err
	}

	// Set headers
	ctx.RequestCtx.SetContentType(contentType)
	ctx.RequestCtx.Response.Header.Set("Cache-Control", "public, max-age=86400") // 24 hours
	ctx.RequestCtx.Response.Header.Set("X-Cache-Status", "HIT") // Could be enhanced to detect cache hits

	// Set response body
	ctx.RequestCtx.SetBody(imageData)

	return nil
}

// GetImageURL constructs a proxy URL for a WordPress image
// Returns URL that will be handled by the proxy
func (p *ImageProxy) GetImageURL(imagePath string) string {
	// If it's already a full URL, extract path
	if strings.HasPrefix(imagePath, "http://") || strings.HasPrefix(imagePath, "https://") {
		u, err := url.Parse(imagePath)
		if err == nil {
			imagePath = u.Path
		}
	}

	// Return proxy URL
	if strings.HasPrefix(imagePath, "/wp-content") {
		return "/wp-images" + imagePath
	}
	return "/wp-images/wp-content" + imagePath
}
