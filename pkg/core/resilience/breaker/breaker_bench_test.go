package breaker

import (
	"context"
	"testing"
	"time"
)

func BenchmarkManager_Execute_ClosedState(b *testing.B) {
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

func BenchmarkManager_Allow(b *testing.B) {
	manager := NewManager()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Allow(ctx)
	}
}

func BenchmarkManager_State(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.State()
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

func BenchmarkManager_StateTransition(b *testing.B) {
	config := Config{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		Timeout:          100 * time.Millisecond,
	}
	manager := NewManagerWithConfig(config)
	ctx := context.Background()

	// Setup: force circuit to open
	fnFail := func(ctx context.Context) error {
		return context.DeadlineExceeded
	}
	for i := 0; i < 3; i++ {
		_ = manager.Execute(ctx, fnFail)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.State()
	}
}
