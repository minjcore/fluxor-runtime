package aimodule

import (
	"testing"
	"time"
)

func TestCache_GetSet(t *testing.T) {
	cache := NewCache(5 * time.Minute)

	// Test Set and Get
	key := "test-key"
	response := &ChatResponse{
		ID:    "test-id",
		Model: "gpt-3.5-turbo",
		Choices: []Choice{
			{
				Message: Message{
					Role:    "assistant",
					Content: "Hello, world!",
				},
			},
		},
	}

	cache.Set(key, response)

	// Get should return the cached response
	cached, found := cache.Get(key)
	if !found {
		t.Error("Expected to find cached response")
	}
	if cached.ID != response.ID {
		t.Errorf("Expected ID %s, got %s", response.ID, cached.ID)
	}
}

func TestCache_Expiration(t *testing.T) {
	cache := NewCache(100 * time.Millisecond)

	key := "test-key"
	response := &ChatResponse{
		ID:    "test-id",
		Model: "gpt-3.5-turbo",
	}

	cache.Set(key, response)

	// Should be found immediately
	_, found := cache.Get(key)
	if !found {
		t.Error("Expected to find cached response immediately")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = cache.Get(key)
	if found {
		t.Error("Expected cached response to be expired")
	}
}

func TestGenerateCacheKey(t *testing.T) {
	req := ChatRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
	}

	key1, err := GenerateCacheKey(req)
	if err != nil {
		t.Fatalf("Failed to generate cache key: %v", err)
	}

	key2, err := GenerateCacheKey(req)
	if err != nil {
		t.Fatalf("Failed to generate cache key: %v", err)
	}

	if key1 != key2 {
		t.Error("Same request should generate same cache key")
	}

	// Different request should generate different key
	req2 := ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	key3, err := GenerateCacheKey(req2)
	if err != nil {
		t.Fatalf("Failed to generate cache key: %v", err)
	}

	if key1 == key3 {
		t.Error("Different requests should generate different cache keys")
	}
}
