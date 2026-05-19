package core

import (
	"context"
	"testing"
)

func TestWithFloxID(t *testing.T) {
	ctx := context.Background()
	floxid := "aggregate-123"
	ctxWithID := WithFloxID(ctx, floxid)

	retrievedID := GetFloxID(ctxWithID)
	if retrievedID != floxid {
		t.Errorf("GetFloxID() = %v, want %v", retrievedID, floxid)
	}
}

func TestGetFloxID_NoID(t *testing.T) {
	ctx := context.Background()
	id := GetFloxID(ctx)
	if id != "" {
		t.Errorf("GetFloxID() = %v, want empty string", id)
	}
}

func TestGenerateFloxID(t *testing.T) {
	id1 := GenerateFloxID()
	id2 := GenerateFloxID()

	if id1 == "" {
		t.Error("GenerateFloxID() returned empty string")
	}
	if id2 == "" {
		t.Error("GenerateFloxID() returned empty string")
	}
	if id1 == id2 {
		t.Error("GenerateFloxID() should generate unique IDs")
	}
}

func TestWithNewFloxID(t *testing.T) {
	ctx := context.Background()
	ctxWithID := WithNewFloxID(ctx)

	id := GetFloxID(ctxWithID)
	if id == "" {
		t.Error("WithNewFloxID() should generate a FloxID")
	}
}
