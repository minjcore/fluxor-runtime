package config

import (
	"os"
	"testing"
)

type TestConfig struct {
	Database struct {
		DSN      string `yaml:"dsn" json:"dsn"`
		MaxConns int    `yaml:"max_conns" json:"max_conns"`
	} `yaml:"database" json:"database"`
	Server struct {
		Port int    `yaml:"port" json:"port"`
		Host string `yaml:"host" json:"host"`
	} `yaml:"server" json:"server"`
}

func TestLoadYAML(t *testing.T) {
	// Create temporary YAML file
	yamlContent := `
database:
  dsn: "postgres://localhost/test"
  max_conns: 25
server:
  port: 8080
  host: "localhost"
`
	tmpFile := createTempFile(t, "test.yaml", yamlContent)
	defer os.Remove(tmpFile)

	var cfg TestConfig
	if err := LoadYAML(tmpFile, &cfg); err != nil {
		t.Fatalf("LoadYAML failed: %v", err)
	}

	if cfg.Database.DSN != "postgres://localhost/test" {
		t.Errorf("Database.DSN = %v, want postgres://localhost/test", cfg.Database.DSN)
	}
	if cfg.Database.MaxConns != 25 {
		t.Errorf("Database.MaxConns = %v, want 25", cfg.Database.MaxConns)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %v, want 8080", cfg.Server.Port)
	}
}

func TestLoadJSON(t *testing.T) {
	// Create temporary JSON file
	jsonContent := `{
  "database": {
    "dsn": "postgres://localhost/test",
    "max_conns": 25
  },
  "server": {
    "port": 8080,
    "host": "localhost"
  }
}`
	tmpFile := createTempFile(t, "test.json", jsonContent)
	defer os.Remove(tmpFile)

	var cfg TestConfig
	if err := LoadJSON(tmpFile, &cfg); err != nil {
		t.Fatalf("LoadJSON failed: %v", err)
	}

	if cfg.Database.DSN != "postgres://localhost/test" {
		t.Errorf("Database.DSN = %v, want postgres://localhost/test", cfg.Database.DSN)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %v, want 8080", cfg.Server.Port)
	}
}

func TestLoadWithEnv(t *testing.T) {
	// Create temporary YAML file
	yamlContent := `
database:
  dsn: "postgres://localhost/test"
  max_conns: 25
server:
  port: 8080
  host: "localhost"
`
	tmpFile := createTempFile(t, "test.yaml", yamlContent)
	defer os.Remove(tmpFile)

	// Set environment variables
	os.Setenv("APP_DATABASE_DSN", "postgres://env/test")
	os.Setenv("APP_SERVER_PORT", "9090")
	defer os.Unsetenv("APP_DATABASE_DSN")
	defer os.Unsetenv("APP_SERVER_PORT")

	var cfg TestConfig
	if err := LoadWithEnv(tmpFile, "APP", &cfg); err != nil {
		t.Fatalf("LoadWithEnv failed: %v", err)
	}

	// Environment variables should override file values
	if cfg.Database.DSN != "postgres://env/test" {
		t.Errorf("Database.DSN = %v, want postgres://env/test", cfg.Database.DSN)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %v, want 9090", cfg.Server.Port)
	}
	// Host should remain from file (no env override)
	if cfg.Server.Host != "localhost" {
		t.Errorf("Server.Host = %v, want localhost", cfg.Server.Host)
	}
}

func TestRequiredFields(t *testing.T) {
	cfg := TestConfig{
		Database: struct {
			DSN      string `yaml:"dsn" json:"dsn"`
			MaxConns int    `yaml:"max_conns" json:"max_conns"`
		}{
			DSN:      "",
			MaxConns: 25,
		},
	}

	// Test with nested field path
	validator := RequiredFields("Database.DSN")
	if err := validator.Validate(&cfg); err == nil {
		t.Error("RequiredFields should fail for empty DSN")
	}

	cfg.Database.DSN = "postgres://localhost/test"
	if err := validator.Validate(&cfg); err != nil {
		t.Errorf("RequiredFields should pass for valid config: %v", err)
	}
}

func TestRangeValidator(t *testing.T) {
	cfg := TestConfig{
		Database: struct {
			DSN      string `yaml:"dsn" json:"dsn"`
			MaxConns int    `yaml:"max_conns" json:"max_conns"`
		}{
			DSN:      "postgres://localhost/test",
			MaxConns: 5,
		},
	}

	// Use nested field path
	validator := RangeValidator("Database.MaxConns", 10, 100)
	if err := validator.Validate(&cfg); err == nil {
		t.Error("RangeValidator should fail for value below minimum")
	}

	cfg.Database.MaxConns = 50
	if err := validator.Validate(&cfg); err != nil {
		t.Errorf("RangeValidator should pass for value in range: %v", err)
	}
}

func createTempFile(t *testing.T, name, content string) string {
	tmpFile := name
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	return tmpFile
}
