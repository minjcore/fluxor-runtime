package core

import (
	"context"
	"testing"
	"time"
)

func TestConsumer_Completion(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	eb := gocmd.EventBus()
	defer eb.Close()

	consumer := eb.Consumer("test.address")

	// Completion channel should be available
	done := consumer.Completion()
	if done == nil {
		t.Error("Completion() should not return nil")
	}

	// Set handler to start processing
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		return nil
	})

	// Unregister should close the mailbox and signal completion
	time.Sleep(100 * time.Millisecond) // Give time for executor to start

	err := consumer.Unregister()
	if err != nil {
		t.Errorf("Unregister() error = %v", err)
	}

	// Wait for completion (with timeout)
	select {
	case <-done:
		// Success - channel closed
	case <-time.After(2 * time.Second):
		t.Error("Completion channel not closed after unregister")
	}
}

func TestConsumer_MultipleConsumers(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	eb := gocmd.EventBus()
	defer eb.Close()

	// Create multiple consumers for same address
	received1 := make(chan bool, 1)
	received2 := make(chan bool, 1)

	consumer1 := eb.Consumer("test.address")
	consumer1.Handler(func(ctx FluxorContext, msg Message) error {
		received1 <- true
		return nil
	})

	consumer2 := eb.Consumer("test.address")
	consumer2.Handler(func(ctx FluxorContext, msg Message) error {
		received2 <- true
		return nil
	})

	// Publish should reach both consumers
	err := eb.Publish("test.address", "test message")
	if err != nil {
		t.Errorf("Publish() error = %v", err)
	}

	// Wait for both consumers to receive
	select {
	case <-received1:
	case <-time.After(1 * time.Second):
		t.Error("Consumer1 did not receive message")
	}

	select {
	case <-received2:
	case <-time.After(1 * time.Second):
		t.Error("Consumer2 did not receive message")
	}
}

func TestConsumer_RequestIDPropagation(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	eb := gocmd.EventBus()
	defer eb.Close()

	// Add request ID to context
	requestID := "test-request-123"
	ctxWithID := WithRequestID(ctx, requestID)

	// Create event bus with context that has request ID
	ebWithID := NewEventBus(ctxWithID, gocmd)
	defer ebWithID.Close()

	// Use channel for safe communication between goroutines
	receivedCh := make(chan string, 1)
	consumer := ebWithID.Consumer("test.address")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		headers := msg.Headers()
		if id, ok := headers["X-Request-ID"]; ok {
			receivedCh <- id
		} else {
			receivedCh <- ""
		}
		return nil
	})

	// Send message
	err := ebWithID.Send("test.address", "test")
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Wait for message processing with timeout
	select {
	case receivedRequestID := <-receivedCh:
		if receivedRequestID != requestID {
			t.Errorf("Request ID propagation: got %v, want %v", receivedRequestID, requestID)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for message with request ID")
	}
}
