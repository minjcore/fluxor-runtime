package web

import (
	"testing"
	"time"
)

// RunWithTimeout runs f and fails the test if it does not complete within timeout.
// Use for tests that block on channels or I/O so they don't hang the test run.
// For full package runs, use: go test -timeout=60s ./pkg/web/...
func RunWithTimeout(t *testing.T, timeout time.Duration, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	select {
	case <-done:
		return
	case <-time.After(timeout):
		t.Fatalf("test timed out after %v", timeout)
	}
}
