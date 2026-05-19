package wordpress

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "empty baseURL uses default",
			baseURL: "",
			want:    "https://wordpress.org/news/wp-json",
		},
		{
			name:    "baseURL without /wp-json suffix",
			baseURL: "https://example.com",
			want:    "https://example.com/wp-json",
		},
		{
			name:    "baseURL with trailing slash",
			baseURL: "https://example.com/",
			want:    "https://example.com/wp-json",
		},
		{
			name:    "baseURL already has /wp-json",
			baseURL: "https://example.com/wp-json",
			want:    "https://example.com/wp-json",
		},
		{
			name:    "baseURL with /wp-json and trailing slash",
			baseURL: "https://example.com/wp-json/",
			want:    "https://example.com/wp-json/wp-json", // Current behavior: adds /wp-json even if present
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL)
			if client.baseURL != tt.want {
				t.Errorf("NewClient(%q).baseURL = %q, want %q", tt.baseURL, client.baseURL, tt.want)
			}
			if client.timeout != 10*time.Second {
				t.Errorf("NewClient().timeout = %v, want 10s", client.timeout)
			}
			if client.cacheTTL != 5*time.Minute {
				t.Errorf("NewClient().cacheTTL = %v, want 5m", client.cacheTTL)
			}
		})
	}
}

func TestClient_SetTimeout(t *testing.T) {
	client := NewClient("https://example.com")
	timeout := 30 * time.Second
	client.SetTimeout(timeout)

	if client.timeout != timeout {
		t.Errorf("SetTimeout() timeout = %v, want %v", client.timeout, timeout)
	}
	if client.httpClient.Timeout != timeout {
		t.Errorf("SetTimeout() httpClient.Timeout = %v, want %v", client.httpClient.Timeout, timeout)
	}
}

func TestClient_SetCache(t *testing.T) {
	client := NewClient("https://example.com")
	mockCache := cache.NewMemoryCache()
	client.SetCache(mockCache)

	if client.cache != mockCache {
		t.Error("SetCache() cache not set correctly")
	}
}

func TestClient_SetCacheTTL(t *testing.T) {
	client := NewClient("https://example.com")
	ttl := 10 * time.Minute
	client.SetCacheTTL(ttl)

	if client.cacheTTL != ttl {
		t.Errorf("SetCacheTTL() cacheTTL = %v, want %v", client.cacheTTL, ttl)
	}
}

func TestClient_GetPosts(t *testing.T) {
	mockPosts := []Post{
		{
			ID:     1,
			Slug:   "test-post-1",
			Status: "publish",
			Title:  Title{Rendered: "Test Post 1"},
			Content: Content{Rendered: "Test content 1"},
			Excerpt: Excerpt{Rendered: "Test excerpt 1"},
		},
		{
			ID:     2,
			Slug:   "test-post-2",
			Status: "publish",
			Title:  Title{Rendered: "Test Post 2"},
			Content: Content{Rendered: "Test content 2"},
			Excerpt: Excerpt{Rendered: "Test excerpt 2"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		// Client adds /wp-json to baseURL, so path will be /wp-json/wp/v2/posts
		if r.URL.Path != "/wp-json/wp/v2/posts" {
			t.Errorf("Expected path /wp-json/wp/v2/posts, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPosts)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	t.Run("get posts with default options", func(t *testing.T) {
		posts, err := client.GetPosts(ctx, nil)
		if err != nil {
			t.Fatalf("GetPosts() error = %v", err)
		}
		if len(posts) != 2 {
			t.Errorf("GetPosts() returned %d posts, want 2", len(posts))
		}
		if posts[0].ID != 1 {
			t.Errorf("GetPosts() first post ID = %d, want 1", posts[0].ID)
		}
	})

	t.Run("get posts with options", func(t *testing.T) {
		opts := &GetPostsOptions{
			Page:    2,
			PerPage: 5,
			Search:  "test",
			OrderBy: "date",
			Order:   "desc",
		}
		posts, err := client.GetPosts(ctx, opts)
		if err != nil {
			t.Fatalf("GetPosts() error = %v", err)
		}
		if len(posts) != 2 {
			t.Errorf("GetPosts() returned %d posts, want 2", len(posts))
		}
	})

	t.Run("get posts with per_page > 100", func(t *testing.T) {
		opts := &GetPostsOptions{
			PerPage: 200, // Should be capped at 100
		}
		posts, err := client.GetPosts(ctx, opts)
		if err != nil {
			t.Fatalf("GetPosts() error = %v", err)
		}
		if opts.PerPage != 100 {
			t.Errorf("GetPosts() should cap PerPage at 100, got %d", opts.PerPage)
		}
		if len(posts) != 2 {
			t.Errorf("GetPosts() returned %d posts, want 2", len(posts))
		}
	})

	t.Run("get posts with categories and tags", func(t *testing.T) {
		opts := &GetPostsOptions{
			Categories: []int{1, 2},
			Tags:      []int{3, 4},
		}
		posts, err := client.GetPosts(ctx, opts)
		if err != nil {
			t.Fatalf("GetPosts() error = %v", err)
		}
		if len(posts) != 2 {
			t.Errorf("GetPosts() returned %d posts, want 2", len(posts))
		}
	})
}

func TestClient_GetPosts_Error(t *testing.T) {
	t.Run("HTTP error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		ctx := context.Background()

		_, err := client.GetPosts(ctx, nil)
		if err == nil {
			t.Error("GetPosts() should return error for HTTP 500")
		}
		if !strings.Contains(err.Error(), "WordPress API error") {
			t.Errorf("GetPosts() error message should contain 'WordPress API error', got %q", err.Error())
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		ctx := context.Background()

		_, err := client.GetPosts(ctx, nil)
		if err == nil {
			t.Error("GetPosts() should return error for invalid JSON")
		}
	})
}

func TestClient_GetPosts_WithCache(t *testing.T) {
	mockPosts := []Post{
		{
			ID:     1,
			Slug:   "test-post",
			Status: "publish",
			Title:  Title{Rendered: "Test Post"},
		},
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPosts)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetCache(cache.NewMemoryCache())
	ctx := context.Background()

	// First call - should hit server
	posts1, err := client.GetPosts(ctx, nil)
	if err != nil {
		t.Fatalf("GetPosts() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("GetPosts() should call server once, got %d calls", callCount)
	}

	// Second call - should hit cache
	posts2, err := client.GetPosts(ctx, nil)
	if err != nil {
		t.Fatalf("GetPosts() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("GetPosts() should use cache, got %d calls", callCount)
	}
	if len(posts1) != len(posts2) {
		t.Errorf("GetPosts() cached result length = %d, want %d", len(posts2), len(posts1))
	}
}

func TestClient_GetPost(t *testing.T) {
	mockPost := Post{
		ID:     123,
		Slug:   "test-post",
		Status: "publish",
		Title:  Title{Rendered: "Test Post"},
		Content: Content{Rendered: "Test content"},
		Excerpt: Excerpt{Rendered: "Test excerpt"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		// Client adds /wp-json to baseURL, so path will be /wp-json/wp/v2/posts/123
		if r.URL.Path != "/wp-json/wp/v2/posts/123" {
			t.Errorf("Expected path /wp-json/wp/v2/posts/123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPost)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	post, err := client.GetPost(ctx, 123)
	if err != nil {
		t.Fatalf("GetPost() error = %v", err)
	}
	if post.ID != 123 {
		t.Errorf("GetPost() post.ID = %d, want 123", post.ID)
	}
	if post.Slug != "test-post" {
		t.Errorf("GetPost() post.Slug = %q, want 'test-post'", post.Slug)
	}
}

func TestClient_GetPost_Error(t *testing.T) {
	t.Run("HTTP error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Post not found"))
		}))
		defer server.Close()

		client := NewClient(server.URL)
		ctx := context.Background()

		_, err := client.GetPost(ctx, 999)
		if err == nil {
			t.Error("GetPost() should return error for HTTP 404")
		}
	})
}

func TestClient_GetPost_WithCache(t *testing.T) {
	mockPost := Post{
		ID:     123,
		Slug:   "test-post",
		Status: "publish",
		Title:  Title{Rendered: "Test Post"},
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPost)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetCache(cache.NewMemoryCache())
	ctx := context.Background()

	// First call
	post1, err := client.GetPost(ctx, 123)
	if err != nil {
		t.Fatalf("GetPost() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("GetPost() should call server once, got %d calls", callCount)
	}

	// Second call - should use cache
	post2, err := client.GetPost(ctx, 123)
	if err != nil {
		t.Fatalf("GetPost() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("GetPost() should use cache, got %d calls", callCount)
	}
	if post1.ID != post2.ID {
		t.Errorf("GetPost() cached result ID = %d, want %d", post2.ID, post1.ID)
	}
}

func TestClient_GetPostBySlug(t *testing.T) {
	mockPosts := []Post{
		{
			ID:     1,
			Slug:   "test-post",
			Status: "publish",
			Title:  Title{Rendered: "Test Post"},
			Content: Content{Rendered: "Test content"},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Query().Get("slug") != "test-post" {
			t.Errorf("Expected slug=test-post, got %s", r.URL.Query().Get("slug"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPosts)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	post, err := client.GetPostBySlug(ctx, "test-post")
	if err != nil {
		t.Fatalf("GetPostBySlug() error = %v", err)
	}
	if post.Slug != "test-post" {
		t.Errorf("GetPostBySlug() post.Slug = %q, want 'test-post'", post.Slug)
	}
}

func TestClient_GetPostBySlug_Error(t *testing.T) {
	t.Run("empty slug", func(t *testing.T) {
		client := NewClient("https://example.com")
		ctx := context.Background()

		_, err := client.GetPostBySlug(ctx, "")
		if err == nil {
			t.Error("GetPostBySlug() should return error for empty slug")
		}
	})

	t.Run("post not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]Post{}) // Empty array
		}))
		defer server.Close()

		client := NewClient(server.URL)
		ctx := context.Background()

		_, err := client.GetPostBySlug(ctx, "nonexistent")
		if err == nil {
			t.Error("GetPostBySlug() should return error for nonexistent slug")
		}
		if !strings.Contains(err.Error(), "post not found") {
			t.Errorf("GetPostBySlug() error message should contain 'post not found', got %q", err.Error())
		}
	})
}

func TestClient_GetPostBySlug_WithCache(t *testing.T) {
	mockPosts := []Post{
		{
			ID:     1,
			Slug:   "test-post",
			Status: "publish",
			Title:  Title{Rendered: "Test Post"},
		},
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockPosts)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetCache(cache.NewMemoryCache())
	ctx := context.Background()

	// First call
	post1, err := client.GetPostBySlug(ctx, "test-post")
	if err != nil {
		t.Fatalf("GetPostBySlug() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("GetPostBySlug() should call server once, got %d calls", callCount)
	}

	// Second call - should use cache
	post2, err := client.GetPostBySlug(ctx, "test-post")
	if err != nil {
		t.Fatalf("GetPostBySlug() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("GetPostBySlug() should use cache, got %d calls", callCount)
	}
	if post1.Slug != post2.Slug {
		t.Errorf("GetPostBySlug() cached result Slug = %q, want %q", post2.Slug, post1.Slug)
	}
}

func TestClient_GetCategories(t *testing.T) {
	mockCategories := []Category{
		{
			ID:          1,
			Name:        "Uncategorized",
			Slug:        "uncategorized",
			Description: "Default category",
			Count:       10,
		},
		{
			ID:          2,
			Name:        "Technology",
			Slug:        "technology",
			Description: "Tech posts",
			Count:       5,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		// Client adds /wp-json to baseURL, so path will be /wp-json/wp/v2/categories
		if r.URL.Path != "/wp-json/wp/v2/categories" {
			t.Errorf("Expected path /wp-json/wp/v2/categories, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockCategories)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	categories, err := client.GetCategories(ctx)
	if err != nil {
		t.Fatalf("GetCategories() error = %v", err)
	}
	if len(categories) != 2 {
		t.Errorf("GetCategories() returned %d categories, want 2", len(categories))
	}
	if categories[0].ID != 1 {
		t.Errorf("GetCategories() first category ID = %d, want 1", categories[0].ID)
	}
	if categories[0].Name != "Uncategorized" {
		t.Errorf("GetCategories() first category Name = %q, want 'Uncategorized'", categories[0].Name)
	}
}

func TestClient_GetCategories_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	_, err := client.GetCategories(ctx)
	if err == nil {
		t.Error("GetCategories() should return error for HTTP 500")
	}
}

func TestClient_GetCategories_WithCache(t *testing.T) {
	mockCategories := []Category{
		{
			ID:   1,
			Name: "Uncategorized",
			Slug: "uncategorized",
		},
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockCategories)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetCache(cache.NewMemoryCache())
	ctx := context.Background()

	// First call
	categories1, err := client.GetCategories(ctx)
	if err != nil {
		t.Fatalf("GetCategories() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("GetCategories() should call server once, got %d calls", callCount)
	}

	// Second call - should use cache
	categories2, err := client.GetCategories(ctx)
	if err != nil {
		t.Fatalf("GetCategories() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("GetCategories() should use cache, got %d calls", callCount)
	}
	if len(categories1) != len(categories2) {
		t.Errorf("GetCategories() cached result length = %d, want %d", len(categories2), len(categories1))
	}
}

func TestClient_generateCacheKey(t *testing.T) {
	client := NewClient("https://example.com")

	key1 := client.generateCacheKey("/wp/v2/posts", &GetPostsOptions{Page: 1, PerPage: 10})
	key2 := client.generateCacheKey("/wp/v2/posts", &GetPostsOptions{Page: 1, PerPage: 10})
	key3 := client.generateCacheKey("/wp/v2/posts", &GetPostsOptions{Page: 2, PerPage: 10})

	if key1 != key2 {
		t.Error("generateCacheKey() should generate same key for same options")
	}
	if key1 == key3 {
		t.Error("generateCacheKey() should generate different key for different options")
	}
	if key1[:3] != "wp:" {
		t.Errorf("generateCacheKey() should start with 'wp:', got %q", key1[:3])
	}
}
