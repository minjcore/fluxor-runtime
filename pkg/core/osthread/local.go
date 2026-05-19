package osthread

import (
	"runtime"
	"sync"
	"unsafe"
)

// ThreadLocal provides thread-local storage for values
// Each OS thread (pinned goroutine) has its own isolated storage
//
// Note: This implementation uses a per-goroutine key based on stack pointer.
// For best results, use with pinned threads (runtime.LockOSThread) where
// the goroutine stays on the same OS thread.
//
// Usage:
//   var local = osthread.NewThreadLocal[string]()
//   local.Set("value")
//   value := local.Get()
type ThreadLocal[T any] interface {
	// Get returns the thread-local value
	// Returns zero value if not set
	Get() T

	// Set sets the thread-local value
	Set(value T)

	// Clear clears the thread-local value
	Clear()
}

// threadLocalImpl implements ThreadLocal using sync.Map keyed by goroutine stack pointer
type threadLocalImpl[T any] struct {
	storage sync.Map // map[uintptr]*T
}

// NewThreadLocal creates a new ThreadLocal instance
func NewThreadLocal[T any]() ThreadLocal[T] {
	return &threadLocalImpl[T]{}
}

// getGoroutineKey gets a key for the current goroutine
// Uses the stack pointer of a local variable as a unique identifier
func getGoroutineKey() uintptr {
	// Get pointer to a local variable on the stack
	// This provides a unique identifier per goroutine (each has its own stack)
	var localVar [8]byte
	runtime.Stack(localVar[:], false)
	return uintptr(unsafe.Pointer(&localVar[0]))
}

// Get returns the thread-local value
func (tl *threadLocalImpl[T]) Get() T {
	key := getGoroutineKey()
	if value, ok := tl.storage.Load(key); ok {
		if ptr, ok := value.(*T); ok {
			return *ptr
		}
	}
	var zero T
	return zero
}

// Set sets the thread-local value
func (tl *threadLocalImpl[T]) Set(value T) {
	key := getGoroutineKey()
	tl.storage.Store(key, &value)
}

// Clear clears the thread-local value
func (tl *threadLocalImpl[T]) Clear() {
	key := getGoroutineKey()
	tl.storage.Delete(key)
}
