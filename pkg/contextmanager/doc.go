// Package contextmanager provides scoped context.Context lifecycle management
// and type-safe context keys for Go applications.
//
// It complements the standard context package with:
//   - Scoped context: WithScope runs a function with a context that is cancelled when the scope exits
//   - Fail-fast helpers: WithCancel, WithTimeout, WithDeadline panic on nil parent (programmer error)
//   - Type-safe keys: Key[T] and WithValue/Value avoid key collisions and unsafe type assertions
//   - Run: run a function with context and propagate cancellation
//
// Use with structural concurrency: create bounded scopes, cancel on exit, and avoid leaking
// goroutines or resources. See README.md for examples.
package contextmanager
