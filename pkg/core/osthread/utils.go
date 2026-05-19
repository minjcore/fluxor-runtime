package osthread

import (
	"runtime"
)

// PinCurrentThread pins the current goroutine to the current OS thread (system thread)
// Returns an Unpin function that should be called when done
//
// Note: This pins a goroutine to an OS thread, not the other way around.
// The goroutine will stay on the same OS thread until Unpin is called.
//
// Usage:
//   unpin := osthread.PinCurrentThread()
//   defer unpin()
//   // CPU-bound work on pinned OS thread
func PinCurrentThread() func() {
	runtime.LockOSThread()
	return runtime.UnlockOSThread
}

// WithPinnedThread executes a function on a goroutine pinned to an OS thread
// The function runs on a goroutine that is locked to a dedicated OS thread
// (system-level thread) using runtime.LockOSThread()
//
// Use cases:
//   - CPU-bound native code (CGO, llama.cpp)
//   - Work requiring CPU affinity
//   - Real-time constraints
//
// Usage:
//   result, err := osthread.WithPinnedThread(func() (interface{}, error) {
//       // CPU-bound work on pinned OS thread
//       return nativeFunction(), nil
//   })
func WithPinnedThread(fn func() (interface{}, error)) (interface{}, error) {
	done := make(chan struct {
		result interface{}
		err    error
	}, 1)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		result, err := fn()
		done <- struct {
			result interface{}
			err    error
		}{result: result, err: err}
	}()

	ret := <-done
	return ret.result, ret.err
}

// NumOSThreads returns the current number of OS threads (system-level threads)
// This is the value of GOMAXPROCS, which limits how many OS threads
// the Go runtime can use to run goroutines.
//
// Note: This is NOT the number of goroutines. Goroutines are lightweight
// and can number in the millions, but they run on a limited number of OS threads.
func NumOSThreads() int {
	return runtime.GOMAXPROCS(0)
}

// SetMaxOSThreads sets the maximum number of OS threads (system-level threads)
// that the Go runtime can use to run goroutines.
// Returns the previous value.
//
// Note: This limits OS threads, not goroutines. Goroutines are scheduled
// onto these OS threads by the Go runtime.
func SetMaxOSThreads(n int) int {
	return runtime.GOMAXPROCS(n)
}

// NumGoroutines returns the current number of goroutines (Go's lightweight threads)
// This is different from OS threads - goroutines are scheduled onto OS threads.
func NumGoroutines() int {
	return runtime.NumGoroutine()
}
