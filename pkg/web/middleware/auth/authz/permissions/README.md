# Permissions Management

The `permissions` package provides fine-grained permission management with definitions and role-based assignment.

## Overview

The permissions package allows you to:
- Define permissions with actions and resources
- Assign permissions to roles or principals
- Check permissions for specific actions on resources
- Use wildcard patterns for flexible matching

## Quick Start

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz/permissions"

// Create permissions manager
manager := permissions.NewManager()

// Define a permission
manager.DefinePermission(&permissions.PermissionDefinition{
    Name:        "users.read",
    Description: "Read user information",
    Actions:     []string{"read", "get"},
    Resources:   []string{"user:*"},
})

// Assign to role
manager.AssignRolePermission("user", "users.read")

// Check permission
principal := &authn.Principal{
    Attributes: map[string]interface{}{
        "roles": []string{"user"},
    },
}

hasPermission := manager.HasPermission(principal, "users.read")
canDo := manager.CheckPermission(principal, "read", "user:123")
```

## Permission Definitions

### Defining Permissions

```go
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

manager.DefinePermission(&permissions.PermissionDefinition{
    Name:        "files.*",
    Description: "All file operations",
    Actions:     []string{"*"}, // All actions
    Resources:   []string{"file:*"},
})
```

### Permission Structure

```go
type PermissionDefinition struct {
    Name        string   // Permission name (e.g., "users.read")
    Description string   // Human-readable description
    Actions     []string // Allowed actions
    Resources   []string // Resource patterns
}
```

## Assigning Permissions

### To Roles

```go
manager.AssignRolePermission("user", "users.read")
manager.AssignRolePermission("user", "users.write")
manager.AssignRolePermission("admin", "users.*") // Wildcard
```

### To Principals

```go
manager.AssignPermission("user123", "users.read")
manager.AssignPermission("user123", "files.read")
```

## Checking Permissions

### Has Permission

Check if a principal has a specific permission:

```go
hasPermission := manager.HasPermission(principal, "users.read")
```

Supports wildcard matching:
- `users.*` matches `users.read`, `users.write`, etc.

### Check Permission

Check if a principal can perform an action on a resource:

```go
canDo := manager.CheckPermission(principal, "read", "user:123")
```

This checks:
1. If the principal has a permission that allows the action
2. If the resource matches the permission's resource patterns

### Get All Permissions

Get all permissions for a principal (from roles and direct assignments):

```go
perms := manager.GetPermissions(principal)
// Returns: ["users.read", "users.write", "files.read"]
```

## Resource Patterns

### Wildcards

```go
Resources: []string{"user:*"}  // Matches user:123, user:456, etc.
Resources: []string{"*"}        // Matches everything
Resources: []string{"file:/path/*"} // Prefix matching
```

### Exact Match

```go
Resources: []string{"user:123"} // Exact match only
```

## Examples

### Basic Usage

```go
manager := permissions.NewManager()

// Define permissions
manager.DefinePermission(&permissions.PermissionDefinition{
    Name:    "users.read",
    Actions: []string{"read"},
    Resources: []string{"user:*"},
})

manager.DefinePermission(&permissions.PermissionDefinition{
    Name:    "users.write",
    Actions: []string{"write", "update"},
    Resources: []string{"user:*"},
})

// Assign to roles
manager.AssignRolePermission("user", "users.read")
manager.AssignRolePermission("admin", "users.read")
manager.AssignRolePermission("admin", "users.write")

// Check permissions
principal := &authn.Principal{
    Attributes: map[string]interface{}{
        "roles": []string{"user"},
    },
}

// User can read
if manager.CheckPermission(principal, "read", "user:123") {
    // Allow
}

// User cannot write
if !manager.CheckPermission(principal, "write", "user:123") {
    // Deny
}
```

### Wildcard Permissions

```go
// Define wildcard permission
manager.DefinePermission(&permissions.PermissionDefinition{
    Name:    "users.*",
    Actions: []string{"*"},
    Resources: []string{"user:*"},
})

// Assign to admin
manager.AssignRolePermission("admin", "users.*")

// Admin has all user permissions
principal := &authn.Principal{
    Attributes: map[string]interface{}{
        "roles": []string{"admin"},
    },
}

manager.HasPermission(principal, "users.read")  // true
manager.HasPermission(principal, "users.write") // true
manager.HasPermission(principal, "users.delete") // true (wildcard match)
```

### Direct Principal Permissions

```go
// Assign directly to principal (overrides role permissions)
manager.AssignPermission("user123", "users.write")

principal := &authn.Principal{
    ID: "user123",
    Attributes: map[string]interface{}{
        "roles": []string{"user"}, // Role only has read
    },
}

// Principal has write (direct assignment)
manager.HasPermission(principal, "users.write") // true
```

## Authorizer

Create an authorizer from the manager:

```go
authorizer := manager.Authorizer()

decision, err := authorizer.Authorize(ctx, &authz.Request{
    Principal: principal,
    Action:    "read",
    Resource:  "user:123",
})

if decision == authz.DecisionAllow {
    // Allow access
}
```

## Permission Hierarchy

Permissions are checked in this order:

1. **Direct Principal Permissions** - Permissions assigned directly to the principal
2. **Role Permissions** - Permissions from all roles assigned to the principal

Both are combined (union), so a principal has all permissions from both sources.

## Best Practices

1. **Use Descriptive Names** - `users.read` is better than `ur`
2. **Group Related Permissions** - Use prefixes like `users.*`, `files.*`
3. **Define Before Assigning** - Always define permissions before assigning
4. **Use Wildcards Sparingly** - Be specific when possible
5. **Document Permissions** - Use descriptions to explain what permissions allow

## Performance Considerations

- Permission checking is O(n) where n is the number of permissions
- Consider caching permission checks for frequently accessed resources
- Use indexes for role-based lookups in large systems

## See Also

- [Authorization Package Documentation](../README.md)
- [Main Auth Middleware Documentation](../../README.md)
