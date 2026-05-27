package middleware

import (
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
)

// APIProtectionMiddleware provides additional security for API endpoints
// This middleware should be used in combination with JWT middleware
type APIProtectionMiddleware struct {
	// AllowedOrigins for CORS (if needed)
	AllowedOrigins []string
	// Rate limiting can be added here
}

// NewAPIProtectionMiddleware creates a new API protection middleware
func NewAPIProtectionMiddleware() *APIProtectionMiddleware {
	return &APIProtectionMiddleware{
		AllowedOrigins: []string{"*"}, // Allow all origins by default
	}
}

// ProtectAPI is a middleware that adds security headers and validates API requests
func (m *APIProtectionMiddleware) ProtectAPI(next web.FastRequestHandler) web.FastRequestHandler {
	return func(ctx *web.FastRequestContext) error {
		path := string(ctx.Path())
		
		// Only apply to API endpoints (paths starting with /api/)
		if !strings.HasPrefix(path, "/api/") {
			return next(ctx)
		}

		// Skip protection for public auth endpoints
		if strings.HasPrefix(path, "/api/auth/login") ||
			strings.HasPrefix(path, "/api/auth/register") ||
			strings.HasPrefix(path, "/api/health") {
			return next(ctx)
		}

		// Add security headers
		ctx.RequestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
		ctx.RequestCtx.Response.Header.Set("X-Frame-Options", "DENY")
		ctx.RequestCtx.Response.Header.Set("X-XSS-Protection", "1; mode=block")
		ctx.RequestCtx.Response.Header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Validate Content-Type for POST requests
		method := string(ctx.Method())
		if method == "POST" || method == "PUT" || method == "PATCH" {
			contentType := string(ctx.RequestCtx.Request.Header.ContentType())
			// Allow JSON and form data
			if !strings.Contains(contentType, "application/json") &&
				!strings.Contains(contentType, "application/x-www-form-urlencoded") &&
				!strings.Contains(contentType, "multipart/form-data") {
				// For API endpoints, require JSON
				if strings.HasPrefix(path, "/api/") {
					return ctx.JSON(400, map[string]interface{}{
						"error":   "invalid_content_type",
						"message": "Content-Type must be application/json for API endpoints",
					})
				}
			}
		}

		return next(ctx)
	}
}

// RequireAuth is a helper middleware that ensures authentication is present
// This is a wrapper around JWT middleware for explicit protection
func RequireAuth(jwtMiddleware web.FastMiddleware) web.FastMiddleware {
	return func(next web.FastRequestHandler) web.FastRequestHandler {
		// Apply JWT middleware first
		protectedHandler := jwtMiddleware(next)
		return protectedHandler
	}
}
