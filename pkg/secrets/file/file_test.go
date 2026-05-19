package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("New() should panic (fail-fast) with nil config")
			}
		}()
		New(nil)
	})

	t.Run("empty path", func(t *testing.T) {
		config := &Config{
			Path: "",
		}
		_, err := New(config)
		if err == nil {
			t.Error("New() expected error for empty path")
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		config := &Config{
			Path:   "test.yaml",
			Format: "invalid",
		}
		_, err := New(config)
		if err == nil {
			t.Error("New() expected error for invalid format")
		}
	})
}

func TestFileProvider_YAML(t *testing.T) {
	// Create temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "secrets.yaml")
	yamlContent := `database:
  host: localhost
  port: 5432
  password: secret123
api:
  key: api-key-value
  secret: api-secret-value
`

	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	// Create provider
	config := DefaultConfig()
	config.Path = yamlFile
	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test GetSecret
	tests := []struct {
		key      string
		expected string
		wantErr  bool
	}{
		{"database.host", "localhost", false},
		{"database.port", "5432", false},
		{"database.password", "secret123", false},
		{"api.key", "api-key-value", false},
		{"api.secret", "api-secret-value", false},
		{"nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			value, err := provider.GetSecret(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSecret(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
				return
			}
			if !tt.wantErr && value != tt.expected {
				t.Errorf("GetSecret(%q) = %q, want %q", tt.key, value, tt.expected)
			}
		})
	}

	// Test GetSecretBytes
	bytes, err := provider.GetSecretBytes("database.host")
	if err != nil {
		t.Errorf("GetSecretBytes() error = %v", err)
	}
	if string(bytes) != "localhost" {
		t.Errorf("GetSecretBytes() = %q, want %q", string(bytes), "localhost")
	}

	// Test ListSecrets
	keys, err := provider.ListSecrets()
	if err != nil {
		t.Errorf("ListSecrets() error = %v", err)
	}
	if len(keys) != 5 {
		t.Errorf("ListSecrets() returned %d keys, want 5", len(keys))
	}
}

func TestFileProvider_JSON(t *testing.T) {
	// Create temporary JSON file
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "secrets.json")
	jsonContent := `{
  "database": {
    "host": "localhost",
    "port": 5432,
    "password": "secret123"
  },
  "api": {
    "key": "api-key-value"
  }
}`

	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0600); err != nil {
		t.Fatalf("Failed to create test JSON file: %v", err)
	}

	// Create provider
	config := DefaultConfig()
	config.Path = jsonFile
	config.Format = "json"
	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test GetSecret
	value, err := provider.GetSecret("database.host")
	if err != nil {
		t.Errorf("GetSecret() error = %v", err)
	}
	if value != "localhost" {
		t.Errorf("GetSecret() = %q, want %q", value, "localhost")
	}
}

func TestFileProvider_Properties(t *testing.T) {
	// Create temporary properties file
	tmpDir := t.TempDir()
	propsFile := filepath.Join(tmpDir, "secrets.properties")
	propsContent := `database.host=localhost
database.port=5432
database.password=secret123
api.key=api-key-value
`

	if err := os.WriteFile(propsFile, []byte(propsContent), 0600); err != nil {
		t.Fatalf("Failed to create test properties file: %v", err)
	}

	// Create provider
	config := DefaultConfig()
	config.Path = propsFile
	config.Format = "properties"
	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test GetSecret
	value, err := provider.GetSecret("database.host")
	if err != nil {
		t.Errorf("GetSecret() error = %v", err)
	}
	if value != "localhost" {
		t.Errorf("GetSecret() = %q, want %q", value, "localhost")
	}
}

func TestFileProvider_KeyPrefix(t *testing.T) {
	// Create temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "secrets.yaml")
	yamlContent := `database:
  host: localhost
  password: secret123
`

	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	// Create provider with key prefix
	config := DefaultConfig()
	config.Path = yamlFile
	config.KeyPrefix = "secrets."
	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test GetSecret with prefix
	value, err := provider.GetSecret("database.host")
	if err != nil {
		t.Errorf("GetSecret() error = %v", err)
	}
	if value != "localhost" {
		t.Errorf("GetSecret() = %q, want %q", value, "localhost")
	}
}

func TestFileProvider_Reload(t *testing.T) {
	// Create temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "secrets.yaml")
	yamlContent := `database:
  host: localhost
`

	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	// Create provider
	config := DefaultConfig()
	config.Path = yamlFile
	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Verify initial value
	value, err := provider.GetSecret("database.host")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}
	if value != "localhost" {
		t.Fatalf("GetSecret() = %q, want %q", value, "localhost")
	}

	// Update file
	newContent := `database:
  host: newhost
`
	if err := os.WriteFile(yamlFile, []byte(newContent), 0600); err != nil {
		t.Fatalf("Failed to update test YAML file: %v", err)
	}

	// Reload
	if err := provider.Reload(); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	// Verify new value
	value, err = provider.GetSecret("database.host")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}
	if value != "newhost" {
		t.Errorf("GetSecret() after reload = %q, want %q", value, "newhost")
	}
}

func TestFileProvider_NonexistentFile(t *testing.T) {
	config := DefaultConfig()
	config.Path = "/nonexistent/file.yaml"
	_, err := New(config)
	if err == nil {
		t.Error("New() expected error for nonexistent file")
	}
}

func TestFileProvider_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.yaml")
	if err := os.WriteFile(emptyFile, []byte(""), 0600); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	config := DefaultConfig()
	config.Path = emptyFile
	_, err := New(config)
	if err == nil {
		t.Error("New() expected error for empty file")
	}
}

func TestNewFromPath(t *testing.T) {
	// Create temporary YAML file
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "secrets.yaml")
	yamlContent := `database:
  host: localhost
`

	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	provider, err := NewFromPath(yamlFile)
	if err != nil {
		t.Fatalf("NewFromPath() error = %v", err)
	}

	value, err := provider.GetSecret("database.host")
	if err != nil {
		t.Errorf("GetSecret() error = %v", err)
	}
	if value != "localhost" {
		t.Errorf("GetSecret() = %q, want %q", value, "localhost")
	}
}

func TestNewFromPath_EmptyPath(t *testing.T) {
	_, err := NewFromPath("")
	if err == nil {
		t.Error("NewFromPath() expected error for empty path")
	}
}

func TestNewWithContext(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "secrets.yaml")
	yamlContent := `database:
  host: localhost
`

	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	ctx := context.Background()
	config := DefaultConfig()
	config.Path = yamlFile
	provider, err := NewWithContext(ctx, config)
	if err != nil {
		t.Fatalf("NewWithContext() error = %v", err)
	}
	defer provider.Close()

	value, err := provider.GetSecret("database.host")
	if err != nil {
		t.Errorf("GetSecret() error = %v", err)
	}
	if value != "localhost" {
		t.Errorf("GetSecret() = %q, want %q", value, "localhost")
	}
}

func TestFileProvider_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "secrets.yaml")
	yamlContent := `database:
  host: localhost
`

	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	config := DefaultConfig()
	config.Path = yamlFile
	provider, err := NewWithContext(ctx, config)
	if err != nil {
		t.Fatalf("NewWithContext() error = %v", err)
	}
	defer provider.Close()

	// Cancel context
	cancel()

	// Wait a bit for cancellation to propagate
	time.Sleep(10 * time.Millisecond)

	// Operations should fail with context cancelled
	_, err = provider.GetSecretWithContext(ctx, "database.host")
	if err == nil {
		t.Error("GetSecretWithContext() expected error after context cancellation")
	}
}

func TestFileProvider_FileWatching(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "secrets.yaml")
	yamlContent := `database:
  host: localhost
`

	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	config := DefaultConfig()
	config.Path = yamlFile
	config.WatchInterval = 100 * time.Millisecond
	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Verify initial value
	value, err := provider.GetSecret("database.host")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}
	if value != "localhost" {
		t.Fatalf("GetSecret() = %q, want %q", value, "localhost")
	}

	// Update file
	newContent := `database:
  host: newhost
`
	if err := os.WriteFile(yamlFile, []byte(newContent), 0600); err != nil {
		t.Fatalf("Failed to update test YAML file: %v", err)
	}

	// Wait for watcher to detect change and reload
	time.Sleep(300 * time.Millisecond)

	// Verify new value was loaded automatically
	value, err = provider.GetSecret("database.host")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}
	if value != "newhost" {
		t.Errorf("GetSecret() after auto-reload = %q, want %q", value, "newhost")
	}
}

func TestFileProvider_HelperFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "secrets.yaml")
	yamlContent := `database:
  host: localhost
  port: 5432
api:
  key: test-key
`

	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	config := DefaultConfig()
	config.Path = yamlFile
	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Test GetSecretOrDefault
	t.Run("GetSecretOrDefault", func(t *testing.T) {
		value := provider.GetSecretOrDefault("database.host", "default")
		if value != "localhost" {
			t.Errorf("GetSecretOrDefault() = %q, want %q", value, "localhost")
		}

		value = provider.GetSecretOrDefault("nonexistent", "default")
		if value != "default" {
			t.Errorf("GetSecretOrDefault() = %q, want %q", value, "default")
		}
	})

	// Test HasSecret
	t.Run("HasSecret", func(t *testing.T) {
		if !provider.HasSecret("database.host") {
			t.Error("HasSecret() = false, want true")
		}

		if provider.HasSecret("nonexistent") {
			t.Error("HasSecret() = true, want false")
		}
	})
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		filePath string
		expected string
	}{
		{"secrets.yaml", "yaml"},
		{"secrets.yml", "yaml"},
		{"secrets.json", "json"},
		{"secrets.properties", "properties"},
		{"secrets.props", "properties"},
		{"secrets.conf", "properties"},
		{"secrets.txt", "yaml"}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := DetectFormat(tt.filePath)
			if result != tt.expected {
				t.Errorf("DetectFormat(%q) = %q, want %q", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestValidateFileExists(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.yaml")
		if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		if err := ValidateFileExists(testFile); err != nil {
			t.Errorf("ValidateFileExists() error = %v, want nil", err)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		if err := ValidateFileExists(""); err == nil {
			t.Error("ValidateFileExists() expected error for empty path")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		err := ValidateFileExists("/nonexistent/file.yaml")
		if err == nil {
			t.Error("ValidateFileExists() expected error for nonexistent file")
		}
		if err != nil && !IsFileNotReadable(err) {
			t.Error("ValidateFileExists() should return ErrFileNotReadable")
		}
	})
}

func TestErrorChecking(t *testing.T) {
	t.Run("IsSecretNotFound", func(t *testing.T) {
		err := &ErrSecretNotFound{Key: "test"}
		if !IsSecretNotFound(err) {
			t.Error("IsSecretNotFound() = false, want true")
		}

		if IsSecretNotFound(nil) {
			t.Error("IsSecretNotFound(nil) = true, want false")
		}
	})

	t.Run("IsInvalidFileFormat", func(t *testing.T) {
		err := &ErrInvalidFileFormat{Path: "test.yaml", Format: "invalid", Err: nil}
		if !IsInvalidFileFormat(err) {
			t.Error("IsInvalidFileFormat() = false, want true")
		}

		if IsInvalidFileFormat(nil) {
			t.Error("IsInvalidFileFormat(nil) = true, want false")
		}
	})

	t.Run("IsFileNotReadable", func(t *testing.T) {
		err := &ErrFileNotReadable{Path: "test.yaml", Err: nil}
		if !IsFileNotReadable(err) {
			t.Error("IsFileNotReadable() = false, want true")
		}

		if IsFileNotReadable(nil) {
			t.Error("IsFileNotReadable(nil) = true, want false")
		}
	})
}

func TestFileProvider_Close(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "secrets.yaml")
	yamlContent := `database:
  host: localhost
`

	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("Failed to create test YAML file: %v", err)
	}

	config := DefaultConfig()
	config.Path = yamlFile
	config.WatchInterval = 100 * time.Millisecond
	provider, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Close should not error
	if err := provider.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Close again should be safe
	if err := provider.Close(); err != nil {
		t.Errorf("Close() second call error = %v, want nil", err)
	}
}
