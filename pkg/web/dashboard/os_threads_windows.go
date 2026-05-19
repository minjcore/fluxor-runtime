//go:build windows
// +build windows

package dashboard

import "runtime"

// getOSThreadCount returns the actual number of OS threads for the current process
// On Windows, we fallback to GOMAXPROCS as approximation
func getOSThreadCount() int {
	// Windows doesn't have easy access to thread count via ps
	// Fallback to GOMAXPROCS as approximation
	return runtime.GOMAXPROCS(0)
}
