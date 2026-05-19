package buckey_storage

import (
	"context"
	"testing"
	"time"
)

func TestTTLStorage_PutWithTTL_Get(t *testing.T) {
	s := NewMemoryStorage()
	ttl := NewTTLStorage(s, 0)
	ctx := context.Background()
	if err := ttl.PutWithTTL(ctx, "k1", []byte("v1"), 50*time.Millisecond); err != nil {
		t.Fatalf("PutWithTTL: %v", err)
	}
	got, err := ttl.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("Get: got %q", got)
	}
	time.Sleep(60 * time.Millisecond)
	_, err = ttl.Get(ctx, "k1")
	if err != ErrNotFound {
		t.Errorf("after TTL: got %v, want ErrNotFound", err)
	}
}

func TestTTLStorage_DefaultTTL(t *testing.T) {
	s := NewMemoryStorage()
	ttl := NewTTLStorage(s, 30*time.Millisecond)
	ctx := context.Background()
	ttl.Put(ctx, "k1", []byte("v1"))
	got, _ := ttl.Get(ctx, "k1")
	if string(got) != "v1" {
		t.Errorf("Get: got %q", got)
	}
	time.Sleep(40 * time.Millisecond)
	_, err := ttl.Get(ctx, "k1")
	if err != ErrNotFound {
		t.Errorf("after default TTL: got %v", err)
	}
}
