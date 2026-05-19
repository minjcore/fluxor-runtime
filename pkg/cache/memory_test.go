package cache

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryCache_Get(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	t.Run("key not found", func(t *testing.T) {
		_, err := cache.Get(ctx, "nonexistent")
		if err == nil {
			t.Error("Get() should return error for nonexistent key")
		}
		if err.Error()[:10] != "key not fo" {
			t.Errorf("Expected 'key not found' error, got %q", err.Error())
		}
	})

	t.Run("get existing key", func(t *testing.T) {
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

	t.Run("get expired key", func(t *testing.T) {
		key := "expired:key"
		value := []byte("expired value")
		if err := cache.Set(ctx, key, value, 100*time.Millisecond); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		_, err := cache.Get(ctx, key)
		if err == nil {
			t.Error("Get() should return error for expired key")
		}
	})

	t.Run("fail-fast empty key", func(t *testing.T) {
		_, err := cache.Get(ctx, "")
		if err == nil {
			t.Error("Get() should return error for empty key")
		}
		if err.Error()[:10] != "fail-fast:" {
			t.Errorf("Expected 'fail-fast:' error, got %q", err.Error())
		}
	})

	t.Run("fail-fast nil context", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Get() should panic with nil context")
			}
		}()
		cache.Get(nil, "key")
	})
}

func TestMemoryCache_Set(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	tests := []struct {
		name    string
		key     string
		value   []byte
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:    "set with TTL",
			key:     "key:1",
			value:   []byte("value1"),
			ttl:     5 * time.Minute,
			wantErr: false,
		},
		{
			name:    "set without TTL",
			key:     "key:2",
			value:   []byte("value2"),
			ttl:     0,
			wantErr: false,
		},
		{
			name:    "fail-fast empty key",
			key:     "",
			value:   []byte("value"),
			ttl:     0,
			wantErr: true,
		},
		{
			name:    "fail-fast nil value",
			key:     "key:3",
			value:   nil,
			ttl:     0,
			wantErr: true,
		},
		{
			name:    "fail-fast negative TTL",
			key:     "key:4",
			value:   []byte("value"),
			ttl:     -1 * time.Second,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.Set(ctx, tt.key, tt.value, tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if err.Error()[:10] != "fail-fast:" {
					t.Errorf("Expected 'fail-fast:' error, got %q", err.Error())
				}
			}
		})
	}

	t.Run("fail-fast nil context", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Set() should panic with nil context")
			}
		}()
		cache.Set(nil, "key", []byte("value"), 0)
	})

	t.Run("overwrite existing key", func(t *testing.T) {
		key := "overwrite:key"
		value1 := []byte("value1")
		value2 := []byte("value2")

		if err := cache.Set(ctx, key, value1, 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		if err := cache.Set(ctx, key, value2, 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		got, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(got) != string(value2) {
			t.Errorf("Get() = %q, want %q", string(got), string(value2))
		}
	})
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	t.Run("delete existing key", func(t *testing.T) {
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

	t.Run("delete nonexistent key", func(t *testing.T) {
		// Should not error, just do nothing
		if err := cache.Delete(ctx, "nonexistent"); err != nil {
			t.Errorf("Delete() error = %v, want no error", err)
		}
	})

	t.Run("fail-fast empty key", func(t *testing.T) {
		err := cache.Delete(ctx, "")
		if err == nil {
			t.Error("Delete() should return error for empty key")
		}
		if err.Error()[:10] != "fail-fast:" {
			t.Errorf("Expected 'fail-fast:' error, got %q", err.Error())
		}
	})

	t.Run("fail-fast nil context", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Delete() should panic with nil context")
			}
		}()
		cache.Delete(nil, "key")
	})
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

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

	t.Run("fail-fast nil context", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("Clear() should panic with nil context")
			}
		}()
		cache.Clear(nil)
	})
}

func TestMemoryCache_Exists(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	t.Run("key exists", func(t *testing.T) {
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
		exists, err := cache.Exists(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if exists {
			t.Error("Exists() = true, want false")
		}
	})

	t.Run("expired key", func(t *testing.T) {
		key := "expired:exists"
		if err := cache.Set(ctx, key, []byte("value"), 100*time.Millisecond); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		time.Sleep(150 * time.Millisecond)

		exists, err := cache.Exists(ctx, key)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if exists {
			t.Error("Exists() = true for expired key, want false")
		}
	})

	t.Run("fail-fast empty key", func(t *testing.T) {
		_, err := cache.Exists(ctx, "")
		if err == nil {
			t.Error("Exists() should return error for empty key")
		}
	})
}

func TestMemoryCache_GetTTL(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	t.Run("key with TTL", func(t *testing.T) {
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
		_, err := cache.GetTTL(ctx, "nonexistent")
		if err == nil {
			t.Error("GetTTL() should return error for nonexistent key")
		}
	})

	t.Run("expired key", func(t *testing.T) {
		key := "expired:ttl"
		if err := cache.Set(ctx, key, []byte("value"), 100*time.Millisecond); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		time.Sleep(150 * time.Millisecond)

		_, err := cache.GetTTL(ctx, key)
		if err == nil {
			t.Error("GetTTL() should return error for expired key")
		}
	})
}

func TestMemoryCache_Stats(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Initial stats
	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("Stats().Size = %d, want 0", stats.Size)
	}
	if stats.Hits != 0 {
		t.Errorf("Stats().Hits = %d, want 0", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Errorf("Stats().Misses = %d, want 0", stats.Misses)
	}

	// Add some keys
	keys := []string{"key:1", "key:2", "key:3"}
	for _, key := range keys {
		if err := cache.Set(ctx, key, []byte("value"), 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
	}

	stats = cache.Stats()
	if stats.Size != int64(len(keys)) {
		t.Errorf("Stats().Size = %d, want %d", stats.Size, len(keys))
	}

	// Generate some hits
	for i := 0; i < 5; i++ {
		_, _ = cache.Get(ctx, "key:1")
	}

	// Generate some misses
	for i := 0; i < 3; i++ {
		_, _ = cache.Get(ctx, "nonexistent")
	}

	stats = cache.Stats()
	if stats.Hits < 5 {
		t.Errorf("Stats().Hits = %d, want at least 5", stats.Hits)
	}
	if stats.Misses < 3 {
		t.Errorf("Stats().Misses = %d, want at least 3", stats.Misses)
	}
	if stats.MemoryUsage <= 0 {
		t.Error("Stats().MemoryUsage should be > 0")
	}
}

func TestMemoryCache_Concurrent(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()

	// Concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 100
	numKeys := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numKeys; j++ {
				key := string(rune('a' + (id*numKeys+j)%26))
				value := []byte{byte(id), byte(j)}
				_ = cache.Set(ctx, key, value, 0)
			}
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numKeys; j++ {
				key := string(rune('a' + j%26))
				_, _ = cache.Get(ctx, key)
			}
		}()
	}

	wg.Wait()

	// Verify no panics occurred
	stats := cache.Stats()
	if stats.Size == 0 {
		t.Error("Cache should have some entries after concurrent operations")
	}
}
