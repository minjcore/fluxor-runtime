// Package authn provides authentication functionality for the Fluxor framework.
//
// Package Independence:
//
// This package is completely independent from the web layer and can be used
// in any Go application (CLI tools, gRPC services, workers, etc.). It only
// depends on Go standard library packages (context, errors, time) and has
// no dependencies on pkg/web or any framework-specific code. The package
// location is for organizational purposes only.
//
// This package defines core authentication interfaces and types that can be
// used with various authentication providers. It includes:
//
//   - Principal: Represents an authenticated entity
//   - Authenticator: Interface for authentication providers
//   - CredentialValidator: Interface for validating credentials
//   - TokenGenerator/TokenValidator: Interfaces for token management
//
// Sub-packages:
//
//   - apikey: API key authentication
//   - oidc: OpenID Connect authentication
//
// Example usage:
//
//	// Using API key authentication
//	import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/apikey"
//
//	manager := apikey.NewManager(store, hasher)
//	authenticator := apikey.NewAuthenticator(manager)
//	principal, err := authenticator.Authenticate(ctx, apiKey)
//
//	// Using OIDC authentication
//	import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/oidc"
//
//	provider := oidc.NewProvider(issuerURL, clientID, clientSecret)
//	authenticator := oidc.NewAuthenticator(provider)
//	principal, err := authenticator.Authenticate(ctx, accessToken)
package authn
