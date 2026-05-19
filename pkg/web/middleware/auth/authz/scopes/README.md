# OAuth2-Style Scope Validation

The `scopes` package provides OAuth2-style scope validation for fine-grained access control.

## Overview

Scopes are used to limit what an access token can do. This package allows you to:
- Define scopes with actions and resources
- Validate scopes against principals
- Check if principals have required scopes
- Create authorizers with scope requirements

## Quick Start

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz/scopes"

// Create scope validator
getScopes := func(p *authn.Principal) []string {
    scopes, _ := p.GetAttribute("scopes")
    if scopesSlice, ok := scopes.([]string); ok {
        return scopesSlice
    }
    return []string{}
}

validator := scopes.NewValidator(getScopes)

// Define scopes
validator.DefineScope(&scopes.ScopeDefinition{
    Name:        "read",
    Description: "Read access",
    Actions:     []string{"read", "get"},
    Resources:   []string{"*"},
})

// Check scope
principal := &authn.Principal{
    Attributes: map[string]interface{}{
        "scopes": []string{"read", "write"},
    },
}

hasScope := validator.HasScope(principal, "read")
```

## Scope Definitions

### Defining Scopes

```go
validator.DefineScope(&scopes.ScopeDefinition{
    Name:        "read",
    Description: "Read access",
    Actions:     []string{"read", "get"},
    Resources:   []string{"*"},
})

validator.DefineScope(&scopes.ScopeDefinition{
    Name:        "write",
    Description: "Write access",
    Actions:     []string{"write", "update", "create"},
    Resources:   []string{"*"},
})

validator.DefineScope(&scopes.ScopeDefinition{
    Name:        "admin",
    Description: "Administrative access",
    Actions:     []string{"*"},
    Resources:   []string{"*"},
})
```

### Scope Structure

```go
type ScopeDefinition struct {
    Name        string   // Scope name (e.g., "read", "write")
    Description string   // Human-readable description
    Actions     []string // Allowed actions
    Resources   []string // Resource patterns
}
```

## Scope Extraction

Scopes are extracted from principals using a function:

```go
getScopes := func(p *authn.Principal) []string {
    // Extract from attributes
    scopes, _ := p.GetAttribute("scopes")
    if scopesSlice, ok := scopes.([]string); ok {
        return scopesSlice
    }
    return []string{}
}

validator := scopes.NewValidator(getScopes)
```

**Common Patterns:**

```go
// From JWT claims
getScopes := func(p *authn.Principal) []string {
    if scopes, ok := p.GetAttribute("scopes"); ok {
        if scopesSlice, ok := scopes.([]string); ok {
            return scopesSlice
        }
    }
    return []string{}
}

// From OAuth2 token
getScopes := func(p *authn.Principal) []string {
    if token, ok := p.GetAttribute("token"); ok {
        if tokenMap, ok := token.(map[string]interface{}); ok {
            if scopes, ok := tokenMap["scope"].(string); ok {
                return strings.Split(scopes, " ")
            }
        }
    }
    return []string{}
}
```

## Checking Scopes

### Has Scope

Check if a principal has a specific scope:

```go
hasScope := validator.HasScope(principal, "read")
```

### Has Any Scope

Check if a principal has any of the specified scopes:

```go
hasAny := validator.HasAnyScope(principal, "read", "write", "admin")
```

### Has All Scopes

Check if a principal has all of the specified scopes:

```go
hasAll := validator.HasAllScopes(principal, "read", "write")
```

### Check Scope

Check if a principal's scopes allow an action on a resource:

```go
canDo := validator.CheckScope(principal, "read", "user:123")
```

This checks:
1. If the principal has a scope that allows the action
2. If the resource matches the scope's resource patterns

## Authorizer

Create an authorizer with required scopes:

```go
// Require specific scopes
authorizer := validator.Authorizer("read", "write")

decision, err := authorizer.Authorize(ctx, &authz.Request{
    Principal: principal,
    Action:    "read",
    Resource:  "user:123",
})
```

The authorizer:
- Requires the principal to have ALL specified scopes
- Then checks if scopes allow the action on the resource

## Examples

### Basic Usage

```go
validator := scopes.NewValidator(getScopes)

// Define scopes
validator.DefineScope(&scopes.ScopeDefinition{
    Name:    "read",
    Actions: []string{"read", "get"},
    Resources: []string{"*"},
})

validator.DefineScope(&scopes.ScopeDefinition{
    Name:    "write",
    Actions: []string{"write", "update", "create"},
    Resources: []string{"*"},
})

// Principal with scopes
principal := &authn.Principal{
    Attributes: map[string]interface{}{
        "scopes": []string{"read", "write"},
    },
}

// Check scopes
validator.HasScope(principal, "read")        // true
validator.HasScope(principal, "admin")     // false
validator.HasAnyScope(principal, "read", "admin") // true
validator.HasAllScopes(principal, "read", "write") // true
```

### OAuth2 Integration

```go
// Extract scopes from OAuth2 token
getScopes := func(p *authn.Principal) []string {
    if token, ok := p.GetAttribute("oauth2_token"); ok {
        if tokenMap, ok := token.(map[string]interface{}); ok {
            if scopeStr, ok := tokenMap["scope"].(string); ok {
                return strings.Split(scopeStr, " ")
            }
        }
    }
    return []string{}
}

validator := scopes.NewValidator(getScopes)

// Define OAuth2-style scopes
validator.DefineScope(&scopes.ScopeDefinition{
    Name:    "read:users",
    Actions: []string{"read"},
    Resources: []string{"user:*"},
})

validator.DefineScope(&scopes.ScopeDefinition{
    Name:    "write:users",
    Actions: []string{"write", "update"},
    Resources: []string{"user:*"},
})

// Check with OAuth2 scope format
principal := &authn.Principal{
    Attributes: map[string]interface{}{
        "oauth2_token": map[string]interface{}{
            "scope": "read:users write:users",
        },
    },
}

validator.HasScope(principal, "read:users") // true
```

### Resource-Specific Scopes

```go
validator.DefineScope(&scopes.ScopeDefinition{
    Name:    "read",
    Actions: []string{"read"},
    Resources: []string{"user:*", "file:*"},
})

validator.DefineScope(&scopes.ScopeDefinition{
    Name:    "admin",
    Actions: []string{"*"},
    Resources: []string{"*"},
})

// Check resource access
principal := &authn.Principal{
    Attributes: map[string]interface{}{
        "scopes": []string{"read"},
    },
}

validator.CheckScope(principal, "read", "user:123")  // true
validator.CheckScope(principal, "read", "file:doc")  // true
validator.CheckScope(principal, "delete", "user:123") // false
```

### Authorizer with Required Scopes

```go
// Create authorizer requiring specific scopes
authorizer := validator.Authorizer("read", "write")

// Principal with required scopes
principal := &authn.Principal{
    Attributes: map[string]interface{}{
        "scopes": []string{"read", "write"},
    },
}

req := &authz.Request{
    Principal: principal,
    Action:    "read",
    Resource:  "user:123",
}

decision, err := authorizer.Authorize(ctx, req)
// Decision: Allow (has required scopes and action is allowed)
```

## Resource Patterns

### Wildcards

```go
Resources: []string{"*"}        // Matches everything
Resources: []string{"user:*"}   // Matches user:123, user:456, etc.
Resources: []string{"file:/path/*"} // Prefix matching
```

### Exact Match

```go
Resources: []string{"user:123"} // Exact match only
```

## Best Practices

1. **Use Standard Scope Names** - Follow OAuth2 conventions (e.g., `read`, `write`, `admin`)
2. **Be Specific** - Use resource-specific scopes when needed (e.g., `read:users`, `write:files`)
3. **Document Scopes** - Use descriptions to explain what scopes allow
4. **Validate Early** - Check scopes as early as possible in the request flow
5. **Scope Hierarchy** - Use broad scopes (like `admin`) for administrative access

## OAuth2 Compatibility

This package is designed to work with OAuth2 tokens:

- Scopes are typically space-separated in tokens: `"read write admin"`
- Extract scopes from token claims or attributes
- Validate scopes against defined scope definitions
- Support standard OAuth2 scope patterns

## See Also

- [Authorization Package Documentation](../README.md)
- [Main Auth Middleware Documentation](../../README.md)
