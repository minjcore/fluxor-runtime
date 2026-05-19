package core

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// DeploymentID is a unique identifier for a deployment
type DeploymentID string

// DeploymentState represents the lifecycle state of a deployed verticle.
//
// This acts as a state machine with the following states and valid transitions:
//
//	State Machine Diagram:
//
//	PENDING (initial state)
//	  ├─> DEPLOYING (when Start() begins)
//	  ├─> FAILED (on failed Start())
//	  └─> UNDEPLOYED (during shutdown/Close())
//
//	DEPLOYING
//	  ├─> RUNNING (on successful Start())
//	  └─> FAILED (on failed Start())
//
//	RUNNING
//	  └─> STOPPING (on UndeployVerticle())
//
//	STOPPING
//	  └─> STOPPED (after Stop() completes)
//
//	STOPPED
//	  └─> UNDEPLOYED (final cleanup)
//
//	FAILED
//	  └─> UNDEPLOYED (final cleanup)
//
// Invalid transitions are prevented by validation checks and will panic (fail-fast).
type DeploymentState string

const (
	// StatePending is the initial state when a verticle is being deployed.
	// Transitions: DEPLOYING, FAILED, or UNDEPLOYED (shutdown)
	StatePending DeploymentState = "PENDING"

	// StateDeploying indicates the verticle Start() is in progress.
	// Transitions: RUNNING (success), FAILED (failure)
	StateDeploying DeploymentState = "DEPLOYING"

	// StateRunning indicates the verticle has successfully started and is running.
	// Transitions: STOPPING (on undeploy)
	StateRunning DeploymentState = "RUNNING"

	// StateStopping indicates the verticle is being stopped.
	// Transitions: STOPPED (after Stop() completes)
	StateStopping DeploymentState = "STOPPING"

	// StateStopped indicates the verticle has been stopped.
	// Transitions: UNDEPLOYED (final cleanup)
	StateStopped DeploymentState = "STOPPED"

	// StateFailed indicates the verticle failed to start.
	// This is a terminal state.
	// Transitions: UNDEPLOYED (final cleanup)
	StateFailed DeploymentState = "FAILED"

	// StateUndeployed indicates the deployment has been fully cleaned up.
	// This is a terminal state.
	// No further transitions allowed.
	StateUndeployed DeploymentState = "UNDEPLOYED"
)

// deploymentCmd represents a command sent to the deployment event loop
type deploymentCmd struct {
	action string // "start", "stop", "undeploy"
	err    error
	reply  chan<- error    // optional reply channel
	ctx    context.Context // for cancellation
}

// ExitNotifier is called when a deployment transitions to FAILED or STOPPED (e.g. for Link/Monitor exit signals).
type ExitNotifier func(id DeploymentID, state DeploymentState, err error)

// Deployment represents a deployed verticle instance with a state machine.
//
// Lifecycle (State Machine):
//   - Created in PENDING state (initial state)
//   - Transitions to DEPLOYING when Start() begins
//   - Transitions to RUNNING on successful Start()
//   - Transitions to FAILED on failed Start() (terminal state)
//   - Transitions to STOPPING on UndeployVerticle() or Close()
//   - Transitions to STOPPED after Stop() completes
//   - Transitions to UNDEPLOYED during final cleanup (terminal state)
//
// State transitions are handled by a single event loop goroutine (race-free).
// State reads use atomic.Value for lock-free access.
type Deployment struct {
	ID           DeploymentID
	Verticle     Verticle
	fluxorCtx    FluxorContext
	startTimeout time.Duration // timeout for Start() method
	onExit       ExitNotifier  // optional; called when transitioning to FAILED or STOPPED

	// Hybrid: Atomic for reads, channel for transitions
	state   atomic.Value       // stores DeploymentState (lock-free reads)
	cmdChan chan deploymentCmd // channel for state transitions (race-free)

	err           error
	mu            sync.RWMutex  // protects err + other fields
	done          chan struct{} // closed when event loop exits
	cmdChanClosed sync.Once     // ensures cmdChan is closed only once

	// Diagnostic tracking fields
	startTime     time.Time         // When Start() was called
	startDuration time.Duration     // How long Start() took
	stateHistory  []StateTransition // History of state changes (max 10)
	verticleType  string            // Type name of the verticle
	historyMu     sync.RWMutex      // Protects stateHistory access
}

// NewDeployment creates a new Deployment instance. onExit is optional and called when the deployment
// transitions to FAILED or STOPPED (for Link/Monitor exit signal delivery).
func NewDeployment(id DeploymentID, verticle Verticle, fluxorCtx FluxorContext, startTimeout time.Duration, onExit ExitNotifier) *Deployment {
	// Get verticle type name using reflection
	verticleType := "unknown"
	if verticle != nil {
		verticleType = fmt.Sprintf("%T", verticle)
	}

	dep := &Deployment{
		ID:           id,
		Verticle:     verticle,
		fluxorCtx:    fluxorCtx,
		startTimeout: startTimeout,
		onExit:       onExit,
		cmdChan:      make(chan deploymentCmd, 1),
		done:         make(chan struct{}),
		stateHistory: make([]StateTransition, 0, 10), // Pre-allocate for 10 transitions
		verticleType: verticleType,
	}
	dep.state.Store(StatePending) // Initialize state

	// Record initial PENDING state
	dep.recordStateTransition(StatePending, StatePending, nil)

	return dep
}

// State returns the current deployment state (lock-free read)
func (d *Deployment) State() DeploymentState {
	if v := d.state.Load(); v != nil {
		return v.(DeploymentState)
	}
	return StatePending
}

// setState sets the deployment state with validation (called from event loop only)
func (d *Deployment) setState(newState DeploymentState) {
	oldState := d.State()
	if !IsValidTransition(oldState, newState) {
		failfast.If(false, "invalid state transition: %s → %s for deployment %s",
			oldState, newState, d.ID)
	}
	d.state.Store(newState)

	// Record state transition for diagnostics
	d.recordStateTransition(oldState, newState, nil)
}

// recordStateTransition records a state transition in the history
// Limits history to last 10 transitions to prevent memory growth
func (d *Deployment) recordStateTransition(from, to DeploymentState, err error) {
	d.historyMu.Lock()
	defer d.historyMu.Unlock()

	transition := StateTransition{
		From:      from,
		To:        to,
		Timestamp: time.Now(),
	}
	if err != nil {
		transition.Error = err.Error()
	}

	// Add to history
	d.stateHistory = append(d.stateHistory, transition)

	// Limit to last 10 transitions
	if len(d.stateHistory) > 10 {
		d.stateHistory = d.stateHistory[len(d.stateHistory)-10:]
	}
}

// Error returns the deployment error (thread-safe)
func (d *Deployment) Error() error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.err
}

// GetDiagnostic returns diagnostic information for this deployment
func (d *Deployment) GetDiagnostic() *DeploymentDiagnostic {
	d.mu.RLock()
	startTime := d.startTime
	startDuration := d.startDuration
	err := d.err
	d.mu.RUnlock()

	d.historyMu.RLock()
	stateHistory := make([]StateTransition, len(d.stateHistory))
	copy(stateHistory, d.stateHistory)
	d.historyMu.RUnlock()

	diagnostic := &DeploymentDiagnostic{
		ID:           string(d.ID),
		State:        d.State(),
		VerticleType: d.verticleType,
		StartTimeout: d.startTimeout,
		StateHistory: stateHistory,
		LastUpdated:  time.Now(),
	}

	if !startTime.IsZero() {
		diagnostic.StartTime = &startTime
		diagnostic.StartDuration = startDuration
	}

	if err != nil {
		diagnostic.Error = err.Error()
	}

	return diagnostic
}

// SetError sets the deployment error (thread-safe)
func (d *Deployment) SetError(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.err = err
}

// IsValidTransition validates if a state transition is allowed
// This function is exported for testing purposes
func IsValidTransition(from, to DeploymentState) bool {
	// UNDEPLOYED is a terminal state - no transitions allowed from it
	if from == StateUndeployed {
		return false
	}

	valid := map[DeploymentState][]DeploymentState{
		StatePending:   {StateDeploying, StateFailed, StateUndeployed},
		StateDeploying: {StateRunning, StateFailed},
		StateRunning:   {StateStopping},
		StateStopping:  {StateStopped},
		StateStopped:   {StateUndeployed},
		StateFailed:    {StateUndeployed},
	}
	allowed, exists := valid[from]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// getEnvironment returns "development" or "production" from FLUXOR_ENV or ENV (default: development).
func getEnvironment() string {
	s := os.Getenv("FLUXOR_ENV")
	if s == "" {
		s = os.Getenv("ENV")
	}
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "production" || s == "prod" {
		return "production"
	}
	return "development"
}

// startEventLoop starts the deployment event loop (single goroutine handles all transitions)
func (d *Deployment) startEventLoop() {
	go func() {
		defer close(d.done)
		for cmd := range d.cmdChan {
			// Check context cancellation before processing command
			if cmd.ctx != nil {
				select {
				case <-cmd.ctx.Done():
					if cmd.reply != nil {
						cmd.reply <- cmd.ctx.Err()
					}
					continue
				default:
				}
			}

			currentState := d.State()

			switch cmd.action {
			case "start":
				if !IsValidTransition(currentState, StateDeploying) {
					if cmd.reply != nil {
						cmd.reply <- fmt.Errorf("invalid transition: %s → DEPLOYING", currentState)
					}
					continue
				}
				d.setState(StateDeploying)
				// Start verticle synchronously - DeployVerticle() waits for Start() to complete
				// Log start time for timeout diagnostics
				startTime := time.Now()

				// Record start time for diagnostics
				d.mu.Lock()
				d.startTime = startTime
				d.mu.Unlock()

				logger := NewDefaultLogger()
				env := getEnvironment()
				logger.Info(fmt.Sprintf("Starting verticle for %s, deployment %s (timeout: %v)", env, d.ID, d.startTimeout))
				err := d.Verticle.Start(d.fluxorCtx)
				elapsed := time.Since(startTime)

				// Record start duration for diagnostics
				d.mu.Lock()
				d.startDuration = elapsed
				d.mu.Unlock()

				if err != nil {
					// Record error before state transition
					d.mu.Lock()
					d.err = err
					d.mu.Unlock()

					// setState will record the transition, but we need to record the error
					d.setState(StateFailed)
					if d.onExit != nil {
						d.onExit(d.ID, StateFailed, err)
					}

					// Update the last transition with error
					d.historyMu.Lock()
					if len(d.stateHistory) > 0 {
						lastIdx := len(d.stateHistory) - 1
						d.stateHistory[lastIdx].Error = err.Error()
					}
					d.historyMu.Unlock()

					d.SetError(err)
					logger.Error(fmt.Sprintf("Verticle Start() failed for %s, deployment %s after %v: %v", env, d.ID, elapsed, err))
					if cmd.reply != nil {
						cmd.reply <- err
					}
				} else {
					d.setState(StateRunning)
					logger.Info(fmt.Sprintf("Verticle Start() completed successfully for %s, deployment %s in %v", env, d.ID, elapsed))
					if cmd.reply != nil {
						cmd.reply <- nil
					}
				}

			case "stop":
				if !IsValidTransition(currentState, StateStopping) {
					if cmd.reply != nil {
						cmd.reply <- fmt.Errorf("invalid transition: %s → STOPPING", currentState)
					}
					continue
				}
				d.setState(StateStopping)
				err := d.Verticle.Stop(d.fluxorCtx)
				if err != nil {
					d.SetError(err)
				}
				d.setState(StateStopped)
				if d.onExit != nil {
					d.onExit(d.ID, StateStopped, err)
				}
				if cmd.reply != nil {
					cmd.reply <- err
				}

			case "undeploy":
				// Handle different states for undeploy
				switch currentState {
				case StateRunning:
					// Must stop first before undeploying
					d.setState(StateStopping)
					err := d.Verticle.Stop(d.fluxorCtx)
					if err != nil {
						d.SetError(err)
					}
					d.setState(StateStopped)
					if d.onExit != nil {
						d.onExit(d.ID, StateStopped, err)
					}
					// Fall through to cleanup

				case StateStopped, StateFailed:
					// Can proceed directly to cleanup
					// These states are valid for undeploy

				case StateUndeployed:
					// Already undeployed, just return success
					if cmd.reply != nil {
						cmd.reply <- nil
					}
					return

				default:
					// Invalid state for undeploy (e.g., PENDING, DEPLOYING, STOPPING)
					if cmd.reply != nil {
						cmd.reply <- fmt.Errorf("cannot undeploy from state: %s", currentState)
					}
					continue
				}

				// Final cleanup - transition to UNDEPLOYED
				// At this point state should be STOPPED or FAILED
				if !IsValidTransition(d.State(), StateUndeployed) {
					if cmd.reply != nil {
						cmd.reply <- fmt.Errorf("invalid transition: %s → UNDEPLOYED", d.State())
					}
					continue
				}
				d.setState(StateUndeployed)
				// Safely close cmdChan using sync.Once to prevent race conditions
				d.cmdChanClosed.Do(func() {
					close(d.cmdChan)
				})
				if cmd.reply != nil {
					cmd.reply <- nil
				}
				return // exit event loop
			}
		}
	}()
}
