package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockRedisClient is a mock implementation of RedisClient for testing
type mockRedisClient struct {
	data       map[string]string
	expiration map[string]time.Time
	err        error // Set this to simulate errors
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		data:       make(map[string]string),
		expiration: make(map[string]time.Time),
	}
}

func (m *mockRedisClient) Get(ctx context.Context, key string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	value, ok := m.data[key]
	if !ok {
		return "", errors.New("key not found")
	}
	// Check expiration
	if exp, ok := m.expiration[key]; ok && !exp.IsZero() && time.Now().After(exp) {
		delete(m.data, key)
		delete(m.expiration, key)
		return "", errors.New("key expired")
	}
	return value, nil
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if m.err != nil {
		return m.err
	}
	m.data[key] = value.(string)
	if expiration > 0 {
		m.expiration[key] = time.Now().Add(expiration)
	}
	return nil
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) error {
	if m.err != nil {
		return m.err
	}
	for _, key := range keys {
		delete(m.data, key)
		delete(m.expiration, key)
	}
	return nil
}

func (m *mockRedisClient) FlushDB(ctx context.Context) error {
	if m.err != nil {
		return m.err
	}
	m.data = make(map[string]string)
	m.expiration = make(map[string]time.Time)
	return nil
}

func (m *mockRedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	count := int64(0)
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			// Check expiration
			if exp, ok := m.expiration[key]; ok && !exp.IsZero() && time.Now().After(exp) {
				continue
			}
			count++
		}
	}
	return count, nil
}

func (m *mockRedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	if m.err != nil {
		return 0, m.err
	}
	if _, ok := m.data[key]; !ok {
		return 0, errors.New("key not found")
	}
	if exp, ok := m.expiration[key]; ok && !exp.IsZero() {
		now := time.Now()
		if now.After(exp) {
			return 0, errors.New("key expired")
		}
		return exp.Sub(now), nil
	}
	return 0, nil // No expiration
}

func TestRedisCache_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("key not found", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		_, err := cache.Get(ctx, "nonexistent")
		if err == nil {
			t.Error("Get() should return error for nonexistent key")
		}
	})

	t.Run("get existing key", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		key := "test:key"
		value := []byte("test value")
		if err := cache.Set(ctx, key, value, 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		got, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(got) != string(value) {
			t.Errorf("Get() = %q, want %q", string(got), string(value))
		}
	})

	t.Run("redis not available", func(t *testing.T) {
		cache := NewRedisCache(nil, "test:")

		_, err := cache.Get(ctx, "key")
		if err == nil {
			t.Error("Get() should return error when Redis not available")
		}
		if err.Error()[:10] != "fail-fast:" {
			t.Errorf("Expected 'fail-fast:' error, got %q", err.Error())
		}
	})

	t.Run("fail-fast empty key", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		_, err := cache.Get(ctx, "")
		if err == nil {
			t.Error("Get() should return error for empty key")
		}
		if err.Error()[:10] != "fail-fast:" {
			t.Errorf("Expected 'fail-fast:' error, got %q", err.Error())
		}
	})

	t.Run("fail-fast nil context", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Get() should panic with nil context")
			}
		}()
		cache.Get(nil, "key")
	})

	t.Run("with prefix", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "app:")

		key := "user:123"
		value := []byte("value")
		if err := cache.Set(ctx, key, value, 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Verify key was stored with prefix
		fullKey := "app:" + key
		if _, ok := mockClient.data[fullKey]; !ok {
			t.Errorf("Key should be stored with prefix, expected %q", fullKey)
		}
	})
}

func TestRedisCache_Set(t *testing.T) {
	ctx := context.Background()

	t.Run("set with TTL", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		key := "key:1"
		value := []byte("value1")
		if err := cache.Set(ctx, key, value, 5*time.Minute); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		got, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(got) != string(value) {
			t.Errorf("Get() = %q, want %q", string(got), string(value))
		}
	})

	t.Run("fail-fast empty key", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		err := cache.Set(ctx, "", []byte("value"), 0)
		if err == nil {
			t.Error("Set() should return error for empty key")
		}
		if err.Error()[:10] != "fail-fast:" {
			t.Errorf("Expected 'fail-fast:' error, got %q", err.Error())
		}
	})

	t.Run("fail-fast nil value", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		err := cache.Set(ctx, "key", nil, 0)
		if err == nil {
			t.Error("Set() should return error for nil value")
		}
		if err.Error()[:10] != "fail-fast:" {
			t.Errorf("Expected 'fail-fast:' error, got %q", err.Error())
		}
	})

	t.Run("fail-fast negative TTL", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		err := cache.Set(ctx, "key", []byte("value"), -1*time.Second)
		if err == nil {
			t.Error("Set() should return error for negative TTL")
		}
		if err.Error()[:10] != "fail-fast:" {
			t.Errorf("Expected 'fail-fast:' error, got %q", err.Error())
		}
	})

	t.Run("redis not available", func(t *testing.T) {
		cache := NewRedisCache(nil, "test:")

		err := cache.Set(ctx, "key", []byte("value"), 0)
		if err == nil {
			t.Error("Set() should return error when Redis not available")
		}
	})
}

func TestRedisCache_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("delete existing key", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		key := "delete:key"
		value := []byte("value")
		if err := cache.Set(ctx, key, value, 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		if err := cache.Delete(ctx, key); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		_, err := cache.Get(ctx, key)
		if err == nil {
			t.Error("Get() should return error after Delete()")
		}
	})

	t.Run("fail-fast empty key", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		err := cache.Delete(ctx, "")
		if err == nil {
			t.Error("Delete() should return error for empty key")
		}
	})
}

func TestRedisCache_Clear(t *testing.T) {
	ctx := context.Background()

	t.Run("clear all keys", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		// Add some keys
		keys := []string{"key:1", "key:2", "key:3"}
		for _, key := range keys {
			if err := cache.Set(ctx, key, []byte("value"), 0); err != nil {
				t.Fatalf("Set() error = %v", err)
			}
		}

		// Clear all
		if err := cache.Clear(ctx); err != nil {
			t.Fatalf("Clear() error = %v", err)
		}

		// Verify all keys are gone
		for _, key := range keys {
			_, err := cache.Get(ctx, key)
			if err == nil {
				t.Errorf("Get() should return error after Clear() for key %q", key)
			}
		}
	})

	t.Run("redis not available", func(t *testing.T) {
		cache := NewRedisCache(nil, "test:")

		err := cache.Clear(ctx)
		if err == nil {
			t.Error("Clear() should return error when Redis not available")
		}
	})
}

func TestRedisCache_Exists(t *testing.T) {
	ctx := context.Background()

	t.Run("key exists", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		key := "exists:key"
		if err := cache.Set(ctx, key, []byte("value"), 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		exists, err := cache.Exists(ctx, key)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if !exists {
			t.Error("Exists() = false, want true")
		}
	})

	t.Run("key does not exist", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		exists, err := cache.Exists(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if exists {
			t.Error("Exists() = true, want false")
		}
	})
}

func TestRedisCache_GetTTL(t *testing.T) {
	ctx := context.Background()

	t.Run("key with TTL", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		key := "ttl:key"
		ttl := 5 * time.Minute
		if err := cache.Set(ctx, key, []byte("value"), ttl); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		got, err := cache.GetTTL(ctx, key)
		if err != nil {
			t.Fatalf("GetTTL() error = %v", err)
		}
		// TTL should be approximately 5 minutes (allow 1 second tolerance)
		if got < 4*time.Minute+59*time.Second || got > 5*time.Minute {
			t.Errorf("GetTTL() = %v, want approximately %v", got, ttl)
		}
	})

	t.Run("key without TTL", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		key := "no:ttl"
		if err := cache.Set(ctx, key, []byte("value"), 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		got, err := cache.GetTTL(ctx, key)
		if err != nil {
			t.Fatalf("GetTTL() error = %v", err)
		}
		if got != 0 {
			t.Errorf("GetTTL() = %v, want 0", got)
		}
	})

	t.Run("nonexistent key", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		_, err := cache.GetTTL(ctx, "nonexistent")
		if err == nil {
			t.Error("GetTTL() should return error for nonexistent key")
		}
	})
}

func TestRedisCache_IsAvailable(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "test:")

		if !cache.IsAvailable() {
			t.Error("IsAvailable() = false, want true")
		}
	})

	t.Run("not available", func(t *testing.T) {
		cache := NewRedisCache(nil, "test:")

		if cache.IsAvailable() {
			t.Error("IsAvailable() = true, want false")
		}
	})
}

func TestNewRedisCache(t *testing.T) {
	t.Run("with prefix", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "app:")

		if cache.prefix != "app:" {
			t.Errorf("prefix = %q, want %q", cache.prefix, "app:")
		}
	})

	t.Run("empty prefix uses default", func(t *testing.T) {
		mockClient := newMockRedisClient()
		cache := NewRedisCache(mockClient, "")

		if cache.prefix != "cache:" {
			t.Errorf("prefix = %q, want %q", cache.prefix, "cache:")
		}
	})
}
