package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// httpClient implements the Client interface
type httpClient struct {
	config      Config
	httpClient  *http.Client
	rateLimiter *rate.Limiter
}

// NewClient creates a new HTTP client
func NewClient(config Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create HTTP client with custom transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
	}

	client := &http.Client{
		Timeout:   config.GetTimeout(),
		Transport: transport,
	}

	// Configure redirect following
	if !config.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", config.MaxRedirects)
			}
			return nil
		}
	}

	// Rate limiter
	limiter := rate.NewLimiter(rate.Limit(config.RateLimit), 1)

	return &httpClient{
		config:      config,
		httpClient:  client,
		rateLimiter: limiter,
	}, nil
}

func (c *httpClient) Get(ctx context.Context, url string, headers map[string]string) (*Response, error) {
	return c.Request(ctx, &Request{
		Method:  "GET",
		URL:     url,
		Headers: headers,
	})
}

func (c *httpClient) Post(ctx context.Context, url string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Request(ctx, &Request{
		Method:  "POST",
		URL:     url,
		Body:    body,
		Headers: headers,
	})
}

func (c *httpClient) Put(ctx context.Context, url string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Request(ctx, &Request{
		Method:  "PUT",
		URL:     url,
		Body:    body,
		Headers: headers,
	})
}

func (c *httpClient) Patch(ctx context.Context, url string, body interface{}, headers map[string]string) (*Response, error) {
	return c.Request(ctx, &Request{
		Method:  "PATCH",
		URL:     url,
		Body:    body,
		Headers: headers,
	})
}

func (c *httpClient) Delete(ctx context.Context, url string, headers map[string]string) (*Response, error) {
	return c.Request(ctx, &Request{
		Method:  "DELETE",
		URL:     url,
		Headers: headers,
	})
}

func (c *httpClient) Request(ctx context.Context, req *Request) (*Response, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}
	if req.Method == "" {
		return nil, fmt.Errorf("request method cannot be empty")
	}
	if req.URL == "" {
		return nil, fmt.Errorf("request URL cannot be empty")
	}

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Build full URL
	fullURL := req.URL
	if c.config.BaseURL != "" && !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		baseURL := strings.TrimSuffix(c.config.BaseURL, "/")
		urlPath := strings.TrimPrefix(req.URL, "/")
		fullURL = baseURL + "/" + urlPath
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		parsedURL, err := url.Parse(fullURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
		query := parsedURL.Query()
		for k, v := range req.QueryParams {
			query.Set(k, v)
		}
		parsedURL.RawQuery = query.Encode()
		fullURL = parsedURL.String()
	}

	// Prepare request body
	var reqBody io.Reader
	contentType := req.ContentType
	if req.Body != nil {
		switch body := req.Body.(type) {
		case string:
			reqBody = strings.NewReader(body)
			if contentType == "" {
				contentType = "text/plain"
			}
		case []byte:
			reqBody = bytes.NewReader(body)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
		case io.Reader:
			reqBody = body
			if contentType == "" {
				contentType = "application/octet-stream"
			}
		case map[string]interface{}:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			reqBody = bytes.NewReader(jsonData)
			if contentType == "" {
				contentType = "application/json"
			}
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			reqBody = bytes.NewReader(jsonData)
			if contentType == "" {
				contentType = "application/json"
			}
		}
	} else {
		if contentType == "" {
			contentType = "application/json"
		}
	}

	// Store body bytes for retries
	var bodyBytes []byte
	if reqBody != nil {
		// Read body into bytes for retries
		if br, ok := reqBody.(*bytes.Reader); ok {
			bodyBytes = make([]byte, br.Len())
			br.Read(bodyBytes)
			br.Seek(0, 0)
		} else if sr, ok := reqBody.(*strings.Reader); ok {
			bodyBytes = make([]byte, sr.Len())
			sr.Read(bodyBytes)
			sr.Seek(0, 0)
		} else {
			// Read from reader into bytes
			var err error
			bodyBytes, err = io.ReadAll(reqBody)
			if err != nil {
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}
		}
	}

	// Perform request with retries
	var lastErr error
	var lastResp *http.Response
	var lastBody []byte

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Create body reader for this attempt
		var attemptBody io.Reader
		if bodyBytes != nil {
			attemptBody = bytes.NewReader(bodyBytes)
		}

		// Create request
		httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, attemptBody)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		// Set headers
		if contentType != "" {
			httpReq.Header.Set("Content-Type", contentType)
		}

		// Add default headers
		for k, v := range c.config.DefaultHeaders {
			httpReq.Header.Set(k, v)
		}

		// Add request headers (override defaults)
		if req.Headers != nil {
			for k, v := range req.Headers {
				httpReq.Header.Set(k, v)
			}
		}

		// Add authentication
		if err := c.addAuth(httpReq); err != nil {
			return nil, fmt.Errorf("failed to add authentication: %w", err)
		}

		// Execute request
		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// Check for retryable errors
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			lastResp = resp
			lastBody = respBody
			continue
		}

		// Success - build response
		return c.buildResponse(resp, respBody), nil
	}

	// All retries failed
	if lastResp != nil {
		return c.buildResponse(lastResp, lastBody), nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// addAuth adds authentication headers to the request
func (c *httpClient) addAuth(req *http.Request) error {
	switch c.config.AuthType {
	case "bearer":
		if c.config.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
		}
	case "basic":
		if c.config.BasicAuthUsername != "" && c.config.BasicAuthPassword != "" {
			req.SetBasicAuth(c.config.BasicAuthUsername, c.config.BasicAuthPassword)
		}
	case "apikey":
		if c.config.APIKey != "" {
			headerName := c.config.APIKeyHeader
			if headerName == "" {
				headerName = "X-API-Key"
			}
			req.Header.Set(headerName, c.config.APIKey)
		}
	case "custom":
		if c.config.CustomAuthHeader != "" && c.config.CustomAuthValue != "" {
			req.Header.Set(c.config.CustomAuthHeader, c.config.CustomAuthValue)
		}
	}
	return nil
}

// buildResponse builds a Response from an HTTP response
func (c *httpClient) buildResponse(resp *http.Response, body []byte) *Response {
	response := &Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       body,
		BodyString: string(body),
		Headers:    make(map[string]string),
	}

	// Copy headers
	for k, v := range resp.Header {
		if len(v) > 0 {
			response.Headers[k] = v[0]
		}
	}

	// Try to parse JSON
	if len(body) > 0 {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			response.JSON = jsonData
		}
	}

	return response
}
