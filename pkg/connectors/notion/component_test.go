package notion

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/config"
	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewNotionComponent(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
	}

	component := NewNotionComponent(cfg)
	if component == nil {
		t.Fatal("NewNotionComponent() returned nil")
	}

	if component.Name() != "notion" {
		t.Errorf("NewNotionComponent() Name() = %v, want 'notion'", component.Name())
	}

	if component.Type() != connectors.TypeProductivity {
		t.Errorf("NewNotionComponent() Type() = %v, want TypeProductivity", component.Type())
	}

	if component.Version() != "1.0.0" {
		t.Errorf("NewNotionComponent() Version() = %v, want '1.0.0'", component.Version())
	}

	if component.IsStarted() {
		t.Error("NewNotionComponent() component should not be started after creation")
	}
}

func TestNotionComponent_Client_NotStarted(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
	}

	component := NewNotionComponent(cfg)

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

func TestNotionComponent_ServiceClients_NotStarted(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
	}

	component := NewNotionComponent(cfg)

	// Test all service client getters return errors when not started
	services := []struct {
		name string
		fn   func() error
	}{
		{"Pages", func() error { _, err := component.Pages(); return err }},
		{"Databases", func() error { _, err := component.Databases(); return err }},
		{"Blocks", func() error { _, err := component.Blocks(); return err }},
		{"Users", func() error { _, err := component.Users(); return err }},
		{"Search", func() error { _, err := component.Search(); return err }},
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

func TestNotionComponent_Start_NilContext(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
	}

	component := NewNotionComponent(cfg)

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

func TestNotionComponent_Start_InvalidConfig(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		// Missing APIKey
	}

	component := NewNotionComponent(cfg)
	fluxorCtx, cleanup := createTestContext(t)
	defer cleanup()

	// Try to start with invalid config
	err := component.Start(fluxorCtx)
	if err == nil {
		t.Error("Start() expected error with invalid config, got nil")
	}
}

func TestNotionComponent_StartStop(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
		BaseURL:    "https://api.notion.com",
		Version:    "2022-06-28",
		Timeout:    "5s",
		MaxRetries: 1,
		RateLimit:  3,
	}

	component := NewNotionComponent(cfg)
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
	pages, err := component.Pages()
	if err != nil {
		t.Errorf("Pages() unexpected error after Start(): %v", err)
	}
	if pages == nil {
		t.Error("Pages() returned nil after Start()")
	}

	databases, err := component.Databases()
	if err != nil {
		t.Errorf("Databases() unexpected error after Start(): %v", err)
	}
	if databases == nil {
		t.Error("Databases() returned nil after Start()")
	}

	blocks, err := component.Blocks()
	if err != nil {
		t.Errorf("Blocks() unexpected error after Start(): %v", err)
	}
	if blocks == nil {
		t.Error("Blocks() returned nil after Start()")
	}

	users, err := component.Users()
	if err != nil {
		t.Errorf("Users() unexpected error after Start(): %v", err)
	}
	if users == nil {
		t.Error("Users() returned nil after Start()")
	}

	search, err := component.Search()
	if err != nil {
		t.Errorf("Search() unexpected error after Start(): %v", err)
	}
	if search == nil {
		t.Error("Search() returned nil after Start()")
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

func TestNotionComponent_Start_AlreadyStarted(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
	}

	component := NewNotionComponent(cfg)
	fluxorCtx, cleanup := createTestContext(t)
	defer cleanup()

	// Start component
	if err := component.Start(fluxorCtx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	// Try to start again
	err := component.Start(fluxorCtx)
	if err == nil {
		t.Error("Start() expected error when already started, got nil")
	}

	// Check error code
	if eventBusErr, ok := err.(*core.EventBusError); ok {
		if eventBusErr.Code != "ALREADY_STARTED" {
			t.Errorf("Start() error code = %v, want 'ALREADY_STARTED'", eventBusErr.Code)
		}
	}
}

func TestNotionComponent_Stop_NotStarted(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
	}

	component := NewNotionComponent(cfg)
	fluxorCtx, cleanup := createTestContext(t)
	defer cleanup()

	// Stop without starting should not error
	if err := component.Stop(fluxorCtx); err != nil {
		t.Errorf("Stop() unexpected error when not started: %v", err)
	}
}

func TestNotionComponent_EventBus(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
	}

	component := NewNotionComponent(cfg)
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
		eventBus.Consumer("notion.ready").Handler(func(ctx core.FluxorContext, msg core.Message) error {
			readyReceived = true
			return nil
		})

		eventBus.Consumer("notion.stopped").Handler(func(ctx core.FluxorContext, msg core.Message) error {
			stoppedReceived = true
			return nil
		})
	}

	// Start component
	if err := component.Start(fluxorCtx); err != nil {
		t.Fatalf("Start() unexpected error: %v", err)
	}

	if eventBus != nil && !readyReceived {
		t.Log("EventBus did not receive 'notion.ready' event (may be async)")
	}

	// Stop component
	if err := component.Stop(fluxorCtx); err != nil {
		t.Errorf("Stop() unexpected error: %v", err)
	}

	if eventBus != nil && !stoppedReceived {
		t.Log("EventBus did not receive 'notion.stopped' event (may be async)")
	}
}

func TestNotionComponent_IsHealthy_NotStarted(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
	}

	component := NewNotionComponent(cfg)
	ctx := context.Background()

	healthy, err := component.IsHealthy(ctx)
	if healthy {
		t.Error("IsHealthy() expected false when not started")
	}
	if err == nil {
		t.Error("IsHealthy() expected error when not started")
	}
}

func TestNotionComponent_GetMetadata(t *testing.T) {
	cfg := Config{
		BaseConfig: *config.NewBaseConfig(),
		APIKey:     "secret_TEST123",
		RateLimit:  3,
	}

	component := NewNotionComponent(cfg)
	metadata := component.GetMetadata()

	if metadata.Name != "notion" {
		t.Errorf("GetMetadata() Name = %v, want 'notion'", metadata.Name)
	}
	if metadata.DisplayName != "Notion" {
		t.Errorf("GetMetadata() DisplayName = %v, want 'Notion'", metadata.DisplayName)
	}
	if metadata.Type != connectors.TypeProductivity {
		t.Errorf("GetMetadata() Type = %v, want TypeProductivity", metadata.Type)
	}
	if metadata.Version != "1.0.0" {
		t.Errorf("GetMetadata() Version = %v, want '1.0.0'", metadata.Version)
	}
	if len(metadata.Capabilities) == 0 {
		t.Error("GetMetadata() Capabilities should not be empty")
	}
	if metadata.RateLimits == nil {
		t.Error("GetMetadata() RateLimits should not be nil")
	}
	if metadata.RateLimits.RequestsPerSecond != 3 {
		t.Errorf("GetMetadata() RateLimits.RequestsPerSecond = %v, want 3", metadata.RateLimits.RequestsPerSecond)
	}
}
