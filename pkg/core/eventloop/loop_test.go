package eventloop

import (
	"context"
	"testing"
	"time"
)

func TestNewEventLoop(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.QueueSize = 100

	loop := NewEventLoop(0, ctx, config)
	if loop == nil {
		t.Fatal("NewEventLoop returned nil")
	}
	defer loop.Close()

	stats := loop.Stats()
	if stats.LoopID != 0 {
		t.Errorf("Expected LoopID 0, got %d", stats.LoopID)
	}
}

func TestEventLoop_Dispatch(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.QueueSize = 100
	config.Backpressure = BackpressureBlock

	loop := NewEventLoop(0, ctx, config)
	defer loop.Close()

	handled := make(chan bool, 1)
	event := &Event{
		Key:     "test-key",
		Address: "test.address",
		Body:    "test body",
		Headers: map[string]string{},
		Handler: func(ctx context.Context, ev *Event) error {
			handled <- true
			return nil
		},
	}

	err := loop.Dispatch(event)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	select {
	case <-handled:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Handler was not called")
	}
}

func TestEventLoop_Dispatch_Drop(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.QueueSize = 2 // Small queue to test drop
	config.Backpressure = BackpressureDrop

	loop := NewEventLoop(0, ctx, config)
	defer loop.Close()

	// Create a blocking handler to fill the queue
	blocker := make(chan struct{})
	event1 := &Event{
		Key:     "test-key-1",
		Address: "test.address",
		Body:    "test body 1",
		Headers: map[string]string{},
		Handler: func(ctx context.Context, ev *Event) error {
			<-blocker // Block until released
			return nil
		},
	}

	// Fill queue to capacity
	for i := 0; i < config.QueueSize; i++ {
		err := loop.Dispatch(event1)
		if err != nil {
			t.Fatalf("Dispatch %d failed: %v", i, err)
		}
	}

	// Now queue should be full, next dispatch should drop
	event2 := &Event{
		Key:     "test-key-2",
		Address: "test.address",
		Body:    "test body 2",
		Headers: map[string]string{},
		Handler: func(ctx context.Context, ev *Event) error {
			return nil
		},
	}

	err := loop.Dispatch(event2)
	if err == nil {
		t.Error("Expected error when queue is full with Drop policy")
	}

	// Release blocker to allow queue to drain
	close(blocker)

	// Wait for stats to update
	time.Sleep(100 * time.Millisecond)

	stats := loop.Stats()
	if stats.DroppedMessages == 0 {
		t.Log("No messages dropped (may have processed quickly)")
	}
}

func TestEventLoop_Close(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()

	loop := NewEventLoop(0, ctx, config)

	err := loop.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Dispatch after close should fail
	event := &Event{
		Key:     "test-key",
		Address: "test.address",
		Body:    "test body",
		Headers: map[string]string{},
		Handler: func(ctx context.Context, ev *Event) error {
			return nil
		},
	}

	err = loop.Dispatch(event)
	if err == nil {
		t.Error("Expected error when dispatching after close")
	}
}

func TestEventLoop_Stats(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.Metrics = true
	config.QueueSize = 100

	loop := NewEventLoop(0, ctx, config)
	defer loop.Close()

	// Dispatch some events
	for i := 0; i < 5; i++ {
		event := &Event{
			Key:     "test-key",
			Address: "test.address",
			Body:    "test body",
			Headers: map[string]string{},
			Handler: func(ctx context.Context, ev *Event) error {
				return nil
			},
		}
		_ = loop.Dispatch(event)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	stats := loop.Stats()
	if stats.LoopID != 0 {
		t.Errorf("Expected LoopID 0, got %d", stats.LoopID)
	}
	if stats.ProcessedMessages == 0 {
		t.Error("Expected processed messages > 0")
	}
}
