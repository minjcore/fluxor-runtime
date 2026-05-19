package web

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fluxorio/fluxor/pkg/core"
)

// Server represents an HTTP server abstraction
type Server interface {
	// Start starts the server
	Start() error

	// Stop stops the server
	Stop() error

	// Router returns the router
	Router() Router
}

// Router handles HTTP routing
type Router interface {
	// GET registers a GET handler
	GET(path string, handler RequestHandler)

	// POST registers a POST handler
	POST(path string, handler RequestHandler)

	// PUT registers a PUT handler
	PUT(path string, handler RequestHandler)

	// DELETE registers a DELETE handler
	DELETE(path string, handler RequestHandler)

	// PATCH registers a PATCH handler
	PATCH(path string, handler RequestHandler)

	// Route registers a handler for any HTTP method
	Route(method, path string, handler RequestHandler)

	// Use adds middleware
	Use(middleware Middleware)
}

// RequestHandler handles HTTP requests
type RequestHandler func(ctx *RequestContext) error

// Middleware is HTTP middleware
type Middleware func(handler RequestHandler) RequestHandler

// RequestContext provides request context
// Extends BaseRequestContext for common data storage functionality
type RequestContext struct {
	*core.BaseRequestContext // Embed base context for data storage
	Context                  context.Context
	Request                  *http.Request
	Response                 http.ResponseWriter
	GoCMD                    core.GoCMD
	EventBus                 core.EventBus
	Params                   map[string]string
}

// JSON writes JSON response
func (c *RequestContext) JSON(statusCode int, data interface{}) error {
	c.Response.Header().Set("Content-Type", "application/json")
	c.Response.WriteHeader(statusCode)
	b, err := json.Marshal(data)
	if err != nil {
		http.Error(c.Response, "failed to encode json", http.StatusInternalServerError)
		return err
	}
	_, err = c.Response.Write(b)
	return err
}

// Text writes text response
func (c *RequestContext) Text(statusCode int, text string) error {
	c.Response.Header().Set("Content-Type", "text/plain")
	c.Response.WriteHeader(statusCode)
	_, err := c.Response.Write([]byte(text))
	return err
}
