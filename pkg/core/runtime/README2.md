# Runtime Package Review (Second Pass)

## Overview
The `core/runtime` package provides essential runtime management functionality including quota management, hooks, health checks, component registry, state management, signal handling, time abstractions, graceful draining, and debug information collection.

## Package Structure
```
core/runtime/
├── clock/      - Time abstraction for testing
├── debug/      - Debug information collection
├── drain/      - Graceful component draining
├── health/     - Health check management
├── hooks/      - Hook registry and execution
├── quota/      - Quota/rate limiting management
├── registry/   - Component registration system
├── signals/    - Signal handling for graceful shutdown
└── state/      - State machine management
```

## Strengths

### 1. **Clean Architecture**
- Well-separated concerns with clear interfaces
- Consistent patterns across subpackages
- Good use of dependency injection via config structs

### 2. **Thread Safety**
- Proper use of `sync.RWMutex` for read-heavy operations
- Atomic operations for counters and flags
- Safe concurrent access patterns (with some exceptions noted below)

### 3. **Comprehensive Features**
- Rich configuration options
- Callback support (sync and async)
- Statistics and metrics collection
- History tracking where applicable

### 4. **Error Handling**
- Consistent error types with codes
- Proper error wrapping (mostly)
- Context-aware error handling

## Critical Issues

### 1. **Hooks: Async Hook Execution Bug** ⚠️ CRITICAL
**Location:** `hooks/hooks.go:318-352`

**Issue:** In `executeParallel`, async hooks have incorrect wait group handling:
```go
wg.Add(1)
go func() {
    defer wg.Done()  // Will always call Done()
    // ...
}()

if info.Async {
    wg.Done() // Subtracts immediately
}
```

**Problem:** 
- If async hook panics before defer, we get a double Done() (from immediate call and never-executed defer)
- The logic is confusing and error-prone
- Race condition: immediate `wg.Done()` might execute before goroutine starts

**Fix:**
```go
for _, info := range hooks {
    info := info
    
    if !info.Async {
        wg.Add(1)
    }
    
    go func() {
        if !info.Async {
            defer wg.Done()
        }
        // ... rest of hook execution
    }()
}
```

### 2. **Quota: Race Condition in Acquire** ⚠️ CRITICAL
**Location:** `quota/quota.go:330-347`

**Issue:** Window-based reset check has a race condition:
```go
quota.mu.Lock()
// Check window
if quota.Window > 0 && now.Sub(quota.LastReset) >= quota.Window {
    quota.Usage = 0  // Direct assignment, not atomic
    // ...
}
currentUsage := atomic.LoadInt64(&quota.Usage)  // Reads after potential reset
```

**Problem:** Between checking the window and resetting, another goroutine might have modified usage atomically, causing inconsistencies.

**Fix:** Ensure atomic operations are used consistently or protect all quota operations with the mutex.

### 4. **State: Wait() Channel Leak** ⚠️ MEDIUM
**Location:** `state/state.go:445-457`

**Issue:** If context is cancelled before state is reached, the channel is never cleaned up:
```go
done := make(chan struct{})
m.waitMu.Lock()
m.waiters[StateStopped] = append(m.waiters[StateStopped], done)
m.waiters[StateError] = append(m.waiters[StateError], done)  // Same channel in two slices!
m.waitMu.Unlock()

select {
case <-done:
    return nil
case <-ctx.Done():
    return ctx.Err()  // Channel left in waiters map!
}
```

**Problem:**
- If context expires, the channel remains in both `waiters[StateStopped]` and `waiters[StateError]`
- Memory leak: channels accumulate over time
- Same channel added to multiple waiter lists could cause double notification

**Fix:**
```go
done := make(chan struct{})
m.waitMu.Lock()
m.waiters[StateStopped] = append(m.waiters[StateStopped], done)
m.waiters[StateError] = append(m.waiters[StateError], done)
m.waitMu.Unlock()

select {
case <-done:
    return nil
case <-ctx.Done():
    // Clean up the channel
    m.waitMu.Lock()
    // Remove from both lists
    m.waiters[StateStopped] = removeChannel(m.waiters[StateStopped], done)
    m.waiters[StateError] = removeChannel(m.waiters[StateError], done)
    m.waitMu.Unlock()
    return ctx.Err()
}
```

### 5. **Signal: Wait() Multiple Stop Calls** ⚠️ LOW-MEDIUM
**Location:** `signals/signal.go:262-293`

**Issue:** `Wait()` calls `Stop()` which might cause issues if called multiple times:
```go
case sig := <-h.sigChan:
    h.Stop()  // Could be called multiple times if Wait() is called concurrently
    return sig, nil

case <-ctx.Done():
    h.Stop()  // Also calls Stop()
    return nil, ctx.Err()
```

**Problem:** While `Stop()` has protection against double-stopping, calling it multiple times unnecessarily is wasteful and could cause timing issues.

**Fix:** Use a flag to ensure Stop() is only called once, or refactor to avoid calling Stop() from Wait().

## Design Concerns

### 1. **Health: Threshold Logic Error** ⚠️ MEDIUM
**Location:** `health/health.go:564-571`

**Issue:** Goroutine threshold check incorrectly sets memory health:
```go
if threshold.MaxGoroutines > 0 && runtimeHealth.NumGoroutines > threshold.MaxGoroutines {
    messages = append(messages, fmt.Sprintf("goroutines (%d) exceeds threshold (%d)",
        runtimeHealth.NumGoroutines, threshold.MaxGoroutines))
    if runtimeHealth.Memory != nil {
        runtimeHealth.Memory.Healthy = false  // WRONG: Why set memory health?
    }
}
```

**Problem:** 
- Goroutine threshold violation should not affect memory health flag
- This is clearly a copy-paste error or misunderstanding of what the flag represents
- Memory health should only be affected by memory-related thresholds

**Fix:**
```go
if threshold.MaxGoroutines > 0 && runtimeHealth.NumGoroutines > threshold.MaxGoroutines {
    messages = append(messages, fmt.Sprintf("goroutines (%d) exceeds threshold (%d)",
        runtimeHealth.NumGoroutines, threshold.MaxGoroutines))
    // Don't modify memory health - goroutines are not memory!
}
```

### 2. **Registry: RegisterMany Rollback Issue** ⚠️ MEDIUM-HIGH
**Location:** `registry/registry.go:556-570`

**Issue:** Rollback logic in `RegisterMany` is incorrect:
```go
func (r *registryManager) RegisterMany(components map[string]Component, opts ...Option) error {
    for name, component := range components {
        if err := r.Register(name, component, opts...); err != nil {
            // Rollback: unregister already registered components
            for registeredName := range components {  // WRONG: iterates all components
                if registeredName != name {
                    r.Unregister(registeredName)  // Unregisters ALL, including ones registered before this call!
                }
            }
            return fmt.Errorf("failed to register %s: %w", name, err)
        }
    }
    return nil
}
```

**Problem:** 
- Rolls back ALL components in the map, even ones that were already registered before this call
- Should only rollback components registered in THIS `RegisterMany` call
- Example scenario:
  1. Component "A" already exists in registry
  2. `RegisterMany({"A": A, "B": B, "C": C})` called
  3. Register "A" succeeds (overwrite or already exists)
  4. Register "B" succeeds
  5. Register "C" fails
  6. Rollback unregisters "A", "B" - but "A" might have existed before!

**Fix:**
```go
func (r *registryManager) RegisterMany(components map[string]Component, opts ...Option) error {
    registered := make([]string, 0, len(components))
    
    for name, component := range components {
        wasExisting := r.Exists(name)
        
        if err := r.Register(name, component, opts...); err != nil {
            // Rollback only components we registered in this call
            for _, regName := range registered {
                r.Unregister(regName)
            }
            return fmt.Errorf("failed to register %s: %w", name, err)
        }
        
        // Only track if we actually registered it (not if it already existed)
        if !wasExisting || r.config.AllowOverwrite {
            registered = append(registered, name)
        }
    }
    return nil
}
```

### 3. **Debug: Memory Allocation in GoroutineDump** ⚠️ LOW
**Location:** `debug/debug.go:336`

**Issue:** Fixed 1MB buffer allocation:
```go
buf := make([]byte, 1024*1024) // 1MB buffer
```

**Problem:** 
- May be too large for small applications
- May be too small for large applications with many goroutines
- Could use a growing buffer strategy

### 4. **Drain: Context Timeout Calculation** ⚠️ LOW
**Location:** `drain/drain.go:158-164`

**Issue:** Timeout calculation could be improved:
```go
timeout := d.config.DefaultTimeout
if deadline, ok := ctx.Deadline(); ok {
    timeout = time.Until(deadline)
    if timeout <= 0 {
        timeout = d.config.DefaultTimeout
    }
}
```

**Problem:** If context already expired, we should return an error immediately rather than using default timeout.

## Code Quality Issues

### 1. **Inconsistent Error Wrapping**
Some packages use `fmt.Errorf("...: %w", err)` while others use `fmt.Errorf("...: %v", err)`. Should consistently use `%w` to preserve error chains.

### 2. **Missing Input Validation**
Some methods don't validate inputs consistently:
- `registry.Get()` returns `nil, false` for empty name (should error)
- Some methods accept empty strings without validation

### 3. **Documentation**
- Some complex methods lack inline comments explaining logic
- Public API documentation could be more detailed
- Package-level documentation (`doc.go`) is just a stub

### 4. **Test Coverage Gaps**
- Missing tests for edge cases (timeouts, context cancellation)
- Limited concurrent execution tests
- Missing tests for error paths in some packages

## Performance Considerations

### 1. **Memory Allocations**
- `registry.ListComponents()` creates many copies (good for safety, but could be optimized for read-only scenarios)
- History tracking could use ring buffers for fixed-size histories

### 2. **Lock Contention**
- Some operations hold locks longer than necessary
- Consider using atomic operations where appropriate (e.g., stats)

### 3. **Channel Buffering**
- Signal handler uses buffered channel (good)
- Consider channel buffer sizes based on expected load

## Additional Issues Found

### 1. **Quota: Auto-Reset Loop Goroutine Leak** ⚠️ LOW
**Location:** `quota/quota.go:688-702`

**Issue:** If `autoResetLoop` starts but manager is never stopped, goroutine leaks:
- No timeout or context support in auto-reset loop
- Only stops when `stopChan` is closed or `ticker` is stopped
- If `Stop()` is never called, goroutine runs forever

**Fix:** Consider adding context support or ensuring Stop() is always called (e.g., via finalizer).

### 2. **Hooks: executeHook Async Logic Inconsistent** ⚠️ MEDIUM
**Location:** `hooks/hooks.go:388-410`

**Issue:** Async hooks in `executeHook` don't track execution properly:
```go
if info.Async {
    go func() {
        hookErr := info.Hook(execCtx)
        atomic.AddInt64(&info.ExecutionCount, 1)
        info.LastExecutionTime = time.Now()
        info.LastError = hookErr  // Not thread-safe writes!
    }()
    return nil
}
```

**Problem:**
- `LastExecutionTime` and `LastError` are written without synchronization
- If called from multiple goroutines concurrently, race condition occurs
- Should use mutex or atomic operations for these fields

### 3. **Registry: Get() Updates LastAccessTime Outside Lock** ⚠️ LOW
**Location:** `registry/registry.go:324-342`

**Issue:** While `Get()` updates `LastAccessTime` inside the lock, `AccessCount` is updated atomically but `LastAccessTime` uses direct assignment:
```go
r.mu.Lock()
defer r.mu.Unlock()
// ...
info.LastAccessTime = now  // OK - inside lock
atomic.AddInt64(&info.AccessCount, 1)  // Mixed approach
```

**Observation:** Actually OK since it's inside the lock, but inconsistent pattern with atomic operations.

### 4. **Error Wrapping Inconsistency** ⚠️ LOW
**Location:** Multiple files

**Issue:** Some places use `%w` (correct), others use `%v`:
- `hooks/hooks.go:280` uses `%w` ✓
- `hooks/hooks.go:358` uses `%v` ✗
- `registry/registry.go:566` uses `%w` ✓
- Many error constructors use `fmt.Errorf` without wrapping

**Recommendation:** Standardize on `%w` for error wrapping throughout.

## Recommendations

### Critical Priority (Fix Immediately)
1. **Fix async hook wait group handling** - Will cause panic in production
2. **Fix quota Stop() panic risk** - Will crash if called twice concurrently
3. **Fix RegisterMany rollback** - Data loss/broken state on partial failures

### High Priority
1. **Fix quota atomicity inconsistency** - Potential race condition on refactoring
2. **Fix health threshold logic** - Incorrect health status reporting
3. **Fix state Wait() channel leak** - Memory leak over time

### Medium Priority
1. **Fix hooks async field writes** - Race condition on concurrent access
2. **Improve error wrapping consistency** - Use `%w` throughout
3. **Add comprehensive input validation** - Fail fast on bad input
4. **Improve documentation** - Add package-level docs and method comments
5. **Add integration tests** - Test component interactions

### Low Priority
1. **Optimize memory allocations** - Profile and optimize hot paths
2. **Reduce lock contention** - Review locking strategies
3. **Add benchmarks** - Track performance over time
4. **Consider observability** - Add metrics hooks/exporters
5. **Fix quota auto-reset goroutine leak** - Add timeout/context support

## Positive Highlights

1. **Excellent test coverage in quota package** - Comprehensive edge case testing
2. **Good use of interfaces** - Makes components testable and swappable
3. **Context support throughout** - Proper timeout and cancellation handling
4. **Clean separation of concerns** - Each package has a single responsibility
5. **Configurable behavior** - Extensive configuration options without complexity

## Test Coverage Analysis

### Well-Tested Packages
- **quota**: Excellent test coverage (1027 lines of tests)
  - Comprehensive edge cases
  - Concurrent execution tests
  - Window reset tests
  - Unlimited quota tests
  
### Needs More Tests
- **hooks**: Good coverage but missing:
  - Async hook panic scenarios
  - Concurrent parallel execution edge cases
  - Wait group handling verification
  
- **state**: Missing tests for:
  - Wait() timeout scenarios
  - Channel cleanup on context cancellation
  - Concurrent state transitions
  
- **registry**: Missing tests for:
  - RegisterMany rollback scenarios
  - Concurrent registration edge cases
  
- **signals**: Missing tests for:
  - Multiple Stop() calls
  - Concurrent Wait() calls
  
- **quota**: Missing tests for:
  - Multiple Stop() calls
  - Auto-reset loop cleanup

## Code Quality Metrics

### Positive Aspects
- Consistent error handling patterns
- Good use of interfaces for testability
- Comprehensive configuration options
- Clear separation of concerns

### Areas for Improvement
- Some inconsistent patterns (atomic vs mutex)
- Missing input validation in some methods
- Package-level documentation is minimal
- Some race conditions in edge cases

## Conclusion

The runtime package is well-structured and follows good Go practices overall. However, there are **several critical bugs** that need immediate attention:

1. **Hooks async execution** - Will panic in production
2. **Quota Stop()** - Can panic if called concurrently
3. **RegisterMany rollback** - Data corruption risk

Additionally, there are **medium-priority issues** that should be addressed:
- Quota atomicity inconsistency
- Health threshold logic error
- State Wait() memory leak

Once the critical issues are fixed, and medium-priority issues are addressed, the package will be production-ready with proper test coverage.

**Overall Grade: B** (Would be A- after fixing critical issues)

**Recommendation:** Do not deploy to production until at least the critical issues are fixed. The async hook bug will cause panics under normal concurrent usage.
