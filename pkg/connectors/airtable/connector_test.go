package airtable

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/connectors"
)

// TestAirtableComponent_ImplementsConnectorInterface verifies that AirtableComponent
// implements the connectors.Connector interface
func TestAirtableComponent_ImplementsConnectorInterface(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key"
	config.BaseID = "test-base"

	component := NewAirtableComponent(config)

	// Verify it implements connectors.Connector interface
	var _ connectors.Connector = component
}

// TestAirtableComponent_Name tests the Name method
func TestAirtableComponent_Name(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key"
	config.BaseID = "test-base"

	component := NewAirtableComponent(config)

	if name := component.Name(); name != "airtable" {
		t.Errorf("Expected name 'airtable', got '%s'", name)
	}
}

// TestAirtableComponent_Type tests the Type method
func TestAirtableComponent_Type(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key"
	config.BaseID = "test-base"

	component := NewAirtableComponent(config)

	if connType := component.Type(); connType != connectors.TypeProductivity {
		t.Errorf("Expected type '%s', got '%s'", connectors.TypeProductivity, connType)
	}
}

// TestAirtableComponent_Version tests the Version method
func TestAirtableComponent_Version(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key"
	config.BaseID = "test-base"

	component := NewAirtableComponent(config)

	if version := component.Version(); version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", version)
	}
}

// TestAirtableComponent_IsHealthy tests the IsHealthy method
func TestAirtableComponent_IsHealthy(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key"
	config.BaseID = "test-base"

	component := NewAirtableComponent(config)

	ctx := context.Background()

	// Should return false when not started
	healthy, err := component.IsHealthy(ctx)
	if healthy {
		t.Error("Expected IsHealthy to return false when not started")
	}
	if err == nil {
		t.Error("Expected IsHealthy to return error when not started")
	}
}

// TestAirtableComponent_GetMetadata tests the GetMetadata method
func TestAirtableComponent_GetMetadata(t *testing.T) {
	config := DefaultConfig()
	config.APIKey = "test-key"
	config.BaseID = "test-base"
	config.RateLimit = 5

	component := NewAirtableComponent(config)

	metadata := component.GetMetadata()

	// Verify basic metadata
	if metadata.Name != "airtable" {
		t.Errorf("Expected metadata name 'airtable', got '%s'", metadata.Name)
	}
	if metadata.DisplayName != "Airtable" {
		t.Errorf("Expected display name 'Airtable', got '%s'", metadata.DisplayName)
	}
	if metadata.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", metadata.Version)
	}
	if metadata.Type != connectors.TypeProductivity {
		t.Errorf("Expected type '%s', got '%s'", connectors.TypeProductivity, metadata.Type)
	}

	// Verify capabilities
	if len(metadata.Capabilities) != 4 {
		t.Errorf("Expected 4 capabilities, got %d", len(metadata.Capabilities))
	}

	expectedCapabilities := map[string]bool{
		"read":     true,
		"write":    true,
		"delete":   true,
		"metadata": true,
	}

	for _, cap := range metadata.Capabilities {
		if _, ok := expectedCapabilities[cap.Name]; !ok {
			t.Errorf("Unexpected capability: %s", cap.Name)
		}
		if !cap.Enabled {
			t.Errorf("Expected capability '%s' to be enabled", cap.Name)
		}
	}

	// Verify rate limits
	if metadata.RateLimits == nil {
		t.Error("Expected rate limits to be set")
	} else if metadata.RateLimits.RequestsPerSecond != 5 {
		t.Errorf("Expected rate limit 5 req/sec, got %d", metadata.RateLimits.RequestsPerSecond)
	}

	// Verify auth methods
	if len(metadata.AuthMethods) != 1 || metadata.AuthMethods[0] != "api_key" {
		t.Errorf("Expected auth method 'api_key', got %v", metadata.AuthMethods)
	}

	// Verify tags
	expectedTags := []string{"productivity", "database", "spreadsheet", "collaboration"}
	if len(metadata.Tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(metadata.Tags))
	}
}

// TestConnectorRegistry_RegisterAirtable tests registering Airtable connector
func TestConnectorRegistry_RegisterAirtable(t *testing.T) {
	registry := connectors.NewConnectorRegistry()

	config := DefaultConfig()
	config.APIKey = "test-key"
	config.BaseID = "test-base"

	component := NewAirtableComponent(config)

	// Register the connector
	if err := registry.Register(component); err != nil {
		t.Fatalf("Failed to register connector: %v", err)
	}

	// Retrieve the connector
	conn, exists := registry.Get("airtable")
	if !exists {
		t.Fatal("Expected connector to be registered")
	}

	if conn.Name() != "airtable" {
		t.Errorf("Expected connector name 'airtable', got '%s'", conn.Name())
	}

	// Test duplicate registration
	err := registry.Register(component)
	if err == nil {
		t.Error("Expected error when registering duplicate connector")
	}
}

// TestConnectorRegistry_ListByType tests listing connectors by type
func TestConnectorRegistry_ListByType(t *testing.T) {
	registry := connectors.NewConnectorRegistry()

	config := DefaultConfig()
	config.APIKey = "test-key"
	config.BaseID = "test-base"

	component := NewAirtableComponent(config)

	// Register the connector
	if err := registry.Register(component); err != nil {
		t.Fatalf("Failed to register connector: %v", err)
	}

	// List productivity connectors
	productivityConns := registry.ListByType(connectors.TypeProductivity)
	if len(productivityConns) != 1 {
		t.Errorf("Expected 1 productivity connector, got %d", len(productivityConns))
	}

	// List database connectors (should be empty)
	dbConns := registry.ListByType(connectors.TypeDatabase)
	if len(dbConns) != 0 {
		t.Errorf("Expected 0 database connectors, got %d", len(dbConns))
	}
}

// TestGlobalRegistry tests the global connector registry
func TestGlobalRegistry(t *testing.T) {
	// Note: This test affects global state, so we clean up after
	config := DefaultConfig()
	config.APIKey = "test-key"
	config.BaseID = "test-base"

	component := NewAirtableComponent(config)

	// Register globally
	if err := connectors.Register(component); err != nil {
		// May already be registered from other tests, try to get it
		if conn, exists := connectors.Get("airtable"); !exists {
			t.Fatalf("Failed to register or retrieve global connector: %v", err)
		} else if conn.Name() != "airtable" {
			t.Errorf("Expected connector name 'airtable', got '%s'", conn.Name())
		}
	}

	// List all connectors
	allConns := connectors.List()
	found := false
	for _, conn := range allConns {
		if conn.Name() == "airtable" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find Airtable connector in global registry")
	}

	// List by type
	productivityConns := connectors.ListByType(connectors.TypeProductivity)
	found = false
	for _, conn := range productivityConns {
		if conn.Name() == "airtable" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find Airtable connector in productivity connectors")
	}
}
