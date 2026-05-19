package core

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type testVerticle struct {
	mu      sync.RWMutex
	started bool
	stopped bool
}

func (v *testVerticle) Start(ctx FluxorContext) error {
	v.mu.Lock()
	v.started = true
	v.mu.Unlock()
	return nil
}

func (v *testVerticle) Stop(ctx FluxorContext) error {
	v.mu.Lock()
	v.stopped = true
	v.mu.Unlock()
	return nil
}

func (v *testVerticle) isStarted() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.started
}

func (v *testVerticle) isStopped() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.stopped
}

func TestGoCMD_DeployVerticle(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	// Test fail-fast: nil verticle
	_, err := gocmd.DeployVerticle(nil)
	if err == nil {
		t.Error("DeployVerticle() with nil verticle should fail")
	}

	// Test valid deployment
	verticle := &testVerticle{}
	deploymentID, err := gocmd.DeployVerticle(verticle)
	if err != nil {
		t.Errorf("DeployVerticle() error = %v", err)
	}
	if deploymentID == "" {
		t.Error("DeployVerticle() returned empty deployment ID")
	}

	// Start() is now synchronous, so verticle should be started immediately
	if !verticle.isStarted() {
		t.Error("Verticle should be started")
	}
}

func TestGoCMD_UndeployVerticle(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	// Test fail-fast: empty deployment ID
	err := gocmd.UndeployVerticle("")
	if err == nil {
		t.Error("UndeployVerticle() with empty ID should fail")
	}

	// Test fail-fast: non-existent deployment
	err = gocmd.UndeployVerticle("non-existent")
	if err == nil {
		t.Error("UndeployVerticle() with non-existent ID should fail")
	}

	// Deploy and undeploy
	verticle := &testVerticle{}
	deploymentID, err := gocmd.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Start() is now synchronous, so verticle should be started immediately
	if !verticle.isStarted() {
		t.Error("Verticle should be started")
	}

	err = gocmd.UndeployVerticle(deploymentID)
	if err != nil {
		t.Errorf("UndeployVerticle() error = %v", err)
	}

	// Wait for async stop to complete (Stop() is still async in goroutine)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if verticle.isStopped() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !verticle.isStopped() {
		t.Error("Verticle should be stopped")
	}
}

func TestGoCMD_EventBus(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	eb := gocmd.EventBus()
	if eb == nil {
		t.Error("EventBus() should not return nil")
	}
}

func TestGoCMD_GetDeploymentState(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	// Test fail-fast: empty deployment ID
	state, err := gocmd.GetDeploymentState("")
	if err == nil {
		t.Error("GetDeploymentState() with empty ID should fail")
	}
	if state != StatePending {
		t.Errorf("GetDeploymentState() with empty ID should return StatePending, got %v", state)
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "INVALID_DEPLOYMENT_ID" {
				t.Errorf("Error code = %v, want 'INVALID_DEPLOYMENT_ID'", e.Code)
			}
		} else {
			t.Errorf("Expected EventBusError, got %T", err)
		}
	}

	// Test fail-fast: non-existent deployment
	state, err = gocmd.GetDeploymentState("non-existent")
	if err == nil {
		t.Error("GetDeploymentState() with non-existent ID should fail")
	}
	if state != StatePending {
		t.Errorf("GetDeploymentState() with non-existent ID should return StatePending, got %v", state)
	}
	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "DEPLOYMENT_NOT_FOUND" {
				t.Errorf("Error code = %v, want 'DEPLOYMENT_NOT_FOUND'", e.Code)
			}
		} else {
			t.Errorf("Expected EventBusError, got %T", err)
		}
	}

	// Test valid deployment
	verticle := &testVerticle{}
	deploymentID, err := gocmd.DeployVerticle(verticle)
	if err != nil {
		t.Fatalf("DeployVerticle() error = %v", err)
	}

	// Wait a bit for deployment to start
	time.Sleep(50 * time.Millisecond)

	state, err = gocmd.GetDeploymentState(deploymentID)
	if err != nil {
		t.Errorf("GetDeploymentState() error = %v", err)
	}
	if state != StateRunning {
		t.Errorf("Expected deployment state RUNNING, got %v", state)
	}
}

func TestNewGoCMDWithOptions_FailFast_EventBusFactoryErrorCancelsContext(t *testing.T) {
	parent := context.Background()

	wantErr := errors.New("factory failed")
	var factoryCtx context.Context

	vx, err := NewGoCMDWithOptions(parent, GoCMDOptions{
		EventBusFactory: func(ctx context.Context, _ GoCMD) (EventBus, error) {
			factoryCtx = ctx
			return nil, wantErr
		},
	})
	if err == nil {
		t.Fatalf("NewGoCMDWithOptions() expected error, got nil (vx=%v)", vx)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("NewGoCMDWithOptions() error = %v, want %v", err, wantErr)
	}
	if factoryCtx == nil {
		t.Fatalf("expected factory to be invoked and capture ctx")
	}

	select {
	case <-factoryCtx.Done():
		// ok: fail-fast cleanup should cancel internal ctx
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("expected internal context to be cancelled on factory error")
	}
}

type failingStartVerticle struct{}

func (v *failingStartVerticle) Start(ctx FluxorContext) error { return errors.New("start failed") }
func (v *failingStartVerticle) Stop(ctx FluxorContext) error  { return nil }

func TestGoCMD_DeployVerticle_FailFast_StartError(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	// Start() is now synchronous, so DeployVerticle() waits for Start() to complete
	// and returns an error immediately if Start() fails
	_, err := gocmd.DeployVerticle(&failingStartVerticle{})
	if err == nil {
		t.Fatalf("DeployVerticle() should return error when Start() fails, got nil")
	}
	if err.Error() != "start failed" {
		t.Errorf("DeployVerticle() error = %v, want 'start failed'", err)
	}

	// Deployment should be removed from map on failure (synchronous)
	if gocmd.DeploymentCount() != 0 {
		t.Errorf("expected 0 deployments after failure, got %d", gocmd.DeploymentCount())
	}
}

func TestNewGoCMD_GoCMDOption_StartTimeout(t *testing.T) {
	ctx := context.Background()
	g := NewGoCMD(ctx, WithStartTimeout(7*time.Second)).(*gocmd)
	defer g.Close()
	if g.startTimeout != 7*time.Second {
		t.Fatalf("startTimeout = %v, want 7s", g.startTimeout)
	}
}

func TestNewGoCMD_GoCMDOption_LastStartTimeoutWins(t *testing.T) {
	ctx := context.Background()
	g := NewGoCMD(ctx, WithStartTimeout(time.Second), WithStartTimeout(8*time.Second)).(*gocmd)
	defer g.Close()
	if g.startTimeout != 8*time.Second {
		t.Fatalf("startTimeout = %v, want 8s", g.startTimeout)
	}
}

func TestNewGoCMD_GoCMDOption_EventBusFactory(t *testing.T) {
	ctx := context.Background()
	var calls int
	g := NewGoCMD(ctx, WithEventBusFactory(func(c context.Context, cmd GoCMD) (EventBus, error) {
		calls++
		return NewEventBus(c, cmd), nil
	}))
	defer g.Close()
	if calls != 1 {
		t.Fatalf("EventBusFactory calls = %d, want 1", calls)
	}
	if g.EventBus() == nil {
		t.Fatal("EventBus() is nil")
	}
}
