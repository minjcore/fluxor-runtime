package buckey_storage

import (
	"context"
	"testing"
)

func TestMemoryStorage_PutGet(t *testing.T) {
	s := NewMemoryStorage()
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

func TestMemoryStorage_GetNotFound(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	_, err := s.Get(ctx, "missing")
	if err != ErrNotFound {
		t.Errorf("Get missing: got %v, want ErrNotFound", err)
	}
}

func TestMemoryStorage_Delete(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	s.Put(ctx, "k1", []byte("v1"))
	if err := s.Delete(ctx, "k1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := s.Get(ctx, "k1")
	if err != ErrNotFound {
		t.Errorf("after Delete: got %v, want ErrNotFound", err)
	}
}

func TestMemoryStorage_List(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	s.Put(ctx, "a/1", []byte("1"))
	s.Put(ctx, "a/2", []byte("2"))
	s.Put(ctx, "b/1", []byte("3"))

	all, err := s.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("List all: got %d keys, want 3", len(all))
	}

	prefixed, err := s.List(ctx, "a/")
	if err != nil {
		t.Fatalf("List a/: %v", err)
	}
	if len(prefixed) != 2 {
		t.Errorf("List a/: got %d keys, want 2", len(prefixed))
	}
	wantSet := map[string]bool{"a/1": true, "a/2": true}
	for _, k := range prefixed {
		if !wantSet[k] {
			t.Errorf("List a/: unexpected key %q", k)
		}
	}
}

func TestMemoryStorage_WithPrefix(t *testing.T) {
	cfg := Config{Backend: BackendMemory, Prefix: "app/"}
	s := NewMemory(&cfg)
	ctx := context.Background()

	s.Put(ctx, "k1", []byte("v1"))
	got, err := s.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("Get: got %q", got)
	}

	keys, err := s.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 1 || keys[0] != "k1" {
		t.Errorf("List: got %v", keys)
	}
}

func TestValidateKey(t *testing.T) {
	if err := ValidateKey(""); err == nil {
		t.Error("empty key should error")
	}
	if err := ValidateKey("ok"); err != nil {
		t.Errorf("valid key: %v", err)
	}
}
