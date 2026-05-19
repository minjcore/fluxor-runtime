package core

import (
	"context"
	"testing"
)

// mockMessage is a minimal mock implementation for testing
type mockMessage struct {
	headers      map[string]string
	body         interface{}
	replyAddress string
}

func (m *mockMessage) Headers() map[string]string {
	if m.headers == nil {
		return make(map[string]string)
	}
	return m.headers
}

func (m *mockMessage) Body() interface{} {
	return m.body
}

func (m *mockMessage) ReplyAddress() string {
	return m.replyAddress
}

func (m *mockMessage) Reply(body interface{}) error {
	return nil
}

func (m *mockMessage) DecodeBody(v interface{}) error {
	return nil
}

func (m *mockMessage) Fail(failureCode int, message string) error {
	return nil
}

func TestNewBaseHandler(t *testing.T) {
	handler := NewBaseHandler("test-handler")
	if handler == nil {
		t.Fatal("NewBaseHandler() returned nil")
	}
	if handler.Name() != "test-handler" {
		t.Errorf("Name() = %v, want 'test-handler'", handler.Name())
	}
}

func TestBaseHandler_Handle_FailFast_NilContext(t *testing.T) {
	handler := NewBaseHandler("test")
	msg := &mockMessage{body: "test"}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Handle() should panic (fail-fast) with nil context")
		}
	}()

	handler.Handle(nil, msg)
}

func TestBaseHandler_Handle_FailFast_NilMessage(t *testing.T) {
	handler := NewBaseHandler("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Handle() should panic (fail-fast) with nil message")
		}
	}()

	handler.Handle(fluxorCtx, nil)
}

func TestBaseHandler_Reply_FailFast_NilMessage(t *testing.T) {
	handler := NewBaseHandler("test")

	defer func() {
		if r := recover(); r == nil {
			t.Error("Reply() should panic (fail-fast) with nil message")
		}
	}()

	handler.Reply(nil, "test body")
}

func TestBaseHandler_Fail_FailFast_NilMessage(t *testing.T) {
	handler := NewBaseHandler("test")

	defer func() {
		if r := recover(); r == nil {
			t.Error("Fail() should panic (fail-fast) with nil message")
		}
	}()

	handler.Fail(nil, 500, "test error")
}

func TestBaseHandler_DecodeBody_FailFast_NilMessage(t *testing.T) {
	handler := NewBaseHandler("test")
	var result map[string]string

	defer func() {
		if r := recover(); r == nil {
			t.Error("DecodeBody() should panic (fail-fast) with nil message")
		}
	}()

	handler.DecodeBody(nil, &result)
}

func TestBaseHandler_DecodeBody_FailFast_EmptyBody(t *testing.T) {
	handler := NewBaseHandler("test")
	msg := &mockMessage{body: nil}
	var result map[string]string

	err := handler.DecodeBody(msg, &result)
	if err == nil {
		t.Error("DecodeBody() should fail-fast with empty body")
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "EMPTY_BODY" {
				t.Errorf("Error code = %v, want 'EMPTY_BODY'", e.Code)
			}
		}
	}
}

func TestBaseHandler_DecodeBody_FailFast_NilTarget(t *testing.T) {
	handler := NewBaseHandler("test")
	msg := &mockMessage{body: []byte(`{"key":"value"}`)}

	err := handler.DecodeBody(msg, nil)
	if err == nil {
		t.Error("DecodeBody() should fail-fast with nil target")
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "NIL_TARGET" {
				t.Errorf("Error code = %v, want 'NIL_TARGET'", e.Code)
			}
		}
	}
}

func TestBaseHandler_EncodeBody_FailFast_NilInput(t *testing.T) {
	handler := NewBaseHandler("test")

	_, err := handler.EncodeBody(nil)
	if err == nil {
		t.Error("EncodeBody() should fail-fast with nil input")
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "NIL_INPUT" {
				t.Errorf("Error code = %v, want 'NIL_INPUT'", e.Code)
			}
		}
	}
}

func TestBaseHandler_SetLogger(t *testing.T) {
	handler := NewBaseHandler("test")
	newLogger := NewDefaultLogger()

	handler.SetLogger(newLogger)
	// No easy way to verify logger was set, but should not panic
}

func TestBaseHandler_Handle_NotImplemented(t *testing.T) {
	handler := NewBaseHandler("test")
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)
	msg := &mockMessage{body: "test"}

	// Default doHandle should return NOT_IMPLEMENTED error
	err := handler.Handle(fluxorCtx, msg)
	if err == nil {
		t.Error("Handle() should return error when doHandle not implemented")
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "NOT_IMPLEMENTED" {
				t.Errorf("Error code = %v, want 'NOT_IMPLEMENTED'", e.Code)
			}
		}
	}
}
