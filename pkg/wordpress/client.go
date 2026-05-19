package wordpress

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
)

// Client is a WordPress REST API client
type Client struct {
	baseURL    string        // WordPress site URL (e.g., "https://example.com")
	httpClient *http.Client  // HTTP client for making requests
	timeout    time.Duration // Request timeout
	cache      cache.Cache   // Optional cache for responses
	cacheTTL   time.Duration // Cache TTL (default: 5 minutes)
}

// NewClient creates a new WordPress client
// baseURL is the WordPress site URL (e.g., "https://example.com" or "https://example.com/wp-json")
func NewClient(baseURL string) *Client {
	// Ensure baseURL ends with /wp-json or add it
	if baseURL == "" {
		baseURL = "https://wordpress.org/news/wp-json" // Default to WordPress news
	} else {
		// If baseURL doesn't contain /wp-json, add it
		if len(baseURL) > 8 && baseURL[len(baseURL)-8:] != "/wp-json" {
			if baseURL[len(baseURL)-1] != '/' {
				baseURL += "/wp-json"
			} else {
				baseURL += "wp-json"
			}
		}
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		timeout:  10 * time.Second,
		cacheTTL: 5 * time.Minute, // Default cache TTL: 5 minutes
	}
}

// SetTimeout sets the request timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
	c.httpClient.Timeout = timeout
}

// SetCache sets the cache backend for the client
func (c *Client) SetCache(cacheBackend cache.Cache) {
	c.cache = cacheBackend
}

// SetCacheTTL sets the cache TTL (default: 5 minutes)
func (c *Client) SetCacheTTL(ttl time.Duration) {
	c.cacheTTL = ttl
}

// generateCacheKey generates a cache key from URL and options
func (c *Client) generateCacheKey(endpoint string, opts interface{}) string {
	data, _ := json.Marshal(map[string]interface{}{
		"baseURL": c.baseURL,
		"endpoint": endpoint,
		"opts": opts,
	})
	hash := sha256.Sum256(data)
	return fmt.Sprintf("wp:%s", hex.EncodeToString(hash[:16]))
}

// Post represents a WordPress post
type Post struct {
	ID          int       `json:"id"`
	Date        string    `json:"date"`
	DateGMT     string    `json:"date_gmt"`
	Modified    string    `json:"modified"`
	ModifiedGMT string    `json:"modified_gmt"`
	Slug        string    `json:"slug"`
	Status      string    `json:"status"`
	Type        string    `json:"type"`
	Link        string    `json:"link"`
	Title       Title     `json:"title"`
	Content     Content   `json:"content"`
	Excerpt     Excerpt   `json:"excerpt"`
	Author      int       `json:"author"`
	FeaturedMedia int     `json:"featured_media"`
	Categories  []int     `json:"categories"`
	Tags        []int     `json:"tags"`
}

// Title represents WordPress post title
type Title struct {
	Rendered string `json:"rendered"`
}

// Content represents WordPress post content
type Content struct {
	Rendered  string `json:"rendered"`
	Protected bool   `json:"protected"`
}

// Excerpt represents WordPress post excerpt
type Excerpt struct {
	Rendered  string `json:"rendered"`
	Protected bool   `json:"protected"`
}

// GetPostsOptions are options for fetching posts
type GetPostsOptions struct {
	Page     int    // Page number (default: 1)
	PerPage  int    // Posts per page (default: 10, max: 100)
	Search   string // Search query
	Categories []int // Filter by category IDs
	Tags      []int // Filter by tag IDs
	OrderBy   string // Order by: date, title, etc. (default: date)
	Order     string // Order: asc or desc (default: desc)
}

// GetPosts fetches posts from WordPress (with caching support)
func (c *Client) GetPosts(ctx context.Context, opts *GetPostsOptions) ([]Post, error) {
	if opts == nil {
		opts = &GetPostsOptions{
			Page:    1,
			PerPage: 10,
			OrderBy: "date",
			Order:   "desc",
		}
	}

	// Check cache first
	if c.cache != nil {
		cacheKey := c.generateCacheKey("/wp/v2/posts", opts)
		if cached, err := c.cache.Get(ctx, cacheKey); err == nil {
			var posts []Post
			if err := json.Unmarshal(cached, &posts); err == nil {
				return posts, nil
			}
		}
	}

	// Build URL
	apiURL := c.baseURL + "/wp/v2/posts"
	u, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	q := u.Query()
	if opts.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", opts.Page))
	}
	if opts.PerPage > 0 {
		if opts.PerPage > 100 {
			opts.PerPage = 100 // WordPress max
		}
		q.Set("per_page", fmt.Sprintf("%d", opts.PerPage))
	}
	if opts.Search != "" {
		q.Set("search", opts.Search)
	}
	if len(opts.Categories) > 0 {
		for _, catID := range opts.Categories {
			q.Add("categories", fmt.Sprintf("%d", catID))
		}
	}
	if len(opts.Tags) > 0 {
		for _, tagID := range opts.Tags {
			q.Add("tags", fmt.Sprintf("%d", tagID))
		}
	}
	if opts.OrderBy != "" {
		q.Set("orderby", opts.OrderBy)
	}
	if opts.Order != "" {
		q.Set("order", opts.Order)
	}

	u.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("WordPress API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var posts []Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache the response
	if c.cache != nil {
		cacheKey := c.generateCacheKey("/wp/v2/posts", opts)
		if data, err := json.Marshal(posts); err == nil {
			_ = c.cache.Set(ctx, cacheKey, data, c.cacheTTL)
		}
	}

	return posts, nil
}

// GetPostBySlug fetches a single post by slug (with caching support)
func (c *Client) GetPostBySlug(ctx context.Context, slug string) (*Post, error) {
	if slug == "" {
		return nil, fmt.Errorf("slug is required")
	}

	// Check cache first
	if c.cache != nil {
		cacheKey := c.generateCacheKey(fmt.Sprintf("/wp/v2/posts?slug=%s", slug), nil)
		if cached, err := c.cache.Get(ctx, cacheKey); err == nil {
			var posts []Post
			if err := json.Unmarshal(cached, &posts); err == nil && len(posts) > 0 {
				return &posts[0], nil
			}
		}
	}

	apiURL := fmt.Sprintf("%s/wp/v2/posts?slug=%s", c.baseURL, url.QueryEscape(slug))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("WordPress API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var posts []Post
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(posts) == 0 {
		return nil, fmt.Errorf("post not found: slug=%s", slug)
	}

	post := &posts[0]

	// Cache the response
	if c.cache != nil {
		cacheKey := c.generateCacheKey(fmt.Sprintf("/wp/v2/posts?slug=%s", slug), nil)
		if data, err := json.Marshal([]Post{*post}); err == nil {
			_ = c.cache.Set(ctx, cacheKey, data, c.cacheTTL)
		}
	}

	return post, nil
}

// GetPost fetches a single post by ID (with caching support)
func (c *Client) GetPost(ctx context.Context, postID int) (*Post, error) {
	// Check cache first
	if c.cache != nil {
		cacheKey := c.generateCacheKey(fmt.Sprintf("/wp/v2/posts/%d", postID), nil)
		if cached, err := c.cache.Get(ctx, cacheKey); err == nil {
			var post Post
			if err := json.Unmarshal(cached, &post); err == nil {
				return &post, nil
			}
		}
	}

	apiURL := fmt.Sprintf("%s/wp/v2/posts/%d", c.baseURL, postID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("WordPress API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache the response
	if c.cache != nil {
		cacheKey := c.generateCacheKey(fmt.Sprintf("/wp/v2/posts/%d", postID), nil)
		if data, err := json.Marshal(post); err == nil {
			_ = c.cache.Set(ctx, cacheKey, data, c.cacheTTL)
		}
	}

	return &post, nil
}

// Category represents a WordPress category
type Category struct {
	ID          int    `json:"id"`
	Count       int    `json:"count"`
	Description string `json:"description"`
	Link        string `json:"link"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
}

// GetCategories fetches categories from WordPress (with caching support)
func (c *Client) GetCategories(ctx context.Context) ([]Category, error) {
	// Check cache first
	if c.cache != nil {
		cacheKey := c.generateCacheKey("/wp/v2/categories", nil)
		if cached, err := c.cache.Get(ctx, cacheKey); err == nil {
			var categories []Category
			if err := json.Unmarshal(cached, &categories); err == nil {
				return categories, nil
			}
		}
	}

	apiURL := c.baseURL + "/wp/v2/categories"

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("WordPress API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var categories []Category
	if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache the response (categories change less frequently, use longer TTL)
	if c.cache != nil {
		cacheKey := c.generateCacheKey("/wp/v2/categories", nil)
		if data, err := json.Marshal(categories); err == nil {
			// Categories cache for 30 minutes (longer than posts)
			_ = c.cache.Set(ctx, cacheKey, data, 30*time.Minute)
		}
	}

	return categories, nil
}
