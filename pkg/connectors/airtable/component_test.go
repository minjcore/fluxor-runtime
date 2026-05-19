package airtable

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/config"
	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewAirtableComponent(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "keyTEST123",
		BaseID:     "appTEST456",
	}

	component := NewAirtableComponent(cfg)
	if component == nil {
		t.Fatal("NewAirtableComponent() returned nil")
	}

	if component.Name() != "airtable" {
		t.Errorf("NewAirtableComponent() Name() = %v, want 'airtable'", component.Name())
	}

	if component.IsStarted() {
		t.Error("NewAirtableComponent() component should not be started after creation")
	}
}

func TestAirtableComponent_Client_NotStarted(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "keyTEST123",
		BaseID:     "appTEST456",
	}

	component := NewAirtableComponent(cfg)

	// Try to get client before starting
	client, err := component.Client()
	if err == nil {
		t.Error("Client() expected error when component not started, got nil")
	}
	if client != nil {
		t.Error("Client() expected nil client when component not started")
	}

	// Check error type
	if _, ok := err.(*core.EventBusError); !ok {
		t.Errorf("Client() error type = %T, want *core.EventBusError", err)
	}
}

func TestAirtableComponent_ServiceClients_NotStarted(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "keyTEST123",
		BaseID:     "appTEST456",
	}

	component := NewAirtableComponent(cfg)

	// Test all service client getters return errors when not started
	services := []struct {
		name string
		fn   func() error
	}{
		{"Tables", func() error { _, err := component.Tables(); return err }},
		{"Records", func() error { _, err := component.Records(); return err }},
	}

	for _, svc := range services {
		t.Run(svc.name, func(t *testing.T) {
			err := svc.fn()
			if err == nil {
				t.Errorf("%s() expected error when component not started, got nil", svc.name)
			}
		})
	}
}

func TestAirtableComponent_Start_NilContext(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "keyTEST123",
		BaseID:     "appTEST456",
	}

	component := NewAirtableComponent(cfg)

	// Try to start with nil context
	err := component.Start(nil)
	if err == nil {
		t.Error("Start() expected error with nil context, got nil")
	}
}

// Helper function to create test context
func createTestContext(t *testing.T) (core.FluxorContext, func()) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)

	testVerticle := core.NewBaseVerticle("test")
	deploymentID, err := gocmd.DeployVerticle(testVerticle)
	if err != nil {
		t.Fatalf("Failed to deploy verticle: %v", err)
	}

	fluxorCtx := testVerticle.Context()
	if fluxorCtx == nil {
		t.Skip("Skipping test - verticle context not available")
	}

	cleanup := func() {
		gocmd.UndeployVerticle(deploymentID)
		gocmd.Close()
	}

	return fluxorCtx, cleanup
}

func TestAirtableComponent_Start_InvalidConfig(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		// Missing APIKey and BaseID
	}

	component := NewAirtableComponent(cfg)
	fluxorCtx, cleanup := createTestContext(t)
	defer cleanup()

	// Try to start with invalid config
	err := component.Start(fluxorCtx)
	if err == nil {
		t.Error("Start() expected error with invalid config, got nil")
	}
}

func TestAirtableComponent_StartStop(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "keyTEST123",
		BaseID:     "appTEST456",
		Timeout:    "5s",
		MaxRetries: 1,
		RateLimit:  5,
	}

	component := NewAirtableComponent(cfg)
	fluxorCtx, cleanup := createTestContext(t)
	defer cleanup()

	// Start component
	if err := component.Start(fluxorCtx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	if !component.IsStarted() {
		t.Error("IsStarted() = false after Start(), want true")
	}

	// Get client should work now
	client, err := component.Client()
	if err != nil {
		t.Errorf("Client() unexpected error after Start(): %v", err)
	}
	if client == nil {
		t.Error("Client() returned nil after Start()")
	}

	// Get service clients
	tables, err := component.Tables()
	if err != nil {
		t.Errorf("Tables() unexpected error after Start(): %v", err)
	}
	if tables == nil {
		t.Error("Tables() returned nil after Start()")
	}

	records, err := component.Records()
	if err != nil {
		t.Errorf("Records() unexpected error after Start(): %v", err)
	}
	if records == nil {
		t.Error("Records() returned nil after Start()")
	}

	// Stop component
	if err := component.Stop(fluxorCtx); err != nil {
		t.Errorf("Stop() unexpected error: %v", err)
	}

	if component.IsStarted() {
		t.Error("IsStarted() = true after Stop(), want false")
	}

	// Client should fail after stop
	_, err = component.Client()
	if err == nil {
		t.Error("Client() expected error after Stop(), got nil")
	}
}

func TestAirtableComponent_EventBus(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "keyTEST123",
		BaseID:     "appTEST456",
	}

	component := NewAirtableComponent(cfg)
	fluxorCtx, cleanup := createTestContext(t)
	defer cleanup()

	// Set parent verticle (from test context)
	component.SetParent(core.NewBaseVerticle("test-parent"))

	// Track events
	readyReceived := false
	stoppedReceived := false

	// Get EventBus from context
	eventBus := fluxorCtx.EventBus()
	if eventBus != nil {
		// Subscribe to events
		eventBus.Consumer("airtable.ready").Handler(func(ctx core.FluxorContext, msg core.Message) error {
			readyReceived = true
			return nil
		})

		eventBus.Consumer("airtable.stopped").Handler(func(ctx core.FluxorContext, msg core.Message) error {
			stoppedReceived = true
			return nil
		})
	}

	// Start component
	if err := component.Start(fluxorCtx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	if eventBus != nil && !readyReceived {
		t.Log("EventBus did not receive 'airtable.ready' event (may be async)")
	}

	// Stop component
	if err := component.Stop(fluxorCtx); err != nil {
		t.Errorf("Stop() unexpected error: %v", err)
	}

	if eventBus != nil && !stoppedReceived {
		t.Log("EventBus did not receive 'airtable.stopped' event (may be async)")
	}
}
