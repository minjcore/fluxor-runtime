package core

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/proto/fluxor/common"
)

// debugBreakpoint is a helper function for debugging.
// It prints the file and line number where it's called from.
// Usage: Call debugBreakpoint() anywhere in your code to see execution point.
func debugBreakpoint() {
	_, file, line, _ := runtime.Caller(1) // Use 1 to get caller's location
	fmt.Printf("DEBUG: %s:%d\n", file, line)
}

// Benchmark data setup
var (
	// Protobuf message
	protobufUser = &common.User{
		Id:        "12345",
		Name:      "John Doe",
		Email:     "john.doe@example.com",
		CreatedAt: 1234567890,
		Active:    true,
	}

	// JSON equivalent
	jsonUser = map[string]interface{}{
		"id":         "12345",
		"name":       "John Doe",
		"email":      "john.doe@example.com",
		"created_at": int64(1234567890),
		"active":     true,
	}
)

// BenchmarkEventBus_EncodeBody_Protobuf benchmarks protobuf encoding in eventbus
func BenchmarkEventBus_EncodeBody_Protobuf(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus().(*eventBus)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eb.encodeBody(protobufUser)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEventBus_EncodeBody_JSON benchmarks JSON encoding in eventbus
func BenchmarkEventBus_EncodeBody_JSON(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus().(*eventBus)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eb.encodeBody(jsonUser)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEventBus_DecodeBody_Protobuf benchmarks protobuf decoding in eventbus
func BenchmarkEventBus_DecodeBody_Protobuf(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus().(*eventBus)

	// Pre-encode the message
	encoded, err := eb.encodeBody(protobufUser)
	if err != nil {
		b.Fatal(err)
	}
	msg := newMessage(encoded, nil, "", eb)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var user common.User
		err := msg.DecodeBody(&user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEventBus_DecodeBody_JSON benchmarks JSON decoding in eventbus
func BenchmarkEventBus_DecodeBody_JSON(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus().(*eventBus)

	// Pre-encode the message
	encoded, err := eb.encodeBody(jsonUser)
	if err != nil {
		b.Fatal(err)
	}
	msg := newMessage(encoded, nil, "", eb)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		err := msg.DecodeBody(&result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEventBus_Send_Protobuf benchmarks sending protobuf message via eventbus
func BenchmarkEventBus_Send_Protobuf(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	// Setup consumer before benchmark
	done := make(chan struct{})
	consumer := eb.Consumer("bench.send.protobuf")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		select {
		case done <- struct{}{}:
		default:
		}
		return nil
	})
	defer consumer.Unregister()

	// Wait for consumer to be ready by sending a test message
	_ = eb.Send("bench.send.protobuf", protobufUser)
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		// Consumer is ready even if message not processed yet
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := eb.Send("bench.send.protobuf", protobufUser)
		if err != nil {
			// Ignore timeout errors in benchmark - they're expected under load
			if err != ErrTimeout {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkEventBus_Send_JSON benchmarks sending JSON message via eventbus
func BenchmarkEventBus_Send_JSON(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	// Setup consumer before benchmark
	done := make(chan struct{})
	consumer := eb.Consumer("bench.send.json")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		select {
		case done <- struct{}{}:
		default:
		}
		return nil
	})
	defer consumer.Unregister()

	// Wait for consumer to be ready by sending a test message
	_ = eb.Send("bench.send.json", jsonUser)
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		// Consumer is ready even if message not processed yet
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := eb.Send("bench.send.json", jsonUser)
		if err != nil {
			// Ignore timeout errors in benchmark - they're expected under load
			if err != ErrTimeout {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkEventBus_Send_Protobuf_Parallel benchmarks parallel protobuf sends
func BenchmarkEventBus_Send_Protobuf_Parallel(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	// Setup consumer before benchmark
	consumer := eb.Consumer("bench.send.protobuf.parallel")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		return nil
	})
	defer consumer.Unregister()

	// Wait a bit for consumer to be ready
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := eb.Send("bench.send.protobuf.parallel", protobufUser)
			if err != nil {
				// Ignore timeout errors in parallel benchmarks - they're expected under high load
				if err == ErrTimeout {
					continue
				}
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkEventBus_Send_JSON_Parallel benchmarks parallel JSON sends
func BenchmarkEventBus_Send_JSON_Parallel(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	// Setup consumer before benchmark
	consumer := eb.Consumer("bench.send.json.parallel")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		return nil
	})
	defer consumer.Unregister()

	// Wait a bit for consumer to be ready
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := eb.Send("bench.send.json.parallel", jsonUser)
			if err != nil {
				// Ignore timeout errors in parallel benchmarks - they're expected under high load
				if err == ErrTimeout {
					continue
				}
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkEventBus_Request_Protobuf benchmarks request-reply with protobuf
func BenchmarkEventBus_Request_Protobuf(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	// Setup handler that replies
	consumer := eb.Consumer("bench.request.protobuf")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		reply := &common.User{
			Id:        "reply",
			Name:      "Reply User",
			Email:     "reply@example.com",
			CreatedAt: 1234567890,
			Active:    true,
		}
		return msg.Reply(reply)
	})
	defer consumer.Unregister()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eb.Request("bench.request.protobuf", protobufUser, 1*time.Second)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEventBus_Request_JSON benchmarks request-reply with JSON
func BenchmarkEventBus_Request_JSON(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	// Setup handler that replies
	consumer := eb.Consumer("bench.request.json")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		reply := map[string]interface{}{
			"id":         "reply",
			"name":       "Reply User",
			"email":      "reply@example.com",
			"created_at": int64(1234567890),
			"active":     true,
		}
		return msg.Reply(reply)
	})
	defer consumer.Unregister()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := eb.Request("bench.request.json", jsonUser, 1*time.Second)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEventBus_Publish_Protobuf benchmarks publishing protobuf message
func BenchmarkEventBus_Publish_Protobuf(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	// Setup multiple consumers
	for i := 0; i < 3; i++ {
		consumer := eb.Consumer("bench.publish.protobuf")
		consumer.Handler(func(ctx FluxorContext, msg Message) error {
			return nil
		})
		defer consumer.Unregister()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := eb.Publish("bench.publish.protobuf", protobufUser)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEventBus_Publish_JSON benchmarks publishing JSON message
func BenchmarkEventBus_Publish_JSON(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	// Setup multiple consumers
	for i := 0; i < 3; i++ {
		consumer := eb.Consumer("bench.publish.json")
		consumer.Handler(func(ctx FluxorContext, msg Message) error {
			return nil
		})
		defer consumer.Unregister()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := eb.Publish("bench.publish.json", jsonUser)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEventBus_EndToEnd_Protobuf benchmarks full round-trip with protobuf
func BenchmarkEventBus_EndToEnd_Protobuf(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	// Setup consumer that decodes and processes
	consumer := eb.Consumer("bench.endtoend.protobuf")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		var user common.User
		if err := msg.DecodeBody(&user); err != nil {
			return err
		}
		// Simulate processing
		_ = user.GetId()
		_ = user.GetName()
		return nil
	})
	defer consumer.Unregister()

	// Wait a bit for consumer to be ready
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := eb.Send("bench.endtoend.protobuf", protobufUser)
		if err != nil {
			// Ignore timeout errors - they can happen if handler is slower than send rate
			if err != ErrTimeout {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkEventBus_EndToEnd_JSON benchmarks full round-trip with JSON
func BenchmarkEventBus_EndToEnd_JSON(b *testing.B) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	eb := gocmd.EventBus()

	// Setup consumer that decodes and processes
	consumer := eb.Consumer("bench.endtoend.json")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		var result map[string]interface{}
		if err := msg.DecodeBody(&result); err != nil {
			return err
		}
		// Simulate processing
		_ = result["id"]
		_ = result["name"]
		return nil
	})
	defer consumer.Unregister()

	// Wait a bit for consumer to be ready
	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := eb.Send("bench.endtoend.json", jsonUser)
		if err != nil {
			// Ignore timeout errors - they can happen if handler is slower than send rate
			if err != ErrTimeout {
				b.Fatal(err)
			}
		}
	}
}
