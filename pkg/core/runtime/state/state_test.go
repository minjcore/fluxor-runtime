package state

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.InitialState != StateInitial {
		t.Errorf("Expected InitialState %s, got %s", StateInitial, config.InitialState)
	}

	if config.HistorySize != 10 {
		t.Errorf("Expected HistorySize 10, got %d", config.HistorySize)
	}

	if !config.ValidateTransitions {
		t.Error("Expected ValidateTransitions to be true")
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateInitial, "initial"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateStopping, "stopping"},
		{StateStopped, "stopped"},
		{StateError, "error"},
	}

	for _, tt := range tests {
		if tt.state.String() != tt.expected {
			t.Errorf("State.String() = %s, want %s", tt.state.String(), tt.expected)
		}
	}
}

func TestState_Valid(t *testing.T) {
	tests := []struct {
		state    State
		expected bool
	}{
		{StateInitial, true},
		{StateStarting, true},
		{StateRunning, true},
		{StateStopping, true},
		{StateStopped, true},
		{StateError, true},
		{State("invalid"), false},
	}

	for _, tt := range tests {
		if tt.state.Valid() != tt.expected {
			t.Errorf("State.Valid() = %v, want %v for state %s", tt.state.Valid(), tt.expected, tt.state)
		}
	}
}

func TestState_IsTerminal(t *testing.T) {
	tests := []struct {
		state    State
		expected bool
	}{
		{StateInitial, false},
		{StateStarting, false},
		{StateRunning, false},
		{StateStopping, false},
		{StateStopped, true},
		{StateError, true},
	}

	for _, tt := range tests {
		if tt.state.IsTerminal() != tt.expected {
			t.Errorf("State.IsTerminal() = %v, want %v for state %s", tt.state.IsTerminal(), tt.expected, tt.state)
		}
	}
}

func TestState_IsActive(t *testing.T) {
	tests := []struct {
		state    State
		expected bool
	}{
		{StateInitial, false},
		{StateStarting, true},
		{StateRunning, true},
		{StateStopping, false},
		{StateStopped, false},
		{StateError, false},
	}

	for _, tt := range tests {
		if tt.state.IsActive() != tt.expected {
			t.Errorf("State.IsActive() = %v, want %v for state %s", tt.state.IsActive(), tt.expected, tt.state)
		}
	}
}

func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.Current() != StateInitial {
		t.Errorf("Expected initial state %s, got %s", StateInitial, manager.Current())
	}
}

func TestManager_Current(t *testing.T) {
	manager := NewManager(DefaultConfig())

	if manager.Current() != StateInitial {
		t.Errorf("Expected current state %s, got %s", StateInitial, manager.Current())
	}
}

func TestManager_Transition(t *testing.T) {
	manager := NewManager(DefaultConfig())

	// Valid transition: initial -> starting
	err := manager.Transition(StateStarting)
	if err != nil {
		t.Fatalf("Transition() error = %v", err)
	}

	if manager.Current() != StateStarting {
		t.Errorf("Expected state %s, got %s", StateStarting, manager.Current())
	}

	// Invalid transition: starting -> stopped (should go through stopping first)
	err = manager.Transition(StateStopped)
	if err == nil {
		t.Error("Expected error for invalid transition")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeInvalidTransition {
		t.Errorf("Expected error code %q, got %q", ErrCodeInvalidTransition, err.Code)
	}
}

func TestManager_Transition_InvalidState(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Transition(State("invalid"))
	if err == nil {
		t.Error("Expected error for invalid state")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeInvalidState {
		t.Errorf("Expected error code %q, got %q", ErrCodeInvalidState, err.Code)
	}
}

func TestManager_Transition_ValidationDisabled(t *testing.T) {
	config := DefaultConfig()
	config.ValidateTransitions = false
	manager := NewManager(config)

	// Should allow any transition when validation is disabled
	err := manager.Transition(StateRunning)
	if err != nil {
		t.Fatalf("Transition() error = %v", err)
	}
}

func TestManager_Start(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	err := manager.Start(ctx, nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if manager.Current() != StateRunning {
		t.Errorf("Expected state %s after Start(), got %s", StateRunning, manager.Current())
	}
}

func TestManager_Start_WithCallback(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	called := false
	err := manager.Start(ctx, func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !called {
		t.Error("Expected onStart callback to be called")
	}

	if manager.Current() != StateRunning {
		t.Errorf("Expected state %s, got %s", StateRunning, manager.Current())
	}
}

func TestManager_Start_WithCallbackError(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	expectedErr := errors.New("start failed")
	err := manager.Start(ctx, func() error {
		return expectedErr
	})

	if err == nil {
		t.Error("Expected error from Start()")
	}

	if manager.Current() != StateError {
		t.Errorf("Expected state %s after error, got %s", StateError, manager.Current())
	}
}

func TestManager_Start_NilContext(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Start(nil, nil)
	if err == nil {
		t.Error("Expected error for nil context")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeNilContext {
		t.Errorf("Expected error code %q, got %q", ErrCodeNilContext, err.Code)
	}
}

func TestManager_Start_InvalidState(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	// Start once
	err := manager.Start(ctx, nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Try to start again (should fail)
	err = manager.Start(ctx, nil)
	if err == nil {
		t.Error("Expected error when starting from running state")
	}
}

func TestManager_Stop(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	// Start first
	err := manager.Start(ctx, nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Stop
	err = manager.Stop(nil)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if manager.Current() != StateStopped {
		t.Errorf("Expected state %s after Stop(), got %s", StateStopped, manager.Current())
	}
}

func TestManager_Stop_WithCallback(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	// Start first
	err := manager.Start(ctx, nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	called := false
	err = manager.Stop(func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if !called {
		t.Error("Expected onStop callback to be called")
	}

	if manager.Current() != StateStopped {
		t.Errorf("Expected state %s, got %s", StateStopped, manager.Current())
	}
}

func TestManager_Stop_WithCallbackError(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	// Start first
	err := manager.Start(ctx, nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	expectedErr := errors.New("stop failed")
	err = manager.Stop(func() error {
		return expectedErr
	})

	if err == nil {
		t.Error("Expected error from Stop()")
	}

	if manager.Current() != StateError {
		t.Errorf("Expected state %s after error, got %s", StateError, manager.Current())
	}
}

func TestManager_Stop_AlreadyStopped(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	// Start and stop
	err := manager.Start(ctx, nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	err = manager.Stop(nil)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Stop again (should be no-op)
	err = manager.Stop(nil)
	if err != nil {
		t.Errorf("Stop() on already stopped state should not error, got %v", err)
	}
}

func TestManager_Wait(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	// Start in a goroutine
	go func() {
		time.Sleep(10 * time.Millisecond)
		manager.Start(ctx, nil)
		time.Sleep(10 * time.Millisecond)
		manager.Stop(nil)
	}()

	// Wait for terminal state
	err := manager.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	if !manager.Current().IsTerminal() {
		t.Errorf("Expected terminal state, got %s", manager.Current())
	}
}

func TestManager_Wait_ContextTimeout(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := manager.Wait(ctx)
	if err == nil {
		t.Error("Expected error from Wait() with timeout")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestManager_WaitForState(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	// Wait for running state in a goroutine
	done := make(chan error, 1)
	go func() {
		time.Sleep(10 * time.Millisecond)
		manager.Start(ctx, nil)
	}()

	go func() {
		done <- manager.WaitForState(ctx, StateRunning)
	}()

	err := <-done
	if err != nil {
		t.Fatalf("WaitForState() error = %v", err)
	}

	if manager.Current() != StateRunning {
		t.Errorf("Expected state %s, got %s", StateRunning, manager.Current())
	}
}

func TestManager_WaitForState_InvalidState(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	err := manager.WaitForState(ctx, State("invalid"))
	if err == nil {
		t.Error("Expected error for invalid state")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeInvalidState {
		t.Errorf("Expected error code %q, got %q", ErrCodeInvalidState, err.Code)
	}
}

func TestManager_OnStateChange(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	var transitions []State
	var mu sync.Mutex

	manager.OnStateChange(func(from, to State) {
		mu.Lock()
		defer mu.Unlock()
		transitions = append(transitions, to)
	})

	err := manager.Start(ctx, nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	mu.Lock()
	if len(transitions) < 2 {
		t.Errorf("Expected at least 2 transitions, got %d", len(transitions))
	}
	mu.Unlock()
}

func TestManager_Stats(t *testing.T) {
	manager := NewManager(DefaultConfig())
	ctx := context.Background()

	stats := manager.Stats()
	if stats.CurrentState != StateInitial {
		t.Errorf("Expected current state %s, got %s", StateInitial, stats.CurrentState)
	}

	// Start
	err := manager.Start(ctx, nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	stats = manager.Stats()
	if stats.CurrentState != StateRunning {
		t.Errorf("Expected current state %s, got %s", StateRunning, stats.CurrentState)
	}

	if stats.TransitionCount < 2 {
		t.Errorf("Expected transition count >= 2, got %d", stats.TransitionCount)
	}

	if stats.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}

	if stats.Uptime <= 0 {
		t.Error("Expected Uptime to be positive")
	}

	// Stop
	err = manager.Stop(nil)
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	stats = manager.Stats()
	if stats.StopTime.IsZero() {
		t.Error("Expected StopTime to be set")
	}
}

func TestManager_History(t *testing.T) {
	config := DefaultConfig()
	config.HistorySize = 5
	manager := NewManager(config)
	ctx := context.Background()

	// Perform some transitions
	manager.Start(ctx, nil)
	manager.Stop(nil)

	history := manager.History()
	if len(history) == 0 {
		t.Error("Expected history to contain transitions")
	}

	// Check that history is limited
	if len(history) > config.HistorySize {
		t.Errorf("Expected history size <= %d, got %d", config.HistorySize, len(history))
	}
}

func TestManager_History_Disabled(t *testing.T) {
	config := DefaultConfig()
	config.HistorySize = 0
	manager := NewManager(config)
	ctx := context.Background()

	manager.Start(ctx, nil)
	manager.Stop(nil)

	history := manager.History()
	if history != nil {
		t.Errorf("Expected nil history when disabled, got %v", history)
	}
}

func TestManager_TransitionTimeout(t *testing.T) {
	config := DefaultConfig()
	config.TransitionTimeout = 10 * time.Millisecond
	manager := NewManager(config)
	ctx := context.Background()

	err := manager.Start(ctx, func() error {
		time.Sleep(100 * time.Millisecond) // Longer than timeout
		return nil
	})

	if err == nil {
		t.Error("Expected timeout error")
	}
	if err, ok := err.(*Error); ok && err.Code != ErrCodeTransitionTimeout {
		t.Errorf("Expected error code %q, got %q", ErrCodeTransitionTimeout, err.Code)
	}
}

func TestManager_ConfigCallbacks(t *testing.T) {
	var syncCalled bool
	var asyncCalled bool
	var mu sync.Mutex

	config := DefaultConfig()
	config.OnStateChange = func(from, to State) {
		mu.Lock()
		defer mu.Unlock()
		syncCalled = true
	}
	config.OnStateChangeAsync = func(from, to State) {
		mu.Lock()
		defer mu.Unlock()
		asyncCalled = true
	}

	manager := NewManager(config)
	ctx := context.Background()

	manager.Start(ctx, nil)

	// Give async callback time to execute
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if !syncCalled {
		t.Error("Expected OnStateChange to be called")
	}
	if !asyncCalled {
		t.Error("Expected OnStateChangeAsync to be called")
	}
	mu.Unlock()
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		from     State
		to       State
		expected bool
	}{
		{StateInitial, StateInitial, true},   // Same state
		{StateInitial, StateStarting, true},  // Valid
		{StateInitial, StateRunning, false},  // Invalid
		{StateStarting, StateRunning, true},  // Valid
		{StateRunning, StateStopping, true},  // Valid
		{StateStopping, StateStopped, true},  // Valid
		{StateStopped, StateStarting, true}, // Valid
		{StateRunning, StateStopped, false},  // Invalid (must go through stopping)
	}

	for _, tt := range tests {
		result := isValidTransition(tt.from, tt.to)
		if result != tt.expected {
			t.Errorf("isValidTransition(%s, %s) = %v, want %v", tt.from, tt.to, result, tt.expected)
		}
	}
}

func TestNewError(t *testing.T) {
	err := NewError(ErrCodeInvalidState, "test message")
	if err == nil {
		t.Fatal("NewError() returned nil")
	}

	if err.Code != ErrCodeInvalidState {
		t.Errorf("Expected error code %s, got %s", ErrCodeInvalidState, err.Code)
	}

	if err.Message != "test message" {
		t.Errorf("Expected error message 'test message', got %s", err.Message)
	}

	expectedStr := "INVALID_STATE: test message"
	if err.Error() != expectedStr {
		t.Errorf("Expected error string %s, got %s", expectedStr, err.Error())
	}
}
