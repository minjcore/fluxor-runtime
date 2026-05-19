package eventloop

import (
	"testing"
)

func TestDefaultKeyExtractor(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		address  string
		body     interface{}
		expected string
	}{
		{
			name:     "X-Route-Key priority",
			headers:  map[string]string{"X-Route-Key": "route-123", "X-User-ID": "user-456"},
			address:  "test.address",
			body:     nil,
			expected: "route-123",
		},
		{
			name:     "X-User-ID fallback",
			headers:  map[string]string{"X-User-ID": "user-456"},
			address:  "test.address",
			body:     nil,
			expected: "user-456",
		},
		{
			name:     "X-Session-ID fallback",
			headers:  map[string]string{"X-Session-ID": "session-789"},
			address:  "test.address",
			body:     nil,
			expected: "session-789",
		},
		{
			name:     "X-Request-ID fallback",
			headers:  map[string]string{"X-Request-ID": "req-abc"},
			address:  "test.address",
			body:     nil,
			expected: "req-abc",
		},
		{
			name:     "No headers",
			headers:  nil,
			address:  "test.address",
			body:     nil,
			expected: "",
		},
		{
			name:     "Empty headers",
			headers:  map[string]string{},
			address:  "test.address",
			body:     nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultKeyExtractor(tt.headers, tt.address, tt.body)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHashKey(t *testing.T) {
	key1 := "test-key"
	key2 := "test-key"
	key3 := "different-key"

	hash1 := HashKey(key1)
	hash2 := HashKey(key2)
	hash3 := HashKey(key3)

	// Same key should produce same hash
	if hash1 != hash2 {
		t.Errorf("Same key should produce same hash: %d != %d", hash1, hash2)
	}

	// Different keys should produce different hashes (with high probability)
	if hash1 == hash3 {
		t.Errorf("Different keys should produce different hashes")
	}

	// Empty key should return 0
	emptyHash := HashKey("")
	if emptyHash != 0 {
		t.Errorf("Empty key should return 0, got %d", emptyHash)
	}
}

func TestRouteKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		numLoops int
		expected int
	}{
		{
			name:     "Valid key and loops",
			key:      "user-123",
			numLoops: 4,
			expected: int(HashKey("user-123") % 4),
		},
		{
			name:     "Empty key",
			key:      "",
			numLoops: 4,
			expected: 0,
		},
		{
			name:     "Zero loops",
			key:      "user-123",
			numLoops: 0,
			expected: 0,
		},
		{
			name:     "Single loop",
			key:      "user-123",
			numLoops: 1,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RouteKey(tt.key, tt.numLoops)
			if tt.numLoops > 0 && tt.key != "" {
				// For valid inputs, verify it's within range
				if result < 0 || result >= tt.numLoops {
					t.Errorf("RouteKey result %d out of range [0, %d)", result, tt.numLoops)
				}
			} else {
				// For edge cases, check expected value
				if result != tt.expected {
					t.Errorf("Expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestExtractRouteKey(t *testing.T) {
	event := &Event{
		Key:     "",
		Address: "test.address",
		Body:    "test body",
		Headers: map[string]string{"X-Route-Key": "route-123"},
	}

	key := ExtractRouteKey(event, nil)
	if key != "route-123" {
		t.Errorf("Expected 'route-123', got %q", key)
	}

	// Test with custom extractor
	customExtractor := func(headers map[string]string, address string, body interface{}) string {
		return "custom-key"
	}

	key = ExtractRouteKey(event, customExtractor)
	if key != "custom-key" {
		t.Errorf("Expected 'custom-key', got %q", key)
	}
}
