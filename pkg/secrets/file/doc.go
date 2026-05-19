// Package file provides a file-based secret provider for reading secrets
// from YAML, JSON, or properties files.
//
// Features:
//   - Support for YAML, JSON, and properties file formats
//   - Auto-detection of file format from extension
//   - Nested key support with dot-notation (e.g., "database.host")
//   - Optional key prefix for namespacing
//   - Thread-safe operations with read-write locks
//   - Reload capability for dynamic secret updates
//   - Automatic file watching with auto-reload on changes
//   - Context support for cancellation and timeouts
//   - Helper functions for common operations
//   - Enhanced error handling with error checking utilities
//   - Fail-fast validation
//
// Quick Start:
//
//	// Create provider from file path (auto-detects format)
//	provider, err := file.NewFromPath("secrets.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get a secret
//	password, err := provider.GetSecret("database.password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get secret as bytes
//	apiKey, err := provider.GetSecretBytes("api.key")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List all secrets
//	keys, err := provider.ListSecrets()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Reload secrets from file
//	if err := provider.Reload(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Close provider (stops file watcher if enabled)
//	defer provider.Close()
//
// Advanced Usage:
//
//	// Create provider with custom configuration and file watching
//	config := file.DefaultConfig()
//	config.Path = "secrets.json"
//	config.Format = "json"  // Explicitly set format
//	config.KeyPrefix = "app."  // Prefix all keys with "app."
//	config.WatchInterval = 2 * time.Second  // Auto-reload on file changes
//
//	provider, err := file.New(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Close()
//
//	// Use context for cancellation and timeouts
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	value, err := provider.GetSecretWithContext(ctx, "database.password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Helper functions
//	value := provider.GetSecretOrDefault("api.key", "default-key")
//	exists := provider.HasSecret("database.host")
//
// File Format Examples:
//
// YAML (secrets.yaml):
//
//	database:
//	  host: localhost
//	  port: 5432
//	  password: secret123
//	api:
//	  key: api-key-value
//	  secret: api-secret-value
//
// JSON (secrets.json):
//
//	{
//	  "database": {
//	    "host": "localhost",
//	    "password": "secret123"
//	  },
//	  "api": {
//	    "key": "api-key-value"
//	  }
//	}
//
// Properties (secrets.properties):
//
//	database.host=localhost
//	database.port=5432
//	database.password=secret123
//	api.key=api-key-value
//
// Security Considerations:
//
//   - Secret files should have restrictive permissions (0600 recommended)
//   - Do not commit secret files to version control
//   - Use environment-specific secret files (dev, staging, prod)
//   - Consider encrypting secret files at rest
//   - Rotate secrets regularly
//
// Path: pkg/secrets/file
package file
