package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// GoCMD is the main entry point for the Fluxor Stream
type GoCMD interface {
	// EventBus returns the event bus
	EventBus() EventBus

	// DeployVerticle deploys a verticle
	DeployVerticle(verticle Verticle) (string, error)

	// DeployService deploys a service (convenience method for BaseService)
	// Services are verticles, so this is equivalent to DeployVerticle but provides
	// a more semantic API for service deployment
	DeployService(service *BaseService) (string, error)

	// UndeployVerticle undeploys a verticle
	UndeployVerticle(deploymentID string) error

	// DeploymentCount returns the number of deployed verticles
	DeploymentCount() int

	// GetDeploymentState returns the state of a deployment
	GetDeploymentState(deploymentID string) (DeploymentState, error)

	// Link creates a bidirectional link with target (exit signals both ways). Requires ctx.DeploymentID() to be set.
	Link(ctx FluxorContext, targetDeploymentID string) error
	// Unlink removes the link with target.
	Unlink(ctx FluxorContext, targetDeploymentID string) error
	// Monitor creates a one-way monitor (caller receives exit when target exits). Returns ref for Demonitor.
	Monitor(ctx FluxorContext, targetDeploymentID string) (MonitorRef, error)
	// Demonitor removes the monitor.
	Demonitor(ref MonitorRef) error

	// GetDeploymentDiagnostic returns detailed diagnostic info for a deployment
	GetDeploymentDiagnostic(deploymentID string) (*DeploymentDiagnostic, error)

	// GetSystemDiagnostic returns overall system diagnostic information
	GetSystemDiagnostic() *SystemDiagnostic

	// GetAllDeploymentDiagnostics returns diagnostics for all deployments
	GetAllDeploymentDiagnostics() []DeploymentDiagnostic

	// Close closes the GoCMD instance
	Close() error

	// Context returns the root context
	Context() context.Context
}

// gocmd implements GoCMD
//
// Ownership and lifecycle:
//   - gocmd owns the EventBus instance (created in constructor, closed in Close())
//   - gocmd owns all deployment records
//   - gocmd owns the root context (rootCtx) and its cancel function
//
// Note: EventBus has a back-reference to GoCMD (circular dependency) to create
// FluxorContext for message handlers. This is intentional and doesn't cause
// memory leaks since both are cleaned up together in Close().
type gocmd struct {
	eventBus     EventBus
	linkManager  *LinkManager
	deployments  map[DeploymentID]*Deployment
	mu           sync.RWMutex
	rootCtx      context.Context    // renamed from 'ctx' for clarity: this is the root context.Context
	rootCancel   context.CancelFunc // renamed from 'cancel' for clarity
	logger       Logger
	closed       bool               // tracks if Close() has been called
	startTimeout time.Duration      // timeout for verticle Start() method
	shutdownTimeout time.Duration  // total timeout for Close(); 0 = default 35s/40s
}

// GoCMDOptions configures GoCMD construction.
type GoCMDOptions struct {
	// EventBusFactory allows swapping the default in-memory EventBus with a custom implementation
	// (e.g., a clustered EventBus backed by NATS).
	//
	// The factory is called after the GoCMD struct is created so implementations can reference GoCMD.
	EventBusFactory func(ctx context.Context, gocmd GoCMD) (EventBus, error)

	// StartTimeout is the timeout for verticle Start() method during deployment.
	// Default: 60 seconds. Must be positive.
	// If zero or negative, defaults to 60 seconds.
	StartTimeout time.Duration

	// ShutdownTimeout is the total time allowed for Close() (undeploy all verticles, then cancel, close EventBus).
	// When set (e.g. 25s for GKE Autopilot with terminationGracePeriodSeconds: 30), per-verticle undeploy
	// uses ShutdownTimeout - 5s. When zero, default 35s per verticle and 40s total are used.
	ShutdownTimeout time.Duration
}

// GoCMDOption configures GoCMD construction when passed to [NewGoCMD].
// Options are applied in order; later options override earlier ones for the same field.
type GoCMDOption func(*GoCMDOptions)

// WithEventBusFactory sets [GoCMDOptions.EventBusFactory].
func WithEventBusFactory(factory func(context.Context, GoCMD) (EventBus, error)) GoCMDOption {
	return func(o *GoCMDOptions) {
		o.EventBusFactory = factory
	}
}

// WithStartTimeout sets [GoCMDOptions.StartTimeout].
func WithStartTimeout(d time.Duration) GoCMDOption {
	return func(o *GoCMDOptions) {
		o.StartTimeout = d
	}
}

// WithShutdownTimeout sets [GoCMDOptions.ShutdownTimeout].
func WithShutdownTimeout(d time.Duration) GoCMDOption {
	return func(o *GoCMDOptions) {
		o.ShutdownTimeout = d
	}
}

func applyGoCMDOptions(opts []GoCMDOption) GoCMDOptions {
	var o GoCMDOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}

// DeploymentState and Deployment types are now in pkg/core/deployment.go
// This file uses the new Deployment type with hybrid state machine (atomic + channel)

// NewGoCMD creates a new GoCMD instance. Pass optional [GoCMDOption] values
// (e.g. [WithEventBusFactory]); construction errors panic via fail-fast (same as default path).
func NewGoCMD(ctx context.Context, opts ...GoCMDOption) GoCMD {
	gx, err := NewGoCMDWithOptions(ctx, applyGoCMDOptions(opts))
	failfast.Err(err) // Fail-fast: default construction should not fail.
	return gx
}

// NewGoCMDWithOptions creates a new GoCMD instance with configurable EventBus.
//
// The provided ctx becomes the parent of the root context. When the parent is
// cancelled, the GoCMD instance will also be cancelled.
func NewGoCMDWithOptions(ctx context.Context, opts GoCMDOptions) (GoCMD, error) {
	rootCtx, rootCancel := context.WithCancel(ctx)

	// Set default start timeout if not configured or invalid
	startTimeout := opts.StartTimeout
	if startTimeout <= 0 {
		startTimeout = 60 * time.Second
	}

	g := &gocmd{
		deployments:     make(map[DeploymentID]*Deployment),
		rootCtx:         rootCtx,
		rootCancel:      rootCancel,
		logger:          NewDefaultLogger(),
		startTimeout:    startTimeout,
		shutdownTimeout: opts.ShutdownTimeout,
	}

	if opts.EventBusFactory != nil {
		bus, err := opts.EventBusFactory(rootCtx, g)
		if err != nil {
			rootCancel()
			return nil, err
		}
		g.eventBus = bus
		g.linkManager = NewLinkManager(g.eventBus)
		return g, nil
	}

	// Default: in-memory EventBus.
	g.eventBus = NewEventBus(rootCtx, g)
	g.linkManager = NewLinkManager(g.eventBus)
	return g, nil
}

func (g *gocmd) EventBus() EventBus {
	return g.eventBus
}

func (g *gocmd) DeployVerticle(verticle Verticle) (string, error) {
	// Fail-fast: validate verticle immediately
	if err := ValidateVerticle(verticle); err != nil {
		return "", err
	}

	deploymentID := DeploymentID(generateDeploymentID())
	fluxorCtx := newFluxorContextWithDeploymentID(g.rootCtx, g, string(deploymentID))

	dep := NewDeployment(deploymentID, verticle, fluxorCtx, g.startTimeout, g.onDeploymentExit)

	// Start event loop
	dep.startEventLoop()

	// Add to map
	g.mu.Lock()
	g.deployments[deploymentID] = dep
	g.mu.Unlock()

	// Send start command
	reply := make(chan error, 1)
	select {
	case dep.cmdChan <- deploymentCmd{action: "start", reply: reply, ctx: g.rootCtx}:
		// Command sent
	case <-g.rootCtx.Done():
		return "", g.rootCtx.Err()
	}

	// Wait for start result
	select {
	case err := <-reply:
		if err != nil {
			// Remove from map on failure
			g.mu.Lock()
			delete(g.deployments, deploymentID)
			g.mu.Unlock()
			g.logger.Error(fmt.Sprintf("verticle start failed for deployment %s: %v", deploymentID, err))
			return string(deploymentID), err
		}
		return string(deploymentID), nil
	case <-time.After(g.startTimeout):
		// Clear error message explaining the timeout cause
		timeoutErr := fmt.Errorf("start timeout after %v for deployment %s: verticle Start() method did not complete within the timeout period. This usually means Start() is blocking on a long-running operation (e.g., network call, file I/O, or server startup). Consider moving blocking operations to background goroutines", g.startTimeout, deploymentID)
		g.logger.Error(fmt.Sprintf("Deployment timeout: %v", timeoutErr))
		// Remove from map on timeout
		g.mu.Lock()
		delete(g.deployments, deploymentID)
		g.mu.Unlock()
		return string(deploymentID), timeoutErr
	}
}

func (g *gocmd) DeployService(service *BaseService) (string, error) {
	// Fail-fast: validate service immediately
	if service == nil {
		return "", &EventBusError{Code: "INVALID_INPUT", Message: "service cannot be nil"}
	}

	// Services are verticles (BaseService extends BaseVerticle)
	// Deploy as verticle - services use the same deployment mechanism
	return g.DeployVerticle(service)
}

func (g *gocmd) UndeployVerticle(deploymentID string) error {
	// Fail-fast: validate deployment ID
	if deploymentID == "" {
		return &EventBusError{Code: "INVALID_DEPLOYMENT_ID", Message: "deployment ID cannot be empty"}
	}

	g.mu.RLock()
	dep, exists := g.deployments[DeploymentID(deploymentID)]
	g.mu.RUnlock()

	if !exists {
		return &EventBusError{Code: "DEPLOYMENT_NOT_FOUND", Message: "Deployment not found: " + deploymentID}
	}

	// Send undeploy command (will handle stop if needed)
	// The undeploy action in event loop will automatically stop if RUNNING
	undeployReply := make(chan error, 1)
	select {
	case dep.cmdChan <- deploymentCmd{action: "undeploy", reply: undeployReply, ctx: g.rootCtx}:
		// Command sent, wait for result
		select {
		case err := <-undeployReply:
			if err != nil {
				return err
			}
		case <-time.After(35 * time.Second):
			// Timeout (30s for stop + 5s buffer)
			// Continue to remove from map even on timeout
		}
	case <-g.rootCtx.Done():
		return g.rootCtx.Err()
	case <-time.After(5 * time.Second):
		// Channel might be closed or blocked, continue to cleanup
	}

	// Remove from map AFTER undeploy completes (or times out)
	g.mu.Lock()
	delete(g.deployments, DeploymentID(deploymentID))
	g.mu.Unlock()

	return nil
}

// DeploymentCount returns the number of deployed verticles
func (g *gocmd) DeploymentCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.deployments)
}

// GetDeploymentState returns the state of a deployment
func (g *gocmd) GetDeploymentState(deploymentID string) (DeploymentState, error) {
	if deploymentID == "" {
		return StatePending, &EventBusError{Code: "INVALID_DEPLOYMENT_ID", Message: "deployment ID cannot be empty"}
	}

	g.mu.RLock()
	dep, exists := g.deployments[DeploymentID(deploymentID)]
	g.mu.RUnlock()

	if !exists {
		return StatePending, &EventBusError{Code: "DEPLOYMENT_NOT_FOUND", Message: "Deployment not found: " + deploymentID}
	}

	return dep.State(), nil
}

// GetDeploymentDiagnostic returns detailed diagnostic info for a deployment
func (g *gocmd) GetDeploymentDiagnostic(deploymentID string) (*DeploymentDiagnostic, error) {
	if deploymentID == "" {
		return nil, &EventBusError{Code: "INVALID_DEPLOYMENT_ID", Message: "deployment ID cannot be empty"}
	}

	g.mu.RLock()
	dep, exists := g.deployments[DeploymentID(deploymentID)]
	g.mu.RUnlock()

	if !exists {
		return nil, &EventBusError{Code: "DEPLOYMENT_NOT_FOUND", Message: "Deployment not found: " + deploymentID}
	}

	return dep.GetDiagnostic(), nil
}

// GetSystemDiagnostic returns overall system diagnostic information
func (g *gocmd) GetSystemDiagnostic() *SystemDiagnostic {
	g.mu.RLock()
	deployments := make([]*Deployment, 0, len(g.deployments))
	for _, dep := range g.deployments {
		deployments = append(deployments, dep)
	}
	startTimeout := g.startTimeout
	g.mu.RUnlock()

	// Collect diagnostics
	diagnostics := make([]DeploymentDiagnostic, 0, len(deployments))
	deploymentsByState := make(map[DeploymentState]int)

	for _, dep := range deployments {
		diag := dep.GetDiagnostic()
		diagnostics = append(diagnostics, *diag)

		state := dep.State()
		deploymentsByState[state]++
	}

	// Calculate health status
	healthStatus := HealthStatusHealthy
	failedCount := deploymentsByState[StateFailed]
	stoppingCount := deploymentsByState[StateStopping]

	if failedCount > 0 {
		healthStatus = HealthStatusUnhealthy
	} else if stoppingCount > 0 {
		healthStatus = HealthStatusDegraded
	}

	return &SystemDiagnostic{
		Timestamp:          time.Now(),
		TotalDeployments:   len(deployments),
		DeploymentsByState: deploymentsByState,
		StartTimeout:       startTimeout,
		HealthStatus:       healthStatus,
		DeploymentDetails:  diagnostics,
	}
}

// GetAllDeploymentDiagnostics returns diagnostics for all deployments
func (g *gocmd) GetAllDeploymentDiagnostics() []DeploymentDiagnostic {
	g.mu.RLock()
	deployments := make([]*Deployment, 0, len(g.deployments))
	for _, dep := range g.deployments {
		deployments = append(deployments, dep)
	}
	g.mu.RUnlock()

	diagnostics := make([]DeploymentDiagnostic, 0, len(deployments))
	for _, dep := range deployments {
		diag := dep.GetDiagnostic()
		diagnostics = append(diagnostics, *diag)
	}

	return diagnostics
}

// Close gracefully shuts down the GoCMD instance.
//
// Shutdown order:
//  1. Undeploy all verticles (calls Stop on each)
//  2. Cancel the root context (signals all children to stop)
//  3. Close the EventBus (which also cancels its internal context - intentionally
//     redundant as defense-in-depth since EventBus.ctx is a child of rootCtx)
func (g *gocmd) Close() error {
	g.mu.Lock()
	// Check if already closed to avoid double-close deadlock
	if g.closed {
		g.mu.Unlock()
		return nil
	}
	g.closed = true
	deployments := make([]*Deployment, 0, len(g.deployments))
	for _, dep := range g.deployments {
		deployments = append(deployments, dep)
	}
	// Remove all from map immediately
	for id := range g.deployments {
		delete(g.deployments, id)
	}
	g.mu.Unlock()

	perDepTimeout := 35 * time.Second
	totalTimeout := 40 * time.Second
	if g.shutdownTimeout > 0 {
		totalTimeout = g.shutdownTimeout
		perDepTimeout = g.shutdownTimeout - 5*time.Second
		if perDepTimeout < time.Second {
			perDepTimeout = time.Second
		}
	}

	// Undeploy all deployments BEFORE canceling context
	// The undeploy action will automatically stop if RUNNING
	var wg sync.WaitGroup
	for _, dep := range deployments {
		wg.Add(1)
		go func(d *Deployment) {
			defer wg.Done()
			undeployReply := make(chan error, 1)
			select {
			case d.cmdChan <- deploymentCmd{action: "undeploy", reply: undeployReply, ctx: g.rootCtx}:
				select {
				case <-undeployReply:
				case <-time.After(perDepTimeout):
				}
			case <-time.After(5 * time.Second):
			}
		}(dep)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(totalTimeout):
		g.logger.Info("Some deployments did not undeploy within timeout during Close()")
	}

	// Cancel root context after undeploy attempts
	g.rootCancel()

	// Close EventBus (its internal cancel is redundant but kept for defense-in-depth)
	return g.eventBus.Close()
}

// Context returns the root context.Context for this GoCMD instance.
// This context is cancelled when Close() is called.
func (g *gocmd) Context() context.Context {
	return g.rootCtx
}

// onDeploymentExit is called by Deployment when it transitions to FAILED or STOPPED; delivers exit signals via LinkManager.
func (g *gocmd) onDeploymentExit(id DeploymentID, state DeploymentState, err error) {
	if g.linkManager != nil {
		g.linkManager.NotifyExit(string(id), state, err)
	}
}

// Link creates a bidirectional link between the caller's deployment and target. Call from a verticle with ctx.DeploymentID() set.
func (g *gocmd) Link(ctx FluxorContext, targetDeploymentID string) error {
	if g.linkManager == nil {
		return &EventBusError{Code: "LINK_UNAVAILABLE", Message: "LinkManager not initialized"}
	}
	selfID := ctx.DeploymentID()
	if selfID == "" {
		return &EventBusError{Code: "NO_DEPLOYMENT_ID", Message: "Link requires deployment context (e.g. call from within a verticle)"}
	}
	return g.linkManager.Link(selfID, targetDeploymentID)
}

// Unlink removes the link with target.
func (g *gocmd) Unlink(ctx FluxorContext, targetDeploymentID string) error {
	if g.linkManager == nil {
		return nil
	}
	selfID := ctx.DeploymentID()
	if selfID == "" {
		return &EventBusError{Code: "NO_DEPLOYMENT_ID", Message: "Unlink requires deployment context"}
	}
	return g.linkManager.Unlink(selfID, targetDeploymentID)
}

// Monitor creates a one-way monitor: caller will receive exit signals when target exits (on address core.exit.<callerID>).
func (g *gocmd) Monitor(ctx FluxorContext, targetDeploymentID string) (MonitorRef, error) {
	if g.linkManager == nil {
		return MonitorRef{}, &EventBusError{Code: "LINK_UNAVAILABLE", Message: "LinkManager not initialized"}
	}
	selfID := ctx.DeploymentID()
	if selfID == "" {
		return MonitorRef{}, &EventBusError{Code: "NO_DEPLOYMENT_ID", Message: "Monitor requires deployment context"}
	}
	return g.linkManager.Monitor(selfID, targetDeploymentID)
}

// Demonitor removes the monitor.
func (g *gocmd) Demonitor(ref MonitorRef) error {
	if g.linkManager == nil {
		return nil
	}
	return g.linkManager.Demonitor(ref)
}

// generateDeploymentID generates a unique deployment ID
func generateDeploymentID() string {
	return fmt.Sprintf("deployment.%s", generateUUID())
}
