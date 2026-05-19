# Authorization Package (authz)

The `authz` package provides authorization functionality for the Fluxor framework. It defines interfaces and implementations for controlling what authenticated users can do.

## Package Independence

**This package is completely independent** from the web layer and can be used in any Go application:

- ✅ **No web dependencies**: Only uses Go standard library (`context`, `errors`, `fmt`)
- ✅ **Reusable**: Can be used in CLI tools, gRPC services, message queue workers, or any Go application
- ✅ **Framework-agnostic**: Not tied to any specific framework or HTTP library
- ✅ **Minimal dependencies**: Only depends on `authn` package (for Principal type) and standard library

The package location (`pkg/web/middleware/auth/authz`) is for organizational purposes only - it does not depend on the web layer.

## Overview

Authorization answers the question: **"What can you do?"**

The package provides:
- Multiple authorization models (RBAC, ABAC, Permissions, Scopes)
- Flexible policy evaluation
- Extensible authorizer interface
- Decision-based authorization

## Core Concepts

### Decision

Authorization decisions are represented by the `Decision` type:

```go
type Decision int

const (
    DecisionDeny    Decision = iota  // Explicitly deny access
    DecisionAllow                     // Explicitly allow access
    DecisionAbstain                  // No opinion (pass to next authorizer)
)
```

### Request

An authorization request contains:

```go
type Request struct {
    Principal *authn.Principal      // Who is requesting
    Action    string                 // What action (e.g., "read", "write", "delete")
    Resource  string                 // What resource (e.g., "user:123", "file:/path")
    Context   map[string]interface{} // Additional context
}
```

### Authorizer Interface

All authorization providers implement the `Authorizer` interface:

```go
type Authorizer interface {
    Authorize(ctx context.Context, req *Request) (Decision, error)
}
```

## Sub-packages

### Role-Based Access Control (`rbac.go`)

Simple and effective RBAC implementation.

**Example:**

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz"

// Create RBAC authorizer
getRoles := func(p *authn.Principal) []string {
    roles, _ := p.GetAttribute("roles")
    if rolesSlice, ok := roles.([]string); ok {
        return rolesSlice
    }
    return []string{}
}

rbac := authz.NewRBACAuthorizer(getRoles)

// Assign permissions to roles
rbac.AssignRole("admin", []authz.Permission{
    {Action: "*", Resource: "*"}, // Admin can do anything
})

rbac.AssignRole("user", []authz.Permission{
    {Action: "read", Resource: "user:*"},
    {Action: "write", Resource: "user:own"},
})

// Authorize a request
decision, err := rbac.Authorize(ctx, &authz.Request{
    Principal: principal,
    Action:    "delete",
    Resource:  "user:123",
})

if decision == authz.DecisionAllow {
    // Allow access
}
```

**Simple Role Checker:**

```go
// Require a specific role
authorizer := authz.RequireRole("admin")
decision, err := authorizer.Authorize(ctx, req)
```

### Attribute-Based Access Control (`abac`)

Flexible ABAC with policy-based evaluation.

**Example:**

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz/abac"

// Create ABAC engine
engine := abac.NewEngine()

// Add policies
engine.AddPolicy(&abac.Policy{
    ID:     "policy1",
    Name:   "Allow admins to delete",
    Effect: "allow",
    PrincipalConditions: map[string]interface{}{
        "role": "admin",
    },
    ActionConditions: map[string]interface{}{
        "action": "delete",
    },
})

// Deny policy (always wins)
engine.AddPolicy(&abac.Policy{
    ID:     "policy2",
    Name:   "Deny delete on weekends",
    Effect: "deny",
    EnvironmentConditions: map[string]interface{}{
        "day_of_week": "saturday",
    },
})

// Create authorizer
authorizer := engine.Authorizer()

// Evaluate request
decision, err := authorizer.Authorize(ctx, &authz.Request{
    Principal: principal,
    Action:    "delete",
    Resource:  "user:123",
    Context: map[string]interface{}{
        "day_of_week": "saturday",
    },
})
```

**Condition Operators:**

ABAC supports various condition operators:

```go
// Equality
PrincipalConditions: map[string]interface{}{
    "role": "admin",
}

// Comparison operators (for numeric values)
PrincipalConditions: map[string]interface{}{
    "age": map[string]interface{}{
        "$gte": 18,  // Greater than or equal
        "$lt":   65, // Less than
    },
}

// In array
PrincipalConditions: map[string]interface{}{
    "department": map[string]interface{}{
        "$in": []interface{}{"engineering", "product"},
    },
}
```

### Permissions (`permissions`)

Fine-grained permission management with definitions.

**Example:**

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz/permissions"

// Create permissions manager
manager := permissions.NewManager()

// Define permissions
manager.DefinePermission(&permissions.PermissionDefinition{
    Name:        "users.read",
    Description: "Read user information",
    Actions:     []string{"read", "get"},
    Resources:   []string{"user:*"},
})

manager.DefinePermission(&permissions.PermissionDefinition{
    Name:        "users.write",
    Description: "Write user information",
    Actions:     []string{"write", "update", "create"},
    Resources:   []string{"user:*"},
})

// Assign permissions to roles
manager.AssignRolePermission("user", "users.read")
manager.AssignRolePermission("admin", "users.read")
manager.AssignRolePermission("admin", "users.write")

// Or assign directly to principals
manager.AssignPermission("user123", "users.read")

// Check permissions
principal := &authn.Principal{
    ID: "user123",
    Attributes: map[string]interface{}{
        "roles": []string{"user"},
    },
}

hasPermission := manager.HasPermission(principal, "users.read")
canDo := manager.CheckPermission(principal, "read", "user:123")

// Create authorizer
authorizer := manager.Authorizer()
decision, err := authorizer.Authorize(ctx, &authz.Request{
    Principal: principal,
    Action:    "read",
    Resource:  "user:123",
})
```

### Scopes (`scopes`)

OAuth2-style scope validation.

**Example:**

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

validator.DefineScope(&scopes.ScopeDefinition{
    Name:        "write",
    Description: "Write access",
    Actions:     []string{"write", "update", "create"},
    Resources:   []string{"*"},
})

// Check scopes
principal := &authn.Principal{
    Attributes: map[string]interface{}{
        "scopes": []string{"read", "write"},
    },
}

hasScope := validator.HasScope(principal, "read")
hasAny := validator.HasAnyScope(principal, "read", "admin")
hasAll := validator.HasAllScopes(principal, "read", "write")

// Create authorizer with required scopes
authorizer := validator.Authorizer("read", "write")
decision, err := authorizer.Authorize(ctx, &authz.Request{
    Principal: principal,
    Action:    "read",
    Resource:  "user:123",
})
```

## Resource Patterns

All authorization models support resource pattern matching:

- `*` - Matches everything
- `user:*` - Matches all user resources
- `user:123` - Exact match
- `file:/path/*` - Prefix matching

## Combining Authorization Models

You can combine multiple authorization models:

```go
// Use RBAC for role-based checks
rbac := authz.NewRBACAuthorizer(getRoles)
rbac.AssignRole("admin", []authz.Permission{
    {Action: "*", Resource: "*"},
})

// Use ABAC for complex policies
abacEngine := abac.NewEngine()
abacEngine.AddPolicy(&abac.Policy{
    Effect: "allow",
    PrincipalConditions: map[string]interface{}{
        "department": "engineering",
    },
})

// Chain authorizers
func combinedAuthorizer(ctx context.Context, req *authz.Request) (authz.Decision, error) {
    // Check RBAC first
    decision, err := rbac.Authorize(ctx, req)
    if err != nil || decision == authz.DecisionDeny {
        return decision, err
    }
    
    // Then check ABAC
    return abacEngine.Authorizer().Authorize(ctx, req)
}
```

## Usage with Middleware

Authorization works seamlessly with authentication middleware:

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth"

// Authenticate first
jwtMiddleware := auth.JWT(auth.JWTConfig{
    SecretKey: "secret",
    ClaimsKey: "user",
})

// Then authorize
adminMiddleware := auth.RequireRole("admin")

// Apply both
router.DELETEFast("/api/users/:id", 
    jwtMiddleware(
        adminMiddleware(
            deleteUserHandler,
        ),
    ),
)
```

## Error Handling

Standard authorization errors:

```go
var (
    ErrUnauthorized   = errors.New("unauthorized")
    ErrForbidden      = errors.New("forbidden")
    ErrInvalidRequest = errors.New("invalid authorization request")
    ErrPolicyNotFound = errors.New("policy not found")
    ErrInvalidPolicy  = errors.New("invalid policy")
)
```

## Custom Authorizers

Create custom authorizers by implementing the `Authorizer` interface:

```go
type CustomAuthorizer struct {
    // Your fields
}

func (a *CustomAuthorizer) Authorize(ctx context.Context, req *authz.Request) (authz.Decision, error) {
    // Your authorization logic
    if shouldAllow(req) {
        return authz.DecisionAllow, nil
    }
    return authz.DecisionDeny, nil
}
```

## Testing

Run tests:

```bash
go test ./pkg/web/middleware/auth/authz/...
```

## See Also

- [Authentication Package](../authn/README.md)
- [Main Auth Middleware Documentation](../README.md)
