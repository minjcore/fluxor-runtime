package concurrency

import (
	"context"
	"testing"
)

// BenchmarkExecutor_Submit benchmarks single task submission
func BenchmarkExecutor_Submit(b *testing.B) {
	ctx := context.Background()
	config := ExecutorConfig{
		Workers:   10,
		QueueSize: 1000,
	}

	executor := NewExecutor(ctx, config)
	defer executor.Shutdown(context.Background())

	task := NewNamedTask("bench-task", func(ctx context.Context) error {
		return nil
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = executor.Submit(task)
		}
	})
}

// BenchmarkExecutor_SubmitBatch benchmarks batch task submission
func BenchmarkExecutor_SubmitBatch(b *testing.B) {
	ctx := context.Background()
	config := ExecutorConfig{
		Workers:   10,
		QueueSize: 1000,
	}

	executor := NewExecutor(ctx, config)
	defer executor.Shutdown(context.Background())

	// Create batch of 10 tasks
	tasks := make([]Task, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = NewNamedTask("bench-task", func(ctx context.Context) error {
			return nil
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = executor.SubmitBatch(tasks)
		}
	})
}

// BenchmarkWorkerPool_Submit benchmarks worker pool single task submission
func BenchmarkWorkerPool_Submit(b *testing.B) {
	ctx := context.Background()
	config := WorkerPoolConfig{
		Workers:   10,
		QueueSize: 1000,
	}

	pool := NewWorkerPool(ctx, config)
	pool.Start()
	defer pool.Stop(context.Background())

	task := NewNamedTask("bench-task", func(ctx context.Context) error {
		return nil
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = pool.Submit(task)
		}
	})
}

// BenchmarkWorkerPool_SubmitBatch benchmarks worker pool batch task submission
func BenchmarkWorkerPool_SubmitBatch(b *testing.B) {
	ctx := context.Background()
	config := WorkerPoolConfig{
		Workers:   10,
		QueueSize: 1000,
	}

	pool := NewWorkerPool(ctx, config)
	pool.Start()
	defer pool.Stop(context.Background())

	// Create batch of 10 tasks
	tasks := make([]Task, 10)
	for i := 0; i < 10; i++ {
		tasks[i] = NewNamedTask("bench-task", func(ctx context.Context) error {
			return nil
		})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = pool.SubmitBatch(tasks)
		}
	})
}

// BenchmarkMailbox_Size benchmarks mailbox size operation
func BenchmarkMailbox_Size(b *testing.B) {
	mb := NewBoundedMailbox(1000)

	// Pre-fill mailbox
	for i := 0; i < 500; i++ {
		_ = mb.Send(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = mb.Size()
		}
	})
}

// BenchmarkExecutor_Stats benchmarks stats retrieval
func BenchmarkExecutor_Stats(b *testing.B) {
	ctx := context.Background()
	config := ExecutorConfig{
		Workers:   10,
		QueueSize: 1000,
	}

	executor := NewExecutor(ctx, config)
	defer executor.Shutdown(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.Stats()
	}
}

// BenchmarkWorkerPool_Stats benchmarks worker pool stats retrieval
func BenchmarkWorkerPool_Stats(b *testing.B) {
	ctx := context.Background()
	config := WorkerPoolConfig{
		Workers:   10,
		QueueSize: 1000,
	}

	pool := NewWorkerPool(ctx, config)
	pool.Start()
	defer pool.Stop(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.Stats()
	}
}
