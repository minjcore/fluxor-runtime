# Error Handler Package

Package `errorhandler` provides comprehensive error handling utilities for Fluxor applications.

## Features

- **Structured Error Types**: Error codes, severity levels, and context
- **Error Wrapping**: Wrap existing errors with additional context
- **Panic Recovery**: Utilities for recovering from panics
- **HTTP Error Transformation**: Convert errors to HTTP responses
- **Error Handler Chains**: Chain multiple error handlers together
- **Error Filtering**: Filter and suppress errors based on conditions

## Installation

```go
import "github.com/fluxorio/fluxor/pkg/errorhandler"
```

## Basic Usage

### Creating Errors

```go
// Create a new error
err := errorhandler.New(errorhandler.ErrorCodeNotFound, "Resource not found")

// Add context
err = err.WithContext("resource_id", "123")
err = err.WithContext("user_id", "456")

// Set severity
err = err.WithSeverity(errorhandler.ErrorSeverityHigh)
```

### Wrapping Errors

```go
// Wrap an existing error
originalErr := errors.New("database connection failed")
wrapped := errorhandler.Wrap(originalErr, errorhandler.ErrorCodeInternal, "Failed to fetch user")

// Check error codes
if errorhandler.Is(err, errorhandler.ErrorCodeNotFound) {
    // Handle not found
}

// Extract FluxorError
if fluxorErr, ok := errorhandler.As(err); ok {
    fmt.Printf("Error code: %s\n", fluxorErr.Code)
    fmt.Printf("Severity: %s\n", fluxorErr.Severity)
}
```

### HTTP Error Transformation

```go
err := errorhandler.New(errorhandler.ErrorCodeNotFound, "User not found")
httpErr := errorhandler.ToHTTPError(err)

// httpErr.Status = 404
// httpErr.Code = "NOT_FOUND"
// httpErr.Message = "User not found"
```

### Panic Recovery

```go
// Recover from panic and return error
defer func() {
    if err := errorhandler.RecoverWithError(); err != nil {
        // Handle recovered error
        log.Printf("Recovered: %v", err)
    }
}()

// Recover and wrap
defer func() {
    if err := errorhandler.RecoverAndWrap(
        errorhandler.ErrorCodeInternal,
        "Unexpected error occurred",
    ); err != nil {
        // Handle error
    }
}()

// Safe function call
err := errorhandler.SafeCall(func() error {
    // Risky operation
    return doSomething()
}, func(panicValue interface{}, stackTrace []byte) error {
    return errorhandler.New(
        errorhandler.ErrorCodeInternal,
        fmt.Sprintf("Panic: %v", panicValue),
    )
})
```

### Error Handlers

```go
// Log handler
logHandler := errorhandler.NewLogHandler()

// Transform handler
transformHandler := errorhandler.NewTransformHandler(func(err error) error {
    // Transform error
    return errorhandler.Wrap(err, errorhandler.ErrorCodeInternal, "Transformed")
})

// Filter handler
filterHandler := errorhandler.NewFilterHandler(
    func(err error) bool {
        // Only handle critical errors
        fluxorErr, ok := errorhandler.As(err)
        return ok && fluxorErr.Severity == errorhandler.ErrorSeverityCritical
    },
    logHandler,
)

// Chain handlers
chain := errorhandler.Chain(logHandler, transformHandler, filterHandler)
err := chain.Handle(someError)
```

## Error Codes

- `ErrorCodeUnknown`: Unknown error
- `ErrorCodeValidation`: Validation error (400)
- `ErrorCodeNotFound`: Not found error (404)
- `ErrorCodeUnauthorized`: Unauthorized error (401)
- `ErrorCodeForbidden`: Forbidden error (403)
- `ErrorCodeConflict`: Conflict error (409)
- `ErrorCodeInternal`: Internal server error (500)
- `ErrorCodeTimeout`: Timeout error (408)
- `ErrorCodeRateLimit`: Rate limit error (429)
- `ErrorCodeServiceUnavailable`: Service unavailable error (503)

## Error Severity

- `ErrorSeverityLow`: Low severity (informational)
- `ErrorSeverityMedium`: Medium severity (default)
- `ErrorSeverityHigh`: High severity (needs attention)
- `ErrorSeverityCritical`: Critical severity (immediate action required)

## Examples

### Web Handler Example

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    user, err := getUser(r.URL.Query().Get("id"))
    if err != nil {
        httpErr := errorhandler.ToHTTPError(err)
        w.WriteHeader(httpErr.Status)
        json.NewEncoder(w).Encode(httpErr)
        return
    }
    // ...
}
```

### Service Layer Example

```go
func (s *UserService) GetUser(id string) (*User, error) {
    user, err := s.repo.FindByID(id)
    if err != nil {
        return nil, errorhandler.Wrap(
            err,
            errorhandler.ErrorCodeNotFound,
            "User not found",
        ).WithContext("user_id", id)
    }
    return user, nil
}
```

### Panic Recovery Example

```go
func riskyOperation() (result string, err error) {
    defer func() {
        if recovered := errorhandler.RecoverWithError(); recovered != nil {
            err = recovered
        }
    }()
    
    // Operation that might panic
    result = doSomethingRisky()
    return result, nil
}
```

## Best Practices

1. **Use appropriate error codes**: Choose error codes that accurately represent the error condition
2. **Add context**: Use `WithContext()` to add relevant information for debugging
3. **Set severity**: Use `WithSeverity()` to indicate the importance of the error
4. **Wrap errors**: Use `Wrap()` to preserve error chains while adding context
5. **Handle panics**: Use recovery utilities for operations that might panic
6. **Transform for APIs**: Use `ToHTTPError()` when returning errors in HTTP responses

## Integration with Other Packages

This package works well with:
- `pkg/core/logger`: For structured logging
- `pkg/web`: For HTTP error handling
- `pkg/core/failfast`: For fail-fast error handling
