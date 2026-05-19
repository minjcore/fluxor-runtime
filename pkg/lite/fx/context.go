package fx

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/fluxorio/fluxor/pkg/lite/core"
)

// JSON is a convenience alias for JSON objects.
type JSON map[string]any

// Context is a thin wrapper around http request/response plus the core context.
type Context struct {
	W       http.ResponseWriter
	R       *http.Request
	Params  map[string]string
	coreCtx *core.FluxorContext
}

func NewContext(w http.ResponseWriter, r *http.Request, cCtx *core.FluxorContext) *Context {
	return &Context{W: w, R: r, Params: make(map[string]string), coreCtx: cCtx}
}

func (c *Context) Ok(data any) error {
	return c.JSON(http.StatusOK, data)
}

func (c *Context) Error(code int, msg string) error {
	return c.JSON(code, JSON{"error": msg})
}

func (c *Context) JSON(code int, data any) error {
	c.W.Header().Set("Content-Type", "application/json")
	c.W.WriteHeader(code)
	return json.NewEncoder(c.W).Encode(data)
}

func (c *Context) Text(code int, text string) error {
	c.W.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.W.WriteHeader(code)
	_, err := io.WriteString(c.W, text)
	return err
}

func (c *Context) BindJSON(dst any) error {
	if c.R.Body == nil {
		return fmt.Errorf("empty body")
	}

	const maxBody = 1 << 20 // 1MiB
	dec := json.NewDecoder(io.LimitReader(c.R.Body, maxBody))
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}
	// Ensure single JSON value (no trailing garbage)
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("invalid json: multiple values")
		}
		return err
	}
	return nil
}

func (c *Context) Query(key string) string {
	return c.R.URL.Query().Get(key)
}

func (c *Context) Header(key string) string {
	return c.R.Header.Get(key)
}

func (c *Context) SetHeader(key, value string) {
	c.W.Header().Set(key, value)
}

func (c *Context) Param(key string) string {
	return c.Params[key]
}

func (c *Context) Log() *slog.Logger {
	l := c.coreCtx.Log()
	if c.R != nil {
		return l.With("method", c.R.Method, "path", c.R.URL.Path)
	}
	return l
}

func (c *Context) Worker() *core.WorkerPool  { return c.coreCtx.Worker() }
func (c *Context) Bus() *core.Bus            { return c.coreCtx.Bus() }
func (c *Context) Core() *core.FluxorContext { return c.coreCtx }
