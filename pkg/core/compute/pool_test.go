package compute

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Workers != 0 {
		t.Errorf("Expected Workers=0 (auto), got %d", config.Workers)
	}
	if config.ThreadsPerWorker != 0 {
		t.Errorf("Expected ThreadsPerWorker=0 (auto), got %d", config.ThreadsPerWorker)
	}
	if config.QueueSize == 0 {
		t.Error("Expected QueueSize > 0")
	}
	if config.BackpressurePolicy != Block {
		t.Errorf("Expected BackpressurePolicy=Block, got %v", config.BackpressurePolicy)
	}
}

func TestNewComputePool(t *testing.T) {
	ctx := context.Background()
	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 2
	config.QueueSize = 10

	pool, err := NewComputePool[int](ctx, handler, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	if pool == nil {
		t.Fatal("Pool is nil")
	}

	if err := pool.Stop(ctx); err != nil {
		t.Errorf("Failed to stop pool: %v", err)
	}
}

func TestComputePool_AutoScaling(t *testing.T) {
	ctx := context.Background()
	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 0          // Auto
	config.ThreadsPerWorker = 0 // Auto

	pool, err := NewComputePool[int](ctx, handler, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	// Auto-calculated workers should be > 0
	numCPU := runtime.GOMAXPROCS(0)
	expectedWorkers := numCPU / 2
	if expectedWorkers < 1 {
		expectedWorkers = 1
	}

	stats := pool.Stats()
	if stats.Workers != expectedWorkers {
		t.Errorf("Expected %d workers (auto), got %d", expectedWorkers, stats.Workers)
	}

	if err := pool.Stop(ctx); err != nil {
		t.Errorf("Failed to stop pool: %v", err)
	}
}

func TestComputePool_StartStop(t *testing.T) {
	ctx := context.Background()
	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 2

	pool, err := NewComputePool[int](ctx, handler, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	if pool.IsRunning() {
		t.Error("Pool should not be running before Start()")
	}

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	if !pool.IsRunning() {
		t.Error("Pool should be running after Start()")
	}

	// Stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Stop(stopCtx); err != nil {
		t.Errorf("Failed to stop pool: %v", err)
	}

	if pool.IsRunning() {
		t.Error("Pool should not be running after Stop()")
	}
}

func TestComputePool_Submit(t *testing.T) {
	ctx := context.Background()
	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		val := payload.(int)
		return val * 2, nil
	}

	config := DefaultConfig()
	config.Workers = 2

	pool, err := NewComputePool[int](ctx, handler, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)

	// Submit job
	future, err := pool.Submit(ctx, "test-key", 42)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Get result
	result, err := future.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get result: %v", err)
	}

	// Note: result is payload type (int), handler result is stored in HandlerResult
	handlerResult, err := future.GetHandlerResult(ctx)
	if err != nil {
		t.Fatalf("Failed to get handler result: %v", err)
	}

	if handlerResult == nil {
		t.Fatal("Handler result is nil")
	}

	resultInt, ok := handlerResult.(int)
	if !ok {
		t.Fatalf("Expected int, got %T", handlerResult)
	}

	if resultInt != 84 {
		t.Errorf("Expected handler result 84, got %d", resultInt)
	}

	_ = result // Suppress unused
}

func TestComputePool_Backpressure_Block(t *testing.T) {
	ctx := context.Background()
	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		time.Sleep(100 * time.Millisecond) // Slow handler
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 1
	config.QueueSize = 2
	config.BackpressurePolicy = Block

	pool, err := NewComputePool[int](ctx, handler, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)

	// Fill queue
	for i := 0; i < 3; i++ {
		_, err := pool.Submit(ctx, "", i)
		if err != nil {
			t.Fatalf("Failed to submit job %d: %v", i, err)
		}
	}

	// Next submit should block (but we'll timeout)
	submitCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = pool.Submit(submitCtx, "", 99)
	if err == nil {
		t.Error("Expected timeout/error when queue is full with Block policy")
	}
}

func TestComputePool_Backpressure_DropNewest(t *testing.T) {
	ctx := context.Background()
	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 1
	config.QueueSize = 2
	config.BackpressurePolicy = DropNewest

	pool, err := NewComputePool[int](ctx, handler, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)

	// Fill queue
	for i := 0; i < 3; i++ {
		_, err := pool.Submit(ctx, "", i)
		if err != nil {
			// With DropNewest, this might fail when queue is full
			_ = err
		}
	}
}

func TestComputePool_CoalesceByKey(t *testing.T) {
	ctx := context.Background()
	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		time.Sleep(50 * time.Millisecond)
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 1
	config.QueueSize = 10
	config.BackpressurePolicy = CoalesceByKey

	pool, err := NewComputePool[int](ctx, handler, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)

	// Submit multiple jobs with same key
	key := "same-key"
	futures := make([]*Future[int], 0, 5)

	for i := 0; i < 5; i++ {
		future, err := pool.Submit(ctx, key, i)
		if err != nil {
			t.Fatalf("Failed to submit job %d: %v", i, err)
		}
		futures = append(futures, future)
	}

	// With CoalesceByKey, older jobs should be dropped
	// Only the newest should complete
	completed := 0
	for _, future := range futures {
		_, err := future.GetWithTimeout(200 * time.Millisecond)
		if err == nil {
			completed++
		}
	}

	// Should have at least one result (newest)
	if completed == 0 {
		t.Error("Expected at least one job to complete with CoalesceByKey")
	}
}

func TestFuture_GetWithTimeout(t *testing.T) {
	ctx := context.Background()
	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		time.Sleep(200 * time.Millisecond)
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 1

	pool, err := NewComputePool[int](ctx, handler, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)

	future, err := pool.Submit(ctx, "", 42)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Get with short timeout (should timeout)
	_, err = future.GetWithTimeout(50 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error")
	}

	// Get with long timeout (should succeed)
	result, err := future.GetWithTimeout(500 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to get result: %v", err)
	}

	_ = result // Suppress unused
}

func TestFuture_IsDone(t *testing.T) {
	ctx := context.Background()
	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 1

	pool, err := NewComputePool[int](ctx, handler, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)

	future, err := pool.Submit(ctx, "", 42)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	if future.IsDone() {
		t.Error("Future should not be done immediately")
	}

	// Wait for result
	_, err = future.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get result: %v", err)
	}

	if !future.IsDone() {
		t.Error("Future should be done after Get()")
	}
}

func TestComputePool_Stats(t *testing.T) {
	ctx := context.Background()
	handler := func(ctx context.Context, payload interface{}) (interface{}, error) {
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 2

	pool, err := NewComputePool[int](ctx, handler, config)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	stats := pool.Stats()
	if stats.Workers != 2 {
		t.Errorf("Expected 2 workers, got %d", stats.Workers)
	}
	if stats.Running {
		t.Error("Pool should not be running before Start()")
	}

	if err := pool.Start(); err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop(ctx)

	stats = pool.Stats()
	if !stats.Running {
		t.Error("Pool should be running after Start()")
	}
	if len(stats.WorkersStats) != 2 {
		t.Errorf("Expected 2 worker stats, got %d", len(stats.WorkersStats))
	}
}
