package authn

import (
	"context"
	"time"
)

// Principal represents an authenticated entity (user, service, etc.)
type Principal struct {
	// ID is the unique identifier of the principal
	ID string

	// Type indicates the type of principal (user, service, api_key, etc.)
	Type string

	// Attributes contains additional attributes about the principal
	Attributes map[string]interface{}

	// AuthenticatedAt is when the principal was authenticated
	AuthenticatedAt time.Time

	// ExpiresAt is when the authentication expires (if applicable)
	ExpiresAt *time.Time
}

// IsExpired checks if the principal's authentication has expired
func (p *Principal) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// GetAttribute retrieves an attribute by key
func (p *Principal) GetAttribute(key string) (interface{}, bool) {
	if p.Attributes == nil {
		return nil, false
	}
	val, ok := p.Attributes[key]
	return val, ok
}

// SetAttribute sets an attribute
func (p *Principal) SetAttribute(key string, value interface{}) {
	if p.Attributes == nil {
		p.Attributes = make(map[string]interface{})
	}
	p.Attributes[key] = value
}

// Authenticator is the interface for authentication providers
type Authenticator interface {
	// Authenticate authenticates a credential and returns a Principal
	Authenticate(ctx context.Context, credential string) (*Principal, error)

	// Name returns the name of the authenticator
	Name() string
}

// CredentialValidator validates a credential
type CredentialValidator interface {
	// Validate validates a credential and returns claims/attributes
	Validate(ctx context.Context, credential string) (map[string]interface{}, error)
}

// TokenGenerator generates authentication tokens
type TokenGenerator interface {
	// Generate generates a token with the given claims
	Generate(ctx context.Context, claims map[string]interface{}, expiresIn time.Duration) (string, error)
}

// TokenValidator validates authentication tokens
type TokenValidator interface {
	// Validate validates a token and returns claims
	Validate(ctx context.Context, token string) (map[string]interface{}, error)
}
