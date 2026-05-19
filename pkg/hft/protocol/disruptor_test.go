package protocol

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

// TestPublishConsumeOrdering verifies that a consumer never reads a slot before
// the producer has written data to it (the cursor-before-write race).
func TestPublishConsumeOrdering(t *testing.T) {
	const count = 100_000
	rb := NewRingBuffer(1024, YieldWait)

	values := make([]int64, count)
	for i := range values {
		values[i] = int64(i + 1)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Producer
	go func() {
		defer wg.Done()
		for i := 0; i < count; {
			seq := rb.Next()
			if seq < 0 {
				continue // buffer full, retry
			}
			rb.Publish(seq, unsafe.Pointer(&values[i]))
			i++
		}
	}()

	// Consumer — verifies it never sees a zero/nil pointer for a claimed slot
	var gotNil int64
	go func() {
		defer wg.Done()
		ctx := context.Background()
		for i := int64(0); i < count; i++ {
			seq, err := rb.WaitFor(i, ctx)
			if err != nil {
				return
			}
			ptr := rb.Get(seq)
			if ptr == nil {
				atomic.AddInt64(&gotNil, 1)
			}
			rb.Consume(seq)
		}
	}()

	wg.Wait()
	if gotNil > 0 {
		t.Errorf("consumer read nil pointer %d times — publish/wait race still present", gotNil)
	}
}

// TestMultiProducerOrdering verifies multi-producer correctness: all published
// values must eventually be read exactly once by the consumer.
func TestMultiProducerOrdering(t *testing.T) {
	const (
		producers = 4
		perProd   = 10_000
		total     = producers * perProd
	)
	rb := NewRingBuffer(4096, YieldWait)

	var counter int64
	seen := make([]int32, total+1) // index 1..total

	var prodWg sync.WaitGroup
	prodWg.Add(producers)
	for p := 0; p < producers; p++ {
		go func(id int) {
			defer prodWg.Done()
			v := int64(id*perProd + 1)
			end := v + perProd
			for v < end {
				seq := rb.Next()
				if seq < 0 {
					continue
				}
				vCopy := v // escape to heap so pointer is stable
				rb.Publish(seq, unsafe.Pointer(&vCopy))
				v++
			}
		}(p)
	}

	// Consumer runs until it has consumed `total` items
	ctx := context.Background()
	var consWg sync.WaitGroup
	consWg.Add(1)
	go func() {
		defer consWg.Done()
		for atomic.LoadInt64(&counter) < int64(total) {
			seq := atomic.LoadInt64(&counter)
			avail, err := rb.WaitFor(seq, ctx)
			if err != nil {
				return
			}
			for s := seq; s <= avail; s++ {
				ptr := rb.Get(s)
				if ptr == nil {
					t.Errorf("nil pointer at seq %d", s)
					continue
				}
				val := *(*int64)(ptr)
				if val >= 1 && int(val) <= total {
					seen[val]++
				}
				rb.Consume(s)
			}
			atomic.StoreInt64(&counter, avail+1)
		}
	}()

	prodWg.Wait()
	consWg.Wait()
}

// TestNextReturnsMinus1WhenFull verifies the non-blocking full check.
func TestNextReturnsMinus1WhenFull(t *testing.T) {
	rb := NewRingBuffer(4, BusySpinWait)

	var v int64 = 99
	claimed := 0
	for i := 0; i < 8; i++ {
		seq := rb.Next()
		if seq < 0 {
			break
		}
		rb.Publish(seq, unsafe.Pointer(&v))
		claimed++
	}
	if claimed != 4 {
		t.Errorf("expected 4 claims on size-4 buffer, got %d", claimed)
	}
}

// TestAvailableCapacity sanity-checks capacity math.
func TestAvailableCapacity(t *testing.T) {
	rb := NewRingBuffer(8, BusySpinWait)
	if cap := rb.AvailableCapacity(); cap != 8 {
		t.Errorf("fresh buffer capacity = %d, want 8", cap)
	}
	var v int64 = 1
	seq := rb.Next()
	rb.Publish(seq, unsafe.Pointer(&v))
	if cap := rb.AvailableCapacity(); cap != 7 {
		t.Errorf("after 1 publish capacity = %d, want 7", cap)
	}
	rb.Consume(seq)
	if cap := rb.AvailableCapacity(); cap != 8 {
		t.Errorf("after consume capacity = %d, want 8", cap)
	}
}

// BenchmarkRingBufferSPSC measures single-producer single-consumer throughput.
func BenchmarkRingBufferSPSC(b *testing.B) {
	rb := NewRingBuffer(1024, BusySpinWait)
	var v int64 = 42
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	go func() {
		for i := int64(0); i < int64(b.N); i++ {
			rb.WaitFor(i, ctx) //nolint
			rb.Consume(i)
		}
	}()

	for i := 0; i < b.N; {
		seq := rb.Next()
		if seq < 0 {
			continue
		}
		rb.Publish(seq, unsafe.Pointer(&v))
		i++
	}
}
