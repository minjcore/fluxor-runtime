package buckey_storage

import (
	"context"
	"testing"
)

func TestFSStorage_PutGet(t *testing.T) {
	dir := t.TempDir()
	s, err := NewFS(dir, "")
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	ctx := context.Background()
	if err := s.Put(ctx, "k1", []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("Get: got %q", got)
	}
}

func TestFSStorage_GetNotFound(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewFS(dir, "")
	ctx := context.Background()
	_, err := s.Get(ctx, "missing")
	if err != ErrNotFound {
		t.Errorf("Get missing: got %v", err)
	}
}

func TestFSStorage_Delete_List(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewFS(dir, "")
	ctx := context.Background()
	s.Put(ctx, "a", []byte("1"))
	s.Put(ctx, "b", []byte("2"))
	keys, err := s.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("List: got %d keys", len(keys))
	}
	_ = s.Delete(ctx, "a")
	_, err = s.Get(ctx, "a")
	if err != ErrNotFound {
		t.Errorf("after Delete: got %v", err)
	}
}
