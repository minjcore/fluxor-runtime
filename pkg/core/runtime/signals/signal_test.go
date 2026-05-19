package signals

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if len(config.Signals) == 0 {
		t.Error("DefaultConfig should have default signals")
	}

	// Verify it includes expected signals
	hasInterrupt := false
	hasSIGTERM := false
	hasSIGINT := false

	for _, sig := range config.Signals {
		if sig == os.Interrupt {
			hasInterrupt = true
		}
		if sig == syscall.SIGTERM {
			hasSIGTERM = true
		}
		if sig == syscall.SIGINT {
			hasSIGINT = true
		}
	}

	if !hasInterrupt {
		t.Error("DefaultConfig should include os.Interrupt")
	}
	if !hasSIGTERM {
		t.Error("DefaultConfig should include syscall.SIGTERM")
	}
	if !hasSIGINT {
		t.Error("DefaultConfig should include syscall.SIGINT")
	}

	if config.QueueSize == 0 {
		t.Error("DefaultConfig should have a queue size")
	}
}

func TestNewHandler(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	if handler == nil {
		t.Fatal("NewHandler() returned nil")
	}

	ch := handler.Channel()
	if ch == nil {
		t.Error("Channel() should not return nil")
	}
}

func TestHandler_Start(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	ctx := context.Background()

	// Should succeed
	err := handler.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Should fail on second start
	err = handler.Start(ctx)
	if err == nil {
		t.Error("Start() should fail when already started")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeAlreadyStarted {
		t.Errorf("Expected error code %q, got %q", ErrCodeAlreadyStarted, err.Code)
	}

	// Cleanup
	handler.Stop()
}

func TestHandler_Start_NilContext(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	err := handler.Start(nil)
	if err == nil {
		t.Error("Start(nil) should fail")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %q, got %q", ErrCodeNilContext, err.Code)
	}
}

func TestHandler_Stop(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	ctx := context.Background()

	// Start handler
	if err := handler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Stop should succeed
	err := handler.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Second stop should also succeed (idempotent)
	err = handler.Stop()
	if err != nil {
		t.Fatalf("Stop() second call error = %v", err)
	}
}

func TestHandler_Stop_NotStarted(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	// Stop should succeed even if not started
	err := handler.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestHandler_Wait(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a goroutine to cancel context after a delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// Wait should return with context error
	sig, err := handler.Wait(ctx)

	// Should get context cancellation error
	if err == nil {
		t.Error("Wait() should return error when context is cancelled")
	}
	if sig != nil {
		t.Errorf("Wait() should return nil signal on context cancellation, got %v", sig)
	}
}

func TestHandler_Wait_NilContext(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	_, err := handler.Wait(nil)
	if err == nil {
		t.Error("Wait(nil) should fail")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %q, got %q", ErrCodeNilContext, err.Code)
	}
}

func TestHandler_Channel(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	ch := handler.Channel()
	if ch == nil {
		t.Error("Channel() should not return nil")
	}
}

func TestHandler_Stats(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	stats := handler.Stats()
	if stats.SignalCount != 0 {
		t.Errorf("Initial SignalCount should be 0, got %d", stats.SignalCount)
	}
	if stats.LastSignal != nil {
		t.Error("Initial LastSignal should be nil")
	}
	if stats.IsStarted {
		t.Error("Initial IsStarted should be false")
	}
	if stats.IsStopped {
		t.Error("Initial IsStopped should be false")
	}
}

func TestWaitForSignal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a goroutine to cancel context after a delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// WaitForSignal should return with context error
	sig, err := WaitForSignal(ctx)

	// Should get context cancellation error
	if err == nil {
		t.Error("WaitForSignal() should return error when context is cancelled")
	}
	if sig != nil {
		t.Errorf("WaitForSignal() should return nil signal on context cancellation, got %v", sig)
	}
}

func TestWaitForSignalWithTimeout(t *testing.T) {
	ctx := context.Background()

	// Should timeout after 10ms
	sig, err := WaitForSignalWithTimeout(ctx, 10*time.Millisecond)

	// Should get timeout error
	if err == nil {
		t.Error("WaitForSignalWithTimeout() should return error on timeout")
	}
	if sig != nil {
		t.Errorf("WaitForSignalWithTimeout() should return nil signal on timeout, got %v", sig)
	}
}

func TestListen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := false
	callback := func(sig os.Signal) {
		called = true
	}

	config := DefaultConfig()
	err := Listen(ctx, config, callback)
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	// Cancel context to stop listening
	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	// Callback should not have been called (no signal sent)
	if called {
		t.Error("Callback should not be called without a signal")
	}
}

func TestListen_NilCallback(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()

	err := Listen(ctx, config, nil)
	if err == nil {
		t.Error("Listen() with nil callback should fail")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeNilCallback {
		t.Errorf("Expected error code %q, got %q", ErrCodeNilCallback, err.Code)
	}
}

func TestListenAsync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := make(chan bool, 1)
	callback := func(sig os.Signal) {
		called <- true
	}

	config := DefaultConfig()
	err := ListenAsync(ctx, config, callback)
	if err != nil {
		t.Fatalf("ListenAsync() error = %v", err)
	}

	// Cancel context to stop listening
	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	// Callback should not have been called (no signal sent)
	select {
	case <-called:
		t.Error("Callback should not be called without a signal")
	default:
		// Success
	}
}

func TestNewHandler_CustomSignals(t *testing.T) {
	config := Config{
		Signals: []os.Signal{syscall.SIGUSR1, syscall.SIGUSR2},
	}

	handler := NewHandler(config)
	if handler == nil {
		t.Fatal("NewHandler() with custom signals returned nil")
	}

	ctx := context.Background()
	err := handler.Start(ctx)
	if err != nil {
		t.Fatalf("Start() with custom signals error = %v", err)
	}
	defer handler.Stop()
}

func TestHandler_ContextCancellation(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	ctx, cancel := context.WithCancel(context.Background())

	err := handler.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Cancel context
	cancel()

	// Give handler time to stop
	time.Sleep(50 * time.Millisecond)

	// Handler should have stopped
	err = handler.Start(context.Background())
	if err == nil {
		t.Error("Handler should not be able to restart after context cancellation")
	}
}

func TestHandler_MultipleStartAttempts(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	ctx := context.Background()

	// First start should succeed
	err := handler.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Second start should fail
	err = handler.Start(ctx)
	if err == nil {
		t.Error("Second Start() should fail")
	}

	handler.Stop()
}

func TestHandler_StopAfterWait(t *testing.T) {
	config := DefaultConfig()
	handler := NewHandler(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start goroutine to cancel after delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	// Wait for context cancellation
	_, _ = handler.Wait(ctx)

	// Stop should still work
	err := handler.Stop()
	if err != nil {
		t.Errorf("Stop() after Wait() error = %v", err)
	}
}

func TestHandler_ContinueOnSignal(t *testing.T) {
	config := DefaultConfig()
	config.ContinueOnSignal = true
	handler := NewHandler(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := handler.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Handler should continue after stop
	time.Sleep(10 * time.Millisecond)
	err = handler.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	// Handler should be stopped
	stats := handler.Stats()
	if !stats.IsStopped {
		t.Error("Handler should be stopped")
	}
}

func TestHandler_ShutdownTimeout(t *testing.T) {
	config := DefaultConfig()
	config.ShutdownTimeout = 10 * time.Millisecond
	handler := NewHandler(config)

	ctx := context.Background()

	err := handler.Start(ctx)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Stop should succeed (no blocking callback)
	err = handler.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestHandler_SignalHistory(t *testing.T) {
	config := DefaultConfig()
	config.SignalHistory = true
	handler := NewHandler(config)

	history := handler.(*signalHandler).GetHistory()
	if history != nil {
		t.Error("Initial history should be nil")
	}
}

func TestGracefulShutdown_WithOptions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := false
	onShutdown := func() {
		called = true
	}

	err := GracefulShutdown(ctx, onShutdown,
		WithShutdownTimeout(100*time.Millisecond),
		WithContinueOnSignal(false),
	)
	if err != nil {
		t.Fatalf("GracefulShutdown() error = %v", err)
	}

	// Cancel context to stop
	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(10 * time.Millisecond)

	// Callback should not have been called (no signal sent)
	if called {
		t.Error("onShutdown should not be called without a signal")
	}
}

func TestGracefulShutdown_NilCallback(t *testing.T) {
	ctx := context.Background()

	err := GracefulShutdown(ctx, nil)
	if err == nil {
		t.Error("GracefulShutdown() with nil callback should fail")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeNilCallback {
		t.Errorf("Expected error code %q, got %q", ErrCodeNilCallback, err.Code)
	}
}

func TestNewError(t *testing.T) {
	err := NewError(ErrCodeAlreadyStarted, "test message")
	if err == nil {
		t.Fatal("NewError() returned nil")
	}

	if err.Code != ErrCodeAlreadyStarted {
		t.Errorf("Error code = %q, want %q", err.Code, ErrCodeAlreadyStarted)
	}

	if err.Message != "test message" {
		t.Errorf("Error message = %q, want %q", err.Message, "test message")
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should return a non-empty string")
	}
}
