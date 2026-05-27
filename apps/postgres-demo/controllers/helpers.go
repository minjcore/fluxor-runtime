package controllers

import (
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/golang-jwt/jwt/v5"
)

// GetUserIDFromContext extracts user_id from JWT token claims
// Returns the username from JWT token, or empty string if not found
func GetUserIDFromContext(ctx *web.FastRequestContext) string {
	claimsInterface := ctx.Get("user")
	if claimsInterface != nil {
		// Check map[string]interface{} first (as per plan)
		if claims, ok := claimsInterface.(map[string]interface{}); ok {
			if username, ok := claims["username"].(string); ok && username != "" {
				return username
			}
		}
		
		// Also handle jwt.MapClaims (actual type used by JWT library)
		if claims, ok := claimsInterface.(jwt.MapClaims); ok {
			if username, ok := claims["username"].(string); ok && username != "" {
				return username
			}
		}
	}
	
	// Fallback to UserID() method
	if uid := ctx.UserID(); uid != "" {
		return uid
	}
	
	return "" // Return empty, let caller decide default
}
