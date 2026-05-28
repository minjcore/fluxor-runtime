// Copyright (c) 2026 Fluxor Framework
// SPDX-License-Identifier: MIT

package handlers

import (
	"crypto/subtle"
	"encoding/base64"
	"log"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
	"golang.org/x/crypto/bcrypt"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled bool
	Users   map[string]string // username -> bcrypt hashed password
}

// AuthMiddleware provides HTTP Basic Authentication
type AuthMiddleware struct {
	config AuthConfig
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config AuthConfig) *AuthMiddleware {
	return &AuthMiddleware{
		config: config,
	}
}

// Middleware returns a middleware function for authentication
func (a *AuthMiddleware) Middleware() web.FastMiddleware {
	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// Skip auth if disabled
			if !a.config.Enabled {
				return next(ctx)
			}

			// Extract credentials from Authorization header
			authHeader := string(ctx.RequestCtx.Request.Header.Peek("Authorization"))
			if authHeader == "" {
				return a.unauthorized(ctx, "missing authorization header")
			}

			// Check for Basic auth scheme
			if !strings.HasPrefix(authHeader, "Basic ") {
				return a.unauthorized(ctx, "invalid authorization scheme")
			}

			// Decode credentials
			credentials, err := base64.StdEncoding.DecodeString(authHeader[6:])
			if err != nil {
				return a.unauthorized(ctx, "invalid credentials encoding")
			}

			// Parse username:password
			parts := strings.SplitN(string(credentials), ":", 2)
			if len(parts) != 2 {
				return a.unauthorized(ctx, "invalid credentials format")
			}

			username := parts[0]
			password := parts[1]

			// Validate credentials
			if !a.validateCredentials(username, password) {
				log.Printf("Authentication failed for user: %s", username)
				return a.unauthorized(ctx, "invalid username or password")
			}

			// Store username in context for logging/auditing
			ctx.RequestCtx.SetUserValue("username", username)

			return next(ctx)
		}
	}
}

// ReadOnlyMiddleware allows read-only operations without auth
// Write operations (POST, PUT, DELETE) require authentication
func (a *AuthMiddleware) ReadOnlyMiddleware() web.FastMiddleware {
	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// Skip auth if disabled
			if !a.config.Enabled {
				return next(ctx)
			}

			// Allow read-only methods without auth
			method := string(ctx.RequestCtx.Method())
			if method == "GET" || method == "HEAD" || method == "OPTIONS" {
				return next(ctx)
			}

			// Require auth for write operations
			return a.Middleware()(next)(ctx)
		}
	}
}

// validateCredentials validates username and password
func (a *AuthMiddleware) validateCredentials(username, password string) bool {
	// Get stored password hash for user
	storedHash, exists := a.config.Users[username]
	if !exists {
		// Perform dummy bcrypt comparison to prevent timing attacks
		bcrypt.CompareHashAndPassword([]byte("$2a$10$dummyhashfortiminattackprevention"), []byte(password))
		return false
	}

	// Compare password with stored hash
	err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	return err == nil
}

// unauthorized sends a 401 response with WWW-Authenticate header
func (a *AuthMiddleware) unauthorized(ctx *web.FastRequestContext, reason string) error {
	ctx.RequestCtx.Response.Header.Set("WWW-Authenticate", `Basic realm="GoProxy Module Registry"`)
	return ctx.JSON(401, map[string]string{
		"error":  "unauthorized",
		"reason": reason,
	})
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword verifies a password against a bcrypt hash
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ConstantTimeCompare performs a constant-time string comparison
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ParseUsers parses a list of user configurations
func ParseUsers(users []struct {
	Username string `json:"username"`
	Password string `json:"password"`
}) map[string]string {
	result := make(map[string]string)
	for _, u := range users {
		result[u.Username] = u.Password
	}
	return result
}
