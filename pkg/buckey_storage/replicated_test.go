package buckey_storage

import (
	"context"
	"testing"
)

func TestReplicatedStorage_PutGet(t *testing.T) {
	cfg := Config{Backend: BackendReplicated, Replicas: 3}
	s := NewReplicated(&cfg)
	ctx := context.Background()

	if err := s.Put(ctx, "k1", []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("Get: got %q, want v1", got)
	}
}

func TestReplicatedStorage_GetNotFound(t *testing.T) {
	s := NewReplicated(nil)
	ctx := context.Background()
	_, err := s.Get(ctx, "missing")
	if err != ErrNotFound {
		t.Errorf("Get missing: got %v, want ErrNotFound", err)
	}
}

func TestReplicatedStorage_Delete(t *testing.T) {
	s := NewReplicated(nil)
	ctx := context.Background()
	s.Put(ctx, "k1", []byte("v1"))
	_ = s.Delete(ctx, "k1")
	_, err := s.Get(ctx, "k1")
	if err != ErrNotFound {
		t.Errorf("after Delete: got %v, want ErrNotFound", err)
	}
}

func TestReplicatedStorage_List(t *testing.T) {
	s := NewReplicated(nil)
	ctx := context.Background()
	s.Put(ctx, "a", []byte("1"))
	s.Put(ctx, "b", []byte("2"))
	keys, err := s.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("List: got %d keys, want 2", len(keys))
	}
}
