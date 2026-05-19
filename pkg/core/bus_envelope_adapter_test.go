package core

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core/eventloop"
)

func TestEnvelopeToEventLoopData(t *testing.T) {
	topic := "test.topic"
	key := "test-key"
	data := []byte("test data")
	meta := map[string]string{"key1": "value1", "X-Route-Key": "route-key"}

	env := NewEnvelope(topic, key, data, meta)
	envData := EnvelopeToEventLoopData(env)

	if envData == nil {
		t.Fatal("EnvelopeToEventLoopData() returned nil")
	}

	if envData.Topic != topic {
		t.Errorf("Topic = %s, want %s", envData.Topic, topic)
	}

	// GetRoutingKey should use priority logic
	expectedKey := env.GetRoutingKey()
	if envData.Key != expectedKey {
		t.Errorf("Key = %s, want %s (from GetRoutingKey)", envData.Key, expectedKey)
	}

	if string(envData.Data) != string(data) {
		t.Errorf("Data = %v, want %v", envData.Data, data)
	}

	if len(envData.Meta) != len(meta) {
		t.Errorf("Meta length = %d, want %d", len(envData.Meta), len(meta))
	}

	if envData.Meta["key1"] != meta["key1"] {
		t.Errorf("Meta[key1] = %s, want %s", envData.Meta["key1"], meta["key1"])
	}
}

func TestEnvelopeToEventLoopData_NilEnvelope(t *testing.T) {
	envData := EnvelopeToEventLoopData(nil)

	if envData != nil {
		t.Errorf("EnvelopeToEventLoopData(nil) = %v, want nil", envData)
	}
}

func TestEnvelopeToEventLoopData_EmptyEnvelope(t *testing.T) {
	env := NewEnvelope("", "", nil, nil)
	envData := EnvelopeToEventLoopData(env)

	if envData == nil {
		t.Fatal("EnvelopeToEventLoopData() returned nil")
	}

	if envData.Topic != "" {
		t.Errorf("Topic = %s, want empty string", envData.Topic)
	}

	if envData.Key != "" {
		t.Errorf("Key = %s, want empty string", envData.Key)
	}

	if envData.Data != nil {
		t.Errorf("Data = %v, want nil", envData.Data)
	}

	if envData.Meta == nil {
		t.Error("Meta should be initialized as empty map, not nil")
	}
}

func TestEnvelopeToEventLoopData_RoutingKeyPriority(t *testing.T) {
	// Test that GetRoutingKey priority is respected
	meta := map[string]string{
		"X-Route-Key":  "route-key",
		"X-User-ID":    "user-id",
		"X-Session-ID": "session-id",
	}
	env := NewEnvelope("test.topic", "", []byte("data"), meta)
	envData := EnvelopeToEventLoopData(env)

	expectedKey := env.GetRoutingKey() // Should be "route-key"
	if envData.Key != expectedKey {
		t.Errorf("Key = %s, want %s (from GetRoutingKey priority)", envData.Key, expectedKey)
	}
}

func TestNewEventLoopGroupAdapter(t *testing.T) {
	ctx := context.Background()
	config := eventloop.DefaultEventLoopConfig()
	group, err := eventloop.NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create EventLoopGroup: %v", err)
	}
	defer group.Close()

	adapter := NewEventLoopGroupAdapter(group)

	if adapter == nil {
		t.Fatal("NewEventLoopGroupAdapter() returned nil")
	}

	if adapter.group != group {
		t.Error("Adapter group does not match provided group")
	}
}

func TestEventLoopGroupAdapter_DispatchEnvelope(t *testing.T) {
	ctx := context.Background()
	config := eventloop.DefaultEventLoopConfig()
	group, err := eventloop.NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create EventLoopGroup: %v", err)
	}
	defer group.Close()

	adapter := NewEventLoopGroupAdapter(group)

	topic := "test.topic"
	key := "test-key"
	data := []byte("test data")
	meta := map[string]string{"key1": "value1"}

	env := NewEnvelope(topic, key, data, meta)

	// Dispatch should succeed (even though there are no handlers)
	err = adapter.DispatchEnvelope(ctx, env)
	if err != nil {
		t.Errorf("DispatchEnvelope() error = %v, want nil", err)
	}
}

func TestEventLoopGroupAdapter_DispatchEnvelope_NilGroup(t *testing.T) {
	adapter := &EventLoopGroupAdapter{group: nil}

	ctx := context.Background()
	env := NewEnvelope("test.topic", "key", []byte("data"), nil)

	err := adapter.DispatchEnvelope(ctx, env)

	if err == nil {
		t.Error("DispatchEnvelope() with nil group should return error")
	}

	// Check error type
	busErr, ok := err.(*EventBusError)
	if !ok {
		t.Errorf("Expected EventBusError, got %T", err)
	}

	if busErr.Code != "NO_EVENTLOOP_GROUP" {
		t.Errorf("Error code = %s, want NO_EVENTLOOP_GROUP", busErr.Code)
	}
}

func TestEventLoopGroupAdapter_DispatchEnvelope_NilContext(t *testing.T) {
	ctx := context.Background()
	config := eventloop.DefaultEventLoopConfig()
	group, err := eventloop.NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create EventLoopGroup: %v", err)
	}
	defer group.Close()

	adapter := NewEventLoopGroupAdapter(group)

	env := NewEnvelope("test.topic", "key", []byte("data"), nil)

	// nil context should be handled by the underlying implementation
	// We test that it doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DispatchEnvelope panicked with nil context: %v", r)
		}
	}()

	_ = adapter.DispatchEnvelope(nil, env)
}

func TestEventLoopGroupAdapter_DispatchEnvelope_NilEnvelope(t *testing.T) {
	ctx := context.Background()
	config := eventloop.DefaultEventLoopConfig()
	group, err := eventloop.NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create EventLoopGroup: %v", err)
	}
	defer group.Close()

	adapter := NewEventLoopGroupAdapter(group)

	// nil envelope should be handled by the underlying implementation
	err = adapter.DispatchEnvelope(ctx, nil)
	// The underlying DispatchEnvelope may return an error for nil envelope
	// We just verify it doesn't panic
	if err != nil {
		// Expected - nil envelope should return error
		t.Logf("DispatchEnvelope correctly returned error for nil envelope: %v", err)
	}
}

func TestEventLoopGroupAdapter_DispatchEnvelope_WithRoutingKey(t *testing.T) {
	ctx := context.Background()
	config := eventloop.DefaultEventLoopConfig()
	group, err := eventloop.NewEventLoopGroup(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create EventLoopGroup: %v", err)
	}
	defer group.Close()

	adapter := NewEventLoopGroupAdapter(group)

	tests := []struct {
		name    string
		key     string
		meta    map[string]string
		wantKey string
	}{
		{
			name:    "Direct key field",
			key:     "direct-key",
			meta:    map[string]string{"X-Route-Key": "meta-key"},
			wantKey: "direct-key",
		},
		{
			name:    "Key from meta X-Route-Key",
			key:     "",
			meta:    map[string]string{"X-Route-Key": "meta-key"},
			wantKey: "meta-key",
		},
		{
			name:    "Key from meta X-User-ID",
			key:     "",
			meta:    map[string]string{"X-User-ID": "user-123"},
			wantKey: "user-123",
		},
		{
			name:    "Empty key",
			key:     "",
			meta:    map[string]string{"Other": "value"},
			wantKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvelope("test.topic", tt.key, []byte("data"), tt.meta)
			envData := EnvelopeToEventLoopData(env)

			if envData.Key != tt.wantKey {
				t.Errorf("Routing key = %s, want %s", envData.Key, tt.wantKey)
			}

			err := adapter.DispatchEnvelope(ctx, env)
			if err != nil {
				t.Errorf("DispatchEnvelope() error = %v", err)
			}
		})
	}
}

func TestEventLoopGroupAdapter_ImplementsDispatcherInterface(t *testing.T) {
	// Verify that EventLoopGroupAdapter implements DispatcherInterface
	var _ DispatcherInterface = (*EventLoopGroupAdapter)(nil)
}
