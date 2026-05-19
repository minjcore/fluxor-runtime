package tcp

import "testing"

func TestBackpressureController_FailFast_CapacityExceeded(t *testing.T) {
	t.Parallel()

	bc := NewBackpressureController(2, 3600)

	if !bc.TryAcquire() {
		t.Fatalf("expected first acquire to succeed")
	}
	if !bc.TryAcquire() {
		t.Fatalf("expected second acquire to succeed")
	}
	if bc.TryAcquire() {
		t.Fatalf("expected third acquire to fail-fast")
	}

	m := bc.GetMetrics()
	if m.RejectedCount < 1 {
		t.Fatalf("expected rejected count to be >= 1, got %d", m.RejectedCount)
	}

	bc.Release()
	if !bc.TryAcquire() {
		t.Fatalf("expected acquire to succeed after release")
	}
}
