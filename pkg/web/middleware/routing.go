package middleware

import (
	"github.com/fluxorio/fluxor/pkg/web"
)

// RoutingMiddleware automatically extracts and sets routing identifiers
// This middleware is optional - FastRequestContext methods can extract routing info directly
// Use this middleware if you want to pre-extract and cache routing identifiers
type RoutingMiddleware struct {
	// ExtractUserID enables automatic UserID extraction from JWT/Header
	ExtractUserID bool
	// ExtractFloxID enables automatic FloxID extraction from Header/Path
	ExtractFloxID bool
	// JWTClaimsKey is the key used to store JWT claims (default: "user")
	JWTClaimsKey string
}

// DefaultRoutingMiddleware creates a routing middleware with default settings
func DefaultRoutingMiddleware() *RoutingMiddleware {
	return &RoutingMiddleware{
		ExtractUserID: true,
		ExtractFloxID: true,
		JWTClaimsKey:  "user",
	}
}

// Handler returns the middleware handler function
func (m *RoutingMiddleware) Handler() web.FastMiddleware {
	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// Extract UserID if enabled
			if m.ExtractUserID {
				userID := ctx.UserID()
				if userID != "" {
					// Cache in context for reuse
					ctx.Set("user_id", userID)
					// Also set in response header for tracing
					ctx.RequestCtx.Response.Header.Set("X-User-ID", userID)
				}
			}

			// Extract FloxID if enabled
			if m.ExtractFloxID {
				floxid := ctx.FloxID()
				if floxid != "" {
					// Cache in context for reuse
					ctx.Set("floxid", floxid)
					// Also set in response header for tracing
					ctx.RequestCtx.Response.Header.Set("X-Flox-ID", floxid)
				}
			}

			// Extract RequestID and set in response header
			if requestID := ctx.RequestID(); requestID != "" {
				ctx.RequestCtx.Response.Header.Set("X-Request-ID", requestID)
			}

			return next(ctx)
		}
	}
}
