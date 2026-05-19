package core

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/proto/fluxor/common"
)

func TestEventBus_Publish(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Test fail-fast: empty address
	err := eb.Publish("", "test")
	if err == nil {
		t.Error("Publish() with empty address should fail")
	}

	// Test fail-fast: nil body
	err = eb.Publish("test.address", nil)
	if err == nil {
		t.Error("Publish() with nil body should fail")
	}

	// Test valid publish
	err = eb.Publish("test.address", "test message")
	if err != nil {
		t.Errorf("Publish() error = %v", err)
	}
}

func TestEventBus_Send(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Test fail-fast: empty address
	err := eb.Send("", "test")
	if err == nil {
		t.Error("Send() with empty address should fail")
	}

	// Test fail-fast: nil body
	err = eb.Send("test.address", nil)
	if err == nil {
		t.Error("Send() with nil body should fail")
	}

	// Test fail-fast: no handlers
	err = eb.Send("test.address", "test")
	if err == nil {
		t.Error("Send() with no handlers should fail")
	}
	if ce, ok := err.(*EventBusError); ok {
		if ce.Code != "NO_HANDLERS" {
			t.Fatalf("Send() error code = %q, want %q", ce.Code, "NO_HANDLERS")
		}
	}

	// Register handler
	consumer := eb.Consumer("test.address")
	received := make(chan bool, 1)
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		received <- true
		return nil
	})

	// Test valid send
	err = eb.Send("test.address", "test message")
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Wait for message
	select {
	case <-received:
	case <-time.After(1 * time.Second):
		t.Error("Message not received")
	}
}

func TestEventBus_Request(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Test fail-fast: invalid timeout
	_, err := eb.Request("test.address", "test", 0)
	if err == nil {
		t.Error("Request() with zero timeout should fail")
	}

	// Test fail-fast: empty address
	_, err = eb.Request("", "test", 1*time.Second)
	if err == nil {
		t.Error("Request() with empty address should fail")
	}

	// Test fail-fast: nil body
	_, err = eb.Request("test.address", nil, 1*time.Second)
	if err == nil {
		t.Error("Request() with nil body should fail")
	}

	// Test fail-fast: no handlers
	_, err = eb.Request("no.handlers", "test", 1*time.Second)
	if err == nil {
		t.Error("Request() with no handlers should fail")
	}
	if ce, ok := err.(*EventBusError); ok {
		if ce.Code != "NO_HANDLERS" {
			t.Fatalf("Request() error code = %q, want %q", ce.Code, "NO_HANDLERS")
		}
	}

	// Register handler
	consumer := eb.Consumer("test.address")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		return msg.Reply("reply")
	})

	// Test valid request
	msg, err := eb.Request("test.address", "test", 1*time.Second)
	if err != nil {
		t.Errorf("Request() error = %v", err)
	}
	if msg == nil {
		t.Error("Request() returned nil message")
	}
}

func TestEventBus_Consumer(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Test fail-fast: empty address should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Consumer() with empty address should panic")
		}
	}()

	eb.Consumer("")
}

func TestConsumer_Handler_FailFast_NilHandlerPanics(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	c := eb.Consumer("test.address")
	defer func() {
		if r := recover(); r == nil {
			t.Error("Handler(nil) should panic (fail-fast)")
		}
	}()
	c.Handler(nil)
}

// TestMessage_Body tests Message.Body() method
func TestMessage_Body(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Setup consumer
	consumer := eb.Consumer("test.body")
	received := make(chan Message, 1)
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		received <- msg
		return nil
	})

	// Send message
	testBody := "test message body"
	err := eb.Send("test.body", testBody)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Wait for message
	select {
	case msg := <-received:
		body := msg.Body()
		if body == nil {
			t.Error("Body() returned nil")
		}
		// Body should be []byte after encoding
		if data, ok := body.([]byte); ok {
			if string(data) != `"test message body"` {
				t.Errorf("Body() = %q, want %q", string(data), `"test message body"`)
			}
		} else {
			t.Errorf("Body() type = %T, want []byte", body)
		}
	case <-time.After(1 * time.Second):
		t.Error("Message not received")
	}
}

// TestMessage_ReplyAddress tests Message.ReplyAddress() method
func TestMessage_ReplyAddress(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Setup handler that replies
	consumer := eb.Consumer("test.reply.address")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		replyAddr := msg.ReplyAddress()
		if replyAddr == "" {
			t.Error("ReplyAddress() should not be empty for request message")
		}
		return msg.Reply("reply")
	})

	// Send request
	reply, err := eb.Request("test.reply.address", "test", 1*time.Second)
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}

	if reply == nil {
		t.Fatal("Request() returned nil message")
	}

	// Reply message should not have reply address
	replyAddr := reply.ReplyAddress()
	if replyAddr != "" {
		t.Errorf("ReplyAddress() for reply message = %q, want empty", replyAddr)
	}
}

// TestMessage_DecodeBody_JSON tests Message.DecodeBody() with JSON
func TestMessage_DecodeBody_JSON(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Setup consumer
	consumer := eb.Consumer("test.decode.json")
	received := make(chan Message, 1)
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		received <- msg
		return nil
	})

	// Send JSON message
	testData := map[string]interface{}{
		"id":   "123",
		"name": "test",
	}
	err := eb.Send("test.decode.json", testData)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Wait for message
	select {
	case msg := <-received:
		var decoded map[string]interface{}
		err := msg.DecodeBody(&decoded)
		if err != nil {
			t.Fatalf("DecodeBody() error = %v", err)
		}

		if decoded["id"] != "123" {
			t.Errorf("DecodeBody() id = %v, want %q", decoded["id"], "123")
		}
		if decoded["name"] != "test" {
			t.Errorf("DecodeBody() name = %v, want %q", decoded["name"], "test")
		}
	case <-time.After(1 * time.Second):
		t.Error("Message not received")
	}
}

// TestMessage_DecodeBody_Protobuf tests Message.DecodeBody() with protobuf
func TestMessage_DecodeBody_Protobuf(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Setup consumer
	consumer := eb.Consumer("test.decode.protobuf")
	received := make(chan Message, 1)
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		received <- msg
		return nil
	})

	// Send protobuf message
	user := &common.User{
		Id:        "123",
		Name:      "John Doe",
		Email:     "john@example.com",
		CreatedAt: 1234567890,
		Active:    true,
	}
	err := eb.Send("test.decode.protobuf", user)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Wait for message
	select {
	case msg := <-received:
		var decoded common.User
		err := msg.DecodeBody(&decoded)
		if err != nil {
			t.Fatalf("DecodeBody() error = %v", err)
		}

		if decoded.GetId() != "123" {
			t.Errorf("DecodeBody() id = %q, want %q", decoded.GetId(), "123")
		}
		if decoded.GetName() != "John Doe" {
			t.Errorf("DecodeBody() name = %q, want %q", decoded.GetName(), "John Doe")
		}
		if decoded.GetEmail() != "john@example.com" {
			t.Errorf("DecodeBody() email = %q, want %q", decoded.GetEmail(), "john@example.com")
		}
	case <-time.After(1 * time.Second):
		t.Error("Message not received")
	}
}

// TestMessage_DecodeBody_InvalidBody tests Message.DecodeBody() with invalid body type
func TestMessage_DecodeBody_InvalidBody(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Create message with non-[]byte body
	msg := newMessage("not bytes", nil, "", eb)

	var result map[string]interface{}
	err := msg.DecodeBody(&result)
	if err == nil {
		t.Error("DecodeBody() with non-[]byte body should fail")
	}
}

// TestMessage_Fail tests Message.Fail() method
func TestMessage_Fail(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Setup handler that fails
	consumer := eb.Consumer("test.fail")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		return msg.Fail(500, "test error")
	})

	// Send request
	reply, err := eb.Request("test.fail", "test", 1*time.Second)
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}

	if reply == nil {
		t.Fatal("Request() returned nil message")
	}

	// Decode failure response
	var failure map[string]interface{}
	err = reply.DecodeBody(&failure)
	if err != nil {
		t.Fatalf("DecodeBody() error = %v", err)
	}

	if failure["failureCode"] != float64(500) {
		t.Errorf("Fail() failureCode = %v, want %d", failure["failureCode"], 500)
	}
	if failure["message"] != "test error" {
		t.Errorf("Fail() message = %q, want %q", failure["message"], "test error")
	}
}

// TestMessage_Fail_NoReplyAddress tests Message.Fail() without reply address
func TestMessage_Fail_NoReplyAddress(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Create message without reply address
	msg := newMessage([]byte("test"), nil, "", eb)

	err := msg.Fail(500, "test error")
	if err == nil {
		t.Error("Fail() without reply address should fail")
	}
	if err != ErrNoReplyAddress {
		t.Errorf("Fail() error = %v, want ErrNoReplyAddress", err)
	}
}

// TestMessage_Reply_NoReplyAddress tests Message.Reply() without reply address
func TestMessage_Reply_NoReplyAddress(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	eb := gocmd.EventBus()
	defer eb.Close()

	// Create message without reply address
	msg := newMessage([]byte("test"), nil, "", eb)

	err := msg.Reply("reply")
	if err == nil {
		t.Error("Reply() without reply address should fail")
	}
	if err != ErrNoReplyAddress {
		t.Errorf("Reply() error = %v, want ErrNoReplyAddress", err)
	}
}
