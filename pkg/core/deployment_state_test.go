package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestDeploymentState_SyncVerticle tests state transitions for synchronous verticles
func TestDeploymentState_SyncVerticle(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := &testVerticle{}
	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Wait for deployment to start (channel-based, async)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		vx.mu.RLock()
		dep, exists := vx.deployments[DeploymentID(deploymentID)]
		var state DeploymentState
		if exists {
			state = dep.State()
		}
		vx.mu.RUnlock()

		if exists && state == StateRunning {
			return // Success
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Check final state
	vx.mu.RLock()
	dep, exists := vx.deployments[DeploymentID(deploymentID)]
	var state DeploymentState
	if exists {
		state = dep.State()
	}
	vx.mu.RUnlock()

	if !exists {
		t.Fatalf("deployment %s should exist", deploymentID)
	}
	if state != StateRunning {
		t.Errorf("expected state RUNNING, got %s", state)
	}
}

// TestDeploymentState_SyncVerticleFailure tests state when Start() fails
func TestDeploymentState_SyncVerticleFailure(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := &failingStartVerticle{}
	_, err := vx.DeployVerticle(verticle)
	// DeployVerticle() waits for Start() to complete via channel reply, so error is returned immediately
	if err == nil {
		t.Fatalf("DeployVerticle() should return error when Start() fails, got nil")
	}

	// Deployment should be removed from map on failure (synchronous)
	vx.mu.RLock()
	count := len(vx.deployments)
	vx.mu.RUnlock()

	if count != 0 {
		t.Errorf("expected 0 deployments after failure, got %d", count)
	}
}

// asyncTestVerticle is an async verticle for testing state transitions
type asyncTestVerticle struct {
	startCalled int32
	stopCalled  int32
	startDelay  time.Duration
	startErr    error
	stopErr     error
	startDone   chan struct{}
	stopDone    chan struct{}
}

func newAsyncTestVerticle(startDelay time.Duration) *asyncTestVerticle {
	return &asyncTestVerticle{
		startDelay: startDelay,
		startDone:  make(chan struct{}),
		stopDone:   make(chan struct{}),
	}
}

func (v *asyncTestVerticle) Start(ctx FluxorContext) error {
	return nil
}

func (v *asyncTestVerticle) Stop(ctx FluxorContext) error {
	return nil
}

func (v *asyncTestVerticle) AsyncStart(ctx FluxorContext, resultHandler func(error)) {
	atomic.AddInt32(&v.startCalled, 1)
	go func() {
		if v.startDelay > 0 {
			time.Sleep(v.startDelay)
		}
		resultHandler(v.startErr)
		close(v.startDone)
	}()
}

func (v *asyncTestVerticle) AsyncStop(ctx FluxorContext, resultHandler func(error)) {
	atomic.AddInt32(&v.stopCalled, 1)
	go func() {
		resultHandler(v.stopErr)
		close(v.stopDone)
	}()
}

// TestDeploymentState_AsyncVerticle_Pending tests that async verticle starts in PENDING state
// TODO: Re-enable when AsyncVerticle support is restored
func TestDeploymentState_AsyncVerticle_Pending(t *testing.T) {
	t.Skip("AsyncVerticle support removed - all verticles use Start() in goroutine now")
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	// Use a longer delay to observe PENDING state
	verticle := newAsyncTestVerticle(100 * time.Millisecond)

	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Immediately after deploy, state should be PENDING
	vx.mu.RLock()
	dep, exists := vx.deployments[DeploymentID(deploymentID)]
	var state DeploymentState
	if exists {
		state = dep.State()
	}
	vx.mu.RUnlock()

	if !exists {
		t.Fatalf("deployment %s should exist", deploymentID)
	}
	if state != StatePending {
		t.Errorf("expected state PENDING immediately after deploy, got %s", state)
	}

	// Wait for async start to complete
	select {
	case <-verticle.startDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("async start timed out")
	}

	// After completion, state should be RUNNING
	vx.mu.RLock()
	if exists {
		state = dep.State()
	}
	vx.mu.RUnlock()

	if state != StateRunning {
		t.Errorf("expected state RUNNING after async complete, got %s", state)
	}
}

// TestDeploymentState_AsyncVerticle_Failed tests state when AsyncStart fails
// TODO: Re-enable when AsyncVerticle support is restored
func TestDeploymentState_AsyncVerticle_Failed(t *testing.T) {
	t.Skip("AsyncVerticle support removed - all verticles use Start() in goroutine now")
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := newAsyncTestVerticle(0)
	verticle.startErr = errors.New("async start failed")

	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() should not return error for async verticle, got %v", err)
	}

	// Wait for async start callback
	select {
	case <-verticle.startDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("async start timed out")
	}

	// Small delay to let callback complete
	time.Sleep(10 * time.Millisecond)

	// After failure, deployment should be removed from map
	vx.mu.RLock()
	_, exists := vx.deployments[DeploymentID(deploymentID)]
	vx.mu.RUnlock()

	if exists {
		t.Errorf("deployment should be removed after async start failure")
	}
}

// TestDeploymentState_Undeploy_Stopping tests state transitions during undeploy
func TestDeploymentState_Undeploy_Stopping(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := &testVerticle{}
	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// DeployVerticle() waits for Start() to complete via channel reply
	// State should be RUNNING after DeployVerticle() returns

	// Undeploy should transition to STOPPING
	err = vx.UndeployVerticle(deploymentID)
	if err != nil {
		t.Errorf("UndeployVerticle() error = %v", err)
	}

	// Wait a bit for undeploy to complete
	time.Sleep(100 * time.Millisecond)

	// Deployment should be removed from map
	vx.mu.RLock()
	_, exists := vx.deployments[DeploymentID(deploymentID)]
	vx.mu.RUnlock()

	if exists {
		t.Errorf("deployment should be removed after undeploy")
	}
}

// TestDeploymentState_Undeploy_PendingRejected tests that pending deployments cannot be undeployed
func TestDeploymentState_Undeploy_PendingRejected(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	// Since Start() is now synchronous, we can't have PENDING state after DeployVerticle returns
	// This test is no longer valid - PENDING state only exists during the synchronous Start() call
	// Skip this test as it's testing an async behavior that no longer exists
	t.Skip("PENDING state no longer exists after DeployVerticle returns - Start() is synchronous")
}

// TestDeploymentState_Undeploy_DoubleUndeploy tests that double undeploy is rejected
func TestDeploymentState_Undeploy_DoubleUndeploy(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := &testVerticle{}
	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// DeployVerticle() waits for Start() to complete via channel reply
	// State should be RUNNING after DeployVerticle() returns

	// First undeploy should succeed
	err = vx.UndeployVerticle(deploymentID)
	if err != nil {
		t.Errorf("first UndeployVerticle() error = %v", err)
	}

	// Second undeploy should fail (deployment not found)
	err = vx.UndeployVerticle(deploymentID)
	if err == nil {
		t.Errorf("second UndeployVerticle() should fail")
	}
}

// TestDeploymentState_AsyncUndeploy tests async verticle undeploy state transitions
// TODO: Re-enable when AsyncVerticle support is restored
func TestDeploymentState_AsyncUndeploy(t *testing.T) {
	t.Skip("AsyncVerticle support removed - all verticles use Start() in goroutine now")
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	verticle := newAsyncTestVerticle(0)

	deploymentID, err := vx.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Wait for async start
	select {
	case <-verticle.startDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("async start timed out")
	}
	time.Sleep(10 * time.Millisecond)

	// Undeploy
	err = vx.UndeployVerticle(deploymentID)
	if err != nil {
		t.Errorf("UndeployVerticle() error = %v", err)
	}

	// Wait for async stop
	select {
	case <-verticle.stopDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("async stop timed out")
	}

	// Verify stop was called
	if atomic.LoadInt32(&verticle.stopCalled) != 1 {
		t.Errorf("AsyncStop should be called once")
	}
}

// TestDeploymentState_ConcurrentDeploy tests concurrent deployments
func TestDeploymentState_ConcurrentDeploy(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	defer vx.Close()

	const numDeployments = 10
	var wg sync.WaitGroup
	deploymentIDs := make(chan string, numDeployments)
	errs := make(chan error, numDeployments)

	for i := 0; i < numDeployments; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			verticle := &testVerticle{}
			id, err := vx.DeployVerticle(verticle)
			if err != nil {
				errs <- err
				return
			}
			deploymentIDs <- id
		}()
	}

	wg.Wait()
	close(deploymentIDs)
	close(errs)

	// Check for errors
	for err := range errs {
		t.Errorf("concurrent deploy error: %v", err)
	}

	// Collect all deployment IDs
	ids := make([]string, 0, numDeployments)
	for id := range deploymentIDs {
		ids = append(ids, id)
	}

	// DeployVerticle() waits for Start() to complete via channel reply
	// All deployments should reach RUNNING state after DeployVerticle() returns

	// Wait for all deployments to reach RUNNING state
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		allRunning := true
		for _, id := range ids {
			vx.mu.RLock()
			dep, exists := vx.deployments[DeploymentID(id)]
			var state DeploymentState
			if exists {
				state = dep.State()
			}
			vx.mu.RUnlock()

			if !exists || state != StateRunning {
				allRunning = false
				break
			}
		}
		if allRunning {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Verify all deployments exist and are RUNNING
	count := 0
	for _, id := range ids {
		count++
		vx.mu.RLock()
		dep, exists := vx.deployments[DeploymentID(id)]
		var state DeploymentState
		if exists {
			state = dep.State()
		}
		vx.mu.RUnlock()

		if !exists {
			t.Errorf("deployment %s should exist", id)
			continue
		}
		if state != StateRunning {
			t.Errorf("deployment %s should be RUNNING, got %s", id, state)
		}
	}

	if count != numDeployments {
		t.Errorf("expected %d deployments, got %d", numDeployments, count)
	}

	// Verify count
	if vx.DeploymentCount() != numDeployments {
		t.Errorf("DeploymentCount() = %d, want %d", vx.DeploymentCount(), numDeployments)
	}
}

// TestDeploymentState_Constants tests that DeploymentState constants have expected string values
func TestDeploymentState_Constants(t *testing.T) {
	if StatePending != "PENDING" {
		t.Errorf("StatePending should be 'PENDING', got %s", StatePending)
	}
	if StateDeploying != "DEPLOYING" {
		t.Errorf("StateDeploying should be 'DEPLOYING', got %s", StateDeploying)
	}
	if StateRunning != "RUNNING" {
		t.Errorf("StateRunning should be 'RUNNING', got %s", StateRunning)
	}
	if StateStopping != "STOPPING" {
		t.Errorf("StateStopping should be 'STOPPING', got %s", StateStopping)
	}
	if StateStopped != "STOPPED" {
		t.Errorf("StateStopped should be 'STOPPED', got %s", StateStopped)
	}
	if StateFailed != "FAILED" {
		t.Errorf("StateFailed should be 'FAILED', got %s", StateFailed)
	}
	if StateUndeployed != "UNDEPLOYED" {
		t.Errorf("StateUndeployed should be 'UNDEPLOYED', got %s", StateUndeployed)
	}
}

// TestIsValidTransition tests the state transition validation
func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		name     string
		from     DeploymentState
		to       DeploymentState
		expected bool
	}{
		{"PENDING to DEPLOYING", StatePending, StateDeploying, true},
		{"PENDING to FAILED", StatePending, StateFailed, true},
		{"PENDING to UNDEPLOYED", StatePending, StateUndeployed, true},
		{"DEPLOYING to RUNNING", StateDeploying, StateRunning, true},
		{"DEPLOYING to FAILED", StateDeploying, StateFailed, true},
		{"RUNNING to STOPPING", StateRunning, StateStopping, true},
		{"STOPPING to STOPPED", StateStopping, StateStopped, true},
		{"STOPPED to UNDEPLOYED", StateStopped, StateUndeployed, true},
		{"FAILED to UNDEPLOYED", StateFailed, StateUndeployed, true},
		{"PENDING to RUNNING (invalid)", StatePending, StateRunning, false},
		{"RUNNING to RUNNING (invalid)", StateRunning, StateRunning, false},
		{"STOPPED to RUNNING (invalid)", StateStopped, StateRunning, false},
		{"UNDEPLOYED to anything (invalid)", StateUndeployed, StateRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("isValidTransition(%s, %s) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

// TestDeploymentState_CloseUndeploysAll tests that Close() undeploys all verticles
func TestDeploymentState_CloseUndeploysAll(t *testing.T) {
	ctx := context.Background()
	vx := NewGoCMD(ctx).(*gocmd)
	// Note: Don't use defer here since we call Close() explicitly in the test

	verticles := make([]*testVerticle, 5)
	for i := 0; i < 5; i++ {
		verticles[i] = &testVerticle{}
		_, err := vx.DeployVerticle(verticles[i])
		if err != nil {
			t.Fatalf("DeployVerticle() error = %v", err)
		}
	}

	// DeployVerticle() waits for Start() to complete via channel reply
	// All verticles should be started after DeployVerticle() returns

	// Verify all started
	for i, v := range verticles {
		if !v.isStarted() {
			t.Errorf("verticle %d should be started", i)
		}
	}

	// Close should undeploy all
	err := vx.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify all stopped
	for i, v := range verticles {
		if !v.isStopped() {
			t.Errorf("verticle %d should be stopped after Close()", i)
		}
	}

	// Verify no deployments remain
	vx.mu.RLock()
	count := len(vx.deployments)
	vx.mu.RUnlock()

	if count != 0 {
		t.Errorf("expected 0 deployments after Close(), got %d", count)
	}
}
