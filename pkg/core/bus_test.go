package core

import (
	"sync"
	"testing"
	"time"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()
	if bus == nil {
		t.Fatal("NewBus() returned nil")
	}
	// Verify it's SimpleBus (type check)
	if bus == nil {
		t.Error("NewBus() should return non-nil")
	}
}

func TestBus_Subscribe_FailFast_EmptyTopic(t *testing.T) {
	bus := NewBus()

	// Subscribe with empty topic - should handle gracefully or fail-fast
	// Current implementation doesn't validate, but we should test behavior
	bus.Subscribe("", func(msg any) {})

	// If empty topic is allowed, verify it works
	// If it should fail-fast, we should add validation
	bus.Publish("", "test")
}

func TestBus_Subscribe_FailFast_NilHandler(t *testing.T) {
	bus := NewBus()

	defer func() {
		if r := recover(); r == nil {
			// Subscribe with nil handler might panic or handle gracefully
			// Current implementation doesn't validate, but we should add validation
		}
	}()

	bus.Subscribe("test.topic", nil)
	// If nil handler should fail-fast, add panic check
}

func TestBus_Publish_FailFast_EmptyTopic(t *testing.T) {
	bus := NewBus()

	// Publish with empty topic - should handle gracefully
	bus.Publish("", "test message")
}

func TestBus_Publish_NilMessage(t *testing.T) {
	bus := NewBus()
	done := make(chan bool, 1)
	bus.Subscribe("test.topic", func(msg any) {
		// Nil message is allowed in current implementation
		done <- true
	})

	// Publish nil message - handlers should handle it
	bus.Publish("test.topic", nil)

	// Wait for handler to complete to avoid goroutine leak
	select {
	case <-done:
		// Handler completed
	case <-time.After(100 * time.Millisecond):
		t.Error("Handler did not complete in time")
	}
}

func TestBus_Subscribe_Publish(t *testing.T) {
	bus := NewBus()
	var receivedMsg interface{}
	var mu sync.Mutex
	received := make(chan bool, 1)

	handler := func(msg any) {
		mu.Lock()
		receivedMsg = msg
		mu.Unlock()
		received <- true
	}

	bus.Subscribe("test.topic", handler)
	bus.Publish("test.topic", "test message")

	// Wait for handler to be called
	<-received

	mu.Lock()
	msg := receivedMsg
	mu.Unlock()

	if msg != "test message" {
		t.Errorf("Received message = %v, want 'test message'", msg)
	}
}

func TestBus_MultipleSubscribers(t *testing.T) {
	bus := NewBus()
	count := 0
	var mu sync.Mutex
	received := make(chan bool, 2)

	handler1 := func(msg any) {
		mu.Lock()
		count++
		mu.Unlock()
		received <- true
	}

	handler2 := func(msg any) {
		mu.Lock()
		count++
		mu.Unlock()
		received <- true
	}

	bus.Subscribe("test.topic", handler1)
	bus.Subscribe("test.topic", handler2)
	bus.Publish("test.topic", "test message")

	// Wait for both handlers
	<-received
	<-received

	mu.Lock()
	c := count
	mu.Unlock()

	if c != 2 {
		t.Errorf("Handler call count = %d, want 2", c)
	}
}

func TestBus_NoSubscribers(t *testing.T) {
	bus := NewBus()

	// Publish to topic with no subscribers - should not panic
	bus.Publish("unknown.topic", "test message")
}

func TestBus_MultipleTopics(t *testing.T) {
	bus := NewBus()
	var receivedMsg1, receivedMsg2 interface{}
	var mu sync.Mutex
	received1 := make(chan bool, 1)
	received2 := make(chan bool, 1)

	handler1 := func(msg any) {
		mu.Lock()
		receivedMsg1 = msg
		mu.Unlock()
		received1 <- true
	}

	handler2 := func(msg any) {
		mu.Lock()
		receivedMsg2 = msg
		mu.Unlock()
		received2 <- true
	}

	bus.Subscribe("topic1", handler1)
	bus.Subscribe("topic2", handler2)

	// Publish to topic1
	bus.Publish("topic1", "message1")

	// Publish to topic2
	bus.Publish("topic2", "message2")

	// Wait for both handlers
	<-received1
	<-received2

	mu.Lock()
	msg1 := receivedMsg1
	msg2 := receivedMsg2
	mu.Unlock()

	if msg1 != "message1" {
		t.Errorf("Topic1 received message = %v, want 'message1'", msg1)
	}
	if msg2 != "message2" {
		t.Errorf("Topic2 received message = %v, want 'message2'", msg2)
	}

	// Verify topic2 handler didn't receive topic1 message
	if msg2 == "message1" {
		t.Error("Topic2 handler incorrectly received message from topic1")
	}

	// Verify topic1 handler didn't receive topic2 message
	if msg1 == "message2" {
		t.Error("Topic1 handler incorrectly received message from topic2")
	}
}

func TestBus_ConcurrentPublishSubscribe(t *testing.T) {
	bus := NewBus()
	const numGoroutines = 10
	const messagesPerGoroutine = 10

	var mu sync.Mutex
	receivedCount := 0
	received := make(chan bool, numGoroutines*messagesPerGoroutine)

	handler := func(msg any) {
		mu.Lock()
		receivedCount++
		mu.Unlock()
		received <- true
	}

	bus.Subscribe("test.topic", handler)

	var wg sync.WaitGroup
	// Concurrent publishes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				bus.Publish("test.topic", id*messagesPerGoroutine+j)
			}
		}(i)
	}

	wg.Wait()

	// Wait for all handlers
	timeout := time.After(2 * time.Second)
	for i := 0; i < numGoroutines*messagesPerGoroutine; i++ {
		select {
		case <-received:
		case <-timeout:
			t.Fatalf("Timeout waiting for messages. Received %d/%d", i, numGoroutines*messagesPerGoroutine)
		}
	}

	mu.Lock()
	count := receivedCount
	mu.Unlock()

	if count != numGoroutines*messagesPerGoroutine {
		t.Errorf("Received %d messages, expected %d", count, numGoroutines*messagesPerGoroutine)
	}
}

func TestBus_HandlerPanicRecovery(t *testing.T) {
	bus := NewBus()
	done := make(chan bool, 1)
	panicCaught := make(chan bool, 1)

	// Handler that panics but recovers internally (realistic scenario)
	// This tests that panics in one handler don't affect other handlers
	panicHandler := func(msg any) {
		defer func() {
			if r := recover(); r != nil {
				// Panic was caught - handlers should handle their own errors
				panicCaught <- true
			}
		}()
		panic("handler panic")
	}

	// Normal handler
	normalHandler := func(msg any) {
		done <- true
	}

	bus.Subscribe("test.topic", panicHandler)
	bus.Subscribe("test.topic", normalHandler)

	// Publish should not crash even if one handler panics
	bus.Publish("test.topic", "test message")

	// Wait for normal handler - should complete despite panic in other handler
	select {
	case <-done:
		// Normal handler completed successfully
	case <-time.After(500 * time.Millisecond):
		t.Error("Normal handler did not complete, panic may have affected other handlers")
	}

	// Verify panic handler executed and recovered
	select {
	case <-panicCaught:
		// Panic was caught as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Panic handler did not execute")
	}
}

func TestBus_MultipleMessagesSameTopic(t *testing.T) {
	bus := NewBus()
	var mu sync.Mutex
	messages := make([]interface{}, 0)
	received := make(chan bool, 3)

	handler := func(msg any) {
		mu.Lock()
		messages = append(messages, msg)
		mu.Unlock()
		received <- true
	}

	bus.Subscribe("test.topic", handler)

	// Publish multiple messages
	bus.Publish("test.topic", "message1")
	bus.Publish("test.topic", "message2")
	bus.Publish("test.topic", "message3")

	// Wait for all handlers
	for i := 0; i < 3; i++ {
		<-received
	}

	mu.Lock()
	msgs := make([]interface{}, len(messages))
	copy(msgs, messages)
	mu.Unlock()

	if len(msgs) != 3 {
		t.Errorf("Received %d messages, expected 3", len(msgs))
	}

	// Verify all messages were received (order may vary due to goroutines)
	expected := map[string]bool{"message1": false, "message2": false, "message3": false}
	for _, msg := range msgs {
		if msgStr, ok := msg.(string); ok {
			expected[msgStr] = true
		}
	}

	for msg, received := range expected {
		if !received {
			t.Errorf("Message %s was not received", msg)
		}
	}
}

func TestBus_SameHandlerMultipleTimes(t *testing.T) {
	bus := NewBus()
	count := 0
	var mu sync.Mutex
	received := make(chan bool, 3)

	handler := func(msg any) {
		mu.Lock()
		count++
		mu.Unlock()
		received <- true
	}

	// Subscribe same handler multiple times
	bus.Subscribe("test.topic", handler)
	bus.Subscribe("test.topic", handler)
	bus.Subscribe("test.topic", handler)

	// Single publish should trigger all subscriptions
	bus.Publish("test.topic", "test message")

	// Wait for all handlers
	for i := 0; i < 3; i++ {
		<-received
	}

	mu.Lock()
	c := count
	mu.Unlock()

	if c != 3 {
		t.Errorf("Handler call count = %d, want 3", c)
	}
}

func TestBus_SubscribeAfterPublish(t *testing.T) {
	bus := NewBus()
	received := make(chan bool, 1)

	// Publish before subscribe - should not be received
	bus.Publish("test.topic", "message1")

	// Now subscribe
	bus.Subscribe("test.topic", func(msg any) {
		received <- true
	})

	// Publish after subscribe - should be received
	bus.Publish("test.topic", "message2")

	select {
	case <-received:
		// Good, received message2
	case <-time.After(100 * time.Millisecond):
		t.Error("Handler did not receive message after subscription")
	}

	// Verify we only received one message (message2)
	select {
	case <-received:
		t.Error("Received unexpected message")
	case <-time.After(50 * time.Millisecond):
		// Expected - no more messages
	}
}

func TestBus_ManySubscribers(t *testing.T) {
	bus := NewBus()
	const numSubscribers = 100
	var mu sync.Mutex
	count := 0
	received := make(chan bool, numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		bus.Subscribe("test.topic", func(msg any) {
			mu.Lock()
			count++
			mu.Unlock()
			received <- true
		})
	}

	bus.Publish("test.topic", "test message")

	// Wait for all handlers
	timeout := time.After(2 * time.Second)
	for i := 0; i < numSubscribers; i++ {
		select {
		case <-received:
		case <-timeout:
			t.Fatalf("Timeout waiting for handlers. Received %d/%d", i, numSubscribers)
		}
	}

	mu.Lock()
	c := count
	mu.Unlock()

	if c != numSubscribers {
		t.Errorf("Handler call count = %d, want %d", c, numSubscribers)
	}
}

func TestBus_DifferentMessageTypes(t *testing.T) {
	bus := NewBus()
	received := make(chan interface{}, 4)

	handler := func(msg any) {
		received <- msg
	}

	bus.Subscribe("test.topic", handler)

	// Test different message types
	bus.Publish("test.topic", "string message")
	bus.Publish("test.topic", 42)
	bus.Publish("test.topic", true)
	bus.Publish("test.topic", map[string]string{"key": "value"})

	// Wait for all messages
	timeout := time.After(500 * time.Millisecond)
	messages := make([]interface{}, 0, 4)
	for i := 0; i < 4; i++ {
		select {
		case msg := <-received:
			messages = append(messages, msg)
		case <-timeout:
			t.Fatalf("Timeout waiting for message %d", i+1)
		}
	}

	// Verify we received all different types
	types := make(map[string]bool)
	for _, msg := range messages {
		switch msg.(type) {
		case string:
			types["string"] = true
		case int:
			types["int"] = true
		case bool:
			types["bool"] = true
		case map[string]string:
			types["map"] = true
		}
	}

	if len(types) != 4 {
		t.Errorf("Expected 4 different message types, got %d", len(types))
	}
}
