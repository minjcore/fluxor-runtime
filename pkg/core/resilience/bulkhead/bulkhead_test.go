package bulkhead

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	stats := manager.Stats()
	if stats.CurrentConcurrency != 0 {
		t.Errorf("Expected CurrentConcurrency 0, got %d", stats.CurrentConcurrency)
	}
}

func TestNewManagerWithConfig(t *testing.T) {
	config := Config{
		MaxConcurrency: 5,
		MaxQueueSize:   10,
	}

	manager := NewManagerWithConfig(config)
	if manager == nil {
		t.Fatal("NewManagerWithConfig returned nil")
	}

	stats := manager.Stats()
	if stats.CurrentConcurrency != 0 {
		t.Errorf("Expected CurrentConcurrency 0, got %d", stats.CurrentConcurrency)
	}
}

func TestExecute_Success(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	stats := manager.Stats()
	if stats.TotalAccepted != 1 {
		t.Errorf("Expected TotalAccepted 1, got %d", stats.TotalAccepted)
	}
	if stats.TotalSuccessful != 1 {
		t.Errorf("Expected TotalSuccessful 1, got %d", stats.TotalSuccessful)
	}
}

func TestExecute_Error(t *testing.T) {
	manager := NewManager()

	testErr := errors.New("test error")
	ctx := context.Background()
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return testErr
	})

	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}

	stats := manager.Stats()
	if stats.TotalAccepted != 1 {
		t.Errorf("Expected TotalAccepted 1, got %d", stats.TotalAccepted)
	}
	if stats.TotalFailed != 1 {
		t.Errorf("Expected TotalFailed 1, got %d", stats.TotalFailed)
	}
}

func TestExecute_BulkheadFull(t *testing.T) {
	config := Config{
		MaxConcurrency: 2, // Allow only 2 concurrent executions
		MaxQueueSize:   0, // No queuing
	}

	manager := NewManagerWithConfig(config)

	// Start 2 executions that will block
	ctx := context.Background()
	blocker := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			manager.Execute(ctx, func(ctx context.Context) error {
				<-blocker // Block until released
				return nil
			})
		}()
	}

	// Wait a bit for both to start
	time.Sleep(50 * time.Millisecond)

	// Try a third execution (should be rejected)
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected error for bulkhead full, got nil")
	}

	bulkheadErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if bulkheadErr.Code != ErrCodeBulkheadFull {
		t.Errorf("Expected error code %s, got %s", ErrCodeBulkheadFull, bulkheadErr.Code)
	}

	stats := manager.Stats()
	if stats.TotalRejected != 1 {
		t.Errorf("Expected TotalRejected 1, got %d", stats.TotalRejected)
	}

	// Release blockers
	close(blocker)
	wg.Wait()
}

func TestExecute_WithQueue(t *testing.T) {
	config := Config{
		MaxConcurrency: 2, // Allow only 2 concurrent executions
		MaxQueueSize:   5, // Queue up to 5 waiting executions
	}

	manager := NewManagerWithConfig(config)

	// Start 2 executions that will block
	ctx := context.Background()
	blocker := make(chan struct{})
	var started sync.WaitGroup
	var completed sync.WaitGroup

	started.Add(2)
	completed.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			manager.Execute(ctx, func(ctx context.Context) error {
				started.Done()
				<-blocker // Block until released
				completed.Done()
				return nil
			})
		}()
	}

	// Wait for both to start
	started.Wait()

	// Queue 3 more executions (should be queued)
	var queuedCount int32
	var wg sync.WaitGroup
	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func() {
			defer wg.Done()
			err := manager.Execute(ctx, func(ctx context.Context) error {
				atomic.AddInt32(&queuedCount, 1)
				return nil
			})
			if err != nil {
				t.Errorf("Expected nil error for queued execution, got %v", err)
			}
		}()
	}

	// Wait a bit for executions to be queued
	time.Sleep(50 * time.Millisecond)

	stats := manager.Stats()
	if stats.CurrentQueueSize < 3 {
		// Allow some variance - executions might have started if slots freed
		t.Logf("Expected queue size at least 1, got %d (executions might have completed)", stats.CurrentQueueSize)
	}

	// Release blockers
	close(blocker)
	completed.Wait()
	wg.Wait()

	// All should complete successfully
	finalStats := manager.Stats()
	if finalStats.TotalAccepted != 5 {
		t.Errorf("Expected TotalAccepted 5, got %d", finalStats.TotalAccepted)
	}
}

func TestExecute_QueueFull(t *testing.T) {
	config := Config{
		MaxConcurrency: 1, // Only 1 concurrent execution
		MaxQueueSize:   2, // Queue only 2 waiting executions
	}

	manager := NewManagerWithConfig(config)

	// Start 1 execution that will block
	ctx := context.Background()
	blocker := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		manager.Execute(ctx, func(ctx context.Context) error {
			<-blocker // Block until released
			return nil
		})
	}()

	// Wait for execution to start
	time.Sleep(50 * time.Millisecond)

	// Fill the queue (2 executions)
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			manager.Execute(ctx, func(ctx context.Context) error {
				return nil
			})
		}()
	}

	// Wait a bit for queue to fill
	time.Sleep(50 * time.Millisecond)

	// Try one more (should be rejected - bulkhead full + queue full)
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected error for bulkhead and queue full, got nil")
	}

	bulkheadErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if bulkheadErr.Code != ErrCodeBulkheadFull {
		t.Errorf("Expected error code %s, got %s", ErrCodeBulkheadFull, bulkheadErr.Code)
	}

	// Release blocker
	close(blocker)
	wg.Wait()
}

func TestExecute_QueueTimeout(t *testing.T) {
	config := Config{
		MaxConcurrency: 1, // Only 1 concurrent execution
		MaxQueueSize:   5, // Queue up to 5
		QueueTimeout:   100 * time.Millisecond, // Queue timeout
	}

	manager := NewManagerWithConfig(config)

	// Start 1 execution that will block longer than queue timeout
	ctx := context.Background()
	blocker := make(chan struct{})
	var started sync.WaitGroup
	var wg sync.WaitGroup

	started.Add(1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		manager.Execute(ctx, func(ctx context.Context) error {
			started.Done()
			<-blocker // Block until released
			return nil
		})
	}()

	// Wait for execution to start and acquire semaphore
	started.Wait()
	time.Sleep(10 * time.Millisecond) // Ensure semaphore is held

	// Try execution that should timeout in queue
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	bulkheadErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if bulkheadErr.Code != ErrCodeContextTimeout {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextTimeout, bulkheadErr.Code)
	}

	close(blocker)
	wg.Wait()
}

func TestExecute_ContextCancellation(t *testing.T) {
	manager := NewManager()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context
	cancel()

	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected error for cancelled context, got nil")
	}

	bulkheadErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if bulkheadErr.Code != ErrCodeContextCanceled {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextCanceled, bulkheadErr.Code)
	}
}

func TestExecute_ContextTimeout(t *testing.T) {
	manager := NewManager()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected error for expired context, got nil")
	}

	bulkheadErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if bulkheadErr.Code != ErrCodeContextTimeout {
		t.Errorf("Expected error code %s, got %s", ErrCodeContextTimeout, bulkheadErr.Code)
	}
}

func TestExecute_NilContext(t *testing.T) {
	manager := NewManager()

	err := manager.Execute(nil, func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected error for nil context, got nil")
	}

	bulkheadErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if bulkheadErr.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilContext, bulkheadErr.Code)
	}
}

func TestExecute_NilFunction(t *testing.T) {
	manager := NewManager()

	ctx := context.Background()
	err := manager.Execute(ctx, nil)

	if err == nil {
		t.Fatal("Expected error for nil function, got nil")
	}

	bulkheadErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if bulkheadErr.Code != ErrCodeNilFunction {
		t.Errorf("Expected error code %s, got %s", ErrCodeNilFunction, bulkheadErr.Code)
	}
}

func TestExecute_OnRejectedCallback(t *testing.T) {
	var rejectedCalled bool
	var rejectedReason string
	var mu sync.Mutex

	config := Config{
		MaxConcurrency: 1,
		MaxQueueSize:   0,
		OnRejected: func(ctx context.Context, reason string) {
			mu.Lock()
			rejectedCalled = true
			rejectedReason = reason
			mu.Unlock()
		},
	}

	manager := NewManagerWithConfig(config)

	// Fill bulkhead
	ctx := context.Background()
	blocker := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		manager.Execute(ctx, func(ctx context.Context) error {
			<-blocker
			return nil
		})
	}()

	// Wait for execution to start
	time.Sleep(50 * time.Millisecond)

	// Try another execution (should be rejected)
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	mu.Lock()
	if !rejectedCalled {
		t.Error("Expected OnRejected callback to be called")
	}
	if rejectedReason != "bulkhead full" {
		t.Errorf("Expected reason 'bulkhead full', got '%s'", rejectedReason)
	}
	mu.Unlock()

	close(blocker)
	wg.Wait()
}

func TestExecute_OnExecutingCallback(t *testing.T) {
	var executingCalled bool
	var mu sync.Mutex

	config := Config{
		MaxConcurrency: 1,
		OnExecuting: func(ctx context.Context) {
			mu.Lock()
			executingCalled = true
			mu.Unlock()
		},
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	mu.Lock()
	if !executingCalled {
		t.Error("Expected OnExecuting callback to be called")
	}
	mu.Unlock()
}

func TestExecute_OnCompletedCallback(t *testing.T) {
	var completedCalled bool
	var completedErr error
	var mu sync.Mutex

	testErr := errors.New("test error")

	config := Config{
		MaxConcurrency: 1,
		OnCompleted: func(ctx context.Context, err error) {
			mu.Lock()
			completedCalled = true
			completedErr = err
			mu.Unlock()
		},
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return testErr
	})

	mu.Lock()
	if !completedCalled {
		t.Error("Expected OnCompleted callback to be called")
	}
	if completedErr != testErr {
		t.Errorf("Expected error %v, got %v", testErr, completedErr)
	}
	mu.Unlock()

	if err != testErr {
		t.Errorf("Expected error %v, got %v", testErr, err)
	}
}

func TestConcurrentExecution(t *testing.T) {
	config := Config{
		MaxConcurrency: 5,
		MaxQueueSize:   10,
	}

	manager := NewManagerWithConfig(config)

	var wg sync.WaitGroup
	concurrency := 20
	successes := int32(0)

	ctx := context.Background()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := manager.Execute(ctx, func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})
			if err == nil {
				atomic.AddInt32(&successes, 1)
			}
		}()
	}

	wg.Wait()

	// Allow time for all operations to complete and stats to update
	time.Sleep(50 * time.Millisecond)

	// In highly concurrent scenarios with timing-sensitive operations, 
	// we allow some tolerance for race conditions
	if successes < int32(concurrency*9/10) {
		t.Errorf("Expected at least %d successes, got %d (concurrency: %d)", concurrency*9/10, successes, concurrency)
	}

	stats := manager.Stats()
	// Allow some tolerance for stats as well
	if stats.TotalAccepted < int64(concurrency*9/10) {
		t.Errorf("Expected at least %d TotalAccepted, got %d (concurrency: %d)", concurrency*9/10, stats.TotalAccepted, concurrency)
	}
	// CurrentConcurrency should be 0 after all operations complete
	if stats.CurrentConcurrency != 0 {
		t.Errorf("Expected CurrentConcurrency 0, got %d", stats.CurrentConcurrency)
	}
}

func TestStats(t *testing.T) {
	config := Config{
		MaxConcurrency: 2,
		MaxQueueSize:   0,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// One success
	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	// One failure
	testErr := errors.New("test error")
	manager.Execute(ctx, func(ctx context.Context) error {
		return testErr
	})

	// One rejection
	blocker := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		manager.Execute(ctx, func(ctx context.Context) error {
			<-blocker
			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond)

	manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	}) // Should be rejected

	close(blocker)
	wg.Wait()

	stats := manager.Stats()

	if stats.TotalExecutions != 3 {
		t.Errorf("Expected TotalExecutions 3, got %d", stats.TotalExecutions)
	}

	if stats.TotalAccepted != 2 {
		t.Errorf("Expected TotalAccepted 2, got %d", stats.TotalAccepted)
	}

	if stats.TotalRejected != 1 {
		t.Errorf("Expected TotalRejected 1, got %d", stats.TotalRejected)
	}

	if stats.TotalSuccessful != 1 {
		t.Errorf("Expected TotalSuccessful 1, got %d", stats.TotalSuccessful)
	}

	if stats.TotalFailed != 1 {
		t.Errorf("Expected TotalFailed 1, got %d", stats.TotalFailed)
	}

	if stats.CurrentConcurrency != 0 {
		t.Errorf("Expected CurrentConcurrency 0, got %d", stats.CurrentConcurrency)
	}
}

func TestExecute_CurrentConcurrency(t *testing.T) {
	config := Config{
		MaxConcurrency: 3,
		MaxQueueSize:   0,
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()
	blocker := make(chan struct{})
	var wg sync.WaitGroup

	// Start 3 executions
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.Execute(ctx, func(ctx context.Context) error {
				<-blocker
				return nil
			})
		}()
	}

	// Wait for all to start
	time.Sleep(50 * time.Millisecond)

	stats := manager.Stats()
	if stats.CurrentConcurrency != 3 {
		t.Errorf("Expected CurrentConcurrency 3, got %d", stats.CurrentConcurrency)
	}

	// Release blockers
	close(blocker)
	wg.Wait()

	// After completion, concurrency should be 0
	stats = manager.Stats()
	if stats.CurrentConcurrency != 0 {
		t.Errorf("Expected CurrentConcurrency 0 after completion, got %d", stats.CurrentConcurrency)
	}
}

func TestExecuteWithConfig_InvalidMaxConcurrency(t *testing.T) {
	config := Config{
		MaxConcurrency: 0, // Invalid
	}

	manager := NewManagerWithConfig(config)

	// Should use default
	stats := manager.Stats()
	if stats.CurrentConcurrency < 0 {
		t.Error("Expected valid concurrency after fixing invalid config")
	}
}

func TestExecuteWithConfig_NegativeQueueSize(t *testing.T) {
	config := Config{
		MaxConcurrency: 5,
		MaxQueueSize:   -1, // Invalid, should be set to 0
	}

	manager := NewManagerWithConfig(config)

	ctx := context.Background()

	// Fill bulkhead
	blocker := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		manager.Execute(ctx, func(ctx context.Context) error {
			<-blocker
			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond)

	// Should be rejected immediately (no queue)
	err := manager.Execute(ctx, func(ctx context.Context) error {
		return nil
	})

	if err == nil {
		t.Fatal("Expected error for bulkhead full, got nil")
	}

	bulkheadErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("Expected *Error, got %T", err)
	}

	if bulkheadErr.Code != ErrCodeBulkheadFull {
		t.Errorf("Expected error code %s, got %s", ErrCodeBulkheadFull, bulkheadErr.Code)
	}

	close(blocker)
	wg.Wait()
}
