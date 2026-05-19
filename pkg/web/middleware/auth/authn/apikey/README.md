# API Key Authentication

The `apikey` package provides secure API key generation, validation, and management.

## Features

- **Secure Key Generation** - Cryptographically secure random key generation
- **Bcrypt Hashing** - Secure storage with bcrypt password hashing
- **Fast Lookup** - Peppered scrypt lookup hashes for efficient key retrieval
- **Key Management** - Full CRUD operations for API keys
- **Expiration Support** - Optional key expiration
- **Revocation** - Key revocation without deletion
- **Scope Support** - Associate scopes/permissions with keys
- **Storage Abstraction** - Pluggable storage backends

## Quick Start

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/apikey"

// Create a key manager
store := apikey.NewMemoryStore() // or your custom Store
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

## Key Manager

### Creating a Manager

```go
manager := apikey.NewManager(store, hasher, opts...)
```

**Options:**
- `WithPrefix(prefix)` - Set key prefix (default: "sk_")
- `WithKeyLength(length)` - Set key length in bytes (default: 32)

### Generating Keys

```go
key, keyRecord, err := manager.Generate(
    ctx,
    principalID,  // Owner of the key
    name,         // Human-readable name
    scopes,       // Permissions/scopes
    expiresIn,    // Optional expiration duration
)
```

**Example:**

```go
// Key that never expires
key, record, err := manager.Generate(ctx, "user123", "Production Key", 
    []string{"read", "write"}, nil)

// Key that expires in 30 days
expiresIn := 30 * 24 * time.Hour
key, record, err := manager.Generate(ctx, "user123", "Temporary Key", 
    []string{"read"}, &expiresIn)
```

### Validating Keys

```go
keyRecord, err := manager.Validate(ctx, apiKey)
if err != nil {
    // Invalid, expired, or revoked key
    return err
}

// Key is valid, use keyRecord
fmt.Printf("Key belongs to: %s\n", keyRecord.PrincipalID)
fmt.Printf("Scopes: %v\n", keyRecord.Scopes)
```

### Key Management

```go
// List all keys for a principal
keys, err := manager.List(ctx, "user123")

// Revoke a key (marks as revoked but doesn't delete)
err := manager.Revoke(ctx, keyID)

// Delete a key permanently
err := manager.Delete(ctx, keyID)
```

## Storage

### Memory Store (Testing/Development)

```go
store := apikey.NewMemoryStore()
```

The memory store is useful for testing and development. For production, implement your own `Store` interface.

### Custom Store Implementation

```go
type Store interface {
    Create(ctx context.Context, key *Key) error
    GetByID(ctx context.Context, id string) (*Key, error)
    GetByHash(ctx context.Context, hash string) (*Key, error)
    ListByPrincipal(ctx context.Context, principalID string) ([]*Key, error)
    Update(ctx context.Context, key *Key) error
    Delete(ctx context.Context, id string) error
    Revoke(ctx context.Context, id string) error
}
```

**Example Database Store:**

```go
type DatabaseStore struct {
    db *sql.DB
}

func (s *DatabaseStore) Create(ctx context.Context, key *Key) error {
    query := `INSERT INTO api_keys (id, key_hash, lookup_hash, ...) VALUES (?, ?, ?, ...)`
    _, err := s.db.ExecContext(ctx, query, key.ID, key.KeyHash, key.LookupHash, ...)
    return err
}

// Implement other methods...
```

## Hashing

### Bcrypt Hasher

```go
hasher := apikey.NewBcryptHasher(10) // cost factor
```

The bcrypt hasher provides:
- `Hash(key string) (string, error)` - Hash a key for storage
- `Verify(key, hash string) bool` - Verify a key against a hash

### Custom Hasher

```go
type Hasher interface {
    Hash(key string) (string, error)
    Verify(key, hash string) bool
}
```

## Key Structure

```go
type Key struct {
    ID          string     // Unique identifier
    KeyHash     string     // Bcrypt hash for verification
    LookupHash  string     // SHA256 hash for fast lookup
    Prefix      string     // Key prefix (e.g., "sk_live_")
    Name        string     // Human-readable name
    PrincipalID string     // Owner of the key
    Scopes      []string   // Permissions/scopes
    CreatedAt   time.Time  // Creation timestamp
    ExpiresAt   *time.Time // Optional expiration
    LastUsedAt  *time.Time // Last usage timestamp
    Revoked     bool       // Revocation status
}
```

## Key Validation

Keys are validated using a two-step process:

1. **Lookup** - Fast lookup using SHA256 hash
2. **Verification** - Secure verification using bcrypt hash

This provides both performance and security.

## Security Considerations

1. **Never store plain keys** - Always use hashed storage
2. **Use strong prefixes** - Distinguish key types (e.g., "sk_live_", "sk_test_")
3. **Rotate keys regularly** - Implement key rotation policies
4. **Monitor usage** - Track `LastUsedAt` for security monitoring
5. **Revoke compromised keys** - Immediate revocation without deletion
6. **Set expiration** - Use expiration for temporary keys

## Example: Full Integration

```go
package main

import (
    "context"
    "fmt"
    "github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/apikey"
)

func main() {
    // Setup
    store := apikey.NewMemoryStore()
    hasher := apikey.NewBcryptHasher(10)
    manager := apikey.NewManager(store, hasher,
        apikey.WithPrefix("sk_live_"),
    )
    
    ctx := context.Background()
    
    // Generate key
    key, record, err := manager.Generate(ctx, "user123", "My API Key", 
        []string{"read", "write"}, nil)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Generated key: %s\n", key)
    fmt.Printf("Key ID: %s\n", record.ID)
    
    // Validate key
    validated, err := manager.Validate(ctx, key)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Validated key for: %s\n", validated.PrincipalID)
    
    // Use with authenticator
    authenticator := apikey.NewAuthenticator(manager)
    principal, err := authenticator.Authenticate(ctx, key)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Authenticated principal: %s\n", principal.ID)
}
```

## Testing

Run tests:

```bash
go test ./pkg/web/middleware/auth/authn/apikey/...
```

## See Also

- [Authentication Package Documentation](../README.md)
- [Main Auth Middleware Documentation](../../README.md)
