// Package state provides runtime state management for components.
//
// The state package provides utilities for managing the lifecycle state of
// runtime components. It supports state transitions, validation, callbacks,
// and state history tracking. All operations are thread-safe and designed
// for concurrent use.
//
// State Types:
//
// The package defines the following states:
//   - StateInitial: The initial/uninitialized state
//   - StateStarting: The state when starting
//   - StateRunning: The active/running state
//   - StateStopping: The state when stopping
//   - StateStopped: The stopped state
//   - StateError: An error state
//
// Basic Usage:
//
//	// Create a state manager with default configuration
//	manager := state.NewManager(state.DefaultConfig())
//
//	// Start the component
//	ctx := context.Background()
//	err := manager.Start(ctx, func() error {
//	    // Initialization logic
//	    return nil
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check current state
//	current := manager.Current()
//	log.Printf("Current state: %s", current)
//
//	// Stop the component
//	err = manager.Stop(func() error {
//	    // Cleanup logic
//	    return nil
//	})
//
// Advanced Usage with Callbacks:
//
//	// Create manager with state change callbacks
//	config := state.Config{
//	    OnStateChange: func(from, to state.State) {
//	        log.Printf("State changed from %s to %s", from, to)
//	    },
//	    OnStateChangeAsync: func(from, to state.State) {
//	        // Non-blocking async callback
//	        metrics.RecordStateChange(from, to)
//	    },
//	}
//
//	manager := state.NewManager(config)
//
//	// Register additional callbacks
//	manager.OnStateChange(func(from, to state.State) {
//	    // Custom logic
//	})
//
// State Transitions:
//
//	// Manual state transition
//	err := manager.Transition(state.StateStarting)
//	if err != nil {
//	    log.Printf("Transition failed: %v", err)
//	}
//
//	// Transitions are validated by default
//	// Invalid transitions return an error
//
// Waiting for States:
//
//	// Wait for a terminal state (stopped or error)
//	ctx := context.Background()
//	err := manager.Wait(ctx)
//	if err != nil {
//	    log.Printf("Wait failed: %v", err)
//	}
//
//	// Wait for a specific state
//	err = manager.WaitForState(ctx, state.StateRunning)
//	if err != nil {
//	    log.Printf("Wait failed: %v", err)
//	}
//
// State Statistics:
//
//	// Get statistics about state transitions
//	stats := manager.Stats()
//	log.Printf("Current state: %s", stats.CurrentState)
//	log.Printf("Transition count: %d", stats.TransitionCount)
//	log.Printf("Uptime: %v", stats.Uptime)
//	log.Printf("Last transition: %s -> %s at %v",
//	    stats.LastTransition.From,
//	    stats.LastTransition.To,
//	    stats.LastTransition.Timestamp)
//
// State History:
//
//	// Enable state history tracking
//	config := state.DefaultConfig()
//	config.HistorySize = 20 // Keep last 20 transitions
//	manager := state.NewManager(config)
//
//	// Get transition history
//	history := manager.History()
//	for _, transition := range history {
//	    log.Printf("%s -> %s at %v",
//	        transition.From,
//	        transition.To,
//	        transition.Timestamp)
//	}
//
// Timeout Configuration:
//
//	// Configure transition timeouts
//	config := state.DefaultConfig()
//	config.TransitionTimeout = 30 * time.Second
//	manager := state.NewManager(config)
//
//	// Start with timeout protection
//	err := manager.Start(ctx, func() error {
//	    // This will timeout if it takes longer than 30 seconds
//	    return performLongOperation()
//	})
//
// Disabling Transition Validation:
//
//	// Allow any state transition (use with caution)
//	config := state.DefaultConfig()
//	config.ValidateTransitions = false
//	manager := state.NewManager(config)
//
//	// Now any transition is allowed
//	manager.Transition(state.StateRunning)
//
// Thread Safety:
//
// All Manager methods are thread-safe and can be called concurrently.
// State reads use atomic operations for lock-free access, while state
// transitions are serialized to ensure consistency.
//
// The Manager interface provides:
//   - Current() - Get the current state (lock-free)
//   - Transition() - Manually transition to a new state
//   - Start() - Start the component (initial -> starting -> running)
//   - Stop() - Stop the component (running -> stopping -> stopped)
//   - Wait() - Wait for a terminal state
//   - WaitForState() - Wait for a specific state
//   - OnStateChange() - Register state change callbacks
//   - Stats() - Get state transition statistics
//   - History() - Get state transition history
package state
