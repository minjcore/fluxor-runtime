package config_test

import (
	"os"
	"testing"

	"github.com/fluxorio/fluxor/pkg/config"
)

func TestConfigWithEnvOverrides(t *testing.T) {
	// Create temporary YAML file
	yamlContent := `
database:
  dsn: "postgres://localhost/test"
  max_conns: 25
server:
  port: 8080
  host: "localhost"
`
	tmpFile := "test_config.yaml"
	if err := os.WriteFile(tmpFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	// Set environment variables
	os.Setenv("APP_DATABASE_DSN", "postgres://env/test")
	os.Setenv("APP_SERVER_PORT", "9090")
	defer os.Unsetenv("APP_DATABASE_DSN")
	defer os.Unsetenv("APP_SERVER_PORT")

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

	var cfg TestConfig
	if err := config.LoadWithEnv(tmpFile, "APP", &cfg); err != nil {
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
