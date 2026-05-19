package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
)

func TestProxyCache_New(t *testing.T) {
	tmpDir := t.TempDir()
	c := cache.NewMemoryCache()

	pc, err := NewProxyCache(tmpDir, "https://proxy.golang.org", c, 1024*1024*1024)
	if err != nil {
		t.Fatalf("Failed to create proxy cache: %v", err)
	}

	if pc.cacheRoot != tmpDir {
		t.Errorf("Expected cache root %s, got %s", tmpDir, pc.cacheRoot)
	}

	if pc.upstreamURL != "https://proxy.golang.org" {
		t.Errorf("Expected upstream URL https://proxy.golang.org, got %s", pc.upstreamURL)
	}
}

func TestProxyCache_CachePath(t *testing.T) {
	tmpDir := t.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://proxy.golang.org", c, 1024*1024*1024)

	tests := []struct {
		name       string
		modulePath string
		wantPrefix string
	}{
		{
			name:       "simple path",
			modulePath: "github.com/gin-gonic/gin",
			wantPrefix: tmpDir,
		},
		{
			name:       "versioned path",
			modulePath: "github.com/gin-gonic/gin/@v/v1.9.1.info",
			wantPrefix: tmpDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := pc.cachePath(tt.modulePath)
			if !filepath.HasPrefix(path, tt.wantPrefix) {
				t.Errorf("Cache path %s doesn't start with %s", path, tt.wantPrefix)
			}
		})
	}
}

func TestProxyCache_CacheKey(t *testing.T) {
	tmpDir := t.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://proxy.golang.org", c, 1024*1024*1024)

	key := pc.cacheKey("test/module")
	if key != "pkg:test/module" {
		t.Errorf("Expected key 'pkg:test/module', got '%s'", key)
	}
}

func TestProxyCache_ServeFromCache_Miss(t *testing.T) {
	tmpDir := t.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://proxy.golang.org", c, 1024*1024*1024)

	req := httptest.NewRequest("GET", "/test/module", nil)
	w := httptest.NewRecorder()

	hit := pc.serveFromCache(w, req, "test/module")
	if hit {
		t.Error("Expected cache miss, got hit")
	}
}

func TestProxyCache_ServeFromCache_Hit(t *testing.T) {
	tmpDir := t.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://proxy.golang.org", c, 1024*1024*1024)

	// Add to cache
	ctx := context.Background()
	key := pc.cacheKey("test/module")
	testData := []byte("test data")
	c.Set(ctx, key, testData, time.Hour)

	req := httptest.NewRequest("GET", "/test/module", nil)
	w := httptest.NewRecorder()

	hit := pc.serveFromCache(w, req, "test/module")
	if !hit {
		t.Error("Expected cache hit, got miss")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("X-Cache") != "HIT" {
		t.Errorf("Expected X-Cache: HIT, got %s", w.Header().Get("X-Cache"))
	}
}

func TestProxyCache_ServeFromCache_DiskFallback(t *testing.T) {
	tmpDir := t.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://proxy.golang.org", c, 1024*1024*1024)

	// Write directly to disk cache
	modulePath := "test/module"
	cachePath := pc.cachePath(modulePath)
	testData := []byte("disk cache data")

	// Use restrictive permissions (0750) to prevent unauthorized access
	os.MkdirAll(filepath.Dir(cachePath), 0750)
	// Use restrictive permissions (0600) to prevent unauthorized access
	os.WriteFile(cachePath, testData, 0600)

	req := httptest.NewRequest("GET", "/test/module", nil)
	w := httptest.NewRecorder()

	hit := pc.serveFromCache(w, req, modulePath)
	if !hit {
		t.Error("Expected disk cache hit, got miss")
	}

	if w.Header().Get("X-Cache") != "HIT-DISK" {
		t.Errorf("Expected X-Cache: HIT-DISK, got %s", w.Header().Get("X-Cache"))
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"test.mod", "text/plain; charset=utf-8"},
		{"test.info", "application/json"},
		{"test.zip", "application/zip"},
		{"test.unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := detectContentType(tt.path)
			if got != tt.expected {
				t.Errorf("detectContentType(%s) = %s, want %s", tt.path, got, tt.expected)
			}
		})
	}
}

func TestProxyCache_StatsHandler(t *testing.T) {
	tmpDir := t.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://proxy.golang.org", c, 1024*1024*1024)

	// Set some stats
	pc.stats.Hits = 100
	pc.stats.Misses = 50
	pc.stats.Errors = 5

	req := httptest.NewRequest("GET", "/_stats", nil)
	w := httptest.NewRecorder()

	pc.statsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type: application/json, got %s", contentType)
	}
}

func TestProxyCache_HealthHandler(t *testing.T) {
	tmpDir := t.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://proxy.golang.org", c, 1024*1024*1024)

	req := httptest.NewRequest("GET", "/_health", nil)
	w := httptest.NewRecorder()

	pc.healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type: application/json, got %s", contentType)
	}
}

func BenchmarkCachePath(b *testing.B) {
	tmpDir := b.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://proxy.golang.org", c, 1024*1024*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pc.cachePath("github.com/gin-gonic/gin/@v/v1.9.1.info")
	}
}

func BenchmarkServeFromCache_Hit(b *testing.B) {
	tmpDir := b.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://proxy.golang.org", c, 1024*1024*1024)

	// Populate cache
	ctx := context.Background()
	key := pc.cacheKey("test/module")
	testData := []byte("test data")
	c.Set(ctx, key, testData, time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test/module", nil)
		w := httptest.NewRecorder()
		pc.serveFromCache(w, req, "test/module")
	}
}
