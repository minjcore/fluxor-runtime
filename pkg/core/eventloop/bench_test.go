package eventloop

import (
	"context"
	"testing"
)

func BenchmarkEventLoopGroup_Dispatch(b *testing.B) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.NumLoops = 4
	config.QueueSize = 10000

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		b.Fatalf("NewEventLoopGroup failed: %v", err)
	}
	defer group.Close()

	event := &Event{
		Key:     "test-key",
		Address: "test.address",
		Body:    "test body",
		Headers: map[string]string{"X-Route-Key": "test-key"},
		Handler: func(ctx context.Context, ev *Event) error {
			return nil
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = group.Dispatch(ctx, event)
		}
	})
}

func BenchmarkEventLoopGroup_Dispatch_WithKeyExtraction(b *testing.B) {
	ctx := context.Background()
	config := DefaultEventLoopConfig()
	config.NumLoops = 4
	config.QueueSize = 10000

	group, err := NewEventLoopGroup(ctx, config)
	if err != nil {
		b.Fatalf("NewEventLoopGroup failed: %v", err)
	}
	defer group.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			event := &Event{
				Key:     "",
				Address: "test.address",
				Body:    "test body",
				Headers: map[string]string{"X-Route-Key": "user-123"},
				Handler: func(ctx context.Context, ev *Event) error {
					return nil
				},
			}
			_ = group.Dispatch(ctx, event)
		}
	})
}

func BenchmarkDispatcher_RouteKey(b *testing.B) {
	keys := []string{"user-1", "user-2", "user-3", "user-4", "user-5"}
	numLoops := 4

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		_ = RouteKey(key, numLoops)
	}
}

func BenchmarkHashKey(b *testing.B) {
	key := "user-123456789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = HashKey(key)
	}
}

func BenchmarkDefaultKeyExtractor(b *testing.B) {
	headers := map[string]string{
		"X-Route-Key":  "route-123",
		"X-User-ID":    "user-456",
		"X-Session-ID": "session-789",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultKeyExtractor(headers, "test.address", nil)
	}
}
