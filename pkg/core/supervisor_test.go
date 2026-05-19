package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

// testWorkerVerticle is a simple test verticle
type testWorkerVerticle struct {
	*BaseVerticle
	id      string
	mu      sync.RWMutex
	started bool
	stopped bool
}

func newTestWorkerVerticle(id string) *testWorkerVerticle {
	return &testWorkerVerticle{
		BaseVerticle: NewBaseVerticle("worker-" + id),
		id:           id,
	}
}

func (v *testWorkerVerticle) Start(ctx FluxorContext) error {
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}
	v.mu.Lock()
	v.started = true
	v.mu.Unlock()
	return nil
}

func (v *testWorkerVerticle) Stop(ctx FluxorContext) error {
	v.mu.Lock()
	v.stopped = true
	v.mu.Unlock()
	return v.BaseVerticle.Stop(ctx)
}

func (v *testWorkerVerticle) isStarted() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.started
}

func (v *testWorkerVerticle) isStopped() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.stopped
}

// failingWorkerVerticle is a worker that fails to start
type failingWorkerVerticle struct {
	*BaseVerticle
	failError error
}

func newFailingWorkerVerticle(err error) *failingWorkerVerticle {
	return &failingWorkerVerticle{
		BaseVerticle: NewBaseVerticle("failing-worker"),
		failError:    err,
	}
}

func (v *failingWorkerVerticle) Start(ctx FluxorContext) error {
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}
	if v.failError != nil {
		return v.failError
	}
	return nil
}

func (v *failingWorkerVerticle) Stop(ctx FluxorContext) error {
	return v.BaseVerticle.Stop(ctx)
}

func TestNewSupervisor(t *testing.T) {
	spec := SupervisorSpec{
		Strategy: RestartStrategyOneForOne,
		Config: RestartConfig{
			MaxRestarts: 5,
			Within:      10 * time.Second,
		},
		Children: []ChildSpec{
			{
				ID: "worker1",
				Factory: func() Verticle {
					return newTestWorkerVerticle("1")
				},
				Restart: RestartPermanent,
			},
		},
	}

	supervisor := NewSupervisor(spec)
	if supervisor == nil {
		t.Fatal("NewSupervisor() returned nil")
	}

	if supervisor.Name() != "supervisor" {
		t.Errorf("Name() = %v, want 'supervisor'", supervisor.Name())
	}

	if supervisor.IsStarted() {
		t.Error("NewSupervisor() should create supervisor that is not started")
	}
}

func TestSupervisor_Start_FailFast_NilContext(t *testing.T) {
	spec := SupervisorSpec{
		Strategy: RestartStrategyOneForOne,
		Children: []ChildSpec{
			{
				ID: "worker1",
				Factory: func() Verticle {
					return newTestWorkerVerticle("1")
				},
				Restart: RestartPermanent,
			},
		},
	}

	supervisor := NewSupervisor(spec)

	// Supervisor extends BaseVerticle, which should handle nil context
	// Testing that it doesn't panic on nil (BaseVerticle.Start will panic)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Start() should panic (fail-fast) with nil context")
		}
	}()

	supervisor.Start(nil)
}

func TestSupervisor_Start_DeploysChildren(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	worker1 := newTestWorkerVerticle("1")
	worker2 := newTestWorkerVerticle("2")

	spec := SupervisorSpec{
		Strategy: RestartStrategyOneForOne,
		Children: []ChildSpec{
			{
				ID: "worker1",
				Factory: func() Verticle {
					return worker1
				},
				Restart: RestartPermanent,
			},
			{
				ID: "worker2",
				Factory: func() Verticle {
					return worker2
				},
				Restart: RestartPermanent,
			},
		},
	}

	supervisor := NewSupervisor(spec)

	err := supervisor.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Wait a bit for children to start
	time.Sleep(100 * time.Millisecond)

	// Verify workers were started
	if !worker1.isStarted() {
		t.Error("worker1 should be started")
	}
	if !worker2.isStarted() {
		t.Error("worker2 should be started")
	}

	// Verify supervisor is started
	if !supervisor.IsStarted() {
		t.Error("Supervisor should be started")
	}

	// Cleanup
	err = supervisor.Stop(fluxorCtx)
	if err != nil {
		t.Errorf("Stop() failed: %v", err)
	}
}

func TestSupervisor_Start_FailFast_NilFactory(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	spec := SupervisorSpec{
		Strategy: RestartStrategyOneForOne,
		Children: []ChildSpec{
			{
				ID: "worker1",
				Factory: func() Verticle {
					return nil // Nil factory
				},
				Restart: RestartPermanent,
			},
		},
	}

	supervisor := NewSupervisor(spec)

	err := supervisor.Start(fluxorCtx)
	if err == nil {
		t.Error("Start() should fail when factory returns nil")
	}
}

// Note: TestSupervisor_Start_FailFast_FailingChild is not included because:
// - DeployVerticle() returns successfully even if child Start() fails (Start() runs asynchronously)
// - The supervisor monitoring will detect failures later via deployment state polling
// - Testing immediate failure detection would require synchronous Start(), which isn't how the system works

func TestSupervisor_Stop_UndeploysChildren(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	worker1 := newTestWorkerVerticle("1")
	worker2 := newTestWorkerVerticle("2")

	spec := SupervisorSpec{
		Strategy: RestartStrategyOneForOne,
		Children: []ChildSpec{
			{
				ID: "worker1",
				Factory: func() Verticle {
					return worker1
				},
				Restart: RestartPermanent,
			},
			{
				ID: "worker2",
				Factory: func() Verticle {
					return worker2
				},
				Restart: RestartPermanent,
			},
		},
	}

	supervisor := NewSupervisor(spec)

	// Start supervisor
	err := supervisor.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Wait for children to start
	time.Sleep(100 * time.Millisecond)

	// Stop supervisor
	err = supervisor.Stop(fluxorCtx)
	if err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Wait for children to stop
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if worker1.isStopped() && worker2.isStopped() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !worker1.isStopped() {
		t.Error("worker1 should be stopped")
	}
	if !worker2.isStopped() {
		t.Error("worker2 should be stopped")
	}
}

func TestSupervisor_Start_FailFast_EmptyChildren(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	spec := SupervisorSpec{
		Strategy: RestartStrategyOneForOne,
		Children: []ChildSpec{}, // Empty children
	}

	supervisor := NewSupervisor(spec)

	// Empty children should be allowed (supervisor with no children)
	err := supervisor.Start(fluxorCtx)
	if err != nil {
		t.Errorf("Start() with empty children should not fail, got: %v", err)
	}

	err = supervisor.Stop(fluxorCtx)
	if err != nil {
		t.Errorf("Stop() failed: %v", err)
	}
}

func TestSupervisor_RestartFrequencyLimit(t *testing.T) {
	// Test that restart frequency limit is properly configured
	spec := SupervisorSpec{
		Strategy: RestartStrategyOneForOne,
		Config: RestartConfig{
			MaxRestarts: 3,
			Within:      5 * time.Second,
		},
		Children: []ChildSpec{
			{
				ID: "worker1",
				Factory: func() Verticle {
					return newTestWorkerVerticle("1")
				},
				Restart: RestartPermanent,
			},
		},
	}

	supervisor := NewSupervisor(spec)
	if supervisor == nil {
		t.Fatal("NewSupervisor() returned nil")
	}

	// Verify config is stored
	if supervisor.spec.Config.MaxRestarts != 3 {
		t.Errorf("MaxRestarts = %v, want 3", supervisor.spec.Config.MaxRestarts)
	}
	if supervisor.spec.Config.Within != 5*time.Second {
		t.Errorf("Within = %v, want 5s", supervisor.spec.Config.Within)
	}
}

func TestSupervisor_RestartStrategies(t *testing.T) {
	strategies := []RestartStrategy{
		RestartStrategyOneForOne,
		RestartStrategyOneForAll,
		RestartStrategyRestForOne,
	}

	for _, strategy := range strategies {
		t.Run(string(strategy), func(t *testing.T) {
			spec := SupervisorSpec{
				Strategy: strategy,
				Children: []ChildSpec{
					{
						ID: "worker1",
						Factory: func() Verticle {
							return newTestWorkerVerticle("1")
						},
						Restart: RestartPermanent,
					},
				},
			}

			supervisor := NewSupervisor(spec)
			if supervisor == nil {
				t.Fatal("NewSupervisor() returned nil")
			}

			if supervisor.spec.Strategy != strategy {
				t.Errorf("Strategy = %v, want %v", supervisor.spec.Strategy, strategy)
			}
		})
	}
}

func TestSupervisor_RestartTypes(t *testing.T) {
	restartTypes := []RestartType{
		RestartPermanent,
		RestartTransient,
		RestartTemporary,
	}

	for _, restartType := range restartTypes {
		t.Run(string(restartType), func(t *testing.T) {
			spec := SupervisorSpec{
				Strategy: RestartStrategyOneForOne,
				Children: []ChildSpec{
					{
						ID: "worker1",
						Factory: func() Verticle {
							return newTestWorkerVerticle("1")
						},
						Restart: restartType,
					},
				},
			}

			supervisor := NewSupervisor(spec)
			if supervisor == nil {
				t.Fatal("NewSupervisor() returned nil")
			}

			if supervisor.spec.Children[0].Restart != restartType {
				t.Errorf("Restart = %v, want %v", supervisor.spec.Children[0].Restart, restartType)
			}
		})
	}
}

func TestSupervisor_Start_FailFast_DoubleStart(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := newFluxorContext(ctx, gocmd)

	spec := SupervisorSpec{
		Strategy: RestartStrategyOneForOne,
		Children: []ChildSpec{
			{
				ID: "worker1",
				Factory: func() Verticle {
					return newTestWorkerVerticle("1")
				},
				Restart: RestartPermanent,
			},
		},
	}

	supervisor := NewSupervisor(spec)

	err := supervisor.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("First Start() failed: %v", err)
	}

	// Second start should fail-fast (BaseVerticle prevents double start)
	err = supervisor.Start(fluxorCtx)
	if err == nil {
		t.Error("Start() should fail-fast when called twice")
	}

	if err != nil {
		if e, ok := err.(*EventBusError); ok {
			if e.Code != "ALREADY_STARTED" {
				t.Errorf("Error code = %v, want 'ALREADY_STARTED'", e.Code)
			}
		} else {
			t.Errorf("Expected EventBusError, got %T", err)
		}
	}

	// Cleanup
	supervisor.Stop(fluxorCtx)
}
