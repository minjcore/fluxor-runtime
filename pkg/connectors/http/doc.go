// Package http provides a generic HTTP client connector for Fluxor.
//
// This package implements the Connector interface and provides a high-level
// API for making HTTP requests to any REST API. It supports multiple
// authentication methods, retries, rate limiting, and flexible request/response handling.
//
// Example usage:
//
//	// Create HTTP component with configuration
//	config := http.DefaultConfig()
//	config.BaseURL = "https://api.example.com"
//	config.AuthType = "bearer"
//	config.BearerToken = "your-token"
//
//	component := http.NewHTTPComponent(config)
//
//	// Start the component
//	ctx := core.NewFluxorContext(...)
//	if err := component.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer component.Stop(ctx)
//
//	// Get client
//	client, _ := component.Client()
//
//	// Make GET request
//	resp, err := client.Get(context.Background(), "/users", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Status: %d\n", resp.StatusCode)
//	fmt.Printf("Body: %s\n", resp.BodyString)
//
//	// Make POST request with JSON body
//	body := map[string]interface{}{
//	    "name": "John Doe",
//	    "email": "john@example.com",
//	}
//	resp, err = client.Post(context.Background(), "/users", body, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Make custom request
//	req := &http.Request{
//	    Method: "PUT",
//	    URL:    "/users/123",
//	    Body:   body,
//	    Headers: map[string]string{
//	        "X-Custom-Header": "value",
//	    },
//	}
//	resp, err = client.Request(context.Background(), req)
//
// Features:
//   - HTTP methods: GET, POST, PUT, PATCH, DELETE, and custom methods
//   - Multiple authentication methods: Bearer token, Basic auth, API key, custom headers
//   - Automatic retries with exponential backoff
//   - Rate limiting support
//   - Flexible request body types: JSON, string, bytes, io.Reader
//   - Response parsing: JSON, string, bytes
//   - Custom headers per request or globally
//   - Query parameters support
//   - Redirect following (configurable)
//   - TLS configuration (including insecure skip for development)
//
// Configuration:
//   - HTTP_BASE_URL: Base URL for all requests (optional)
//   - HTTP_TIMEOUT: Request timeout (default: 30s)
//   - HTTP_MAX_RETRIES: Maximum retries (default: 3)
//   - HTTP_RATE_LIMIT: Requests per second (default: 100)
//   - HTTP_AUTH_TYPE: Authentication type: bearer, basic, apikey, custom, none (default: none)
//   - HTTP_BEARER_TOKEN: Bearer token (for authType=bearer)
//   - HTTP_BASIC_AUTH_USERNAME: Basic auth username (for authType=basic)
//   - HTTP_BASIC_AUTH_PASSWORD: Basic auth password (for authType=basic)
//   - HTTP_API_KEY: API key (for authType=apikey)
//   - HTTP_API_KEY_HEADER: Header name for API key (default: X-API-Key)
//   - HTTP_CUSTOM_AUTH_HEADER: Custom auth header name (for authType=custom)
//   - HTTP_CUSTOM_AUTH_VALUE: Custom auth header value (for authType=custom)
//   - HTTP_FOLLOW_REDIRECTS: Follow redirects (default: true)
//   - HTTP_MAX_REDIRECTS: Maximum redirects (default: 10)
//   - HTTP_INSECURE_SKIP_VERIFY: Skip TLS verification (default: false, WARNING: dev only)
//   - HTTP_DEBUG: Enable debug logging (default: false)
package http
