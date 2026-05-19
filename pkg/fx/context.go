package fx

import (
	"fmt"
	"io"
	"net/http"

	"github.com/fluxorio/fluxor/pkg/core"
)

// Context is a wrapper for HTTP Handler with Fluxor core context integration
type Context struct {
	W       http.ResponseWriter
	R       *http.Request
	coreCtx core.FluxorContext
}

// NewContext creates a new Context instance
func NewContext(w http.ResponseWriter, r *http.Request, cCtx core.FluxorContext) *Context {
	return &Context{W: w, R: r, coreCtx: cCtx}
}

// JSON writes JSON response using core.JSONEncode (standardized)
func (c *Context) JSON(code int, data any) error {
	c.W.Header().Set("Content-Type", "application/json")
	c.W.WriteHeader(code)
	jsonData, err := core.JSONEncode(data)
	if err != nil {
		return fmt.Errorf("json encode error: %w", err)
	}
	_, err = c.W.Write(jsonData)
	if err != nil {
		return fmt.Errorf("write response error: %w", err)
	}
	return nil
}

// Ok writes a 200 OK JSON response (Dev UX)
func (c *Context) Ok(data any) error {
	return c.JSON(http.StatusOK, data)
}

// Error writes an error JSON response (Dev UX)
func (c *Context) Error(code int, msg string) error {
	return c.JSON(code, JSON{"error": msg})
}

// Text writes a plain text response
func (c *Context) Text(code int, text string) error {
	c.W.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.W.WriteHeader(code)
	_, err := io.WriteString(c.W, text)
	return err
}

// BindJSON binds JSON request body to a struct using core.JSONDecode
func (c *Context) BindJSON(dst any) error {
	if c.R.Body == nil {
		return fmt.Errorf("empty request body")
	}
	defer c.R.Body.Close()

	body, err := io.ReadAll(c.R.Body)
	if err != nil {
		return fmt.Errorf("read body error: %w", err)
	}

	if len(body) == 0 {
		return fmt.Errorf("empty request body")
	}

	if err := core.JSONDecode(body, dst); err != nil {
		return fmt.Errorf("json decode error: %w", err)
	}

	return nil
}

// Query returns the query parameter value for the given key
func (c *Context) Query(key string) string {
	return c.R.URL.Query().Get(key)
}

// Header returns the request header value for the given key
func (c *Context) Header(key string) string {
	return c.R.Header.Get(key)
}

// SetHeader sets a response header
func (c *Context) SetHeader(key, value string) {
	c.W.Header().Set(key, value)
}

// EventBus returns the EventBus from core context
func (c *Context) EventBus() core.EventBus {
	return c.coreCtx.EventBus()
}

// GoCMD returns the GoCMD instance from core context (kept as GoCMD for backward compatibility)
func (c *Context) GoCMD() core.GoCMD {
	return c.coreCtx.GoCMD()
}
