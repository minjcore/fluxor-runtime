package web

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/valyala/fasthttp"
)

// SSRHandler handles Server-Side Rendering by proxying requests to a Node.js SSR server
// This allows Go backend to serve SSR React applications
type SSRHandler struct {
	ssrServerURL string        // URL of the Node.js SSR server (e.g., "http://localhost:3001")
	httpClient   *http.Client  // HTTP client for making requests to SSR server
	timeout      time.Duration // Request timeout
}

// NewSSRHandler creates a new SSR handler
// ssrServerURL is the base URL of the Node.js SSR server (e.g., "http://localhost:3001")
func NewSSRHandler(ssrServerURL string) *SSRHandler {
	if ssrServerURL == "" {
		ssrServerURL = "http://localhost:3001"
	}

	return &SSRHandler{
		ssrServerURL: ssrServerURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		timeout: 10 * time.Second,
	}
}

// SetTimeout sets the request timeout for SSR requests
func (h *SSRHandler) SetTimeout(timeout time.Duration) {
	h.timeout = timeout
	h.httpClient.Timeout = timeout
}

// Handle handles a FastHTTP request by proxying it to the SSR server
// This should be used as a FastRouter handler
func (h *SSRHandler) Handle(ctx *FastRequestContext) error {
	// Build the full URL for the SSR server
	requestURI := string(ctx.RequestCtx.RequestURI())
	url := h.ssrServerURL + requestURI

	// Create HTTP request
	req, err := http.NewRequest(
		string(ctx.Method()),
		url,
		bytes.NewReader(ctx.RequestCtx.PostBody()),
	)
	if err != nil {
		ctx.Error(fmt.Sprintf("Failed to create SSR request: %v", err), fasthttp.StatusInternalServerError)
		return err
	}

	// Copy headers from FastHTTP request to HTTP request
	// Preserve important headers for SSR
	ctx.RequestCtx.Request.Header.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		// Skip hop-by-hop headers that shouldn't be forwarded
		if keyStr != "Connection" && keyStr != "Keep-Alive" && 
		   keyStr != "Proxy-Authenticate" && keyStr != "Proxy-Authorization" &&
		   keyStr != "Te" && keyStr != "Trailers" && keyStr != "Transfer-Encoding" &&
		   keyStr != "Upgrade" {
			req.Header.Set(keyStr, string(value))
		}
	})

	// Set Host header to match SSR server (important for virtual hosting)
	req.Host = ""
	
	// Make request to SSR server
	resp, err := h.httpClient.Do(req)
	if err != nil {
		// Network error - SSR server might be down
		ctx.Error(fmt.Sprintf("SSR server error: %v (URL: %s)", err, url), fasthttp.StatusBadGateway)
		return err
	}
	defer resp.Body.Close()

	// Copy status code (including 403 if SSR server returns it)
	ctx.RequestCtx.SetStatusCode(resp.StatusCode)

	// Copy headers from HTTP response to FastHTTP response
	for key, values := range resp.Header {
		// Skip hop-by-hop headers
		if key != "Connection" && key != "Keep-Alive" && 
		   key != "Proxy-Authenticate" && key != "Proxy-Authorization" &&
		   key != "Te" && key != "Trailers" && key != "Transfer-Encoding" &&
		   key != "Upgrade" {
			for _, value := range values {
				ctx.RequestCtx.Response.Header.Add(key, value)
			}
		}
	}

	// Copy response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.Error(fmt.Sprintf("Failed to read SSR response: %v", err), fasthttp.StatusBadGateway)
		return err
	}

	ctx.RequestCtx.SetBody(body)
	return nil
}

// HandleWithContext handles a FastHTTP request with FluxorContext
// This allows access to EventBus and other Fluxor features
func (h *SSRHandler) HandleWithContext(fluxorCtx core.FluxorContext) func(*FastRequestContext) error {
	return func(ctx *FastRequestContext) error {
		// You can use fluxorCtx here for EventBus, GoCMD, etc.
		// For now, just delegate to Handle
		return h.Handle(ctx)
	}
}

// ProxyHandler returns a handler function that can be used with FastRouter
func (h *SSRHandler) ProxyHandler() func(*FastRequestContext) error {
	return h.Handle
}
