# contextmanager

Scoped `context.Context` lifecycle management and type-safe context keys for Go.

## Features

- **WithScope** – Run a function with a context that is cancelled when the scope exits (like a context manager / `defer cancel()`).
- **WithCancel / WithTimeout / WithDeadline** – Fail-fast wrappers (panic on nil parent or invalid args).
- **Run** – Run a function with a context and propagate its error.
- **Key[T]** – Type-safe context keys to avoid collisions and unsafe type assertions.

## Usage

### Scoped context

```go
err := contextmanager.WithScope(ctx, func(scoped context.Context) error {
    return doWork(scoped) // scoped is cancelled when this returns
})
```

### Type-safe keys

```go
var requestIDKey = contextmanager.NewKey[string]("request_id")

ctx = contextmanager.WithValue(ctx, requestIDKey, "abc-123")
id, ok := contextmanager.Value(ctx, requestIDKey)
id = contextmanager.MustValue(ctx, requestIDKey) // zero value if not set
```

### Timeout

```go
ctx, cancel := contextmanager.WithTimeout(parent, 5*time.Second)
defer cancel()
// use ctx...
```

## Knowing the error from a context

When a context is done (cancelled or deadline exceeded), use **`ctx.Err()`** to get the reason:

| `ctx.Err()`              | Meaning |
|-------------------------|--------|
| `nil`                   | Context is not done yet. |
| `context.Canceled`      | Context was cancelled (e.g. `cancel()` called, or parent cancelled). |
| `context.DeadlineExceeded` | Context deadline passed (e.g. `WithTimeout` or `WithDeadline`). |

**Check after `Done()`:**

```go
select {
case <-ctx.Done():
    return ctx.Err()  // context.Canceled or context.DeadlineExceeded
default:
    // still running
}
```

**In `WithScope`:** the function can return `scoped.Err()` when you exit due to cancellation so the caller sees the reason:

```go
_ = contextmanager.WithScope(appCtx, func(scoped context.Context) error {
    for {
        select {
        case <-scoped.Done():
            return scoped.Err()  // propagate Canceled or DeadlineExceeded
        default:
            // do work
        }
    }
})
```

**Which context failed?** Use **named variables** (`appCtx`, `scoped`, `reqCtx`) and check `ctx.Err()` on the context you're waiting on. Use **`errors.Is(err, context.Canceled)`** or **`errors.Is(err, context.DeadlineExceeded)`** to tell the kind of error.

## Managing many contexts

When an app has several contexts (root, request, scoped, handler), keep them clear with **naming** and **hierarchy**:

| Name / variable   | Meaning | Where it comes from | Use for |
|-------------------|--------|----------------------|---------|
| **rootCtx** / **appCtx** | Process lifetime | `entrypoint` → `GoCMD.Context()` | Shutdown signal; pass to long-lived goroutines. |
| **fxCtx** | Fluxor runtime (EventBus, GoCMD, Config) | `Verticle.Start(ctx)`, handler `ctx` | Use `fxCtx.Context()` when you need Go's `context.Context`. |
| **scoped** | Short-lived scope | `contextmanager.WithScope(parent, fn)` | Work that must end when scope exits; cancel when fn returns. |
| **reqCtx** / **goCtx** | Per-request or per-handler | `fluxorCtx.Context()` in a handler | Timeouts, cancellation for a single request or sub-call. |

Rules of thumb:

1. **One parent per scope** – Derive child contexts from the right parent (e.g. request timeout from `ctx.Context()`, not from `context.Background()`).
2. **Name by role** – Use `appCtx`, `scoped`, `reqCtx` (or `goCtx`) so it's obvious which context you have.
3. **Scoped work** – Use `WithScope(appCtx, fn)` for background loops so they exit when `appCtx` is cancelled.
4. **Handlers** – In Fluxor handlers you get `core.FluxorContext` (fxCtx); call `fxCtx.Context()` for Go context (e.g. `WithTimeout(fxCtx.Context(), d)` for sub-operations).

## Fail-fast

All functions panic on invalid arguments (nil parent, non-positive timeout) per project rules. Use for programmer errors; return errors for expected runtime failures.
