# Authentication Package (authn)

The `authn` package provides authentication functionality for the Fluxor framework. It defines core interfaces and types for authenticating users and services.

## Package Independence

**This package is completely independent** from the web layer and can be used in any Go application:

- ✅ **No web dependencies**: Only uses Go standard library (`context`, `errors`, `time`)
- ✅ **Reusable**: Can be used in CLI tools, gRPC services, message queue workers, or any Go application
- ✅ **Framework-agnostic**: Not tied to any specific framework or HTTP library
- ✅ **Minimal external deps**: Only `golang.org/x/crypto/bcrypt` for API key hashing (security-critical)

The package location (`pkg/web/middleware/auth/authn`) is for organizational purposes only - it does not depend on the web layer.

## Overview

Authentication answers the question: **"Who are you?"**

The package provides:
- Core authentication interfaces and types
- Principal representation (authenticated entities)
- Multiple authentication providers (API keys, OIDC)
- Extensible architecture for custom authenticators

## Core Concepts

### Principal

A `Principal` represents an authenticated entity (user, service, API key, etc.):

```go
type Principal struct {
    ID              string                 // Unique identifier
    Type            string                 // Type: "user", "api_key", "oidc", etc.
    Attributes      map[string]interface{} // Additional attributes
    AuthenticatedAt time.Time              // When authentication occurred
    ExpiresAt       *time.Time             // Optional expiration
}
```

### Authenticator Interface

All authentication providers implement the `Authenticator` interface:

```go
type Authenticator interface {
    Authenticate(ctx context.Context, credential string) (*Principal, error)
    Name() string
}
```

## Sub-packages

### API Key Authentication (`apikey`)

Secure API key generation, validation, and management.

**Features:**
- Secure key generation with configurable prefixes
- Bcrypt hashing for key storage
- SHA256 lookup hashes for fast retrieval
- Key expiration and revocation
- Scope-based permissions
- In-memory and persistent storage support

**Example:**

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/apikey"

// Create a key manager
store := apikey.NewMemoryStore() // or your custom Store implementation
hasher := apikey.NewBcryptHasher(10)
manager := apikey.NewManager(store, hasher, 
    apikey.WithPrefix("sk_live_"),
    apikey.WithKeyLength(32),
)

// Generate a new API key
ctx := context.Background()
key, keyRecord, err := manager.Generate(ctx, "user123", "My API Key", 
    []string{"read", "write"}, nil) // nil = no expiration

// Validate an API key
keyRecord, err := manager.Validate(ctx, key)

// Create an authenticator
authenticator := apikey.NewAuthenticator(manager)
principal, err := authenticator.Authenticate(ctx, key)
```

**Key Management:**

```go
// List all keys for a principal
keys, err := manager.List(ctx, "user123")

// Revoke a key
err := manager.Revoke(ctx, keyID)

// Delete a key
err := manager.Delete(ctx, keyID)
```

### OpenID Connect Authentication (`oidc`)

Full OIDC provider integration with discovery and token validation.

**Features:**
- OIDC discovery document caching
- Token introspection
- User info retrieval
- Automatic provider configuration

**Example:**

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/oidc"

// Create an OIDC provider
provider := oidc.NewProvider(
    "https://accounts.google.com",
    "your-client-id",
    "your-client-secret",
)

// Discover provider configuration
doc, err := provider.Discover(ctx)

// Get user information
userInfo, err := provider.GetUserInfo(ctx, accessToken)

// Create an authenticator
authenticator := oidc.NewAuthenticator(provider)
principal, err := authenticator.Authenticate(ctx, accessToken)
```

## Usage with Middleware

The authn package is designed to work seamlessly with the auth middleware:

```go
import (
    "github.com/fluxorio/fluxor/pkg/web/middleware/auth"
    "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/apikey"
)

// Setup
store := apikey.NewMemoryStore()
hasher := apikey.NewBcryptHasher(10)
manager := apikey.NewManager(store, hasher)
authenticator := apikey.NewAuthenticator(manager)

// Create middleware
apiKeyMiddleware := auth.APIKey(auth.APIKeyWithAuthenticator(authenticator))

// Use in routes
router.GETFast("/api/protected", apiKeyMiddleware(handler))
```

## Principal Attributes

Principals can store arbitrary attributes that can be used for authorization:

```go
principal := &authn.Principal{
    ID:   "user123",
    Type: "user",
    Attributes: map[string]interface{}{
        "roles":  []string{"admin", "user"},
        "email":  "user@example.com",
        "scopes": []string{"read", "write"},
    },
}

// Get attribute
role, ok := principal.GetAttribute("roles")

// Set attribute
principal.SetAttribute("department", "engineering")
```

## Error Handling

The package defines standard authentication errors:

```go
var (
    ErrInvalidCredential    = errors.New("invalid credential")
    ErrExpiredCredential     = errors.New("credential has expired")
    ErrMissingCredential     = errors.New("credential is missing")
    ErrUnauthorized          = errors.New("unauthorized")
    ErrTokenExpired          = errors.New("token has expired")
    ErrTokenInvalid          = errors.New("token is invalid")
    ErrProviderUnavailable   = errors.New("authentication provider unavailable")
)
```

## Custom Authenticators

You can create custom authenticators by implementing the `Authenticator` interface:

```go
type CustomAuthenticator struct {
    // Your fields
}

func (a *CustomAuthenticator) Authenticate(ctx context.Context, credential string) (*authn.Principal, error) {
    // Validate credential
    // Return principal or error
}

func (a *CustomAuthenticator) Name() string {
    return "custom"
}
```

## Testing

Run tests:

```bash
go test ./pkg/web/middleware/auth/authn/...
```

## See Also

- [Authorization Package](../authz/README.md)
- [Main Auth Middleware Documentation](../README.md)
