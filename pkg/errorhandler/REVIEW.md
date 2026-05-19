# Error Handler Package Review

## Overview
Comprehensive review of the `errorhandler` package dated 2024.

## Strengths ✅

1. **Well-structured design**: Clear separation of concerns across files
2. **Good documentation**: README provides clear examples
3. **Flexible handler pattern**: Chainable handlers with good composition
4. **Structured errors**: FluxorError provides rich context
5. **HTTP integration**: Good mapping between error codes and HTTP status codes

## Critical Issues 🔴

### 1. Error Chain Traversal Not Implemented

**Location**: `types.go` - `As()` and `Is()` functions

**Problem**: These functions only check direct type assertion, not error chains using `errors.As()` and `errors.Is()` from the standard library.

**Current Code**:
```go
func As(err error) (*FluxorError, bool) {
    if err == nil {
        return nil, false
    }
    if fluxorErr, ok := err.(*FluxorError); ok {
        return fluxorErr, true
    }
    return nil, false
}
```

**Issue**: If a FluxorError is wrapped in another error, this won't find it.

**Fix**: Use `errors.As()` to traverse the error chain:
```go
func As(err error) (*FluxorError, bool) {
    if err == nil {
        return nil, false
    }
    var fluxorErr *FluxorError
    if errors.As(err, &fluxorErr) {
        return fluxorErr, true
    }
    return nil, false
}
```

Same issue exists in `Is()` function.

### 2. RetryHandler Not Implemented

**Location**: `handler.go` - `RetryHandler.Handle()`

**Problem**: The retry logic is just a placeholder that returns an error saying "retry not implemented".

**Current Code**:
```go
func (h *RetryHandler) Handle(err error) error {
    // ...
    if h.RetryFunc != nil && h.RetryFunc(err) && h.MaxRetries > 0 {
        // Retry logic would be implemented here
        // This is a placeholder for retry functionality
        return fmt.Errorf("retry not implemented: %w", err)
    }
    // ...
}
```

**Recommendation**: Either implement retry logic or remove this handler until it's needed.

### 3. RecoverWithError() Design Issue

**Location**: `recovery.go` - `RecoverWithError()`

**Problem**: This function doesn't actually recover from panics. By the time it's called, the panic must already be recovered. The function checks `recover()` but if called outside a defer, it will always return nil.

**Current Usage Pattern** (from README):
```go
defer func() {
    if err := errorhandler.RecoverWithError(); err != nil {
        // Handle recovered error
    }
}()
```

**Issue**: This pattern doesn't work because `RecoverWithError()` calls `recover()` internally, but the defer already recovered the panic.

**Fix**: The function should be used differently, or the design should be changed. Consider:
```go
defer func() {
    if r := recover(); r != nil {
        err := errorhandler.RecoverWithError() // This won't work!
        // ...
    }
}()
```

Better design would be:
```go
defer func() {
    if err := errorhandler.RecoverWithError(); err != nil {
        // Handle
    }
}()
// But RecoverWithError needs to be called from within the defer
```

## Design Issues 🟡

### 4. Chain Behavior - Early Termination

**Location**: `handler.go` - `Chain()`

**Current Behavior**: Chain stops processing if any handler returns `nil`.

**Question**: Is this the desired behavior? Sometimes you might want all handlers to process the error, even if one "handles" it by returning nil.

**Recommendation**: Document this behavior clearly, or consider adding a `ChainAll()` variant that processes all handlers.

### 5. Context Preservation in Wrap()

**Location**: `types.go` - `Wrap()`

**Current Code**:
```go
if existing, ok := err.(*FluxorError); ok {
    fluxorErr.Context = existing.Context  // Overwrites, doesn't merge
    fluxorErr.Severity = existing.Severity
}
```

**Issue**: If the new FluxorError already has context, it gets overwritten. Consider merging contexts instead.

### 6. Missing Error Code: Unprocessable Entity (422)

**Location**: `types.go` and `http.go`

**Issue**: HTTP 422 (Unprocessable Entity) is commonly used for validation errors that are syntactically correct but semantically invalid. Currently only 400 (Bad Request) is available.

**Recommendation**: Add `ErrorCodeUnprocessableEntity` mapped to HTTP 422.

### 7. ToHTTPError Loses Error Chain

**Location**: `http.go` - `ToHTTPError()`

**Current Code**:
```go
fluxorErr, ok := As(err)
if !ok {
    // If it's not a FluxorError, create one
    fluxorErr = Wrap(err, ErrorCodeInternal, err.Error())
}
```

**Issue**: If `As()` is fixed to traverse chains, this should work better. But the current implementation loses the original error's context if it's wrapped multiple times.

## Testing Issues 🟡

### 8. Incomplete Test Coverage

**Missing Tests**:
- `FilterHandler` - no tests
- `TransformHandler` - no tests  
- `SuppressHandler` - no tests
- `RetryHandler` - no tests (though it's not implemented)
- `Chain()` with multiple handlers and nil returns
- `Wrap()` with context preservation
- `ToHTTPError()` with non-FluxorError wrapped errors
- `RecoverAndWrap()` - basic test exists but could be more comprehensive
- `SafeCallWithDefault()` - no tests
- Error unwrapping in chains

### 9. RecoverWithError Test Workaround

**Location**: `errorhandler_test.go` - `TestRecoverWithError()`

The test manually creates the error instead of actually using `RecoverWithError()` because of the design issue mentioned above.

## Code Quality Issues 🟢

### 10. Inconsistent Error Handling

Some handlers check for `nil` errors, others don't. Consider standardizing:
- All handlers should check `if err == nil { return nil }` at the start
- Or document that handlers should not be called with nil errors

### 11. Missing Validation

- `NewRetryHandler()` doesn't validate `maxRetries >= 0`
- `NewFilterHandler()` doesn't validate that filter or handler are not nil
- Consider adding validation or documenting expected behavior

### 12. HTTPError Context Mutability

**Location**: `http.go` - `WithContext()`

Returns a new `HTTPError` (value type), which is good, but the original context map is still shared if not copied. Consider:
```go
func (e HTTPError) WithContext(key string, value interface{}) HTTPError {
    newCtx := make(map[string]interface{})
    for k, v := range e.Context {
        newCtx[k] = v
    }
    newCtx[key] = value
    e.Context = newCtx
    return e
}
```

## Recommendations

### High Priority
1. **Fix `As()` and `Is()`** to use `errors.As()` and `errors.Is()` for proper error chain traversal
2. **Implement or remove `RetryHandler`** - don't leave placeholder code
3. **Fix `RecoverWithError()` design** - clarify the intended usage pattern
4. **Add missing tests** for all handlers

### Medium Priority
5. **Add ErrorCodeUnprocessableEntity** for HTTP 422
6. **Improve context merging** in `Wrap()`
7. **Document Chain behavior** regarding early termination
8. **Add input validation** to constructors

### Low Priority
9. **Consider adding metrics/hooks** for error tracking
10. **Consider adding error rate limiting** handler
11. **Consider adding structured logging** integration
12. **Add benchmarks** for performance-critical paths

## Example Fixes

### Fix 1: Proper Error Chain Traversal

```go
// types.go
import "errors"

func Is(err error, code ErrorCode) bool {
    if err == nil {
        return false
    }
    
    var fluxorErr *FluxorError
    if errors.As(err, &fluxorErr) {
        return fluxorErr.Code == code
    }
    
    return false
}

func As(err error) (*FluxorError, bool) {
    if err == nil {
        return nil, false
    }
    
    var fluxorErr *FluxorError
    if errors.As(err, &fluxorErr) {
        return fluxorErr, true
    }
    
    return nil, false
}
```

### Fix 2: Add Unprocessable Entity

```go
// types.go
const (
    // ... existing codes ...
    ErrorCodeUnprocessableEntity ErrorCode = "UNPROCESSABLE_ENTITY"
)

// http.go
func StatusCodeFromErrorCode(code ErrorCode) int {
    switch code {
    // ... existing cases ...
    case ErrorCodeUnprocessableEntity:
        return http.StatusUnprocessableEntity
    // ...
    }
}
```

### Fix 3: Improve Wrap Context Merging

```go
func Wrap(err error, code ErrorCode, message string) *FluxorError {
    if err == nil {
        return nil
    }
    
    fluxorErr := New(code, message)
    fluxorErr.Cause = err
    
    if existing, ok := err.(*FluxorError); ok {
        // Merge contexts instead of overwriting
        for k, v := range existing.Context {
            fluxorErr.Context[k] = v
        }
        // Only set severity if not already set
        if fluxorErr.Severity == ErrorSeverityMedium {
            fluxorErr.Severity = existing.Severity
        }
    }
    
    return fluxorErr
}
```

## Conclusion

The package has a solid foundation with good design patterns, but needs fixes for error chain traversal, completion of placeholder code, and improved test coverage. The issues are fixable and the package structure is sound.
