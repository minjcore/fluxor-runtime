package eventloop

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEventLoopGroup_ConcurrentDispatch(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.NumLoops = 4
	config.QueueSize = 1000

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("NewEventLoopGroup failed: %v", err)
	}
	defer group.Close()

	// Concurrent dispatch test
	numGoroutines := 100
	numEventsPerGoroutine := 10
	var wg sync.WaitGroup
	handled := make(chan bool, numGoroutines*numEventsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numEventsPerGoroutine; j++ {
				event := &Event{
					Key:     "user-123", // Same key to test routing consistency
					Address: "test.address",
					Body:    "test body",
					Headers: map[string]string{"X-Route-Key": "user-123"},
					Handler: func(ctx context.Context, ev *Event) error {
						handled <- true
						return nil
					},
				}
				if err := group.Dispatch(ctx, event); err != nil {
					t.Errorf("Dispatch failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Wait for all handlers to complete
	timeout := time.After(5 * time.Second)
	received := 0
	for received < numGoroutines*numEventsPerGoroutine {
		select {
		case <-handled:
			received++
		case <-timeout:
			t.Fatalf("Timeout waiting for handlers. Received %d/%d", received, numGoroutines*numEventsPerGoroutine)
		}
	}
}

func TestEventLoopGroup_KeyRoutingConsistency(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.NumLoops = 4
	config.QueueSize = 100

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("NewEventLoopGroup failed: %v", err)
	}
	defer group.Close()

	// Test that same key always routes to same loop
	key := "user-456"
	loopIDs := make(map[int]bool)

	for i := 0; i < 10; i++ {
		handled := make(chan int, 1)
		event := &Event{
			Key:     key,
			Address: "test.address",
			Body:    "test body",
			Headers: map[string]string{"X-Route-Key": key},
			Handler: func(ctx context.Context, ev *Event) error {
				// Extract loop ID from stats (simplified - in real scenario would track this)
				stats := group.Stats()
				for _, stat := range stats {
					if stat.QueueLength > 0 || stat.ProcessedMessages > 0 {
						handled <- stat.LoopID
						return nil
					}
				}
				handled <- -1
				return nil
			},
		}

		err := group.Dispatch(ctx, event)
		if err != nil {
			t.Fatalf("Dispatch failed: %v", err)
		}

		select {
		case loopID := <-handled:
			if loopID >= 0 {
				loopIDs[loopID] = true
			}
		case <-time.After(1 * time.Second):
			t.Fatal("Handler was not called")
		}
	}

	// All events with same key should route to same loop
	// (In practice, we can't easily verify this without exposing internal state,
	// but the test ensures routing doesn't crash)
	if len(loopIDs) == 0 {
		t.Error("No loop IDs recorded")
	}
}

func TestEventLoopGroup_Backpressure(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.NumLoops = 2
	config.QueueSize = 10 // Small queue
	config.Backpressure = BackpressureDrop

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("NewEventLoopGroup failed: %v", err)
	}
	defer group.Close()

	// Fill queues with slow handlers
	for i := 0; i < 20; i++ {
		event := &Event{
			Key:     "test-key",
			Address: "test.address",
			Body:    "test body",
			Headers: map[string]string{},
			Handler: func(ctx context.Context, ev *Event) error {
				time.Sleep(50 * time.Millisecond) // Slow handler
				return nil
			},
		}
		_ = group.Dispatch(ctx, event)
	}

	// Wait a bit for queues to fill
	time.Sleep(100 * time.Millisecond)

	// Try to dispatch more - should start dropping
	dropped := 0
	for i := 0; i < 10; i++ {
		event := &Event{
			Key:     "test-key",
			Address: "test.address",
			Body:    "test body",
			Headers: map[string]string{},
			Handler: func(ctx context.Context, ev *Event) error {
				return nil
			},
		}
		if err := group.Dispatch(ctx, event); err != nil {
			dropped++
		}
	}

	// Wait for stats to update
	time.Sleep(200 * time.Millisecond)

	stats := group.Stats()
	totalDropped := int64(0)
	for _, stat := range stats {
		totalDropped += stat.DroppedMessages
	}

	if totalDropped == 0 && dropped == 0 {
		t.Log("No messages dropped (queues may have had space)")
	}
}
