package eventloop

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestNewEventLoopGroup(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("NewEventLoopGroup failed: %v", err)
	}
	defer group.Close()

	stats := group.Stats()
	expectedLoops := runtime.GOMAXPROCS(0)
	if len(stats) != expectedLoops {
		t.Errorf("Expected %d loops, got %d", expectedLoops, len(stats))
	}
}

func TestEventLoopGroup_Dispatch(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.QueueSize = 100

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("NewEventLoopGroup failed: %v", err)
	}
	defer group.Close()

	// Test dispatch with key
	handled := make(chan bool, 1)
	event := &Event{
		Key:     "test-key",
		Address: "test.address",
		Body:    "test body",
		Headers: map[string]string{"X-Route-Key": "test-key"},
		Handler: func(ctx context.Context, ev *Event) error {
			handled <- true
			return nil
		},
	}

	err = group.Dispatch(ctx, event)
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

func TestEventLoopGroup_DispatchByKey(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.QueueSize = 100

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("NewEventLoopGroup failed: %v", err)
	}
	defer group.Close()

	// Test dispatch by key
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

	err = group.DispatchByKey("test-key", event)
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

func TestEventLoopGroup_Close(t *testing.T) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("NewEventLoopGroup failed: %v", err)
	}

	err = group.Close()
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

	err = group.Dispatch(ctx, event)
	if err == nil {
		t.Error("Expected error when dispatching after close")
	}
}
