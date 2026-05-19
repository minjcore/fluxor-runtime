package oidc

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
)

// Authenticator authenticates requests using OIDC tokens
type Authenticator struct {
	provider *Provider
}

// NewAuthenticator creates a new OIDC authenticator
func NewAuthenticator(provider *Provider) *Authenticator {
	return &Authenticator{
		provider: provider,
	}
}

// Authenticate authenticates an OIDC token and returns a Principal
func (a *Authenticator) Authenticate(ctx context.Context, credential string) (*authn.Principal, error) {
	if credential == "" {
		return nil, authn.ErrMissingCredential
	}

	// Introspect the token
	claims, err := a.provider.IntrospectToken(ctx, credential)
	if err != nil {
		return nil, fmt.Errorf("token introspection failed: %w", err)
	}

	// Check if token is active
	active, ok := claims["active"].(bool)
	if !ok || !active {
		return nil, authn.ErrInvalidCredential
	}

	// Extract subject (user ID)
	sub, ok := claims["sub"].(string)
	if !ok {
		// Try alternative claim names
		if userID, ok := claims["user_id"].(string); ok {
			sub = userID
		} else if email, ok := claims["email"].(string); ok {
			sub = email
		} else {
			return nil, fmt.Errorf("subject not found in token claims")
		}
	}

	// Get user info if available
	userInfo, err := a.provider.GetUserInfo(ctx, credential)
	if err != nil {
		// If userinfo fails, use claims from token
		userInfo = claims
	}

	// Extract expiration
	var expiresAt *time.Time
	if exp, ok := claims["exp"].(float64); ok {
		t := time.Unix(int64(exp), 0)
		expiresAt = &t
		if time.Now().After(t) {
			return nil, authn.ErrExpiredCredential
		}
	}

	// Create principal
	principal := &authn.Principal{
		ID:              sub,
		Type:            "oidc",
		AuthenticatedAt: time.Now(),
		ExpiresAt:       expiresAt,
		Attributes:      make(map[string]interface{}),
	}

	// Copy relevant claims to attributes
	for k, v := range userInfo {
		principal.Attributes[k] = v
	}

	// Ensure standard fields
	if email, ok := userInfo["email"].(string); ok {
		principal.Attributes["email"] = email
	}
	if name, ok := userInfo["name"].(string); ok {
		principal.Attributes["name"] = name
	}

	return principal, nil
}

// Name returns the name of the authenticator
func (a *Authenticator) Name() string {
	return "oidc"
}
