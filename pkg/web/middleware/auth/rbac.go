package auth

import (
	"fmt"

	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/golang-jwt/jwt/v5"
)

// User represents an authenticated user
type User struct {
	ID    string
	Roles []string
	// Additional user data
	Data map[string]interface{}
}

// RBACConfig configures Role-Based Access Control
type RBACConfig struct {
	// GetUser extracts user from request context
	GetUser func(ctx *web.FastRequestContext) (*User, error)

	// GetRoles extracts roles from user
	GetRoles func(user *User) []string
}

// RequireRole creates middleware that requires a specific role
func RequireRole(role string) web.FastMiddleware {
	return RequireAnyRole(role)
}

// RequireAnyRole creates middleware that requires any of the specified roles
func RequireAnyRole(roles ...string) web.FastMiddleware {
	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// Try to get user from context (default key: "user")
			userInterface := ctx.Get("user")
			if userInterface == nil {
				ctx.RequestCtx.SetStatusCode(401)
				ctx.RequestCtx.SetContentType("application/json")
				if _, err := ctx.RequestCtx.WriteString(`{"error":"unauthorized","message":"user not found in context"}`); err != nil {
					// Best-effort response write; ignore on error.
				}
				return nil
			}

			// Extract user roles
			var userRoles []string
			switch u := userInterface.(type) {
			case *User:
				userRoles = u.Roles
			case jwt.MapClaims:
				// Extract roles from JWT claims
				if rolesClaim, ok := u["roles"].([]interface{}); ok {
					for _, r := range rolesClaim {
						if roleStr, ok := r.(string); ok {
							userRoles = append(userRoles, roleStr)
						}
					}
				}
			case map[string]interface{}:
				// Extract roles from map
				if rolesClaim, ok := u["roles"].([]interface{}); ok {
					for _, r := range rolesClaim {
						if roleStr, ok := r.(string); ok {
							userRoles = append(userRoles, roleStr)
						}
					}
				}
			default:
				ctx.RequestCtx.SetStatusCode(403)
				ctx.RequestCtx.SetContentType("application/json")
				if _, err := ctx.RequestCtx.WriteString(`{"error":"forbidden","message":"invalid user type"}`); err != nil {
					// Best-effort response write; ignore on error.
				}
				return nil
			}

			// Check if user has any of the required roles
			hasRole := false
			for _, requiredRole := range roles {
				for _, userRole := range userRoles {
					if userRole == requiredRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				ctx.RequestCtx.SetStatusCode(403)
				ctx.RequestCtx.SetContentType("application/json")
				if _, err := ctx.RequestCtx.WriteString(fmt.Sprintf(`{"error":"forbidden","message":"insufficient permissions, required roles: %v"}`, roles)); err != nil {
					// Best-effort response write; ignore on error.
				}
				return nil
			}

			return next(ctx)
		}
	}
}

// RequireAllRoles creates middleware that requires all of the specified roles
func RequireAllRoles(roles ...string) web.FastMiddleware {
	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// Try to get user from context
			userInterface := ctx.Get("user")
			if userInterface == nil {
				ctx.RequestCtx.SetStatusCode(401)
				ctx.RequestCtx.SetContentType("application/json")
				if _, err := ctx.RequestCtx.WriteString(`{"error":"unauthorized","message":"user not found in context"}`); err != nil {
					// Best-effort response write; ignore on error.
				}
				return nil
			}

			// Extract user roles
			var userRoles []string
			switch u := userInterface.(type) {
			case *User:
				userRoles = u.Roles
			case jwt.MapClaims:
				if rolesClaim, ok := u["roles"].([]interface{}); ok {
					for _, r := range rolesClaim {
						if roleStr, ok := r.(string); ok {
							userRoles = append(userRoles, roleStr)
						}
					}
				}
			case map[string]interface{}:
				if rolesClaim, ok := u["roles"].([]interface{}); ok {
					for _, r := range rolesClaim {
						if roleStr, ok := r.(string); ok {
							userRoles = append(userRoles, roleStr)
						}
					}
				}
			default:
				ctx.RequestCtx.SetStatusCode(403)
				ctx.RequestCtx.SetContentType("application/json")
				if _, err := ctx.RequestCtx.WriteString(`{"error":"forbidden","message":"invalid user type"}`); err != nil {
					// Best-effort response write; ignore on error.
				}
				return nil
			}

			// Check if user has all required roles
			roleMap := make(map[string]bool)
			for _, role := range userRoles {
				roleMap[role] = true
			}

			for _, requiredRole := range roles {
				if !roleMap[requiredRole] {
					ctx.RequestCtx.SetStatusCode(403)
					ctx.RequestCtx.SetContentType("application/json")
					if _, err := ctx.RequestCtx.WriteString(fmt.Sprintf(`{"error":"forbidden","message":"missing required role: %s"}`, requiredRole)); err != nil {
						// Best-effort response write; ignore on error.
					}
					return nil
				}
			}

			return next(ctx)
		}
	}
}
