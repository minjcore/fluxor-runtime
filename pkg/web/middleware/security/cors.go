package security

import (
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
)

// CORSConfig configures CORS (Cross-Origin Resource Sharing)
type CORSConfig struct {
	// AllowedOrigins is a list of allowed origins (use "*" for all)
	AllowedOrigins []string

	// AllowedMethods is a list of allowed HTTP methods
	AllowedMethods []string

	// AllowedHeaders is a list of allowed request headers
	AllowedHeaders []string

	// ExposedHeaders is a list of headers that can be exposed to the client
	ExposedHeaders []string

	// AllowCredentials indicates whether credentials can be included
	AllowCredentials bool

	// MaxAge is the maximum age for preflight requests (in seconds)
	MaxAge int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS middleware handles CORS headers
func CORS(config CORSConfig) web.FastMiddleware {
	// Normalize allowed origins
	allowedOriginsMap := make(map[string]bool)
	allowAllOrigins := false
	for _, origin := range config.AllowedOrigins {
		if origin == "*" {
			allowAllOrigins = true
			break
		}
		allowedOriginsMap[origin] = true
	}

	// Normalize allowed methods
	allowedMethodsStr := strings.Join(config.AllowedMethods, ", ")

	// Normalize allowed headers
	allowedHeadersStr := strings.Join(config.AllowedHeaders, ", ")

	// Normalize exposed headers
	exposedHeadersStr := strings.Join(config.ExposedHeaders, ", ")

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			origin := string(ctx.RequestCtx.Request.Header.Peek("Origin"))

			// Handle preflight OPTIONS request
			if string(ctx.Method()) == "OPTIONS" {
				// Set CORS headers
				if allowAllOrigins {
					ctx.RequestCtx.Response.Header.Set("Access-Control-Allow-Origin", "*")
				} else if origin != "" && allowedOriginsMap[origin] {
					ctx.RequestCtx.Response.Header.Set("Access-Control-Allow-Origin", origin)
				}

				ctx.RequestCtx.Response.Header.Set("Access-Control-Allow-Methods", allowedMethodsStr)
				ctx.RequestCtx.Response.Header.Set("Access-Control-Allow-Headers", allowedHeadersStr)

				if exposedHeadersStr != "" {
					ctx.RequestCtx.Response.Header.Set("Access-Control-Expose-Headers", exposedHeadersStr)
				}

				if config.AllowCredentials {
					ctx.RequestCtx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
				}

				if config.MaxAge > 0 {
					ctx.RequestCtx.Response.Header.Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
				}

				ctx.RequestCtx.SetStatusCode(204) // No Content
				return nil
			}

			// Handle actual request
			if origin != "" {
				if allowAllOrigins {
					ctx.RequestCtx.Response.Header.Set("Access-Control-Allow-Origin", "*")
				} else if allowedOriginsMap[origin] {
					ctx.RequestCtx.Response.Header.Set("Access-Control-Allow-Origin", origin)
				}

				if exposedHeadersStr != "" {
					ctx.RequestCtx.Response.Header.Set("Access-Control-Expose-Headers", exposedHeadersStr)
				}

				if config.AllowCredentials {
					ctx.RequestCtx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
				}
			}

			return next(ctx)
		}
	}
}
