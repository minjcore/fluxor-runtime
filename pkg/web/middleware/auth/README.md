# Authentication & Authorization Middleware

This package provides comprehensive authentication and authorization middleware for the Fluxor web framework.

## Overview

The auth middleware package is organized into two main components:

- **authn** - Authentication (who you are)
- **authz** - Authorization (what you can do)

## Package Independence

**Important**: The `authn` and `authz` packages are **completely independent** from the web layer. They:

- ✅ Only depend on Go standard library and each other
- ✅ Can be used in any Go application (CLI tools, gRPC services, workers, etc.)
- ✅ Have no dependencies on `pkg/web` or any framework-specific code
- ✅ Are reusable and framework-agnostic

The middleware layer (`api_key.go`, `jwt.go`, `oauth2.go`, `rbac.go`) provides web-specific adapters that use these independent packages. See [DEPENDENCIES.md](./DEPENDENCIES.md) for a complete dependency analysis.

## Quick Start

### Authentication Middleware

#### JWT Authentication

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth"

// Create JWT middleware
jwtMiddleware := auth.JWT(auth.JWTConfig{
    SecretKey:   "your-secret-key",
    ClaimsKey:   "user",
    TokenLookup: "header:Authorization",
    AuthScheme:  "Bearer",
})

// Apply to routes
router.GETFast("/api/protected", jwtMiddleware(handler))
```

#### API Key Authentication

```go
import (
    "github.com/fluxorio/fluxor/pkg/web/middleware/auth"
    "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/apikey"
)

// Using the new authn package
store := apikey.NewMemoryStore()
hasher := apikey.NewBcryptHasher(10)
manager := apikey.NewManager(store, hasher)
authenticator := apikey.NewAuthenticator(manager)

apiKeyMiddleware := auth.APIKey(auth.APIKeyWithAuthenticator(authenticator))

// Or using simple validator
validKeys := map[string]map[string]interface{}{
    "sk_test_123": {
        "user_id": "123",
        "roles":   []string{"user"},
    },
}
apiKeyMiddleware := auth.APIKey(auth.DefaultAPIKeyConfig(
    auth.SimpleAPIKeyValidator(validKeys),
))
```

#### OAuth2/OIDC Authentication

```go
import (
    "github.com/fluxorio/fluxor/pkg/web/middleware/auth"
    "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/oidc"
)

// Using OIDC provider
provider := oidc.NewProvider(
    "https://accounts.google.com",
    "your-client-id",
    "your-client-secret",
)
authenticator := oidc.NewAuthenticator(provider)

oauth2Middleware := auth.OAuth2(auth.OAuth2WithAuthenticator(authenticator))
```

### Authorization Middleware

#### Role-Based Access Control (RBAC)

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth"

// Require specific role
adminMiddleware := auth.RequireRole("admin")

// Require any of the specified roles
userOrAdminMiddleware := auth.RequireAnyRole("user", "admin")

// Require all specified roles
superUserMiddleware := auth.RequireAllRoles("admin", "moderator")

// Apply to routes
router.DELETEFast("/api/users/:id", adminMiddleware(deleteUserHandler))
```

## Package Structure

```
auth/
├── authn/              # Authentication
│   ├── apikey/         # API key authentication
│   └── oidc/           # OpenID Connect authentication
├── authz/              # Authorization
│   ├── abac/           # Attribute-Based Access Control
│   ├── permissions/    # Fine-grained permissions
│   ├── scopes/         # OAuth2-style scopes
│   └── rbac.go         # Role-Based Access Control
├── api_key.go          # API key middleware
├── jwt.go              # JWT middleware
├── oauth2.go           # OAuth2 middleware
└── rbac.go             # RBAC middleware helpers
```

## Features

### Authentication Features

- **JWT Token Validation** - Validate JWT tokens with configurable secret keys
- **API Key Authentication** - Secure API key management with bcrypt hashing
- **OAuth2/OIDC Support** - Full OpenID Connect provider integration
- **Principal Management** - Structured principal objects with attributes
- **Token Expiration** - Built-in expiration checking

### Authorization Features

- **RBAC** - Role-Based Access Control with flexible permission mapping
- **ABAC** - Attribute-Based Access Control with policy engine
- **Permissions** - Fine-grained permission management
- **Scopes** - OAuth2-style scope validation
- **Wildcard Support** - Pattern matching for resources and actions

## Examples

See the [examples directory](../../../../examples) for complete working examples.

## Testing

Run all tests:

```bash
go test ./pkg/web/middleware/auth/...
```

Run tests with coverage:

```bash
go test -cover ./pkg/web/middleware/auth/...
```

## Documentation

- [Authentication (authn) Documentation](./authn/README.md)
- [Authorization (authz) Documentation](./authz/README.md)

## License

Copyright (c) 2024-2026 Fluxor Framework
All rights reserved.
