package queue

import (
	"testing"
	"time"
)

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with URL",
			config: Config{
				URL: "amqp://guest:guest@localhost:5672/",
			},
			wantErr: false,
		},
		{
			name: "valid config with host/port",
			config: Config{
				Host: "localhost",
				Port: 5672,
			},
			wantErr: false,
		},
		{
			name: "invalid config - empty host",
			config: Config{
				Host: "",
				Port: 5672,
			},
			wantErr: true,
		},
		{
			name: "invalid config - negative port",
			config: Config{
				Host: "localhost",
				Port: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid config - negative timeout",
			config: Config{
				Host:              "localhost",
				Port:              5672,
				ConnectionTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDefaultConfig tests default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if err := config.Validate(); err != nil {
		t.Errorf("DefaultConfig() should be valid: %v", err)
	}
	if config.Host != "localhost" {
		t.Errorf("DefaultConfig().Host = %v, want localhost", config.Host)
	}
	if config.Port != 5672 {
		t.Errorf("DefaultConfig().Port = %v, want 5672", config.Port)
	}
	if config.VHost != "/" {
		t.Errorf("DefaultConfig().VHost = %v, want /", config.VHost)
	}
}

// TestMessageJSONEncode tests message JSON encoding
func TestMessageJSONEncode(t *testing.T) {
	msg := Message{
		Body: map[string]interface{}{
			"test": "value",
		},
	}

	data, err := JSONEncode(msg.Body)
	if err != nil {
		t.Fatalf("JSONEncode() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("JSONEncode() returned empty data")
	}
}

// TestValidateQueueName tests queue name validation
func TestValidateQueueName(t *testing.T) {
	tests := []struct {
		name    string
		queue   string
		wantErr bool
	}{
		{"valid name", "my.queue", false},
		{"empty name", "", true},
		{"too long name", string(make([]byte, 256)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateQueueName(tt.queue)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateQueueName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateExchangeName tests exchange name validation
func TestValidateExchangeName(t *testing.T) {
	tests := []struct {
		name     string
		exchange string
		wantErr  bool
	}{
		{"valid name", "my.exchange", false},
		{"empty name", "", true},
		{"too long name", string(make([]byte, 256)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExchangeName(tt.exchange)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExchangeName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestComponentCreation tests component creation
func TestComponentCreation(t *testing.T) {
	config := DefaultConfig()
	config.Host = "localhost"
	config.Port = 5672

	// Should not panic with valid config
	comp := NewQueueComponent(config)
	if comp == nil {
		t.Fatal("NewQueueComponent() returned nil")
	}
	if comp.Name() != "queue" {
		t.Errorf("Component.Name() = %v, want queue", comp.Name())
	}
}

// TestComponentCreationInvalidConfig tests component creation with invalid config
func TestComponentCreationInvalidConfig(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewQueueComponent() should panic with invalid config")
		}
	}()

	config := Config{} // Invalid config
	_ = NewQueueComponent(config)
}

// Note: Integration tests requiring actual RabbitMQ server are not included
// These would require a running RabbitMQ instance and should be in a separate file
// with build tags like //go:build integration
