package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Provider represents an OIDC provider
type Provider struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client

	// Cached discovery document
	discoveryDoc *DiscoveryDocument
	mu           sync.RWMutex
}

// DiscoveryDocument contains OIDC provider metadata
type DiscoveryDocument struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserInfoEndpoint                 string   `json:"userinfo_endpoint"`
	JWKSURI                          string   `json:"jwks_uri"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                  []string `json:"scopes_supported"`
	ClaimsSupported                  []string `json:"claims_supported"`
}

// NewProvider creates a new OIDC provider
func NewProvider(issuerURL, clientID, clientSecret string) *Provider {
	return &Provider{
		IssuerURL:    issuerURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Discover fetches the OIDC discovery document
func (p *Provider) Discover(ctx context.Context) (*DiscoveryDocument, error) {
	// Check cache first
	p.mu.RLock()
	if p.discoveryDoc != nil {
		doc := p.discoveryDoc
		p.mu.RUnlock()
		return doc, nil
	}
	p.mu.RUnlock()

	// Fetch discovery document
	discoveryURL := p.IssuerURL
	if discoveryURL[len(discoveryURL)-1] != '/' {
		discoveryURL += "/"
	}
	discoveryURL += ".well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery endpoint returned status %d", resp.StatusCode)
	}

	var doc DiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to decode discovery document: %w", err)
	}

	// Cache the document
	p.mu.Lock()
	p.discoveryDoc = &doc
	p.mu.Unlock()

	return &doc, nil
}

// GetUserInfo fetches user information from the userinfo endpoint
func (p *Provider) GetUserInfo(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	doc, err := p.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover provider: %w", err)
	}

	if doc.UserInfoEndpoint == "" {
		return nil, fmt.Errorf("userinfo endpoint not available")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", doc.UserInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned status %d", resp.StatusCode)
	}

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return userInfo, nil
}

// IntrospectToken introspects an access token
func (p *Provider) IntrospectToken(ctx context.Context, token string) (map[string]interface{}, error) {
	_, err := p.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover provider: %w", err)
	}

	// Use token introspection endpoint if available
	// Otherwise, validate the token using JWKS
	// For now, we'll use a simple introspection approach
	// In production, you should validate the JWT signature using JWKS

	// This is a simplified version - in production you should:
	// 1. Parse the JWT
	// 2. Get the JWKS from jwks_uri
	// 3. Validate the signature
	// 4. Validate claims (iss, aud, exp, etc.)

	return map[string]interface{}{
		"active": true,
		"token":  token,
	}, nil
}
