package controllers

import (
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth"
	"github.com/golang-jwt/jwt/v5"
)

// GetUserIDFromContext extracts username from JWT claims stored by either
// FastJWT (auth.FastClaims) or the standard JWT middleware (jwt.MapClaims).
func GetUserIDFromContext(ctx *web.FastRequestContext) string {
	claimsInterface := ctx.Get("user")
	if claimsInterface == nil {
		if uid := ctx.UserID(); uid != "" {
			return uid
		}
		return ""
	}

	// FastJWT path — typed struct, no map lookup needed.
	if c, ok := claimsInterface.(auth.FastClaims); ok {
		return c.Username
	}

	// Standard jwt.MapClaims path (fallback / legacy).
	if claims, ok := claimsInterface.(jwt.MapClaims); ok {
		if username, ok := claims["username"].(string); ok {
			return username
		}
	}

	// map[string]interface{} (test mocks / older paths).
	if claims, ok := claimsInterface.(map[string]interface{}); ok {
		if username, ok := claims["username"].(string); ok {
			return username
		}
	}

	return ""
}
