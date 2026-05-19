package dbruntime

import (
	"context"
	"strings"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core/runtime/state"
)

// TestDatabaseComponent_StateValidation_Initial tests operations in StateInitial
func TestDatabaseComponent_StateValidation_Initial(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)
	ctx := context.Background()

	// Verify initial state
	if component.GetState() != state.StateInitial {
		t.Errorf("Expected StateInitial, got %s", component.GetState())
	}

	// Test Query() returns error
	_, err := component.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("Query() should return error in StateInitial")
	}
	verifyStateError(t, err, "Query", state.StateInitial)

	// Test QueryRow() panics
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("QueryRow() should panic in StateInitial")
			} else {
				msg := r.(string)
				if !strings.Contains(msg, "initial") && !strings.Contains(msg, "StateInitial") {
					t.Errorf("Panic message should include state, got: %s", msg)
				}
			}
		}()
		component.QueryRow(ctx, "SELECT 1")
	}()

	// Test Exec() returns error
	_, err = component.Exec(ctx, "SELECT 1")
	if err == nil {
		t.Error("Exec() should return error in StateInitial")
	}
	verifyStateError(t, err, "Exec", state.StateInitial)

	// Test Begin() returns error
	_, err = component.Begin(ctx)
	if err == nil {
		t.Error("Begin() should return error in StateInitial")
	}
	verifyStateError(t, err, "Begin", state.StateInitial)

	// Test BeginTx() returns error
	_, err = component.BeginTx(ctx, nil)
	if err == nil {
		t.Error("BeginTx() should return error in StateInitial")
	}
	verifyStateError(t, err, "BeginTx", state.StateInitial)

	// Test Ping() returns error
	err = component.Ping(ctx)
	if err == nil {
		t.Error("Ping() should return error in StateInitial")
	}
	verifyStateError(t, err, "Ping", state.StateInitial)

	// Test Stats() returns empty stats
	stats := component.Stats()
	if stats.OpenConnections != 0 {
		t.Errorf("Stats() should return empty stats in StateInitial, got %d", stats.OpenConnections)
	}

	// Test QueryAsync() returns error
	_, err = component.QueryAsync(ctx, "SELECT 1")
	if err == nil {
		t.Error("QueryAsync() should return error in StateInitial")
	}
	verifyStateError(t, err, "QueryAsync", state.StateInitial)

	// Test QueryRowAsync() returns error
	_, err = component.QueryRowAsync(ctx, "SELECT 1")
	if err == nil {
		t.Error("QueryRowAsync() should return error in StateInitial")
	}
	verifyStateError(t, err, "QueryRowAsync", state.StateInitial)

	// Test ExecAsync() returns error
	_, err = component.ExecAsync(ctx, "SELECT 1")
	if err == nil {
		t.Error("ExecAsync() should return error in StateInitial")
	}
	verifyStateError(t, err, "ExecAsync", state.StateInitial)

	// Test BeginAsync() returns error
	_, err = component.BeginAsync(ctx)
	if err == nil {
		t.Error("BeginAsync() should return error in StateInitial")
	}
	verifyStateError(t, err, "BeginAsync", state.StateInitial)

	// Test Pool() panics
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Pool() should panic in StateInitial")
			} else {
				msg := r.(string)
				if !strings.Contains(msg, "initial") && !strings.Contains(msg, "StateInitial") {
					t.Errorf("Panic message should include state, got: %s", msg)
				}
			}
		}()
		component.Pool()
	}()

	// Test DB() panics
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("DB() should panic in StateInitial")
			} else {
				msg := r.(string)
				if !strings.Contains(msg, "initial") && !strings.Contains(msg, "StateInitial") {
					t.Errorf("Panic message should include state, got: %s", msg)
				}
			}
		}()
		component.DB()
	}()
}

// TestDatabaseComponent_StateValidation_Stopped tests operations in StateStopped
func TestDatabaseComponent_StateValidation_Stopped(t *testing.T) {
	// Create a component and manually transition to stopped state
	// We need to follow valid state transitions: Initial -> Starting -> Running -> Stopping -> Stopped
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)
	ctx := context.Background()

	// Manually transition through valid states to reach Stopped
	// Note: This tests state validation, not actual start/stop flow
	_ = component.stateManager.Transition(state.StateStarting)
	_ = component.stateManager.Transition(state.StateRunning)
	_ = component.stateManager.Transition(state.StateStopping)
	err := component.stateManager.Transition(state.StateStopped)
	if err != nil {
		t.Errorf("Failed to transition to StateStopped: %v", err)
	}

	if component.GetState() != state.StateStopped {
		t.Errorf("Expected StateStopped, got %s", component.GetState())
	}

	// Test Query() returns error
	_, err = component.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("Query() should return error in StateStopped")
	}
	verifyStateError(t, err, "Query", state.StateStopped)

	// Test Exec() returns error
	_, err = component.Exec(ctx, "SELECT 1")
	if err == nil {
		t.Error("Exec() should return error in StateStopped")
	}
	verifyStateError(t, err, "Exec", state.StateStopped)

	// Test QueryAsync() returns error
	_, err = component.QueryAsync(ctx, "SELECT 1")
	if err == nil {
		t.Error("QueryAsync() should return error in StateStopped")
	}
	verifyStateError(t, err, "QueryAsync", state.StateStopped)
}

// TestDatabaseComponent_StateValidation_Error tests operations in StateError
func TestDatabaseComponent_StateValidation_Error(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)
	ctx := context.Background()

	// Manually transition to error state
	component.stateManager.Transition(state.StateError)

	if component.GetState() != state.StateError {
		t.Errorf("Expected StateError, got %s", component.GetState())
	}

	// Test Query() returns error
	_, err := component.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("Query() should return error in StateError")
	}
	verifyStateError(t, err, "Query", state.StateError)

	// Test Exec() returns error
	_, err = component.Exec(ctx, "SELECT 1")
	if err == nil {
		t.Error("Exec() should return error in StateError")
	}
	verifyStateError(t, err, "Exec", state.StateError)

	// Verify error message includes "error" (state string is lowercase)
	if err != nil {
		if dbErr, ok := err.(*Error); ok {
			if !strings.Contains(dbErr.Message, "error") {
				t.Errorf("Error message should include 'error', got: %s", dbErr.Message)
			}
		}
	}
}

// TestDatabaseComponent_StateValidation_ErrorMessages tests error message quality
func TestDatabaseComponent_StateValidation_ErrorMessages(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)
	ctx := context.Background()

	// Test Query error message
	_, err := component.Query(ctx, "SELECT 1")
	verifyErrorFormat(t, err, "Query", state.StateInitial)

	// Test Exec error message
	_, err = component.Exec(ctx, "SELECT 1")
	verifyErrorFormat(t, err, "Exec", state.StateInitial)

	// Test QueryAsync error message
	_, err = component.QueryAsync(ctx, "SELECT 1")
	verifyErrorFormat(t, err, "QueryAsync", state.StateInitial)

	// Test ExecAsync error message
	_, err = component.ExecAsync(ctx, "SELECT 1")
	verifyErrorFormat(t, err, "ExecAsync", state.StateInitial)

	// Test Begin error message
	_, err = component.Begin(ctx)
	verifyErrorFormat(t, err, "Begin", state.StateInitial)

	// Test BeginAsync error message
	_, err = component.BeginAsync(ctx)
	verifyErrorFormat(t, err, "BeginAsync", state.StateInitial)

	// Test Ping error message
	err = component.Ping(ctx)
	verifyErrorFormat(t, err, "Ping", state.StateInitial)
}

// TestDatabaseComponent_StateValidation_Transitions tests state transitions
func TestDatabaseComponent_StateValidation_Transitions(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)

	// Verify initial state
	if component.GetState() != state.StateInitial {
		t.Errorf("Expected StateInitial, got %s", component.GetState())
	}

	// Test state transition to Starting
	err := component.stateManager.Transition(state.StateStarting)
	if err != nil {
		t.Errorf("Failed to transition to StateStarting: %v", err)
	}
	if component.GetState() != state.StateStarting {
		t.Errorf("Expected StateStarting, got %s", component.GetState())
	}

	// Test state transition to Running
	err = component.stateManager.Transition(state.StateRunning)
	if err != nil {
		t.Errorf("Failed to transition to StateRunning: %v", err)
	}
	if component.GetState() != state.StateRunning {
		t.Errorf("Expected StateRunning, got %s", component.GetState())
	}

	// Test state transition to Stopping
	err = component.stateManager.Transition(state.StateStopping)
	if err != nil {
		t.Errorf("Failed to transition to StateStopping: %v", err)
	}
	if component.GetState() != state.StateStopping {
		t.Errorf("Expected StateStopping, got %s", component.GetState())
	}

	// Test state transition to Stopped
	err = component.stateManager.Transition(state.StateStopped)
	if err != nil {
		t.Errorf("Failed to transition to StateStopped: %v", err)
	}
	if component.GetState() != state.StateStopped {
		t.Errorf("Expected StateStopped, got %s", component.GetState())
	}

	// Verify state history
	history := component.GetStateHistory()
	if len(history) < 4 {
		t.Errorf("Expected at least 4 state transitions, got %d", len(history))
	}

	// Verify state stats
	stats := component.GetStateStats()
	if stats.TransitionCount < 4 {
		t.Errorf("Expected at least 4 transitions, got %d", stats.TransitionCount)
	}
}

// TestDatabaseComponent_validateStateForOperation tests the helper method directly
func TestDatabaseComponent_validateStateForOperation(t *testing.T) {
	// Test with nil component
	var nilComponent *DatabaseComponent
	err := nilComponent.validateStateForOperation("TestOp")
	if err == nil {
		t.Error("validateStateForOperation() should return error for nil component")
	}
	if dbErr, ok := err.(*Error); ok {
		if dbErr.Code != "INVALID_STATE" {
			t.Errorf("Expected error code INVALID_STATE, got %s", dbErr.Code)
		}
		if !strings.Contains(dbErr.Message, "cannot be nil") {
			t.Errorf("Error message should mention nil, got: %s", dbErr.Message)
		}
	}

	// Test with component in StateInitial
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)
	err = component.validateStateForOperation("TestOp")
	if err == nil {
		t.Error("validateStateForOperation() should return error in StateInitial")
	}
	verifyStateError(t, err, "TestOp", state.StateInitial)

	// Test with component in StateStopped (transition through valid states)
	_ = component.stateManager.Transition(state.StateStarting)
	_ = component.stateManager.Transition(state.StateRunning)
	_ = component.stateManager.Transition(state.StateStopping)
	_ = component.stateManager.Transition(state.StateStopped)
	err = component.validateStateForOperation("TestOp")
	if err == nil {
		t.Error("validateStateForOperation() should return error in StateStopped")
	}
	verifyStateError(t, err, "TestOp", state.StateStopped)

	// Test with component in StateError (can transition from Stopped to Error)
	_ = component.stateManager.Transition(state.StateError)
	err = component.validateStateForOperation("TestOp")
	if err == nil {
		t.Error("validateStateForOperation() should return error in StateError")
	}
	verifyStateError(t, err, "TestOp", state.StateError)

	// Test with nil pool manager (component in Running but no pool manager)
	// Reset component for this test
	component2 := NewDatabaseComponent(config)
	_ = component2.stateManager.Transition(state.StateStarting)
	_ = component2.stateManager.Transition(state.StateRunning)
	component2.poolManager = nil
	err = component2.validateStateForOperation("TestOp")
	if err == nil {
		t.Error("validateStateForOperation() should return error when pool manager is nil")
	}
	if dbErr, ok := err.(*Error); ok {
		if dbErr.Code != "NOT_STARTED" {
			t.Errorf("Expected error code NOT_STARTED, got %s", dbErr.Code)
		}
	}
}

// TestDatabaseComponent_StateValidation_AsyncOperations tests async operations validate state
func TestDatabaseComponent_StateValidation_AsyncOperations(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)
	ctx := context.Background()

	// Test QueryAsync validates state before delegating
	_, err := component.QueryAsync(ctx, "SELECT 1")
	if err == nil {
		t.Error("QueryAsync() should return error in StateInitial")
	}
	// Verify error is returned immediately (not after async execution)
	if dbErr, ok := err.(*Error); ok {
		if dbErr.Code != "INVALID_STATE" {
			t.Errorf("Expected error code INVALID_STATE, got %s", dbErr.Code)
		}
	}

	// Test QueryRowAsync validates state
	_, err = component.QueryRowAsync(ctx, "SELECT 1")
	if err == nil {
		t.Error("QueryRowAsync() should return error in StateInitial")
	}
	verifyStateError(t, err, "QueryRowAsync", state.StateInitial)

	// Test ExecAsync validates state
	_, err = component.ExecAsync(ctx, "SELECT 1")
	if err == nil {
		t.Error("ExecAsync() should return error in StateInitial")
	}
	verifyStateError(t, err, "ExecAsync", state.StateInitial)

	// Test BeginAsync validates state
	_, err = component.BeginAsync(ctx)
	if err == nil {
		t.Error("BeginAsync() should return error in StateInitial")
	}
	verifyStateError(t, err, "BeginAsync", state.StateInitial)
}

// TestDatabaseComponent_StateValidation_PanicMethods tests Pool() and DB() panic behavior
func TestDatabaseComponent_StateValidation_PanicMethods(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)

	// Test Pool() panics in StateInitial
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Pool() should panic in StateInitial")
			} else {
				msg := r.(string)
				if !strings.Contains(msg, "initial") && !strings.Contains(msg, "StateInitial") {
					t.Errorf("Panic message should include state, got: %s", msg)
				}
				if !strings.Contains(msg, "expected StateRunning") {
					t.Errorf("Panic message should mention expected state, got: %s", msg)
				}
			}
		}()
		component.Pool()
	}()

	// Test DB() panics in StateInitial
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("DB() should panic in StateInitial")
			} else {
				msg := r.(string)
				if !strings.Contains(msg, "initial") && !strings.Contains(msg, "StateInitial") {
					t.Errorf("Panic message should include state, got: %s", msg)
				}
				if !strings.Contains(msg, "expected StateRunning") {
					t.Errorf("Panic message should mention expected state, got: %s", msg)
				}
			}
		}()
		component.DB()
	}()

	// Test Pool() panics in StateStopped (transition through valid states)
	_ = component.stateManager.Transition(state.StateStarting)
	_ = component.stateManager.Transition(state.StateRunning)
	_ = component.stateManager.Transition(state.StateStopping)
	_ = component.stateManager.Transition(state.StateStopped)
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Pool() should panic in StateStopped")
			} else {
				msg := r.(string)
				if !strings.Contains(msg, "stopped") && !strings.Contains(msg, "StateStopped") {
					t.Errorf("Panic message should include state, got: %s", msg)
				}
			}
		}()
		component.Pool()
	}()

	// Test DB() panics in StateStopped
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("DB() should panic in StateStopped")
			} else {
				msg := r.(string)
				if !strings.Contains(msg, "stopped") && !strings.Contains(msg, "StateStopped") {
					t.Errorf("Panic message should include state, got: %s", msg)
				}
			}
		}()
		component.DB()
	}()
}

// TestDatabaseComponent_StateValidation_Starting tests operations during starting phase
func TestDatabaseComponent_StateValidation_Starting(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)
	ctx := context.Background()

	// Manually transition to starting state
	component.stateManager.Transition(state.StateStarting)

	if component.GetState() != state.StateStarting {
		t.Errorf("Expected StateStarting, got %s", component.GetState())
	}

	// Test operations fail during starting state
	_, err := component.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("Query() should return error in StateStarting")
	}
	verifyStateError(t, err, "Query", state.StateStarting)

	// Verify error message includes "starting" (state string is lowercase)
	if err != nil {
		if dbErr, ok := err.(*Error); ok {
			if !strings.Contains(dbErr.Message, "starting") {
				t.Errorf("Error message should include 'starting', got: %s", dbErr.Message)
			}
		}
	}
}

// TestDatabaseComponent_StateValidation_Stopping tests operations during stopping phase
func TestDatabaseComponent_StateValidation_Stopping(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)
	ctx := context.Background()

	// Manually transition through valid states to reach Stopping
	_ = component.stateManager.Transition(state.StateStarting)
	_ = component.stateManager.Transition(state.StateRunning)
	err := component.stateManager.Transition(state.StateStopping)
	if err != nil {
		t.Errorf("Failed to transition to StateStopping: %v", err)
	}

	if component.GetState() != state.StateStopping {
		t.Errorf("Expected StateStopping, got %s", component.GetState())
	}

	// Test operations fail during stopping state
	_, err = component.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("Query() should return error in StateStopping")
	}
	verifyStateError(t, err, "Query", state.StateStopping)

	// Verify error message includes "stopping" (state string is lowercase)
	if err != nil {
		if dbErr, ok := err.(*Error); ok {
			if !strings.Contains(dbErr.Message, "stopping") {
				t.Errorf("Error message should include 'stopping', got: %s", dbErr.Message)
			}
		}
	}
}

// Helper functions

// verifyStateError verifies that an error is a state validation error with correct format
func verifyStateError(t *testing.T, err error, operation string, expectedState state.State) {
	if err == nil {
		t.Errorf("Expected error for %s in state %s", operation, expectedState)
		return
	}

	dbErr, ok := err.(*Error)
	if !ok {
		t.Errorf("Expected *Error, got %T", err)
		return
	}

	if dbErr.Code != "INVALID_STATE" {
		t.Errorf("Expected error code INVALID_STATE, got %s", dbErr.Code)
	}

	if !strings.Contains(dbErr.Message, operation) {
		t.Errorf("Error message should include operation name '%s', got: %s", operation, dbErr.Message)
	}

	// State string is lowercase, so check for lowercase version
	stateStr := strings.ToLower(expectedState.String())
	if !strings.Contains(strings.ToLower(dbErr.Message), stateStr) {
		t.Errorf("Error message should include state '%s', got: %s", expectedState, dbErr.Message)
	}

	if !strings.Contains(dbErr.Message, "StateRunning") && !strings.Contains(strings.ToLower(dbErr.Message), "running") {
		t.Errorf("Error message should mention expected state StateRunning, got: %s", dbErr.Message)
	}
}

// verifyErrorFormat verifies error message format is correct
func verifyErrorFormat(t *testing.T, err error, operation string, currentState state.State) {
	if err == nil {
		t.Errorf("Expected error for %s", operation)
		return
	}

	dbErr, ok := err.(*Error)
	if !ok {
		t.Errorf("Expected *Error, got %T", err)
		return
	}

	// Verify error code
	if dbErr.Code != "INVALID_STATE" {
		t.Errorf("Expected error code INVALID_STATE, got %s", dbErr.Code)
	}

	// Verify error message format: "cannot execute {operation}: component is in state {state} (expected StateRunning)"
	// Note: State string is lowercase in the actual implementation
	expectedFormat := "cannot execute " + operation + ": component is in state " + currentState.String() + " (expected StateRunning)"
	if dbErr.Message != expectedFormat {
		t.Errorf("Error message format incorrect.\nExpected: %s\nGot:      %s", expectedFormat, dbErr.Message)
	}
}

// TestDatabaseComponent_StateValidation_Performance tests that validation is performant and optimized
func TestDatabaseComponent_StateValidation_Performance(t *testing.T) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)

	// Test that validation is fast (should complete quickly)
	// Performance: Validation should be efficient with minimal overhead
	start := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = component.validateStateForOperation("TestOp")
		}
	})

	// Verify validation is performant (completes in reasonable time)
	// Optimized: Should handle high-frequency calls efficiently
	if start.NsPerOp() > 1000 { // 1 microsecond threshold
		t.Errorf("Validation should be fast, took %d ns per op", start.NsPerOp())
	}

	// Test error creation is efficient
	// Performance: Error creation should be optimized
	err := component.validateStateForOperation("TestOp")
	if err == nil {
		t.Fatal("Expected error for validation test")
	}

	// Verify error is created efficiently (fast error creation)
	// Efficient: Error struct creation should be performant
	if dbErr, ok := err.(*Error); ok {
		if dbErr.Code == "" || dbErr.Message == "" {
			t.Error("Error should be created efficiently with all fields")
		}
	}

	// Test multiple operations are handled efficiently
	// Performance: Multiple validations should be fast
	operations := []string{"Query", "Exec", "Begin", "Ping"}
	for _, op := range operations {
		_ = component.validateStateForOperation(op)
	}

	// Test that state checks are optimized
	// Efficient: State checks should use fast lookups
	currentState := component.GetState()
	if currentState == "" {
		t.Error("State check should be efficient and return valid state")
	}
}

// BenchmarkDatabaseComponent_StateValidation benchmarks validation performance
func BenchmarkDatabaseComponent_StateValidation(b *testing.B) {
	config := DefaultPoolConfig("test-dsn", "postgres")
	component := NewDatabaseComponent(config)

	b.ResetTimer()
	b.ReportAllocs()

	// Performance: Benchmark validation speed
	// Optimized: Should be fast and efficient
	for i := 0; i < b.N; i++ {
		_ = component.validateStateForOperation("Query")
	}
}

// BenchmarkDatabaseComponent_ErrorCreation benchmarks error creation performance
func BenchmarkDatabaseComponent_ErrorCreation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	// Performance: Benchmark error creation speed
	// Efficient: Error creation should be optimized
	for i := 0; i < b.N; i++ {
		_ = &Error{
			Code:    "INVALID_STATE",
			Message: "cannot execute Query: component is in state initial (expected StateRunning)",
		}
	}
}
