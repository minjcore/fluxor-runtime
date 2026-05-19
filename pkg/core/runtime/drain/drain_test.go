package drain

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.DefaultTimeout != 30*time.Second {
		t.Errorf("Expected DefaultTimeout 30s, got %v", config.DefaultTimeout)
	}

	if config.Parallel {
		t.Error("Expected Parallel to be false")
	}
}

func TestNewDrainer(t *testing.T) {
	config := DefaultConfig()
	drainer := NewDrainer(config)

	if drainer == nil {
		t.Fatal("NewDrainer returned nil")
	}

	stats := drainer.Stats()
	if stats.IsDraining {
		t.Error("Expected IsDraining to be false")
	}
}

func TestRegister(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	mockDrainable := &mockDrainable{}
	err := drainer.Register("test", mockDrainable)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	components := drainer.Components()
	if len(components) != 1 || components[0] != "test" {
		t.Errorf("Expected 1 component named 'test', got %v", components)
	}
}

func TestRegister_EmptyName(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	mockDrainable := &mockDrainable{}
	err := drainer.Register("", mockDrainable)
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestRegister_NilDrainable(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	err := drainer.Register("test", nil)
	if err == nil {
		t.Error("Expected error for nil drainable")
	}
}

func TestRegister_Duplicate(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	mockDrainable := &mockDrainable{}
	err := drainer.Register("test", mockDrainable)
	if err != nil {
		t.Fatalf("First Register failed: %v", err)
	}

	err = drainer.Register("test", mockDrainable)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
}

func TestUnregister(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	mockDrainable := &mockDrainable{}
	drainer.Register("test", mockDrainable)

	if len(drainer.Components()) != 1 {
		t.Fatal("Expected 1 component before unregister")
	}

	drainer.Unregister("test")

	if len(drainer.Components()) != 0 {
		t.Error("Expected 0 components after unregister")
	}
}

func TestDrain_Success(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	mockDrainable := &mockDrainable{
		drainFunc: func(ctx context.Context) error {
			return nil
		},
	}

	drainer.Register("test", mockDrainable)

	ctx := context.Background()
	err := drainer.Drain(ctx, "test")
	if err != nil {
		t.Fatalf("Drain failed: %v", err)
	}

	stats := drainer.Stats()
	if stats.TotalDrained != 1 {
		t.Errorf("Expected TotalDrained 1, got %d", stats.TotalDrained)
	}
}

func TestDrain_NotFound(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	ctx := context.Background()
	err := drainer.Drain(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent component")
	}
}

func TestDrain_Timeout(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	mockDrainable := &mockDrainable{
		drainFunc: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}

	drainer.Register("test", mockDrainable)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := drainer.Drain(ctx, "test")
	if err == nil {
		t.Error("Expected timeout error")
	}

	stats := drainer.Stats()
	if stats.TotalTimeouts != 1 {
		t.Errorf("Expected TotalTimeouts 1, got %d", stats.TotalTimeouts)
	}
}

func TestDrainAll_Sequential(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	var order []string
	var mu sync.Mutex

	mock1 := &mockDrainable{
		drainFunc: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "1")
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	mock2 := &mockDrainable{
		drainFunc: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "2")
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	drainer.Register("1", mock1)
	drainer.Register("2", mock2)

	ctx := context.Background()
	err := drainer.DrainAll(ctx)
	if err != nil {
		t.Fatalf("DrainAll failed: %v", err)
	}

	if len(order) != 2 || order[0] != "1" || order[1] != "2" {
		t.Errorf("Expected sequential order [1, 2], got %v", order)
	}

	stats := drainer.Stats()
	if stats.TotalDrained != 2 {
		t.Errorf("Expected TotalDrained 2, got %d", stats.TotalDrained)
	}
}

func TestDrainAll_Parallel(t *testing.T) {
	config := DefaultConfig()
	config.Parallel = true
	drainer := NewDrainer(config)

	var mu sync.Mutex
	var startTimes []time.Time

	mock1 := &mockDrainable{
		drainFunc: func(ctx context.Context) error {
			mu.Lock()
			startTimes = append(startTimes, time.Now())
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	}

	mock2 := &mockDrainable{
		drainFunc: func(ctx context.Context) error {
			mu.Lock()
			startTimes = append(startTimes, time.Now())
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	}

	drainer.Register("1", mock1)
	drainer.Register("2", mock2)

	ctx := context.Background()
	err := drainer.DrainAll(ctx)
	if err != nil {
		t.Fatalf("DrainAll failed: %v", err)
	}

	if len(startTimes) != 2 {
		t.Fatalf("Expected 2 start times, got %d", len(startTimes))
	}

	// In parallel mode, start times should be very close
	diff := startTimes[1].Sub(startTimes[0])
	if diff > 20*time.Millisecond {
		t.Errorf("Expected parallel execution, but start times differ by %v", diff)
	}

	stats := drainer.Stats()
	if stats.TotalDrained != 2 {
		t.Errorf("Expected TotalDrained 2, got %d", stats.TotalDrained)
	}
}

func TestDrainAll_WithErrors(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	mock1 := &mockDrainable{
		drainFunc: func(ctx context.Context) error {
			return nil
		},
	}

	mock2 := &mockDrainable{
		drainFunc: func(ctx context.Context) error {
			return errors.New("drain failed")
		},
	}

	drainer.Register("1", mock1)
	drainer.Register("2", mock2)

	ctx := context.Background()
	err := drainer.DrainAll(ctx)
	if err == nil {
		t.Error("Expected error from DrainAll")
	}

	stats := drainer.Stats()
	if stats.TotalDrained != 1 {
		t.Errorf("Expected TotalDrained 1, got %d", stats.TotalDrained)
	}
	if stats.TotalFailed != 1 {
		t.Errorf("Expected TotalFailed 1, got %d", stats.TotalFailed)
	}
}

func TestDrainAll_NilContext(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	err := drainer.DrainAll(nil)
	if err == nil {
		t.Error("Expected error for nil context")
	}
}

func TestDrainAll_AlreadyDraining(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	mockDrainable := &mockDrainable{
		drainFunc: func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}

	drainer.Register("test", mockDrainable)

	ctx := context.Background()

	// Start first drain in goroutine
	var firstErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		firstErr = drainer.DrainAll(ctx)
	}()

	// Wait a bit to ensure first drain started
	time.Sleep(10 * time.Millisecond)

	// Try to start second drain
	err := drainer.DrainAll(ctx)
	if err == nil {
		t.Error("Expected error for concurrent drain")
	}

	wg.Wait()
	_ = firstErr // ignore first error for test
}

func TestStats(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	stats := drainer.Stats()
	if stats.TotalDrained != 0 {
		t.Errorf("Expected TotalDrained 0, got %d", stats.TotalDrained)
	}
	if stats.TotalFailed != 0 {
		t.Errorf("Expected TotalFailed 0, got %d", stats.TotalFailed)
	}
	if stats.IsDraining {
		t.Error("Expected IsDraining to be false")
	}
}

func TestReset(t *testing.T) {
	drainer := NewDrainer(DefaultConfig())

	mockDrainable := &mockDrainable{
		drainFunc: func(ctx context.Context) error {
			return nil
		},
	}

	drainer.Register("test", mockDrainable)
	drainer.Drain(context.Background(), "test")

	stats := drainer.Stats()
	if stats.TotalDrained == 0 {
		t.Fatal("Expected TotalDrained > 0 before reset")
	}

	drainer.Reset()

	stats = drainer.Stats()
	if stats.TotalDrained != 0 {
		t.Errorf("Expected TotalDrained 0 after reset, got %d", stats.TotalDrained)
	}
}

// mockDrainable is a test implementation of Drainable.
type mockDrainable struct {
	drainFunc func(ctx context.Context) error
}

func (m *mockDrainable) Drain(ctx context.Context) error {
	if m.drainFunc != nil {
		return m.drainFunc(ctx)
	}
	return nil
}
