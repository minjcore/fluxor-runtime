package core

import (
	"fmt"
	"sync"
	"time"
)

// RestartStrategy defines how a supervisor restarts children when one fails
type RestartStrategy string

const (
	// RestartStrategyOneForOne restarts only the failed child
	RestartStrategyOneForOne RestartStrategy = "one-for-one"
	// RestartStrategyOneForAll restarts all children when one fails
	RestartStrategyOneForAll RestartStrategy = "one-for-all"
	// RestartStrategyRestForOne restarts the failed child and all children started after it
	RestartStrategyRestForOne RestartStrategy = "rest-for-one"
)

// RestartType defines when a child should be restarted
type RestartType string

const (
	// RestartPermanent child is always restarted on failure
	RestartPermanent RestartType = "permanent"
	// RestartTransient child is restarted only on abnormal termination (error)
	RestartTransient RestartType = "transient"
	// RestartTemporary child is never restarted
	RestartTemporary RestartType = "temporary"
)

// RestartConfig defines restart frequency limits
type RestartConfig struct {
	// MaxRestarts is the maximum number of restarts allowed
	MaxRestarts int
	// Within is the time window for counting restarts
	Within time.Duration
}

// ChildSpec defines a child verticle specification
type ChildSpec struct {
	// ID is a unique identifier for this child
	ID string
	// Factory creates a new verticle instance (called for initial start and restarts)
	Factory func() Verticle
	// Restart defines when this child should be restarted
	Restart RestartType
}

// SupervisorSpec defines supervisor configuration
type SupervisorSpec struct {
	// Strategy defines how children are restarted
	Strategy RestartStrategy
	// Config defines restart frequency limits
	Config RestartConfig
	// Children are the child verticles to supervise
	Children []ChildSpec
}

// SupervisedChild tracks a supervised child verticle
type SupervisedChild struct {
	Spec         ChildSpec
	DeploymentID string
	State        DeploymentState
	StartIndex   int // Order in which children were started (for rest-for-one strategy)
	mu           sync.RWMutex
}

// SupervisorVerticle supervises child verticles and restarts them on failure
type SupervisorVerticle struct {
	*BaseVerticle
	spec         SupervisorSpec
	children     map[string]*SupervisedChild
	restartCount map[string][]time.Time // Track restart timestamps for frequency limits
	mu           sync.RWMutex
	monitoring   bool
	monitorStop  chan struct{}
	stopOnce     sync.Once
}

// NewSupervisor creates a new supervisor verticle
func NewSupervisor(spec SupervisorSpec) *SupervisorVerticle {
	return &SupervisorVerticle{
		BaseVerticle: NewBaseVerticle("supervisor"),
		spec:         spec,
		children:     make(map[string]*SupervisedChild),
		restartCount: make(map[string][]time.Time),
		monitorStop:  make(chan struct{}),
	}
}

// Start starts the supervisor and all child verticles
func (sv *SupervisorVerticle) Start(ctx FluxorContext) error {
	// Start base verticle first
	if err := sv.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	sv.mu.Lock()
	defer sv.mu.Unlock()

	// Deploy all children
	for i, childSpec := range sv.spec.Children {
		child := &SupervisedChild{
			Spec:       childSpec,
			State:      StatePending,
			StartIndex: i,
		}

		// Create verticle instance using factory
		verticle := childSpec.Factory()
		if verticle == nil {
			sv.undeployChildren(ctx)
			return fmt.Errorf("factory for child %s returned nil verticle", childSpec.ID)
		}

		// Deploy child
		deploymentID, err := ctx.GoCMD().DeployVerticle(verticle)
		if err != nil {
			// If deployment fails, stop all already deployed children and return error
			sv.undeployChildren(ctx)
			return fmt.Errorf("failed to deploy child %s: %w", childSpec.ID, err)
		}

		child.DeploymentID = deploymentID
		sv.children[childSpec.ID] = child
	}

	// Start monitoring
	sv.monitoring = true
	go sv.monitorChildren(ctx)

	return nil
}

// Stop stops the supervisor and all child verticles
func (sv *SupervisorVerticle) Stop(ctx FluxorContext) error {
	sv.mu.Lock()
	sv.monitoring = false
	sv.mu.Unlock()

	// Close monitorStop channel once
	sv.stopOnce.Do(func() {
		close(sv.monitorStop)
	})

	// Wait a bit for monitoring goroutine to stop
	select {
	case <-sv.monitorStop:
	case <-time.After(1 * time.Second):
	}

	// Undeploy all children
	sv.mu.Lock()
	sv.undeployChildren(ctx)
	sv.mu.Unlock()

	// Stop base verticle
	return sv.BaseVerticle.Stop(ctx)
}

// monitorChildren monitors child deployments and restarts failed children
func (sv *SupervisorVerticle) monitorChildren(ctx FluxorContext) {
	ticker := time.NewTicker(500 * time.Millisecond) // Check every 500ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.GoCMD().Context().Done():
			return
		case <-sv.monitorStop:
			return
		case <-ticker.C:
			sv.checkChildren(ctx)
		}
	}
}

// checkChildren checks the state of all children and restarts failed ones
func (sv *SupervisorVerticle) checkChildren(ctx FluxorContext) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	if !sv.monitoring {
		return
	}

	var failedChildren []string

	// Check all children for failures
	for id, child := range sv.children {
		// Get deployment state
		// Note: We need to access deployment state, but GoCMD doesn't expose it
		// For now, we'll use a workaround: try to get state from deployment
		// This is a limitation - in a full implementation, we'd need a way to query deployment state
		state := sv.getChildState(ctx, child.DeploymentID)
		child.State = state

		if state == StateFailed {
			// Check if this child should be restarted
			if child.Spec.Restart == RestartTemporary {
				continue // Don't restart temporary children
			}

			failedChildren = append(failedChildren, id)
		}
	}

	// Handle failed children according to restart strategy
	for _, failedID := range failedChildren {
		sv.handleChildFailure(ctx, failedID)
	}
}

// handleChildFailure handles a child failure according to restart strategy
func (sv *SupervisorVerticle) handleChildFailure(ctx FluxorContext, failedID string) {
	child := sv.children[failedID]
	if child == nil {
		return
	}

	// Check restart frequency limits
	if !sv.canRestart(failedID) {
		// Restart limit exceeded - shutdown supervisor
		sv.logger().Error(fmt.Sprintf("Supervisor: restart limit exceeded for child %s, shutting down", failedID))
		// In a full implementation, we'd escalate to parent supervisor
		// For now, we just log and stop monitoring
		sv.monitoring = false
		return
	}

	// Record restart
	sv.recordRestart(failedID)

	// Determine which children to restart based on strategy
	var childrenToRestart []string
	switch sv.spec.Strategy {
	case RestartStrategyOneForOne:
		childrenToRestart = []string{failedID}
	case RestartStrategyOneForAll:
		// Restart all children
		for id := range sv.children {
			childrenToRestart = append(childrenToRestart, id)
		}
	case RestartStrategyRestForOne:
		// Restart failed child and all children started after it
		for id, c := range sv.children {
			if c.StartIndex >= child.StartIndex {
				childrenToRestart = append(childrenToRestart, id)
			}
		}
	}

	// Undeploy and restart children
	for _, id := range childrenToRestart {
		if err := sv.restartChild(ctx, id); err != nil {
			sv.logger().Error(fmt.Sprintf("Supervisor: failed to restart child %s: %v", id, err))
		}
	}
}

// restartChild undeploys and redeploys a child
func (sv *SupervisorVerticle) restartChild(ctx FluxorContext, childID string) error {
	child := sv.children[childID]
	if child == nil {
		return fmt.Errorf("child %s not found", childID)
	}

	// Undeploy old deployment
	if child.DeploymentID != "" {
		if err := ctx.GoCMD().UndeployVerticle(child.DeploymentID); err != nil {
			// Log but continue - deployment might already be undeployed
			sv.logger().Error(fmt.Sprintf("Supervisor: error undeploying child %s: %v", childID, err))
		}
	}

	// Create new verticle instance using factory
	verticle := child.Spec.Factory()
	if verticle == nil {
		return fmt.Errorf("factory for child %s returned nil verticle", childID)
	}

	// Redeploy
	deploymentID, err := ctx.GoCMD().DeployVerticle(verticle)
	if err != nil {
		return fmt.Errorf("failed to redeploy child %s: %w", childID, err)
	}

	child.DeploymentID = deploymentID
	child.State = StatePending
	return nil
}

// canRestart checks if a child can be restarted based on frequency limits
func (sv *SupervisorVerticle) canRestart(childID string) bool {
	config := sv.spec.Config
	if config.MaxRestarts == 0 {
		return true // No limit
	}

	now := time.Now()
	cutoff := now.Add(-config.Within)

	// Get restart history
	restarts := sv.restartCount[childID]
	if len(restarts) == 0 {
		return true
	}

	// Count restarts within the time window
	count := 0
	for _, t := range restarts {
		if t.After(cutoff) {
			count++
		}
	}

	return count < config.MaxRestarts
}

// recordRestart records a restart timestamp
func (sv *SupervisorVerticle) recordRestart(childID string) {
	now := time.Now()
	sv.restartCount[childID] = append(sv.restartCount[childID], now)

	// Clean up old restart timestamps (older than the time window)
	config := sv.spec.Config
	cutoff := now.Add(-config.Within)
	restarts := sv.restartCount[childID]
	validRestarts := restarts[:0]
	for _, t := range restarts {
		if t.After(cutoff) {
			validRestarts = append(validRestarts, t)
		}
	}
	sv.restartCount[childID] = validRestarts
}

// undeployChildren undeploys all children
// This is called from Stop() which has context access
func (sv *SupervisorVerticle) undeployChildren(ctx FluxorContext) {
	for _, child := range sv.children {
		if child.DeploymentID != "" {
			// Best effort - log errors but continue
			if err := ctx.GoCMD().UndeployVerticle(child.DeploymentID); err != nil {
				sv.logger().Error(fmt.Sprintf("Supervisor: error undeploying child %s: %v", child.Spec.ID, err))
			}
		}
	}
}

// getChildState gets the deployment state of a child
func (sv *SupervisorVerticle) getChildState(ctx FluxorContext, deploymentID string) DeploymentState {
	state, err := ctx.GoCMD().GetDeploymentState(deploymentID)
	if err != nil {
		// If deployment not found, consider it FAILED
		return StateFailed
	}
	return state
}

// logger returns the logger instance
func (sv *SupervisorVerticle) logger() Logger {
	return NewDefaultLogger()
}
