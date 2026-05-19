package compute

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// Mock FluxorContext for testing
type mockFluxorContext struct {
	ctx    context.Context
	gocmd  core.GoCMD
	config map[string]interface{}
}

func (m *mockFluxorContext) Context() context.Context {
	return m.ctx
}

func (m *mockFluxorContext) EventBus() core.EventBus {
	return m.gocmd.EventBus()
}

func (m *mockFluxorContext) GoCMD() core.GoCMD {
	return m.gocmd
}

func (m *mockFluxorContext) Config() map[string]interface{} {
	return m.config
}

func (m *mockFluxorContext) SetConfig(key string, value interface{}) {
	m.config[key] = value
}

func (m *mockFluxorContext) Deploy(v core.Verticle) (string, error) {
	return m.gocmd.DeployVerticle(v)
}

func (m *mockFluxorContext) Undeploy(id string) error {
	return m.gocmd.UndeployVerticle(id)
}

func (m *mockFluxorContext) DeploymentID() string {
	return ""
}

func newMockFluxorContext() core.FluxorContext {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	return &mockFluxorContext{
		ctx:    ctx,
		gocmd:  gocmd,
		config: make(map[string]interface{}),
	}
}

func TestNewComputeComponent(t *testing.T) {
	handler := func(ctx context.Context, payload int) (interface{}, error) {
		return payload * 2, nil
	}

	config := DefaultConfig()
	config.Workers = 2

	component := NewComputeComponent("test-component", handler, config)
	if component == nil {
		t.Fatal("Component is nil")
	}

	if component.Name() != "test-component" {
		t.Errorf("Expected name 'test-component', got '%s'", component.Name())
	}
}

func TestComputeComponent_StartStop(t *testing.T) {
	handler := func(ctx context.Context, payload int) (interface{}, error) {
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 1

	component := NewComputeComponent("test", handler, config)

	ctx := newMockFluxorContext()
	defer ctx.GoCMD().Close()

	if err := component.Start(ctx); err != nil {
		t.Fatalf("Failed to start component: %v", err)
	}

	// Wait a bit for pool to start
	time.Sleep(100 * time.Millisecond)

	stats := component.Stats()
	if !stats.Running {
		t.Error("Component should be running after Start()")
	}

	if err := component.Stop(ctx); err != nil {
		t.Errorf("Failed to stop component: %v", err)
	}

	stats = component.Stats()
	if stats.Running {
		t.Error("Component should not be running after Stop()")
	}
}

func TestComputeComponent_Submit(t *testing.T) {
	handler := func(ctx context.Context, payload int) (interface{}, error) {
		return payload * 2, nil
	}

	config := DefaultConfig()
	config.Workers = 1

	component := NewComputeComponent("test", handler, config)

	ctx := newMockFluxorContext()
	defer ctx.GoCMD().Close()

	if err := component.Start(ctx); err != nil {
		t.Fatalf("Failed to start component: %v", err)
	}
	defer component.Stop(ctx)

	// Wait a bit for workers to initialize
	time.Sleep(100 * time.Millisecond)

	// Submit job
	future, err := component.Submit(ctx, "test-key", 21)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Get handler result
	handlerResult, err := future.GetHandlerResult(ctx.Context())
	if err != nil {
		t.Fatalf("Failed to get handler result: %v", err)
	}

	if handlerResult.(int) != 42 {
		t.Errorf("Expected handler result 42, got %v", handlerResult)
	}
}

func TestComputeComponent_SubmitSync(t *testing.T) {
	handler := func(ctx context.Context, payload int) (interface{}, error) {
		return payload * 2, nil
	}

	config := DefaultConfig()
	config.Workers = 1

	component := NewComputeComponent("test", handler, config)

	ctx := newMockFluxorContext()
	defer ctx.GoCMD().Close()

	if err := component.Start(ctx); err != nil {
		t.Fatalf("Failed to start component: %v", err)
	}
	defer component.Stop(ctx)

	// Wait a bit for workers to initialize
	time.Sleep(100 * time.Millisecond)

	// SubmitSync (blocks)
	result, err := component.SubmitSync(ctx, "test-key", 21)
	if err != nil {
		t.Fatalf("Failed to submit sync: %v", err)
	}

	// Result is payload type (int), not handler result
	if result != 21 {
		t.Errorf("Expected result 21 (payload), got %d", result)
	}
}

func TestNewComputeVerticle(t *testing.T) {
	handler := func(ctx context.Context, payload int) (interface{}, error) {
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 1

	verticle := NewComputeVerticle("test-verticle", handler, config)
	if verticle == nil {
		t.Fatal("Verticle is nil")
	}

	if verticle.Name() != "test-verticle" {
		t.Errorf("Expected name 'test-verticle', got '%s'", verticle.Name())
	}
}

func TestComputeVerticle_StartStop(t *testing.T) {
	handler := func(ctx context.Context, payload int) (interface{}, error) {
		return payload, nil
	}

	config := DefaultConfig()
	config.Workers = 1

	verticle := NewComputeVerticle("test", handler, config)

	ctx := newMockFluxorContext()
	defer ctx.GoCMD().Close()

	if err := verticle.Start(ctx); err != nil {
		t.Fatalf("Failed to start verticle: %v", err)
	}

	// Wait a bit for pool to start
	time.Sleep(100 * time.Millisecond)

	stats := verticle.Stats()
	if !stats.Running {
		t.Error("Verticle should be running after Start()")
	}

	if err := verticle.Stop(ctx); err != nil {
		t.Errorf("Failed to stop verticle: %v", err)
	}

	stats = verticle.Stats()
	if stats.Running {
		t.Error("Verticle should not be running after Stop()")
	}
}

func TestComputeVerticle_Submit(t *testing.T) {
	handler := func(ctx context.Context, payload int) (interface{}, error) {
		return payload * 3, nil
	}

	config := DefaultConfig()
	config.Workers = 1

	verticle := NewComputeVerticle("test", handler, config)

	ctx := newMockFluxorContext()
	defer ctx.GoCMD().Close()

	if err := verticle.Start(ctx); err != nil {
		t.Fatalf("Failed to start verticle: %v", err)
	}
	defer verticle.Stop(ctx)

	// Wait a bit for workers to initialize
	time.Sleep(100 * time.Millisecond)

	// Submit job
	future, err := verticle.Submit(ctx, "key", 14)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Get handler result
	handlerResult, err := future.GetHandlerResult(ctx.Context())
	if err != nil {
		t.Fatalf("Failed to get handler result: %v", err)
	}

	if handlerResult.(int) != 42 {
		t.Errorf("Expected handler result 42, got %v", handlerResult)
	}
}
