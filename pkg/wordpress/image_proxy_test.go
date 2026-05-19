package wordpress

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestNewImageProxy(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "baseURL with /wp-json",
			baseURL: "https://example.com/wp-json",
			want:    "https://example.com",
		},
		{
			name:    "baseURL without /wp-json",
			baseURL: "https://example.com",
			want:    "https://example.com",
		},
		{
			name:    "baseURL with trailing slash",
			baseURL: "https://example.com/",
			want:    "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := NewImageProxy(tt.baseURL)
			if proxy.wpBaseURL != tt.want {
				t.Errorf("NewImageProxy(%q).wpBaseURL = %q, want %q", tt.baseURL, proxy.wpBaseURL, tt.want)
			}
			if proxy.timeout != 30*time.Second {
				t.Errorf("NewImageProxy().timeout = %v, want 30s", proxy.timeout)
			}
			if proxy.cacheTTL != 24*time.Hour {
				t.Errorf("NewImageProxy().cacheTTL = %v, want 24h", proxy.cacheTTL)
			}
		})
	}
}

func TestImageProxy_SetCache(t *testing.T) {
	proxy := NewImageProxy("https://example.com")
	mockCache := cache.NewMemoryCache()
	proxy.SetCache(mockCache)

	if proxy.cache != mockCache {
		t.Error("SetCache() cache not set correctly")
	}
}

func TestImageProxy_SetCacheTTL(t *testing.T) {
	proxy := NewImageProxy("https://example.com")
	ttl := 12 * time.Hour
	proxy.SetCacheTTL(ttl)

	if proxy.cacheTTL != ttl {
		t.Errorf("SetCacheTTL() cacheTTL = %v, want %v", proxy.cacheTTL, ttl)
	}
}

func TestImageProxy_SetTimeout(t *testing.T) {
	proxy := NewImageProxy("https://example.com")
	timeout := 60 * time.Second
	proxy.SetTimeout(timeout)

	if proxy.timeout != timeout {
		t.Errorf("SetTimeout() timeout = %v, want %v", proxy.timeout, timeout)
	}
	if proxy.httpClient.Timeout != timeout {
		t.Errorf("SetTimeout() httpClient.Timeout = %v, want %v", proxy.httpClient.Timeout, timeout)
	}
}

func TestImageProxy_ProxyImage(t *testing.T) {
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/wp-content/uploads/2024/01/image.jpg" {
			t.Errorf("Expected path /wp-content/uploads/2024/01/image.jpg, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(imageData)
	}))
	defer server.Close()

	proxy := NewImageProxy(server.URL)
	ctx := context.Background()

	t.Run("proxy image with relative path", func(t *testing.T) {
		data, contentType, err := proxy.ProxyImage(ctx, "/wp-content/uploads/2024/01/image.jpg")
		if err != nil {
			t.Fatalf("ProxyImage() error = %v", err)
		}
		if len(data) != len(imageData) {
			t.Errorf("ProxyImage() data length = %d, want %d", len(data), len(imageData))
		}
		if contentType != "image/jpeg" {
			t.Errorf("ProxyImage() contentType = %q, want 'image/jpeg'", contentType)
		}
	})

	t.Run("proxy image with full URL", func(t *testing.T) {
		fullURL := server.URL + "/wp-content/uploads/2024/01/image.jpg"
		data, contentType, err := proxy.ProxyImage(ctx, fullURL)
		if err != nil {
			t.Fatalf("ProxyImage() error = %v", err)
		}
		if len(data) != len(imageData) {
			t.Errorf("ProxyImage() data length = %d, want %d", len(data), len(imageData))
		}
		if contentType != "image/jpeg" {
			t.Errorf("ProxyImage() contentType = %q, want 'image/jpeg'", contentType)
		}
	})

	t.Run("proxy image with path without leading slash", func(t *testing.T) {
		data, _, err := proxy.ProxyImage(ctx, "wp-content/uploads/2024/01/image.jpg")
		if err != nil {
			t.Fatalf("ProxyImage() error = %v", err)
		}
		if len(data) != len(imageData) {
			t.Errorf("ProxyImage() data length = %d, want %d", len(data), len(imageData))
		}
	})
}

func TestImageProxy_ProxyImage_Error(t *testing.T) {
	t.Run("HTTP error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		proxy := NewImageProxy(server.URL)
		ctx := context.Background()

		_, _, err := proxy.ProxyImage(ctx, "/wp-content/uploads/image.jpg")
		if err == nil {
			t.Error("ProxyImage() should return error for HTTP 404")
		}
		if !strings.Contains(err.Error(), "WordPress image error") {
			t.Errorf("ProxyImage() error message should contain 'WordPress image error', got %q", err.Error())
		}
	})
}

func TestImageProxy_ProxyImage_WithCache(t *testing.T) {
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(imageData)
	}))
	defer server.Close()

	proxy := NewImageProxy(server.URL)
	proxy.SetCache(cache.NewMemoryCache())
	ctx := context.Background()

	// First call - should hit server
	data1, contentType1, err := proxy.ProxyImage(ctx, "/wp-content/uploads/2024/01/image.jpg")
	if err != nil {
		t.Fatalf("ProxyImage() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("ProxyImage() should call server once, got %d calls", callCount)
	}

	// Second call - should hit cache
	data2, contentType2, err := proxy.ProxyImage(ctx, "/wp-content/uploads/2024/01/image.jpg")
	if err != nil {
		t.Fatalf("ProxyImage() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("ProxyImage() should use cache, got %d calls", callCount)
	}
	if len(data1) != len(data2) {
		t.Errorf("ProxyImage() cached data length = %d, want %d", len(data2), len(data1))
	}
	if contentType1 != contentType2 {
		t.Errorf("ProxyImage() cached contentType = %q, want %q", contentType2, contentType1)
	}
}

func TestImageProxy_detectContentType(t *testing.T) {
	proxy := NewImageProxy("https://example.com")

	tests := []struct {
		name        string
		imagePath   string
		wantContent string
	}{
		{"JPEG", "/image.jpg", "image/jpeg"},
		{"JPEG uppercase", "/IMAGE.JPG", "image/jpeg"},
		{"PNG", "/image.png", "image/png"},
		{"GIF", "/image.gif", "image/gif"},
		{"WebP", "/image.webp", "image/webp"},
		{"SVG", "/image.svg", "image/svg+xml"},
		{"ICO", "/favicon.ico", "image/x-icon"},
		{"unknown extension", "/image.xyz", "image/jpeg"}, // Default
		{"no extension", "/image", "image/jpeg"},         // Default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proxy.detectContentType(tt.imagePath)
			if got != tt.wantContent {
				t.Errorf("detectContentType(%q) = %q, want %q", tt.imagePath, got, tt.wantContent)
			}
		})
	}
}

func TestImageProxy_GetImageURL(t *testing.T) {
	proxy := NewImageProxy("https://example.com")

	tests := []struct {
		name     string
		imagePath string
		want     string
	}{
		{
			name:      "relative path with /wp-content",
			imagePath: "/wp-content/uploads/2024/01/image.jpg",
			want:      "/wp-images/wp-content/uploads/2024/01/image.jpg",
		},
		{
			name:      "relative path without /wp-content",
			imagePath: "/uploads/2024/01/image.jpg",
			want:      "/wp-images/wp-content/uploads/2024/01/image.jpg",
		},
		{
			name:      "full HTTP URL",
			imagePath: "http://example.com/wp-content/uploads/2024/01/image.jpg",
			want:      "/wp-images/wp-content/uploads/2024/01/image.jpg",
		},
		{
			name:      "full HTTPS URL",
			imagePath: "https://example.com/wp-content/uploads/2024/01/image.jpg",
			want:      "/wp-images/wp-content/uploads/2024/01/image.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proxy.GetImageURL(tt.imagePath)
			if got != tt.want {
				t.Errorf("GetImageURL(%q) = %q, want %q", tt.imagePath, got, tt.want)
			}
		})
	}
}

func TestImageProxy_HandleFastHTTP(t *testing.T) {
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(imageData)
	}))
	defer server.Close()

	proxy := NewImageProxy(server.URL)
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	t.Run("handle /wp-images path", func(t *testing.T) {
		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.SetRequestURI("/wp-images/wp-content/uploads/2024/01/image.jpg")

		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := proxy.HandleFastHTTP(fastCtx)
		if err != nil {
			t.Fatalf("HandleFastHTTP() error = %v", err)
		}

		if reqCtx.Response.StatusCode() != fasthttp.StatusOK {
			t.Errorf("HandleFastHTTP() status = %d, want %d", reqCtx.Response.StatusCode(), fasthttp.StatusOK)
		}

		contentType := string(reqCtx.Response.Header.Peek("Content-Type"))
		if contentType != "image/jpeg" {
			t.Errorf("HandleFastHTTP() Content-Type = %q, want 'image/jpeg'", contentType)
		}

		cacheControl := string(reqCtx.Response.Header.Peek("Cache-Control"))
		if cacheControl == "" {
			t.Error("HandleFastHTTP() should set Cache-Control header")
		}
	})

	t.Run("handle /wp-content path", func(t *testing.T) {
		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.SetRequestURI("/wp-content/uploads/2024/01/image.jpg")

		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := proxy.HandleFastHTTP(fastCtx)
		if err != nil {
			t.Fatalf("HandleFastHTTP() error = %v", err)
		}

		if reqCtx.Response.StatusCode() != fasthttp.StatusOK {
			t.Errorf("HandleFastHTTP() status = %d, want %d", reqCtx.Response.StatusCode(), fasthttp.StatusOK)
		}
	})

	t.Run("error on missing image path", func(t *testing.T) {
		reqCtx := &fasthttp.RequestCtx{}
		reqCtx.Request.SetRequestURI("/wp-images")

		fastCtx := &web.FastRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			RequestCtx:         reqCtx,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             make(map[string]string),
		}

		err := proxy.HandleFastHTTP(fastCtx)
		if err == nil {
			t.Error("HandleFastHTTP() should return error for missing image path")
		}

		if reqCtx.Response.StatusCode() != fasthttp.StatusBadRequest {
			t.Errorf("HandleFastHTTP() status = %d, want %d", reqCtx.Response.StatusCode(), fasthttp.StatusBadRequest)
		}
	})
}

func TestImageProxy_HandleFastHTTP_Error(t *testing.T) {
	// Server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	proxy := NewImageProxy(server.URL)
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/wp-images/wp-content/uploads/nonexistent.jpg")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	err := proxy.HandleFastHTTP(fastCtx)
	if err == nil {
		t.Error("HandleFastHTTP() should return error for HTTP 404")
	}

	if reqCtx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Errorf("HandleFastHTTP() status = %d, want %d", reqCtx.Response.StatusCode(), fasthttp.StatusBadGateway)
	}
}
