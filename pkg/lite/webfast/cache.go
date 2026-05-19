package webfast

import (
	"bytes"
	"strings"

	"github.com/fluxorio/fluxor/pkg/lite/fx"
)

// CacheConfig configures HTTP caching headers for responses.
//
// This is a simple "header middleware" intended for high-RPS endpoints with
// mostly-static responses.
type CacheConfig struct {
	// CacheControl sets the `Cache-Control` header value verbatim.
	// Example: `public, max-age=60, immutable`
	CacheControl string

	// ETag sets the `ETag` header. If set, middleware will short-circuit with 304
	// when `If-None-Match` matches (strong match on exact bytes, or within a comma-separated list).
	ETag string

	// Vary sets the `Vary` header value (comma-separated).
	Vary []string
}

// Cache is a middleware that sets caching headers and supports conditional GET via ETag.
func Cache(cfg CacheConfig) Middleware {
	cacheControlKey := []byte("Cache-Control")
	etagKey := []byte("ETag")
	varyKey := []byte("Vary")
	ifNoneMatchKey := "If-None-Match"

	ccVal := []byte(cfg.CacheControl)
	etagVal := []byte(cfg.ETag)

	var varyVal []byte
	if len(cfg.Vary) > 0 {
		varyVal = []byte(strings.Join(cfg.Vary, ", "))
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *fx.FastContext) error {
			// Conditional request
			if len(etagVal) > 0 {
				inm := c.RC.Request.Header.Peek(ifNoneMatchKey)
				if etagMatch(inm, etagVal) {
					c.RC.SetStatusCode(304)
					c.RC.Response.Header.SetBytesKV(etagKey, etagVal)
					if len(ccVal) > 0 {
						c.RC.Response.Header.SetBytesKV(cacheControlKey, ccVal)
					}
					if len(varyVal) > 0 {
						c.RC.Response.Header.SetBytesKV(varyKey, varyVal)
					}
					return nil
				}
			}

			// Apply headers for normal response
			if len(ccVal) > 0 {
				c.RC.Response.Header.SetBytesKV(cacheControlKey, ccVal)
			}
			if len(etagVal) > 0 {
				c.RC.Response.Header.SetBytesKV(etagKey, etagVal)
			}
			if len(varyVal) > 0 {
				c.RC.Response.Header.SetBytesKV(varyKey, varyVal)
			}

			return next(c)
		}
	}
}

func etagMatch(ifNoneMatch []byte, etag []byte) bool {
	if len(ifNoneMatch) == 0 || len(etag) == 0 {
		return false
	}

	// Fast path: exact match
	if bytes.Equal(bytes.TrimSpace(ifNoneMatch), etag) {
		return true
	}

	// Handle comma-separated list: W/"x", "y"
	parts := bytes.Split(ifNoneMatch, []byte(","))
	for _, p := range parts {
		if bytes.Equal(bytes.TrimSpace(p), etag) {
			return true
		}
	}
	return false
}
