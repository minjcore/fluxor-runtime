package eventloop

import (
	"context"
	"testing"
	"time"
)

func TestDispatcher_Dispatch(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.NumLoops = 4
	config.QueueSize = 100

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("NewEventLoopGroup failed: %v", err)
	}
	defer group.Close()

	dispatcher := group.Dispatcher()

	// Test dispatch with key - should route to same loop
	key := "user-123"
	handled1 := make(chan bool, 1)
	handled2 := make(chan bool, 1)

	event1 := &Event{
		Key:     key,
		Address: "test.address",
		Body:    "test body 1",
		Headers: map[string]string{"X-Route-Key": key},
		Handler: func(ctx context.Context, ev *Event) error {
			handled1 <- true
			return nil
		},
	}

	event2 := &Event{
		Key:     key,
		Address: "test.address",
		Body:    "test body 2",
		Headers: map[string]string{"X-Route-Key": key},
		Handler: func(ctx context.Context, ev *Event) error {
			handled2 <- true
			return nil
		},
	}

	// Both events with same key should route to same loop
	err = dispatcher.Dispatch(ctx, event1)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	err = dispatcher.Dispatch(ctx, event2)
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	select {
	case <-handled1:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Handler 1 was not called")
	}

	select {
	case <-handled2:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Handler 2 was not called")
	}
}

func TestDispatcher_DispatchByKey(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.NumLoops = 4
	config.QueueSize = 100

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("NewEventLoopGroup failed: %v", err)
	}
	defer group.Close()

	dispatcher := group.Dispatcher()

	// Test dispatch by key
	key := "user-456"
	handled := make(chan bool, 1)

	event := &Event{
		Key:     key,
		Address: "test.address",
		Body:    "test body",
		Headers: map[string]string{},
		Handler: func(ctx context.Context, ev *Event) error {
			handled <- true
			return nil
		},
	}

	err = dispatcher.DispatchByKey(key, event)
	if err != nil {
		t.Fatalf("DispatchByKey failed: %v", err)
	}

	select {
	case <-handled:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Handler was not called")
	}
}

func TestDispatcher_Stats(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.NumLoops = 4

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("NewEventLoopGroup failed: %v", err)
	}
	defer group.Close()

	dispatcher := group.Dispatcher()

	stats := dispatcher.Stats()
	if len(stats) != 4 {
		t.Errorf("Expected 4 loops, got %d", len(stats))
	}

	for i, stat := range stats {
		if stat.LoopID != i {
			t.Errorf("Expected LoopID %d, got %d", i, stat.LoopID)
		}
	}
}
