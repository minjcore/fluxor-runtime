# Dependencies Report

This document provides a comprehensive overview of all dependencies for the authentication and authorization packages.

## Overview

The `authn` and `authz` packages are **completely independent** from the web layer. They can be used in any Go application, not just web applications. The packages only depend on:

- Standard library packages
- Each other (authz depends on authn, which is intentional)
- External dependencies (only when necessary, e.g., bcrypt for API key hashing)

## Package Independence

### Key Points

1. **No web layer dependencies**: Neither `authn` nor `authz` import any packages from `pkg/web` or any other framework-specific code
2. **Reusable**: These packages can be used in CLI tools, gRPC services, message queue workers, or any other Go application
3. **Standard library only**: Core packages use only Go standard library
4. **Minimal external deps**: Only one external dependency (bcrypt) for security-critical functionality

## authn Package Dependencies

### Core Package (`authn`)

**Location**: `pkg/web/middleware/auth/authn/`

**Dependencies**:
- `context` (standard library)
- `errors` (standard library)
- `time` (standard library)

**Files**:
- `types.go` - Core types (Principal, Authenticator interface)
- `errors.go` - Error definitions
- `doc.go` - Package documentation

**External Dependencies**: None

### API Key Sub-package (`authn/apikey`)

**Location**: `pkg/web/middleware/auth/authn/apikey/`

**Dependencies**:
- Standard library:
  - `context`
  - `crypto/rand`
  - `crypto/sha256`
  - `encoding/base64`
  - `fmt`
  - `sync`
  - `time`
- Internal:
  - `github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn` (parent package)
- External:
  - `golang.org/x/crypto/bcrypt` (for secure key hashing)

**Files**:
- `authenticator.go` - API key authenticator implementation
- `key.go` - Key management and validation
- `memory_store.go` - In-memory store implementation

**External Dependencies**: 
- `golang.org/x/crypto/bcrypt` - Required for secure password hashing (industry standard)

### OIDC Sub-package (`authn/oidc`)

**Location**: `pkg/web/middleware/auth/authn/oidc/`

**Dependencies**:
- Standard library:
  - `context`
  - `encoding/json`
  - `fmt`
  - `net/http`
  - `sync`
  - `time`
- Internal:
  - `github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn` (parent package)

**Files**:
- `authenticator.go` - OIDC authenticator implementation
- `provider.go` - OIDC provider with discovery

**External Dependencies**: None

## authz Package Dependencies

### Core Package (`authz`)

**Location**: `pkg/web/middleware/auth/authz/`

**Dependencies**:
- Standard library:
  - `context`
  - `errors`
  - `fmt`
- Internal:
  - `github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn` (for Principal type)

**Files**:
- `types.go` - Core types (Decision, Request, Authorizer interface)
- `errors.go` - Error definitions
- `rbac.go` - Role-Based Access Control implementation
- `doc.go` - Package documentation

**External Dependencies**: None

**Note**: The dependency on `authn` is intentional and necessary because authorization requires the `Principal` type from authentication.

### ABAC Sub-package (`authz/abac`)

**Location**: `pkg/web/middleware/auth/authz/abac/`

**Dependencies**:
- Standard library:
  - `context`
  - `fmt`
  - `reflect`
- Internal:
  - `github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz` (parent package)

**Files**:
- `engine.go` - ABAC policy engine

**External Dependencies**: None

### Permissions Sub-package (`authz/permissions`)

**Location**: `pkg/web/middleware/auth/authz/permissions/`

**Dependencies**:
- Standard library:
  - `context`
  - `fmt`
  - `strings`
- Internal:
  - `github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn` (for Principal type)
  - `github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz` (parent package)

**Files**:
- `manager.go` - Permission manager

**External Dependencies**: None

### Scopes Sub-package (`authz/scopes`)

**Location**: `pkg/web/middleware/auth/authz/scopes/`

**Dependencies**:
- Standard library:
  - `context`
  - `fmt`
  - `strings`
- Internal:
  - `github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn` (for Principal type)
  - `github.com/fluxorio/fluxor/pkg/web/middleware/auth/authz` (parent package)

**Files**:
- `validator.go` - Scope validator

**External Dependencies**: None

## Dependency Graph

```
authn (core)
├── Standard library only
│
├── authn/apikey
│   ├── Standard library
│   ├── authn (parent)
│   └── golang.org/x/crypto/bcrypt (external)
│
└── authn/oidc
    ├── Standard library
    └── authn (parent)

authz (core)
├── Standard library
└── authn (for Principal type)
    │
    ├── authz/abac
    │   ├── Standard library
    │   └── authz (parent)
    │
    ├── authz/permissions
    │   ├── Standard library
    │   ├── authn (for Principal)
    │   └── authz (parent)
    │
    └── authz/scopes
        ├── Standard library
        ├── authn (for Principal)
        └── authz (parent)
```

## Verification

### Check Dependencies

To verify dependencies, run:

```bash
# Check authn package
go list -f '{{.ImportPath}}: {{.Imports}}' ./pkg/web/middleware/auth/authn/...

# Check authz package
go list -f '{{.ImportPath}}: {{.Imports}}' ./pkg/web/middleware/auth/authz/...
```

### Verify No Web Dependencies

To ensure no web layer dependencies:

```bash
# Should return no results
grep -r "pkg/web" pkg/web/middleware/auth/authn/*.go
grep -r "pkg/web" pkg/web/middleware/auth/authz/*.go

# Should only show internal imports (authn/authz)
grep -r "github.com/fluxorio/fluxor/pkg" pkg/web/middleware/auth/authn/*.go
grep -r "github.com/fluxorio/fluxor/pkg" pkg/web/middleware/auth/authz/*.go
```

## Summary

✅ **All packages are independent** from the web layer  
✅ **Only standard library and necessary external dependencies**  
✅ **No circular dependencies**  
✅ **Reusable in any Go application**  
✅ **Clean separation of concerns**

The packages are designed to be framework-agnostic and can be used independently of the Fluxor web framework.
