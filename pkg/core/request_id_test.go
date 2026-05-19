package core

import (
	"context"
	"testing"
)

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-id"

	ctxWithID := WithRequestID(ctx, requestID)

	retrievedID := GetRequestID(ctxWithID)
	if retrievedID != requestID {
		t.Errorf("GetRequestID() = %v, want %v", retrievedID, requestID)
	}
}

func TestGetRequestID_NoID(t *testing.T) {
	ctx := context.Background()

	id := GetRequestID(ctx)
	if id != "" {
		t.Errorf("GetRequestID() = %v, want empty string", id)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	if id1 == "" {
		t.Error("GenerateRequestID() returned empty string")
	}

	if id2 == "" {
		t.Error("GenerateRequestID() returned empty string")
	}

	if id1 == id2 {
		t.Error("GenerateRequestID() should generate unique IDs")
	}
}

func TestWithNewRequestID(t *testing.T) {
	ctx := context.Background()

	ctxWithID := WithNewRequestID(ctx)

	id := GetRequestID(ctxWithID)
	if id == "" {
		t.Error("WithNewRequestID() should generate a request ID")
	}
}
