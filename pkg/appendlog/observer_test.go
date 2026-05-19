package appendlog

import (
	"sync/atomic"
	"testing"
	"time"
)

type testObserver struct {
	enqueued   int64
	persisted  int64
	rejected   int64
	rotated    int64
	recovered  int64
	lastReason atomic.Value
}

func (o *testObserver) OnRecover(RecoverInfo) { atomic.AddInt64(&o.recovered, 1) }
func (o *testObserver) OnAppendEnqueued(AppendInfo) {
	atomic.AddInt64(&o.enqueued, 1)
}
func (o *testObserver) OnAppendPersisted(PersistInfo) {
	atomic.AddInt64(&o.persisted, 1)
}
func (o *testObserver) OnAppendRejected(RejectInfo) { atomic.AddInt64(&o.rejected, 1) }
func (o *testObserver) OnRotate(info RotateInfo) {
	atomic.AddInt64(&o.rotated, 1)
	o.lastReason.Store(info.Reason)
}

func TestFSStore_Observer_SeesAppendAndPersist(t *testing.T) {
	dir := t.TempDir()
	obs := &testObserver{}

	s, err := NewFSStore(FSStoreConfig{
		Dir:              dir,
		MaxSegmentBytes:  1024,
		MaxBufferedBytes: 1024,
		Durability:       DurabilityFsync,
		Observer:         obs,
	})
	if err != nil {
		t.Fatalf("NewFSStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if _, err := s.Append([]byte("x")); err != nil {
		t.Fatalf("append: %v", err)
	}

	// Persist happens asynchronously; wait briefly.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&obs.enqueued) >= 1 && atomic.LoadInt64(&obs.persisted) >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt64(&obs.enqueued) < 1 {
		t.Fatalf("expected observer enqueued>=1")
	}
	if atomic.LoadInt64(&obs.persisted) < 1 {
		t.Fatalf("expected observer persisted>=1")
	}
}
