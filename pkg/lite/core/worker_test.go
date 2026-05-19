package core_test

import (
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/lite/core"
)

func TestWorkerPool_Submit(t *testing.T) {
	wp := core.NewWorkerPool(1, 10)
	defer wp.Shutdown()

	done := make(chan struct{})
	wp.Submit(func() {
		close(done)
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for task")
	}
}
