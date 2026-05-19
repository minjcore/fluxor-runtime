package core

import (
	"testing"
)

func TestNewEnvelope(t *testing.T) {
	topic := "test.topic"
	key := "test-key"
	data := []byte("test data")
	meta := map[string]string{"key1": "value1"}

	env := NewEnvelope(topic, key, data, meta)

	if env == nil {
		t.Fatal("NewEnvelope() returned nil")
	}

	if env.Topic != topic {
		t.Errorf("Topic = %s, want %s", env.Topic, topic)
	}

	if env.Key != key {
		t.Errorf("Key = %s, want %s", env.Key, key)
	}

	if string(env.Data) != string(data) {
		t.Errorf("Data = %v, want %v", env.Data, data)
	}

	if env.Meta["key1"] != meta["key1"] {
		t.Errorf("Meta[key1] = %s, want %s", env.Meta["key1"], meta["key1"])
	}
}

func TestNewEnvelope_NilMeta(t *testing.T) {
	env := NewEnvelope("test.topic", "key", []byte("data"), nil)

	if env == nil {
		t.Fatal("NewEnvelope() returned nil")
	}

	if env.Meta == nil {
		t.Error("Meta should be initialized as empty map, not nil")
	}

	if len(env.Meta) != 0 {
		t.Errorf("Meta should be empty, got %d items", len(env.Meta))
	}
}

func TestEnvelope_GetRoutingKey_Priority(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		meta        map[string]string
		expectedKey string
	}{
		{
			name:        "Key field takes highest priority",
			key:         "key-field",
			meta:        map[string]string{"X-Route-Key": "route-key", "X-User-ID": "user-id"},
			expectedKey: "key-field",
		},
		{
			name:        "X-Route-Key takes priority over others",
			key:         "",
			meta:        map[string]string{"X-Route-Key": "route-key", "X-User-ID": "user-id", "X-Session-ID": "session-id"},
			expectedKey: "route-key",
		},
		{
			name:        "X-User-ID takes priority over X-Session-ID and X-Request-ID",
			key:         "",
			meta:        map[string]string{"X-User-ID": "user-id", "X-Session-ID": "session-id", "X-Request-ID": "request-id"},
			expectedKey: "user-id",
		},
		{
			name:        "X-Session-ID takes priority over X-Request-ID",
			key:         "",
			meta:        map[string]string{"X-Session-ID": "session-id", "X-Request-ID": "request-id"},
			expectedKey: "session-id",
		},
		{
			name:        "X-Request-ID is used when others are missing",
			key:         "",
			meta:        map[string]string{"X-Request-ID": "request-id"},
			expectedKey: "request-id",
		},
		{
			name:        "Empty string when no keys present",
			key:         "",
			meta:        map[string]string{"Other-Key": "value"},
			expectedKey: "",
		},
		{
			name:        "Empty string when meta is nil",
			key:         "",
			meta:        nil,
			expectedKey: "",
		},
		{
			name:        "Empty string when key exists but is empty",
			key:         "",
			meta:        map[string]string{"X-Route-Key": "", "X-User-ID": "user-id"},
			expectedKey: "user-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &Envelope{
				Topic: "test.topic",
				Key:   tt.key,
				Data:  []byte("data"),
				Meta:  tt.meta,
			}

			result := env.GetRoutingKey()
			if result != tt.expectedKey {
				t.Errorf("GetRoutingKey() = %s, want %s", result, tt.expectedKey)
			}
		})
	}
}

func TestEnvelope_SetRoutingKey(t *testing.T) {
	env := NewEnvelope("test.topic", "", []byte("data"), nil)

	key := "new-routing-key"
	env.SetRoutingKey(key)

	if env.Key != key {
		t.Errorf("Key = %s, want %s", env.Key, key)
	}

	if env.Meta == nil {
		t.Fatal("Meta should be initialized")
	}

	if env.Meta["X-Route-Key"] != key {
		t.Errorf("Meta[X-Route-Key] = %s, want %s", env.Meta["X-Route-Key"], key)
	}
}

func TestEnvelope_SetRoutingKey_WithExistingMeta(t *testing.T) {
	meta := map[string]string{"key1": "value1"}
	env := NewEnvelope("test.topic", "", []byte("data"), meta)

	key := "new-routing-key"
	env.SetRoutingKey(key)

	if env.Key != key {
		t.Errorf("Key = %s, want %s", env.Key, key)
	}

	if env.Meta["X-Route-Key"] != key {
		t.Errorf("Meta[X-Route-Key] = %s, want %s", env.Meta["X-Route-Key"], key)
	}

	// Existing meta should be preserved
	if env.Meta["key1"] != "value1" {
		t.Errorf("Existing meta was overwritten: Meta[key1] = %s, want value1", env.Meta["key1"])
	}
}

func TestEnvelope_GetMeta(t *testing.T) {
	meta := map[string]string{"key1": "value1", "key2": "value2"}
	env := NewEnvelope("test.topic", "", []byte("data"), meta)

	tests := []struct {
		key      string
		expected string
	}{
		{"key1", "value1"},
		{"key2", "value2"},
		{"nonexistent", ""},
	}

	for _, tt := range tests {
		result := env.GetMeta(tt.key)
		if result != tt.expected {
			t.Errorf("GetMeta(%s) = %s, want %s", tt.key, result, tt.expected)
		}
	}
}

func TestEnvelope_GetMeta_NilMeta(t *testing.T) {
	env := &Envelope{
		Topic: "test.topic",
		Key:   "",
		Data:  []byte("data"),
		Meta:  nil,
	}

	result := env.GetMeta("any-key")
	if result != "" {
		t.Errorf("GetMeta() with nil Meta = %s, want empty string", result)
	}
}

func TestEnvelope_SetMeta(t *testing.T) {
	env := NewEnvelope("test.topic", "", []byte("data"), nil)

	env.SetMeta("key1", "value1")
	if env.GetMeta("key1") != "value1" {
		t.Errorf("SetMeta failed: GetMeta(key1) = %s, want value1", env.GetMeta("key1"))
	}

	env.SetMeta("key2", "value2")
	if env.GetMeta("key2") != "value2" {
		t.Errorf("SetMeta failed: GetMeta(key2) = %s, want value2", env.GetMeta("key2"))
	}

	// Update existing key
	env.SetMeta("key1", "updated-value")
	if env.GetMeta("key1") != "updated-value" {
		t.Errorf("SetMeta update failed: GetMeta(key1) = %s, want updated-value", env.GetMeta("key1"))
	}
}

func TestEnvelope_SetMeta_NilMeta(t *testing.T) {
	env := &Envelope{
		Topic: "test.topic",
		Key:   "",
		Data:  []byte("data"),
		Meta:  nil,
	}

	env.SetMeta("key1", "value1")

	if env.Meta == nil {
		t.Fatal("SetMeta should initialize Meta map if nil")
	}

	if env.GetMeta("key1") != "value1" {
		t.Errorf("SetMeta with nil Meta failed: GetMeta(key1) = %s, want value1", env.GetMeta("key1"))
	}
}

func TestEnvelope_GetRoutingKey_AfterSetRoutingKey(t *testing.T) {
	env := NewEnvelope("test.topic", "", []byte("data"), nil)

	// Set via SetRoutingKey
	key := "set-via-method"
	env.SetRoutingKey(key)

	// GetRoutingKey should return the key
	result := env.GetRoutingKey()
	if result != key {
		t.Errorf("GetRoutingKey() after SetRoutingKey() = %s, want %s", result, key)
	}
}

func TestEnvelope_EmptyKeyAndMeta(t *testing.T) {
	env := NewEnvelope("test.topic", "", []byte("data"), map[string]string{})

	result := env.GetRoutingKey()
	if result != "" {
		t.Errorf("GetRoutingKey() with empty key and meta = %s, want empty string", result)
	}
}

func TestEnvelope_MetaWithEmptyValues(t *testing.T) {
	meta := map[string]string{
		"X-Route-Key":  "",
		"X-User-ID":    "",
		"X-Session-ID": "",
		"X-Request-ID": "",
		"Other-Key":    "value",
	}
	env := NewEnvelope("test.topic", "", []byte("data"), meta)

	result := env.GetRoutingKey()
	if result != "" {
		t.Errorf("GetRoutingKey() with empty routing key values = %s, want empty string", result)
	}
}
