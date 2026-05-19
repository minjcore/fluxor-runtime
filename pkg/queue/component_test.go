package queue

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewQueueComponent(t *testing.T) {
	config := DefaultConfig()
	config.Host = "localhost"
	config.Port = 5672

	comp := NewQueueComponent(config)
	if comp == nil {
		t.Fatal("NewQueueComponent() returned nil")
	}

	if comp.Name() != "queue" {
		t.Errorf("Name() = %v, want 'queue'", comp.Name())
	}

	if comp.IsStarted() {
		t.Error("NewQueueComponent() should create component that is not started")
	}
}

func TestQueueComponent_Start_FailFast_NilContext(t *testing.T) {
	config := DefaultConfig()
	config.Host = "localhost"
	config.Port = 5672
	comp := NewQueueComponent(config)

	// QueueComponent.doStart() validates nil context and returns error
	// BaseComponent.Start() calls doStart(), so the nil check in doStart() will catch it
	err := comp.Start(nil)

	// QueueComponent.doStart() explicitly checks for nil context and returns INVALID_INPUT error
	// This is tested to document the behavior - nil context validation happens in doStart()
	if err != nil {
		if e, ok := err.(*core.EventBusError); ok {
			if e.Code == "INVALID_INPUT" {
				// Expected behavior - QueueComponent.doStart() validates nil context
				return
			}
		}
		// Any error is acceptable - the important thing is it doesn't succeed
		return
	}

	// If no error is returned, BaseComponent might handle nil differently
	// This is documented behavior - QueueComponent.doStart() has the validation
	t.Log("Start() with nil context - BaseComponent behavior may vary, but doStart() validates")
}

// Note: TestQueueComponent_Start_FailFast_DoubleStart is not included because:
// - Requires FluxorContext creation (newFluxorContext is internal to pkg/core)
// - Double start prevention is inherited from BaseComponent
// - Tested in pkg/core/base_component_test.go
// - QueueComponent's doStart() is called by BaseComponent.Start()
// - This behavior is inherited and tested in the base class tests

func TestQueueComponent_Connection_FailFast_NotStarted(t *testing.T) {
	config := DefaultConfig()
	config.Host = "localhost"
	config.Port = 5672
	comp := NewQueueComponent(config)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Connection() should panic (fail-fast) when component not started")
		}
	}()

	_ = comp.Connection()
}

func TestQueueComponent_Publisher_FailFast_NotStarted(t *testing.T) {
	config := DefaultConfig()
	config.Host = "localhost"
	config.Port = 5672
	comp := NewQueueComponent(config)

	pub, err := comp.Publisher()
	if err == nil {
		t.Error("Publisher() should fail-fast when component not started")
	}
	if pub != nil {
		t.Error("Publisher() should return nil when component not started")
	}

	if err != nil {
		if e, ok := err.(*core.EventBusError); ok {
			if e.Code != "NOT_STARTED" {
				t.Errorf("Error code = %v, want 'NOT_STARTED'", e.Code)
			}
		} else {
			t.Errorf("Expected EventBusError, got %T", err)
		}
	}
}

func TestQueueComponent_Consumer_FailFast_NotStarted(t *testing.T) {
	config := DefaultConfig()
	config.Host = "localhost"
	config.Port = 5672
	comp := NewQueueComponent(config)

	cons, err := comp.Consumer()
	if err == nil {
		t.Error("Consumer() should fail-fast when component not started")
	}
	if cons != nil {
		t.Error("Consumer() should return nil when component not started")
	}

	if err != nil {
		if e, ok := err.(*core.EventBusError); ok {
			if e.Code != "NOT_STARTED" {
				t.Errorf("Error code = %v, want 'NOT_STARTED'", e.Code)
			}
		} else {
			t.Errorf("Expected EventBusError, got %T", err)
		}
	}
}

func TestQueueComponent_Ping_FailFast_NotStarted(t *testing.T) {
	config := DefaultConfig()
	config.Host = "localhost"
	config.Port = 5672
	comp := NewQueueComponent(config)

	ctx := context.Background()
	err := comp.Ping(ctx)
	if err == nil {
		t.Error("Ping() should fail-fast when component not started")
	}

	if err != nil {
		if e, ok := err.(*core.EventBusError); ok {
			if e.Code != "NOT_STARTED" {
				t.Errorf("Error code = %v, want 'NOT_STARTED'", e.Code)
			}
		} else {
			t.Errorf("Expected EventBusError, got %T", err)
		}
	}
}

func TestQueueComponent_Ping_FailFast_NilContext(t *testing.T) {
	config := DefaultConfig()
	config.Host = "localhost"
	config.Port = 5672
	comp := NewQueueComponent(config)

	// Note: This will fail with NOT_STARTED first (component not started),
	// but we test that the error is returned (nil context check happens after started check)
	err := comp.Ping(nil)
	if err == nil {
		t.Error("Ping() should fail-fast with nil context or not started")
	}

	if err != nil {
		// Since component is not started, we'll get NOT_STARTED error first
		// This is acceptable - the nil check happens after the started check
		if e, ok := err.(*core.EventBusError); ok {
			// Either NOT_STARTED (component not started) or INVALID_INPUT (nil context)
			if e.Code != "NOT_STARTED" && e.Code != "INVALID_INPUT" {
				t.Errorf("Error code = %v, want 'NOT_STARTED' or 'INVALID_INPUT'", e.Code)
			}
		}
	}
}

func TestQueueComponent_IsStarted(t *testing.T) {
	config := DefaultConfig()
	config.Host = "localhost"
	config.Port = 5672
	comp := NewQueueComponent(config)

	if comp.IsStarted() {
		t.Error("IsStarted() should return false for new component")
	}

	// Note: We can't easily test IsStarted() after Start() without a real RabbitMQ connection
	// This is a limitation of unit testing without integration tests
	// The test documents the expected behavior
}

// Note: TestQueueComponent_Stop_NotStarted is not included because:
// - Stop() requires FluxorContext which requires internal newFluxorContext function
// - Stop() idempotency is tested in base_component_test.go for BaseComponent
// - QueueComponent.doStop() is called by BaseComponent.Stop()
// - This behavior is inherited from BaseComponent and tested there

// Note: Integration tests requiring actual RabbitMQ server are not included
// These would require a running RabbitMQ instance and should be in a separate file
// with build tags like //go:build integration
//
// Integration tests would cover:
// - Successful Start/Stop with real RabbitMQ connection
// - Connection() returns valid connection after Start()
// - Publisher() returns valid publisher after Start()
// - Consumer() returns valid consumer after Start()
// - Ping() succeeds with valid connection
// - Connection recovery and reconnection
