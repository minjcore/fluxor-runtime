package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func BenchmarkManager_Execute_Success(b *testing.B) {
	manager := NewManager()
	ctx := context.Background()
	fn := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Execute(ctx, fn)
	}
}

func BenchmarkManager_Execute_WithRetries(b *testing.B) {
	manager := NewManager()
	ctx := context.Background()
	attempts := 0
	fn := func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempts = 0
		_ = manager.Execute(ctx, fn)
	}
}

func BenchmarkManager_ExecuteWithConfig(b *testing.B) {
	manager := NewManager()
	ctx := context.Background()
	config := Config{
		MaxRetries: 3,
		Backoff:    NewFixedBackoff(1 * time.Millisecond),
		Predicate:  AlwaysRetry,
	}
	fn := func(ctx context.Context) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.ExecuteWithConfig(ctx, fn, config)
	}
}

func BenchmarkManager_Stats(b *testing.B) {
	manager := NewManager()
	ctx := context.Background()
	fn := func(ctx context.Context) error {
		return nil
	}
	
	// Execute some operations first
	for i := 0; i < 100; i++ {
		_ = manager.Execute(ctx, fn)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Stats()
	}
}
