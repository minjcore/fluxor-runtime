package state

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// State represents the runtime state.
type State string

const (
	// StateInitial represents the initial/uninitialized state.
	StateInitial State = "initial"

	// StateStarting represents the state when starting.
	StateStarting State = "starting"

	// StateRunning represents the active/running state.
	StateRunning State = "running"

	// StateStopping represents the state when stopping.
	StateStopping State = "stopping"

	// StateStopped represents the stopped state.
	StateStopped State = "stopped"

	// StateError represents an error state.
	StateError State = "error"
)

// String returns the string representation of the state.
func (s State) String() string {
	return string(s)
}

// Valid returns true if the state is valid.
func (s State) Valid() bool {
	return s == StateInitial || s == StateStarting || s == StateRunning ||
		s == StateStopping || s == StateStopped || s == StateError
}

// IsTerminal returns true if the state is a terminal state (stopped or error).
func (s State) IsTerminal() bool {
	return s == StateStopped || s == StateError
}

// IsActive returns true if the state is an active state (starting or running).
func (s State) IsActive() bool {
	return s == StateStarting || s == StateRunning
}

// Manager provides state management for runtime components.
type Manager interface {
	// Current returns the current state.
	Current() State

	// Transition attempts to transition to a new state.
	// Returns an error if the transition is invalid.
	Transition(to State) error

	// Start transitions to the starting state, then to running.
	// If onStart is provided, it's called during the starting phase.
	Start(ctx context.Context, onStart func() error) error

	// Stop transitions to the stopping state, then to stopped.
	// If onStop is provided, it's called during the stopping phase.
	Stop(onStop func() error) error

	// Wait waits for the state to reach a terminal state (stopped or error).
	Wait(ctx context.Context) error

	// WaitForState waits for the state to reach the specified state.
	WaitForState(ctx context.Context, target State) error

	// OnStateChange registers a callback that's called when the state changes.
	OnStateChange(callback func(from, to State))

	// Stats returns statistics about state transitions.
	Stats() Stats

	// History returns the state transition history.
	History() []Transition
}

// Config configures state management behavior.
type Config struct {
	// InitialState is the initial state. Defaults to StateInitial.
	InitialState State

	// OnStateChange is called when the state changes.
	OnStateChange func(from, to State)

	// OnStateChangeAsync is called asynchronously when the state changes.
	OnStateChangeAsync func(from, to State)

	// TransitionTimeout is the maximum time to wait for a transition to complete.
	// Zero means no timeout.
	TransitionTimeout time.Duration

	// HistorySize is the maximum number of state transitions to keep in history.
	// Zero means no history. Defaults to 10.
	HistorySize int

	// ValidateTransitions enables validation of state transitions.
	// If true, only valid transitions are allowed.
	ValidateTransitions bool
}

// DefaultConfig returns the default state manager configuration.
func DefaultConfig() Config {
	return Config{
		InitialState:        StateInitial,
		HistorySize:         10,
		ValidateTransitions: true,
	}
}

// Transition represents a state transition.
type Transition struct {
	From      State
	To        State
	Timestamp time.Time
	Error     error
}

// Stats contains statistics about state transitions.
type Stats struct {
	// CurrentState is the current state.
	CurrentState State

	// TransitionCount is the total number of transitions.
	TransitionCount int64

	// LastTransition is the last transition that occurred.
	LastTransition Transition

	// StartTime is when the manager was started (entered starting state).
	StartTime time.Time

	// StopTime is when the manager was stopped (entered stopped state).
	StopTime time.Time

	// Uptime is the duration the manager has been in running state.
	Uptime time.Duration
}

// stateManager implements the Manager interface.
type stateManager struct {
	config Config

	// Current state (atomic for lock-free reads)
	currentState atomic.Value // stores State

	// State transition management
	mu              sync.RWMutex
	transitionChan  chan State
	callbacks       []func(from, to State)
	history         []Transition
	historyMu       sync.RWMutex

	// Statistics
	transitionCount int64
	startTime       atomic.Value // stores time.Time
	stopTime        atomic.Value // stores time.Time
	lastTransition  atomic.Value // stores Transition

	// Context for state operations
	ctx    context.Context
	cancel context.CancelFunc

	// Wait groups for state synchronization
	waiters map[State][]chan struct{}
	waitMu  sync.Mutex
}

// NewManager creates a new state manager with the given configuration.
func NewManager(config Config) Manager {
	if config.InitialState == "" {
		config.InitialState = DefaultConfig().InitialState
	}
	// Respect HistorySize as-is: if 0, no history; if > 0, use it.
	// DefaultConfig() already sets HistorySize to 10, so users who use
	// DefaultConfig() get history. Users who explicitly set it to 0 get no history.

	ctx, cancel := context.WithCancel(context.Background())

	manager := &stateManager{
		config:         config,
		transitionChan: make(chan State, 1),
		callbacks:      make([]func(from, to State), 0),
		history:        make([]Transition, 0, config.HistorySize),
		waiters:        make(map[State][]chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Set initial state
	manager.currentState.Store(config.InitialState)

	// Record initial state transition only if history is enabled
	if config.HistorySize > 0 {
		manager.recordTransition(config.InitialState, config.InitialState, nil)
	}

	return manager
}

// Current returns the current state (lock-free read).
func (m *stateManager) Current() State {
	if v := m.currentState.Load(); v != nil {
		return v.(State)
	}
	return StateInitial
}

// Transition attempts to transition to a new state.
func (m *stateManager) Transition(to State) error {
	if !to.Valid() {
		return NewError(ErrCodeInvalidState, fmt.Sprintf("invalid state: %s", to))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	current := m.Current()

	// Validate transition if enabled
	if m.config.ValidateTransitions {
		if !isValidTransition(current, to) {
			return NewError(ErrCodeInvalidTransition,
				fmt.Sprintf("invalid transition from %s to %s", current, to))
		}
	}

	// Perform the transition
	return m.transitionLocked(current, to, nil)
}

// transitionLocked performs a state transition (must be called with lock held).
func (m *stateManager) transitionLocked(from, to State, err error) error {
	// Update state atomically
	m.currentState.Store(to)

	// Record transition
	transition := Transition{
		From:      from,
		To:        to,
		Timestamp: time.Now(),
		Error:     err,
	}

	atomic.AddInt64(&m.transitionCount, 1)
	m.lastTransition.Store(transition)

	// Update timestamps
	if to == StateStarting {
		m.startTime.Store(transition.Timestamp)
	} else if to == StateStopped {
		m.stopTime.Store(transition.Timestamp)
	}

	// Record in history
	m.recordTransition(from, to, err)

	// Notify callbacks
	m.notifyCallbacks(from, to)

	// Notify waiters
	m.notifyWaiters(to)

	return nil
}

// recordTransition records a state transition in history.
func (m *stateManager) recordTransition(from, to State, err error) {
	if m.config.HistorySize <= 0 {
		return
	}

	m.historyMu.Lock()
	defer m.historyMu.Unlock()

	transition := Transition{
		From:      from,
		To:        to,
		Timestamp: time.Now(),
		Error:     err,
	}

	m.history = append(m.history, transition)
	if len(m.history) > m.config.HistorySize {
		m.history = m.history[1:]
	}
}

// notifyCallbacks notifies all registered callbacks of a state change.
func (m *stateManager) notifyCallbacks(from, to State) {
	// Call synchronous callbacks
	if m.config.OnStateChange != nil {
		m.config.OnStateChange(from, to)
	}

	// Call registered callbacks
	for _, callback := range m.callbacks {
		if callback != nil {
			callback(from, to)
		}
	}

	// Call asynchronous callback
	if m.config.OnStateChangeAsync != nil {
		go m.config.OnStateChangeAsync(from, to)
	}
}

// notifyWaiters notifies all waiters waiting for a specific state.
func (m *stateManager) notifyWaiters(state State) {
	m.waitMu.Lock()
	defer m.waitMu.Unlock()

	waiters := m.waiters[state]
	for _, ch := range waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	delete(m.waiters, state)
}

// Start transitions to the starting state, then to running.
func (m *stateManager) Start(ctx context.Context, onStart func() error) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	current := m.Current()

	// Validate we can start
	if current != StateInitial && current != StateStopped {
		return NewError(ErrCodeInvalidTransition,
			fmt.Sprintf("cannot start from state %s", current))
	}

	// Transition to starting
	if err := m.transitionLocked(current, StateStarting, nil); err != nil {
		return err
	}

	// Call onStart if provided
	if onStart != nil {
		var startErr error
		if m.config.TransitionTimeout > 0 {
			done := make(chan error, 1)
			go func() {
				done <- onStart()
			}()
			select {
			case startErr = <-done:
			case <-time.After(m.config.TransitionTimeout):
				startErr = NewError(ErrCodeTransitionTimeout,
					fmt.Sprintf("start timeout after %v", m.config.TransitionTimeout))
			}
		} else {
			startErr = onStart()
		}

		if startErr != nil {
			m.transitionLocked(StateStarting, StateError, startErr)
			return startErr
		}
	}

	// Transition to running
	return m.transitionLocked(StateStarting, StateRunning, nil)
}

// Stop transitions to the stopping state, then to stopped.
func (m *stateManager) Stop(onStop func() error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	current := m.Current()

	// Validate we can stop
	if current == StateStopped || current == StateError {
		return nil // Already stopped
	}

	if current != StateRunning && current != StateStarting {
		return NewError(ErrCodeInvalidTransition,
			fmt.Sprintf("cannot stop from state %s", current))
	}

	// Transition to stopping
	if err := m.transitionLocked(current, StateStopping, nil); err != nil {
		return err
	}

	// Call onStop if provided
	if onStop != nil {
		var stopErr error
		if m.config.TransitionTimeout > 0 {
			done := make(chan error, 1)
			go func() {
				done <- onStop()
			}()
			select {
			case stopErr = <-done:
			case <-time.After(m.config.TransitionTimeout):
				stopErr = NewError(ErrCodeTransitionTimeout,
					fmt.Sprintf("stop timeout after %v", m.config.TransitionTimeout))
			}
		} else {
			stopErr = onStop()
		}

		if stopErr != nil {
			m.transitionLocked(StateStopping, StateError, stopErr)
			return stopErr
		}
	}

	// Transition to stopped
	return m.transitionLocked(StateStopping, StateStopped, nil)
}

// Wait waits for the state to reach a terminal state (stopped or error).
func (m *stateManager) Wait(ctx context.Context) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	current := m.Current()
	if current.IsTerminal() {
		return nil
	}

	// Wait for either stopped or error state
	done := make(chan struct{})
	m.waitMu.Lock()
	m.waiters[StateStopped] = append(m.waiters[StateStopped], done)
	m.waiters[StateError] = append(m.waiters[StateError], done)
	m.waitMu.Unlock()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// Clean up the channel to prevent memory leak
		m.waitMu.Lock()
		m.waiters[StateStopped] = removeChannel(m.waiters[StateStopped], done)
		m.waiters[StateError] = removeChannel(m.waiters[StateError], done)
		m.waitMu.Unlock()
		return ctx.Err()
	}
}

// WaitForState waits for the state to reach the specified state.
func (m *stateManager) WaitForState(ctx context.Context, target State) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	if !target.Valid() {
		return NewError(ErrCodeInvalidState, fmt.Sprintf("invalid target state: %s", target))
	}

	current := m.Current()
	if current == target {
		return nil
	}

	done := make(chan struct{})
	m.waitMu.Lock()
	m.waiters[target] = append(m.waiters[target], done)
	m.waitMu.Unlock()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// Clean up the channel to prevent memory leak
		m.waitMu.Lock()
		m.waiters[target] = removeChannel(m.waiters[target], done)
		m.waitMu.Unlock()
		return ctx.Err()
	}
}

// OnStateChange registers a callback that's called when the state changes.
func (m *stateManager) OnStateChange(callback func(from, to State)) {
	if callback == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.callbacks = append(m.callbacks, callback)
}

// Stats returns statistics about state transitions.
func (m *stateManager) Stats() Stats {
	current := m.Current()

	var lastTransition Transition
	if v := m.lastTransition.Load(); v != nil {
		lastTransition = v.(Transition)
	}

	var startTime time.Time
	if v := m.startTime.Load(); v != nil {
		startTime = v.(time.Time)
	}

	var stopTime time.Time
	if v := m.stopTime.Load(); v != nil {
		stopTime = v.(time.Time)
	}

	var uptime time.Duration
	if current == StateRunning && !startTime.IsZero() {
		uptime = time.Since(startTime)
	} else if !startTime.IsZero() && !stopTime.IsZero() {
		uptime = stopTime.Sub(startTime)
	}

	return Stats{
		CurrentState:    current,
		TransitionCount: atomic.LoadInt64(&m.transitionCount),
		LastTransition:  lastTransition,
		StartTime:       startTime,
		StopTime:        stopTime,
		Uptime:          uptime,
	}
}

// History returns the state transition history.
func (m *stateManager) History() []Transition {
	m.historyMu.RLock()
	defer m.historyMu.RUnlock()

	if len(m.history) == 0 {
		return nil
	}

	history := make([]Transition, len(m.history))
	copy(history, m.history)
	return history
}

// isValidTransition checks if a transition from one state to another is valid.
func isValidTransition(from, to State) bool {
	// Same state is always valid
	if from == to {
		return true
	}

	// Define valid transitions
	validTransitions := map[State][]State{
		StateInitial: {StateStarting, StateError},
		StateStarting: {StateRunning, StateError, StateStopping},
		StateRunning:  {StateStopping, StateError},
		StateStopping: {StateStopped, StateError},
		StateStopped:  {StateStarting, StateError},
		StateError:    {StateInitial, StateStarting},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedState := range allowed {
		if allowedState == to {
			return true
		}
	}

	return false
}

// removeChannel removes a channel from a slice.
func removeChannel(channels []chan struct{}, ch chan struct{}) []chan struct{} {
	result := make([]chan struct{}, 0, len(channels))
	for _, c := range channels {
		if c != ch {
			result = append(result, c)
		}
	}
	return result
}
