// Package oidc provides OpenID Connect (OIDC) authentication functionality.
//
// This package includes:
//   - OIDC provider discovery
//   - Token validation and introspection
//   - User info retrieval
//   - Support for standard OIDC flows
//
// Example usage:
//
//	// Create an OIDC provider
//	provider := oidc.NewProvider(
//		"https://accounts.google.com",
//		"your-client-id",
//		"your-client-secret",
//	)
//
//	// Discover provider configuration
//	doc, err := provider.Discover(ctx)
//
//	// Get user info
//	userInfo, err := provider.GetUserInfo(ctx, accessToken)
//
//	// Create an authenticator
//	authenticator := oidc.NewAuthenticator(provider)
//	principal, err := authenticator.Authenticate(ctx, accessToken)
package oidc
