package backoff

import (
	"context"
	"testing"
	"time"
)

func BenchmarkFixedBackoff_Delay(b *testing.B) {
	strategy := NewFixedBackoff(100 * time.Millisecond)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strategy.Delay(i % 10)
	}
}

func BenchmarkExponentialBackoff_Delay(b *testing.B) {
	strategy := NewExponentialBackoff(100*time.Millisecond, 10*time.Second, 2.0, false)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strategy.Delay(i % 10)
	}
}

func BenchmarkExponentialBackoff_DelayWithJitter(b *testing.B) {
	strategy := NewExponentialBackoff(100*time.Millisecond, 10*time.Second, 2.0, true)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strategy.Delay(i % 10)
	}
}

func BenchmarkLinearBackoff_Delay(b *testing.B) {
	strategy := NewLinearBackoff(100*time.Millisecond, 10*time.Second, 100*time.Millisecond)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = strategy.Delay(i % 10)
	}
}

func BenchmarkManager_Wait(b *testing.B) {
	manager := NewManager()
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Wait(ctx, i%10)
	}
}

func BenchmarkManager_Delay(b *testing.B) {
	manager := NewManager()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Delay(i % 10)
	}
}
