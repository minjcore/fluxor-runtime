package cache

import (
	"context"
	"testing"
	"time"
)

func TestNewMultiCache(t *testing.T) {
	t.Run("with caches", func(t *testing.T) {
		memCache1 := NewMemoryCache()
		memCache2 := NewMemoryCache()

		cache := NewMultiCache(memCache1, memCache2)
		if cache == nil {
			t.Fatal("NewMultiCache() returned nil")
		}
	})

	t.Run("fail-fast no caches", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("NewMultiCache() should panic with no caches")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			if err.Error()[:10] != "fail-fast:" {
				t.Errorf("Expected 'fail-fast:' error, got %q", err.Error())
			}
		}()
		NewMultiCache()
	})

	t.Run("fail-fast nil cache", func(t *testing.T) {
		memCache := NewMemoryCache()

		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("NewMultiCache() should panic with nil cache")
			}
		}()
		NewMultiCache(memCache, nil)
	})
}

func TestMultiCache_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("get from first cache", func(t *testing.T) {
		memCache1 := NewMemoryCache()
		memCache2 := NewMemoryCache()
		multiCache := NewMultiCache(memCache1, memCache2)

		key := "test:key"
		value := []byte("test value")

		// Set in first cache only
		if err := memCache1.Set(ctx, key, value, 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Get from multi-cache (should find in first cache)
		got, err := multiCache.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(got) != string(value) {
			t.Errorf("Get() = %q, want %q", string(got), string(value))
		}

		// Verify value was propagated to second cache (propagation is async, wait a bit)
		time.Sleep(50 * time.Millisecond)
		got2, err := memCache2.Get(ctx, key)
		if err != nil {
			t.Errorf("Value should be propagated to second cache, Get() error = %v", err)
		}
		if string(got2) != string(value) {
			t.Errorf("Propagated value = %q, want %q", string(got2), string(value))
		}
	})

	t.Run("get from second cache", func(t *testing.T) {
		memCache1 := NewMemoryCache()
		memCache2 := NewMemoryCache()
		multiCache := NewMultiCache(memCache1, memCache2)

		key := "test:key2"
		value := []byte("test value 2")

		// Set in second cache only
		if err := memCache2.Set(ctx, key, value, 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Get from multi-cache (should find in second cache)
		got, err := multiCache.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(got) != string(value) {
			t.Errorf("Get() = %q, want %q", string(got), string(value))
		}
	})

	t.Run("key not found in any cache", func(t *testing.T) {
		memCache1 := NewMemoryCache()
		memCache2 := NewMemoryCache()
		multiCache := NewMultiCache(memCache1, memCache2)

		_, err := multiCache.Get(ctx, "nonexistent")
		if err == nil {
			t.Error("Get() should return error for nonexistent key")
		}
	})

	t.Run("fail-fast empty key", func(t *testing.T) {
		memCache := NewMemoryCache()
		multiCache := NewMultiCache(memCache)

		_, err := multiCache.Get(ctx, "")
		if err == nil {
			t.Error("Get() should return error for empty key")
		}
		if err.Error()[:10] != "fail-fast:" {
			t.Errorf("Expected 'fail-fast:' error, got %q", err.Error())
		}
	})

	t.Run("fail-fast nil context", func(t *testing.T) {
		memCache := NewMemoryCache()
		multiCache := NewMultiCache(memCache)

		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Get() should panic with nil context")
			}
		}()
		multiCache.Get(nil, "key")
	})
}

func TestMultiCache_Set(t *testing.T) {
	ctx := context.Background()

	t.Run("set in all caches", func(t *testing.T) {
		memCache1 := NewMemoryCache()
		memCache2 := NewMemoryCache()
		multiCache := NewMultiCache(memCache1, memCache2)

		key := "test:key"
		value := []byte("test value")

		if err := multiCache.Set(ctx, key, value, 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Verify value is in both caches
		got1, err := memCache1.Get(ctx, key)
		if err != nil {
			t.Errorf("Value should be in first cache, Get() error = %v", err)
		}
		if string(got1) != string(value) {
			t.Errorf("First cache value = %q, want %q", string(got1), string(value))
		}

		got2, err := memCache2.Get(ctx, key)
		if err != nil {
			t.Errorf("Value should be in second cache, Get() error = %v", err)
		}
		if string(got2) != string(value) {
			t.Errorf("Second cache value = %q, want %q", string(got2), string(value))
		}
	})

	t.Run("set with TTL", func(t *testing.T) {
		memCache1 := NewMemoryCache()
		memCache2 := NewMemoryCache()
		multiCache := NewMultiCache(memCache1, memCache2)

		key := "test:ttl"
		value := []byte("test value")
		ttl := 5 * time.Minute

		if err := multiCache.Set(ctx, key, value, ttl); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Verify TTL is set in both caches
		ttl1, err := memCache1.GetTTL(ctx, key)
		if err != nil {
			t.Fatalf("GetTTL() error = %v", err)
		}
		if ttl1 < 4*time.Minute+59*time.Second || ttl1 > 5*time.Minute {
			t.Errorf("First cache TTL = %v, want approximately %v", ttl1, ttl)
		}

		ttl2, err := memCache2.GetTTL(ctx, key)
		if err != nil {
			t.Fatalf("GetTTL() error = %v", err)
		}
		if ttl2 < 4*time.Minute+59*time.Second || ttl2 > 5*time.Minute {
			t.Errorf("Second cache TTL = %v, want approximately %v", ttl2, ttl)
		}
	})

	t.Run("fail-fast empty key", func(t *testing.T) {
		memCache := NewMemoryCache()
		multiCache := NewMultiCache(memCache)

		err := multiCache.Set(ctx, "", []byte("value"), 0)
		if err == nil {
			t.Error("Set() should return error for empty key")
		}
	})

	t.Run("fail-fast nil value", func(t *testing.T) {
		memCache := NewMemoryCache()
		multiCache := NewMultiCache(memCache)

		err := multiCache.Set(ctx, "key", nil, 0)
		if err == nil {
			t.Error("Set() should return error for nil value")
		}
	})

	t.Run("fail-fast negative TTL", func(t *testing.T) {
		memCache := NewMemoryCache()
		multiCache := NewMultiCache(memCache)

		err := multiCache.Set(ctx, "key", []byte("value"), -1*time.Second)
		if err == nil {
			t.Error("Set() should return error for negative TTL")
		}
	})
}

func TestMultiCache_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("delete from all caches", func(t *testing.T) {
		memCache1 := NewMemoryCache()
		memCache2 := NewMemoryCache()
		multiCache := NewMultiCache(memCache1, memCache2)

		key := "test:key"
		value := []byte("test value")

		// Set in both caches
		if err := memCache1.Set(ctx, key, value, 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		if err := memCache2.Set(ctx, key, value, 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Delete from multi-cache
		if err := multiCache.Delete(ctx, key); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify key is deleted from both caches
		_, err1 := memCache1.Get(ctx, key)
		if err1 == nil {
			t.Error("Key should be deleted from first cache")
		}

		_, err2 := memCache2.Get(ctx, key)
		if err2 == nil {
			t.Error("Key should be deleted from second cache")
		}
	})

	t.Run("fail-fast empty key", func(t *testing.T) {
		memCache := NewMemoryCache()
		multiCache := NewMultiCache(memCache)

		err := multiCache.Delete(ctx, "")
		if err == nil {
			t.Error("Delete() should return error for empty key")
		}
	})
}

func TestMultiCache_Clear(t *testing.T) {
	ctx := context.Background()

	t.Run("clear all caches", func(t *testing.T) {
		memCache1 := NewMemoryCache()
		memCache2 := NewMemoryCache()
		multiCache := NewMultiCache(memCache1, memCache2)

		// Add keys to both caches
		keys := []string{"key:1", "key:2", "key:3"}
		for _, key := range keys {
			if err := memCache1.Set(ctx, key, []byte("value"), 0); err != nil {
				t.Fatalf("Set() error = %v", err)
			}
			if err := memCache2.Set(ctx, key, []byte("value"), 0); err != nil {
				t.Fatalf("Set() error = %v", err)
			}
		}

		// Clear all
		if err := multiCache.Clear(ctx); err != nil {
			t.Fatalf("Clear() error = %v", err)
		}

		// Verify all keys are gone from both caches
		for _, key := range keys {
			_, err1 := memCache1.Get(ctx, key)
			if err1 == nil {
				t.Errorf("Key %q should be deleted from first cache", key)
			}

			_, err2 := memCache2.Get(ctx, key)
			if err2 == nil {
				t.Errorf("Key %q should be deleted from second cache", key)
			}
		}
	})
}

func TestMultiCache_Exists(t *testing.T) {
	ctx := context.Background()

	t.Run("exists in first cache", func(t *testing.T) {
		memCache1 := NewMemoryCache()
		memCache2 := NewMemoryCache()
		multiCache := NewMultiCache(memCache1, memCache2)

		key := "exists:key"
		if err := memCache1.Set(ctx, key, []byte("value"), 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		exists, err := multiCache.Exists(ctx, key)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if !exists {
			t.Error("Exists() = false, want true")
		}
	})

	t.Run("does not exist in any cache", func(t *testing.T) {
		memCache1 := NewMemoryCache()
		memCache2 := NewMemoryCache()
		multiCache := NewMultiCache(memCache1, memCache2)

		exists, err := multiCache.Exists(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if exists {
			t.Error("Exists() = true, want false")
		}
	})
}

func TestMultiCache_GetTTL(t *testing.T) {
	ctx := context.Background()

	t.Run("get TTL from first cache", func(t *testing.T) {
		memCache1 := NewMemoryCache()
		memCache2 := NewMemoryCache()
		multiCache := NewMultiCache(memCache1, memCache2)

		key := "ttl:key"
		ttl := 5 * time.Minute

		if err := memCache1.Set(ctx, key, []byte("value"), ttl); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		got, err := multiCache.GetTTL(ctx, key)
		if err != nil {
			t.Fatalf("GetTTL() error = %v", err)
		}
		if got < 4*time.Minute+59*time.Second || got > 5*time.Minute {
			t.Errorf("GetTTL() = %v, want approximately %v", got, ttl)
		}
	})
}
