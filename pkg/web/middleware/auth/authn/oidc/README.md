# OpenID Connect (OIDC) Authentication

The `oidc` package provides OpenID Connect authentication functionality with provider discovery and token validation.

## Features

- **OIDC Discovery** - Automatic provider configuration discovery
- **Token Introspection** - Validate access tokens
- **User Info Retrieval** - Fetch user information from userinfo endpoint
- **Caching** - Discovery document caching for performance
- **Standard Compliance** - Full OIDC specification support

## Quick Start

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/oidc"

// Create an OIDC provider
provider := oidc.NewProvider(
    "https://accounts.google.com",  // Issuer URL
    "your-client-id",
    "your-client-secret",
)

// Create an authenticator
authenticator := oidc.NewAuthenticator(provider)

// Authenticate a token
ctx := context.Background()
principal, err := authenticator.Authenticate(ctx, accessToken)
```

## Provider

### Creating a Provider

```go
provider := oidc.NewProvider(issuerURL, clientID, clientSecret)
```

The provider automatically:
- Discovers the OIDC configuration
- Caches the discovery document
- Handles token validation

### Discovery

The provider automatically discovers OIDC configuration from the well-known endpoint:

```
GET {issuer}/.well-known/openid-configuration
```

**Example:**

```go
doc, err := provider.Discover(ctx)
if err != nil {
    return err
}

fmt.Printf("Authorization endpoint: %s\n", doc.AuthorizationEndpoint)
fmt.Printf("Token endpoint: %s\n", doc.TokenEndpoint)
fmt.Printf("UserInfo endpoint: %s\n", doc.UserInfoEndpoint)
fmt.Printf("JWKS URI: %s\n", doc.JWKSURI)
```

### User Info

Retrieve user information from the userinfo endpoint:

```go
userInfo, err := provider.GetUserInfo(ctx, accessToken)
if err != nil {
    return err
}

email := userInfo["email"].(string)
name := userInfo["name"].(string)
```

### Token Introspection

Introspect an access token:

```go
claims, err := provider.IntrospectToken(ctx, accessToken)
if err != nil {
    return err
}

active := claims["active"].(bool)
sub := claims["sub"].(string)
```

## Authenticator

The authenticator validates tokens and creates principals:

```go
authenticator := oidc.NewAuthenticator(provider)

principal, err := authenticator.Authenticate(ctx, accessToken)
if err != nil {
    return err
}

// Principal contains:
// - ID: Subject (sub claim)
// - Type: "oidc"
// - Attributes: All user info and claims
// - ExpiresAt: Token expiration
```

**Principal Attributes:**

The authenticator extracts standard OIDC claims:
- `sub` - Subject (user ID)
- `email` - Email address
- `name` - Full name
- `exp` - Expiration timestamp
- All other claims from userinfo and token

## Supported Providers

The package works with any OIDC-compliant provider:

- **Google** - `https://accounts.google.com`
- **Microsoft Azure AD** - `https://login.microsoftonline.com/{tenant-id}`
- **Auth0** - `https://{domain}.auth0.com`
- **Keycloak** - `https://{keycloak-server}/realms/{realm}`
- **Okta** - `https://{domain}.okta.com`
- Any OIDC-compliant provider

## Example: Full Integration

```go
package main

import (
    "context"
    "fmt"
    "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/oidc"
)

func main() {
    // Create provider
    provider := oidc.NewProvider(
        "https://accounts.google.com",
        "your-client-id",
        "your-client-secret",
    )
    
    ctx := context.Background()
    
    // Discover configuration
    doc, err := provider.Discover(ctx)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Discovered endpoints:\n")
    fmt.Printf("  Authorization: %s\n", doc.AuthorizationEndpoint)
    fmt.Printf("  Token: %s\n", doc.TokenEndpoint)
    fmt.Printf("  UserInfo: %s\n", doc.UserInfoEndpoint)
    
    // Get user info
    accessToken := "your-access-token"
    userInfo, err := provider.GetUserInfo(ctx, accessToken)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("User email: %s\n", userInfo["email"])
    
    // Authenticate
    authenticator := oidc.NewAuthenticator(provider)
    principal, err := authenticator.Authenticate(ctx, accessToken)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Authenticated principal: %s\n", principal.ID)
    fmt.Printf("Email: %s\n", principal.Attributes["email"])
}
```

## Discovery Document

The discovery document contains provider metadata:

```go
type DiscoveryDocument struct {
    Issuer                           string
    AuthorizationEndpoint            string
    TokenEndpoint                    string
    UserInfoEndpoint                 string
    JWKSURI                          string
    ResponseTypesSupported           []string
    SubjectTypesSupported            []string
    IDTokenSigningAlgValuesSupported []string
    ScopesSupported                  []string
    ClaimsSupported                  []string
}
```

## Caching

The provider caches the discovery document to avoid repeated requests. The cache is stored in memory and persists for the lifetime of the provider instance.

## Error Handling

Common errors:

- Provider discovery failures
- Invalid token format
- Expired tokens
- Invalid token signatures (when implemented)
- Network errors

## Security Considerations

1. **Token Validation** - Always validate token signatures (implement JWKS validation)
2. **HTTPS Only** - Use HTTPS for all OIDC endpoints
3. **Token Storage** - Never store tokens in logs or error messages
4. **Expiration** - Always check token expiration
5. **Scope Validation** - Validate scopes match expected permissions

## Future Enhancements

The current implementation provides basic OIDC support. Future enhancements may include:

- JWT signature validation using JWKS
- ID token validation
- Refresh token support
- PKCE support
- Token revocation checking

## See Also

- [Authentication Package Documentation](../README.md)
- [Main Auth Middleware Documentation](../../README.md)
