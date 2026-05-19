package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
)

// TestIntegration_ProxyCacheFlow tests the complete flow
func TestIntegration_ProxyCacheFlow(t *testing.T) {
	tmpDir := t.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://example.com", c, 1024*1024*1024)

	// Create upstream mock server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("upstream content"))
	}))
	defer upstream.Close()

	pc.upstreamURL = upstream.URL

	// First request - should hit upstream
	req1 := httptest.NewRequest("GET", "/test/package", nil)
	w1 := httptest.NewRecorder()
	pc.handleRequest(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w1.Code)
	}

	if w1.Header().Get("X-Cache") != "MISS" {
		t.Errorf("Expected X-Cache: MISS, got %s", w1.Header().Get("X-Cache"))
	}

	// Second request - should hit cache
	req2 := httptest.NewRequest("GET", "/test/package", nil)
	w2 := httptest.NewRecorder()
	pc.handleRequest(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w2.Code)
	}

	if w2.Header().Get("X-Cache") != "HIT" && w2.Header().Get("X-Cache") != "HIT-DISK" {
		t.Errorf("Expected cache hit, got %s", w2.Header().Get("X-Cache"))
	}

	// Stats should show hit rate
	if pc.stats.Hits == 0 {
		t.Error("Expected at least one cache hit")
	}
}

// TestIntegration_RateLimiter tests rate limiting
func TestIntegration_RateLimiter(t *testing.T) {
	limiter := NewRateLimiter(5) // 5 requests per second

	// Allow first 5 requests
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if limiter.Allow() {
		t.Error("6th request should be denied")
	}

	// Wait for token refill
	time.Sleep(300 * time.Millisecond)

	// Should allow one more request
	if !limiter.Allow() {
		t.Error("Request should be allowed after refill")
	}
}

// TestIntegration_CacheWarmer tests cache warming
func TestIntegration_CacheWarmer(t *testing.T) {
	// Create mock server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("content for %s", r.URL.Path)))
	}))
	defer upstream.Close()

	warmer := NewCacheWarmer(upstream.URL+"/", 2, false)
	warmer.AddPackage("package1")
	warmer.AddPackage("package2")
	warmer.AddPackage("package3")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := warmer.Warm(ctx)
	if err != nil {
		t.Fatalf("Cache warming failed: %v", err)
	}
}

// TestIntegration_CacheWarmer_FromFile tests cache warming from file
func TestIntegration_CacheWarmer_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	pkgFile := tmpDir + "/packages.txt"

	content := `# Comment line
github.com/gin-gonic/gin
github.com/gorilla/mux
# Another comment

golang.org/x/net
`

	os.WriteFile(pkgFile, []byte(content), 0644)

	warmer := NewCacheWarmer("http://localhost:8080/", 2, false)
	err := warmer.AddPackagesFromFile(pkgFile)
	if err != nil {
		t.Fatalf("Failed to add packages from file: %v", err)
	}

	if len(warmer.packages) != 3 {
		t.Errorf("Expected 3 packages, got %d", len(warmer.packages))
	}
}

// TestIntegration_Middleware tests middleware stacking
func TestIntegration_Middleware(t *testing.T) {
	tmpDir := t.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://example.com", c, 1024*1024*1024)

	limiter := NewRateLimiter(10)
	logger := &RequestLogger{}

	// Create handler with middleware
	handler := http.HandlerFunc(pc.handleRequest)
	wrapped := CORSMiddleware([]string{"*"})(
		RateLimitMiddleware(limiter)(
			LoggingMiddleware(logger)(handler),
		),
	)

	// Test request with Origin header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// Check CORS headers - should have Access-Control-Allow-Origin
	corsHeader := w.Header().Get("Access-Control-Allow-Origin")
	if corsHeader == "" {
		t.Error("Expected CORS headers in response")
	}
}

// TestIntegration_StatsEndpoint tests stats collection
func TestIntegration_StatsEndpoint(t *testing.T) {
	tmpDir := t.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://example.com", c, 1024*1024*1024)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))
	defer upstream.Close()

	pc.upstreamURL = upstream.URL

	// Make some requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test/pkg", nil)
		w := httptest.NewRecorder()
		pc.handleRequest(w, req)
	}

	// Check stats
	req := httptest.NewRequest("GET", "/_stats", nil)
	w := httptest.NewRecorder()
	pc.statsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	body, _ := io.ReadAll(w.Body)
	if len(body) == 0 {
		t.Error("Expected stats response")
	}
}

// BenchmarkIntegration_FullProxyFlow benchmarks complete proxy flow
func BenchmarkIntegration_FullProxyFlow(b *testing.B) {
	tmpDir := b.TempDir()
	c := cache.NewMemoryCache()
	pc, _ := NewProxyCache(tmpDir, "https://example.com", c, 1024*1024*1024)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test data"))
	}))
	defer upstream.Close()

	pc.upstreamURL = upstream.URL

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test/package", nil)
		w := httptest.NewRecorder()
		pc.handleRequest(w, req)
	}
}
