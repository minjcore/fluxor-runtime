package buckey_storage

import (
	"context"
	"testing"
)

func TestReplicatedStorage_GetQuorum_Repair(t *testing.T) {
	cfg := Config{Backend: BackendReplicated, Replicas: 3}
	r := NewReplicated(&cfg)
	ctx := context.Background()

	r.Put(ctx, "k1", []byte("v1"))
	got, err := r.GetQuorum(ctx, "k1", 2)
	if err != nil {
		t.Fatalf("GetQuorum: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("GetQuorum: got %q, want v1", got)
	}

	// Divergent: write different value to one replica
	r.replicas[2].Put(ctx, "k1", []byte("v1-bad"))
	got, err = r.GetQuorum(ctx, "k1", 2)
	if err != nil {
		t.Fatalf("GetQuorum after divergence: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("GetQuorum: got %q, want v1 (majority)", got)
	}
	// Repair should have fixed replica 2
	b, _ := r.replicas[2].Get(ctx, "k1")
	if string(b) != "v1" {
		t.Errorf("after repair replica 2: got %q", b)
	}
}

func TestReplicatedStorage_RepairKey(t *testing.T) {
	r := NewReplicated(nil)
	ctx := context.Background()
	r.Put(ctx, "x", []byte("same"))
	got, err := r.RepairKey(ctx, "x")
	if err != nil {
		t.Fatalf("RepairKey: %v", err)
	}
	if string(got) != "same" {
		t.Errorf("RepairKey: got %q", got)
	}
}
