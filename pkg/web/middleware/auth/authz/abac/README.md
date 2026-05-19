# Attribute-Based Access Control (ABAC)

The `abac` package provides flexible Attribute-Based Access Control with a policy engine.

## Overview

ABAC uses attributes (characteristics) of principals, resources, actions, and environment to make authorization decisions. This provides fine-grained, context-aware access control.

## Key Concepts

- **Policies** - Rules that define access based on attributes
- **Conditions** - Attribute-based matching criteria
- **Effect** - Allow or deny decision
- **Evaluation Order** - Deny always wins, then first matching allow

## Quick Start

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz/abac"

// Create ABAC engine
engine := abac.NewEngine()

// Add a policy
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

// Create authorizer
authorizer := engine.Authorizer()

// Evaluate request
decision, err := authorizer.Authorize(ctx, &authz.Request{
    Principal: principal,
    Action:    "delete",
    Resource:  "user:123",
})
```

## Policies

### Policy Structure

```go
type Policy struct {
    ID                      string                 // Unique identifier
    Name                    string                 // Human-readable name
    Effect                  string                 // "allow" or "deny"
    PrincipalConditions     map[string]interface{} // Principal attributes
    ResourceConditions      map[string]interface{} // Resource attributes
    ActionConditions        map[string]interface{} // Action attributes
    EnvironmentConditions   map[string]interface{} // Context/environment
}
```

### Adding Policies

```go
engine := abac.NewEngine()

// Allow policy
engine.AddPolicy(&abac.Policy{
    ID:     "allow-admins",
    Effect: "allow",
    PrincipalConditions: map[string]interface{}{
        "role": "admin",
    },
})

// Deny policy (always wins)
engine.AddPolicy(&abac.Policy{
    ID:     "deny-weekends",
    Effect: "deny",
    EnvironmentConditions: map[string]interface{}{
        "day_of_week": "saturday",
    },
})
```

### Removing Policies

```go
engine.RemovePolicy("policy-id")
```

## Conditions

### Simple Equality

```go
PrincipalConditions: map[string]interface{}{
    "role": "admin",
    "department": "engineering",
}
```

### Comparison Operators

For numeric values:

```go
PrincipalConditions: map[string]interface{}{
    "age": map[string]interface{}{
        "$gte": 18,  // Greater than or equal
        "$lt":   65, // Less than
    },
    "salary": map[string]interface{}{
        "$gt": 50000,  // Greater than
        "$lte": 200000, // Less than or equal
    },
}
```

### Array Operators

```go
PrincipalConditions: map[string]interface{}{
    "department": map[string]interface{}{
        "$in": []interface{}{"engineering", "product"},
    },
    "role": map[string]interface{}{
        "$ne": "guest", // Not equal
    },
}
```

### Multiple Conditions

All conditions must match (AND logic):

```go
PrincipalConditions: map[string]interface{}{
    "role": "admin",
    "department": "engineering",
    "active": true,
}
```

## Evaluation Logic

1. **Deny Always Wins** - If any deny policy matches, access is denied
2. **First Allow Wins** - If an allow policy matches, access is granted
3. **Default Deny** - If no policies match, access is denied

**Example:**

```go
// Policy 1: Allow admins
engine.AddPolicy(&abac.Policy{
    ID:     "allow-admins",
    Effect: "allow",
    PrincipalConditions: map[string]interface{}{
        "role": "admin",
    },
})

// Policy 2: Deny on weekends (always wins)
engine.AddPolicy(&abac.Policy{
    ID:     "deny-weekends",
    Effect: "deny",
    EnvironmentConditions: map[string]interface{}{
        "day_of_week": "saturday",
    },
})

// Request on Saturday with admin role
// Result: DENY (deny policy wins)
```

## Examples

### Role-Based Access

```go
engine.AddPolicy(&abac.Policy{
    ID:     "admin-full-access",
    Effect: "allow",
    PrincipalConditions: map[string]interface{}{
        "role": "admin",
    },
})
```

### Department-Based Access

```go
engine.AddPolicy(&abac.Policy{
    ID:     "engineering-access",
    Effect: "allow",
    PrincipalConditions: map[string]interface{}{
        "department": "engineering",
    },
    ResourceConditions: map[string]interface{}{
        "resource": "code:*", // Engineering can access code resources
    },
})
```

### Time-Based Access

```go
engine.AddPolicy(&abac.Policy{
    ID:     "business-hours",
    Effect: "allow",
    EnvironmentConditions: map[string]interface{}{
        "hour": map[string]interface{}{
            "$gte": 9,
            "$lt": 17,
        },
        "day_of_week": map[string]interface{}{
            "$in": []interface{}{"monday", "tuesday", "wednesday", "thursday", "friday"},
        },
    },
})
```

### Resource Ownership

```go
engine.AddPolicy(&abac.Policy{
    ID:     "own-resources",
    Effect: "allow",
    PrincipalConditions: map[string]interface{}{
        "user_id": "${resource.owner_id}", // Dynamic matching
    },
    ResourceConditions: map[string]interface{}{
        "resource": "user:*",
    },
})
```

### IP-Based Access

```go
engine.AddPolicy(&abac.Policy{
    ID:     "internal-network",
    Effect: "allow",
    EnvironmentConditions: map[string]interface{}{
        "ip": map[string]interface{}{
            "$in": []interface{}{"10.0.0.0/8", "192.168.0.0/16"},
        },
    },
})
```

## Request Context

Provide additional context in authorization requests:

```go
req := &authz.Request{
    Principal: principal,
    Action:    "delete",
    Resource:  "user:123",
    Context: map[string]interface{}{
        "ip":         "192.168.1.1",
        "hour":       14,
        "day_of_week": "monday",
        "user_agent": "Mozilla/5.0",
    },
}
```

## Authorizer

Create an authorizer from the engine:

```go
authorizer := engine.Authorizer()

decision, err := authorizer.Authorize(ctx, req)
if decision == authz.DecisionAllow {
    // Allow access
}
```

## Best Practices

1. **Order Matters** - Place deny policies first for clarity
2. **Be Specific** - Use specific conditions to avoid unintended access
3. **Test Policies** - Test all policy combinations
4. **Document Policies** - Use descriptive names and comments
5. **Monitor Decisions** - Log authorization decisions for auditing

## Performance Considerations

- Policy evaluation is sequential
- Consider policy ordering for performance
- Cache policy evaluation results when possible
- Use indexes for frequently checked attributes

## See Also

- [Authorization Package Documentation](../README.md)
- [Main Auth Middleware Documentation](../../README.md)
