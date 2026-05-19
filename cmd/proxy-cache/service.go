// Package main provides the cache service layer
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
)

// ProxyCache handles package caching with fail-fast validation
type ProxyCache struct {
	cacheRoot    string
	upstreamURL  string
	cache        cache.Cache
	diskCache    *DiskCache
	stats        *CacheStats
	maxCacheSize int64
	cacheTTL     time.Duration
	upstreamTimeout time.Duration
	verbose      bool
}

// CacheStats tracks cache performance metrics with thread-safe atomic operations
type CacheStats struct {
	Hits   atomic.Int64
	Misses atomic.Int64
	Errors atomic.Int64
	Bytes  atomic.Int64
}

// NewProxyCache creates a new proxy cache instance
func NewProxyCache(cfg *Config, c cache.Cache) (*ProxyCache, error) {
	if cfg.CacheDir == "" {
		return nil, fmt.Errorf("fail-fast: cache root cannot be empty")
	}
	if cfg.Upstream == "" {
		return nil, fmt.Errorf("fail-fast: upstream URL cannot be empty")
	}

	// Use restrictive permissions (0750) to prevent unauthorized access
	if err := os.MkdirAll(cfg.CacheDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	diskCache := NewDiskCache(cfg.CacheDir, cfg.MaxCacheSize)

	return &ProxyCache{
		cacheRoot:       cfg.CacheDir,
		upstreamURL:     cfg.Upstream,
		cache:           c,
		diskCache:       diskCache,
		stats:           &CacheStats{},
		maxCacheSize:    cfg.MaxCacheSize,
		cacheTTL:        cfg.CacheTTL,
		upstreamTimeout: cfg.UpstreamTimeout,
		verbose:         cfg.Verbose,
	}, nil
}

// cachePath generates a deterministic cache path from module path
func (p *ProxyCache) cachePath(modulePath string) string {
	h := sha256.Sum256([]byte(modulePath))
	hash := hex.EncodeToString(h[:])
	// Create two-level directory structure for better filesystem performance
	return filepath.Join(p.cacheRoot, hash[:2], hash[2:4], hash)
}

// cacheKey generates a cache key for in-memory/redis cache
func (p *ProxyCache) cacheKey(modulePath string) string {
	return "pkg:" + modulePath
}

// ServeFromCache attempts to serve content from cache
func (p *ProxyCache) ServeFromCache(w http.ResponseWriter, r *http.Request, modulePath string) bool {
	ctx := r.Context()

	// Try memory/redis cache first (faster)
	key := p.cacheKey(modulePath)
	if data, err := p.cache.Get(ctx, key); err == nil {
		p.stats.Hits.Add(1)
		if p.verbose {
			log.Printf("CACHE HIT (memory): %s", modulePath)
		}
		w.Header().Set("X-Cache", "HIT")
		w.Header().Set("Content-Type", detectContentType(modulePath))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return true
	}

	// Try disk cache as fallback
	cachePath := p.cachePath(modulePath)
	if data, err := os.ReadFile(cachePath); err == nil {
		p.stats.Hits.Add(1)
		if p.verbose {
			log.Printf("CACHE HIT (disk): %s", modulePath)
		}

		// Populate memory cache for next request
		go func() {
			p.cache.Set(context.Background(), key, data, p.cacheTTL)
		}()

		w.Header().Set("X-Cache", "HIT-DISK")
		w.Header().Set("Content-Type", detectContentType(modulePath))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return true
	}

	return false
}

// FetchAndCache fetches from upstream and caches the response
func (p *ProxyCache) FetchAndCache(w http.ResponseWriter, r *http.Request, modulePath string) {
	upstreamURL := p.upstreamURL + r.URL.Path
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	if p.verbose {
		log.Printf("CACHE MISS → FETCH: %s", upstreamURL)
	}

	p.stats.Misses.Add(1)

	// Create request to upstream
	ctx, cancel := context.WithTimeout(r.Context(), p.upstreamTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, r.Method, upstreamURL, nil)
	if err != nil {
		p.stats.Errors.Add(1)
		log.Printf("ERROR creating request: %v", err)
		http.Error(w, "request creation failed", http.StatusInternalServerError)
		return
	}

	// Copy headers from original request
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Fetch from upstream
	client := &http.Client{Timeout: p.upstreamTimeout}
	resp, err := client.Do(req)
	if err != nil {
		p.stats.Errors.Add(1)
		log.Printf("ERROR fetching upstream: %v", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read response data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		p.stats.Errors.Add(1)
		log.Printf("ERROR reading response: %v", err)
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	// Only cache successful responses
	if resp.StatusCode == http.StatusOK {
		p.stats.Bytes.Add(int64(len(data)))

		// Save to disk cache
		cachePath := p.cachePath(modulePath)
		// Use restrictive permissions (0750) to prevent unauthorized access
		if err := os.MkdirAll(filepath.Dir(cachePath), 0750); err == nil {
			// Use restrictive permissions (0600) to prevent unauthorized access
			if err := os.WriteFile(cachePath, data, 0600); err != nil {
				log.Printf("ERROR writing disk cache: %v", err)
			} else {
				p.diskCache.Add(cachePath, int64(len(data)))
			}
		}

		// Save to memory/redis cache
		key := p.cacheKey(modulePath)
		if err := p.cache.Set(r.Context(), key, data, p.cacheTTL); err != nil {
			log.Printf("ERROR writing memory cache: %v", err)
		}
	}

	// Forward response to client
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(resp.StatusCode)
	w.Write(data)
}

// GetStats returns current cache statistics
func (p *ProxyCache) GetStats() map[string]interface{} {
	hits := p.stats.Hits.Load()
	misses := p.stats.Misses.Load()
	errors := p.stats.Errors.Load()
	bytes := p.stats.Bytes.Load()
	
	total := hits + misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"hits":             hits,
		"misses":           misses,
		"errors":           errors,
		"total_requests":   total,
		"hit_rate":         hitRate,
		"cache_bytes":      bytes,
		"disk_cache_size":  p.diskCache.Size(),
		"disk_cache_files": p.diskCache.FileCount(),
	}
}

// detectContentType determines content type from file path
func detectContentType(path string) string {
	if strings.HasSuffix(path, ".mod") {
		return "text/plain; charset=utf-8"
	}
	if strings.HasSuffix(path, ".info") {
		return "application/json"
	}
	if strings.HasSuffix(path, ".zip") {
		return "application/zip"
	}
	return "application/octet-stream"
}
