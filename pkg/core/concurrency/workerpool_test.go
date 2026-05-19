package concurrency

import (
	"context"
	"testing"
	"time"
)

func TestNewWorkerPool(t *testing.T) {
	ctx := context.Background()
	config := DefaultWorkerPoolConfig()

	pool := NewWorkerPool(ctx, config)

	if pool == nil {
		t.Error("NewWorkerPool() should not return nil")
	}
}

func TestWorkerPool_StartStop(t *testing.T) {
	ctx := context.Background()
	config := WorkerPoolConfig{
		Workers:   2,
		QueueSize: 10,
	}

	pool := NewWorkerPool(ctx, config)

	// Test start
	err := pool.Start()
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if !pool.IsRunning() {
		t.Error("IsRunning() should return true after Start()")
	}

	// Test double start
	err = pool.Start()
	if err == nil {
		t.Error("Start() when already running should fail")
	}

	// Test stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = pool.Stop(stopCtx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if pool.IsRunning() {
		t.Error("IsRunning() should return false after Stop()")
	}
}

func TestWorkerPool_Submit(t *testing.T) {
	ctx := context.Background()
	config := WorkerPoolConfig{
		Workers:   2,
		QueueSize: 10,
	}

	pool := NewWorkerPool(ctx, config)
	pool.Start()
	defer pool.Stop(context.Background())

	// Test nil task
	err := pool.Submit(nil)
	if err == nil {
		t.Error("Submit() with nil task should fail")
	}

	// Test submit when not running
	pool2 := NewWorkerPool(ctx, config)
	err = pool2.Submit(NewNamedTask("test", func(ctx context.Context) error {
		return nil
	}))
	if err == nil {
		t.Error("Submit() when not running should fail")
	}

	// Test valid submit
	task := NewNamedTask("test-task", func(ctx context.Context) error {
		return nil
	})

	err = pool.Submit(task)
	if err != nil {
		t.Errorf("Submit() error = %v", err)
	}

	// Wait for task to complete
	time.Sleep(100 * time.Millisecond)
}

func TestWorkerPool_Workers(t *testing.T) {
	ctx := context.Background()
	config := WorkerPoolConfig{
		Workers:   5,
		QueueSize: 10,
	}

	pool := NewWorkerPool(ctx, config)

	if pool.Workers() != 5 {
		t.Errorf("Workers() = %d, want 5", pool.Workers())
	}
}

func TestWorkerPool_Stats(t *testing.T) {
	ctx := context.Background()
	config := WorkerPoolConfig{
		Workers:   3,
		QueueSize: 20,
	}

	pool := NewWorkerPool(ctx, config)
	pool.Start()
	defer pool.Stop(context.Background())

	stats := pool.Stats()

	if stats.ActiveWorkers != 3 {
		t.Errorf("Stats().ActiveWorkers = %d, want 3", stats.ActiveWorkers)
	}

	if stats.QueueCapacity != 20 {
		t.Errorf("Stats().QueueCapacity = %d, want 20", stats.QueueCapacity)
	}

	// Submit some tasks
	for i := 0; i < 5; i++ {
		pool.Submit(NewNamedTask("test", func(ctx context.Context) error {
			return nil
		}))
	}

	time.Sleep(100 * time.Millisecond)

	stats = pool.Stats()
	if stats.CompletedTasks < 5 {
		t.Errorf("Expected at least 5 completed tasks, got %d", stats.CompletedTasks)
	}
}

func TestWorkerPool_SubmitBatch(t *testing.T) {
	ctx := context.Background()
	config := WorkerPoolConfig{
		Workers:   2,
		QueueSize: 100,
	}

	pool := NewWorkerPool(ctx, config)
	pool.Start()
	defer pool.Stop(context.Background())

	// Create batch of tasks
	tasks := make([]Task, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = NewNamedTask("batch-task", func(ctx context.Context) error {
			return nil
		})
	}

	// Submit batch
	err := pool.SubmitBatch(tasks)
	if err != nil {
		t.Errorf("SubmitBatch() error = %v", err)
	}

	// Wait for tasks to complete
	time.Sleep(200 * time.Millisecond)

	stats := pool.Stats()
	if stats.CompletedTasks < 10 {
		t.Errorf("Expected at least 10 completed tasks, got %d", stats.CompletedTasks)
	}
}
