package network_test

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/base/bridge/network"
)

// startServer launches bridge-server on an ephemeral port and returns a stop func.
func startServer(t *testing.T, port int) func() {
	t.Helper()
	cmd := exec.Command("../server/bridge-server", fmt.Sprintf("%d", port))
	if err := cmd.Start(); err != nil {
		t.Skipf("bridge-server not available: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	return func() { cmd.Process.Kill() }
}

const testPort = 19091
const testURL = "http://localhost:19091"

func TestPool_BasicGet(t *testing.T) {
	stop := startServer(t, testPort)
	defer stop()

	pool := network.NewPool(testURL, 4, 30*time.Second)
	defer pool.Close()

	ctx := context.Background()
	client, release, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer release()

	got, err := client.Add(10, 32)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if got != 42 {
		t.Errorf("Add(10,32) = %d, want 42", got)
	}
}

func TestPool_ReleaseReturnsToPool(t *testing.T) {
	stop := startServer(t, testPort)
	defer stop()

	pool := network.NewPool(testURL, 2, 30*time.Second)
	defer pool.Close()

	ctx := context.Background()

	// Check out and return 5 times — pool should reuse the same 2 slots.
	for i := 0; i < 5; i++ {
		c, rel, err := pool.Get(ctx)
		if err != nil {
			t.Fatalf("iteration %d Get: %v", i, err)
		}
		v, err := c.Add(i, 1)
		if err != nil {
			t.Fatalf("iteration %d Add: %v", i, err)
		}
		if v != i+1 {
			t.Errorf("Add(%d,1) = %d, want %d", i, v, i+1)
		}
		rel()
	}

	stats := pool.Stats()
	if stats.Size > 2 {
		t.Errorf("pool grew beyond maxSize 2: size=%d", stats.Size)
	}
	if stats.Active != 0 {
		t.Errorf("active=%d after all releases, want 0", stats.Active)
	}
}

func TestPool_Concurrent(t *testing.T) {
	stop := startServer(t, testPort)
	defer stop()

	const (
		maxConns   = 4
		goroutines = 20
		iterations = 5
	)
	pool := network.NewPool(testURL, maxConns, 30*time.Second)
	defer pool.Close()

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				c, rel, err := pool.Get(ctx)
				cancel()
				if err != nil {
					errors <- fmt.Errorf("g%d i%d Get: %w", id, i, err)
					return
				}
				got, err := c.Add(id, i)
				rel()
				if err != nil {
					errors <- fmt.Errorf("g%d i%d Add: %w", id, i, err)
					return
				}
				if got != id+i {
					errors <- fmt.Errorf("g%d i%d Add(%d,%d)=%d want %d", id, i, id, i, got, id+i)
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}

	stats := pool.Stats()
	t.Logf("pool stats: %+v", stats)
	if stats.Size > maxConns {
		t.Errorf("size %d exceeded maxSize %d", stats.Size, maxConns)
	}
	if stats.Active != 0 {
		t.Errorf("active=%d after all releases", stats.Active)
	}
}

func TestPool_IdleEviction(t *testing.T) {
	stop := startServer(t, testPort)
	defer stop()

	pool := network.NewPool(testURL, 4, 200*time.Millisecond)
	defer pool.Close()

	ctx := context.Background()
	c, rel, err := pool.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	c.Add(1, 1)
	rel()

	stats := pool.Stats()
	if stats.Size != 1 {
		t.Fatalf("expected 1 slot after first use, got %d", stats.Size)
	}

	// Wait for idle eviction (idleTimeout=200ms, evict loop runs at 100ms).
	time.Sleep(500 * time.Millisecond)

	stats = pool.Stats()
	if stats.Size != 0 {
		t.Errorf("expected 0 slots after eviction, got %d", stats.Size)
	}
}

func TestPool_ContextCancel_WhenFull(t *testing.T) {
	stop := startServer(t, testPort)
	defer stop()

	pool := network.NewPool(testURL, 1, 30*time.Second)
	defer pool.Close()

	ctx := context.Background()
	_, rel, err := pool.Get(ctx) // occupy the only slot
	if err != nil {
		t.Fatal(err)
	}
	defer rel()

	// Second Get should block; we cancel it.
	cancelCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, _, err = pool.Get(cancelCtx)
	if err == nil {
		t.Fatal("expected error when pool is full and context cancelled")
	}
}

func TestPool_DoubleRelease(t *testing.T) {
	stop := startServer(t, testPort)
	defer stop()

	pool := network.NewPool(testURL, 2, 30*time.Second)
	defer pool.Close()

	ctx := context.Background()
	_, rel, err := pool.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	rel()
	rel() // second call must be a no-op, not panic or double-return to pool

	stats := pool.Stats()
	if stats.Active != 0 {
		t.Errorf("active=%d after double release", stats.Active)
	}
	if stats.Size > 1 {
		t.Errorf("size=%d after double release, pool grew", stats.Size)
	}
}
