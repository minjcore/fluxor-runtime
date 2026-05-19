package apikey

import (
	"context"
	"time"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
)

// Authenticator authenticates requests using API keys
type Authenticator struct {
	manager *Manager
}

// NewAuthenticator creates a new API key authenticator
func NewAuthenticator(manager *Manager) *Authenticator {
	return &Authenticator{
		manager: manager,
	}
}

// Authenticate authenticates an API key and returns a Principal
func (a *Authenticator) Authenticate(ctx context.Context, credential string) (*authn.Principal, error) {
	if credential == "" {
		return nil, authn.ErrMissingCredential
	}

	// Validate the API key
	key, err := a.manager.Validate(ctx, credential)
	if err != nil {
		return nil, err
	}

	// Create principal from key
	principal := &authn.Principal{
		ID:             key.PrincipalID,
		Type:           "api_key",
		AuthenticatedAt: time.Now(),
		Attributes: map[string]interface{}{
			"key_id":   key.ID,
			"key_name": key.Name,
			"scopes":   key.Scopes,
		},
	}

	if key.ExpiresAt != nil {
		principal.ExpiresAt = key.ExpiresAt
	}

	return principal, nil
}

// Name returns the name of the authenticator
func (a *Authenticator) Name() string {
	return "api_key"
}
