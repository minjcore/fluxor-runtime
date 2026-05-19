// Package config provides a comprehensive configuration management system
// for loading, validating, and watching configuration files.
//
// Features:
//   - Load configuration from YAML, JSON, and Properties files
//   - Environment variable overrides with prefix support
//   - Configuration validation with custom validators
//   - File watching with automatic reload
//   - Type-safe configuration access
//
// Quick Start:
//
//	// Load config from file
//	var cfg MyConfig
//	if err := config.Load("config.yaml", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Load with environment variable overrides
//	if err := config.LoadWithEnv("config.yaml", "APP", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use Manager for advanced features
//	mgr, err := config.NewManagerFromFile("config.yaml", &cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	mgr.AddValidator(config.RequiredFields("Database.Host", "Database.Port"))
//	if err := mgr.Validate(); err != nil {
//	    log.Fatal(err)
//	}
package config

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// ============================================================================
// Package Overview
// ============================================================================
//
// This package provides configuration management with the following capabilities:
//
// 1. LOADING: Load config from YAML, JSON, or Properties files
//    - Auto-detects file type by extension
//    - Supports nested structs and pointers
//
// 2. ENVIRONMENT VARIABLES: Override config values via environment variables
//    - Format: PREFIX_FIELD_SUBFIELD (e.g., APP_DATABASE_HOST)
//    - Automatically converts string values to appropriate types
//
// 3. VALIDATION: Validate configuration using validators
//    - Built-in validators: RequiredFields, RangeValidator, StringLengthValidator
//    - Support for struct tag validation: validate:"required,range:1-100"
//
// 4. WATCHING: Watch config files for changes and auto-reload
//    - Configurable check interval
//    - Automatic validation after reload
//    - Custom reload callbacks
//
// 5. MANAGER: High-level API for managing configuration lifecycle
//    - Load, validate, and watch in one place
//    - Type-safe access with GetTyped<T>()
//
// ============================================================================
// BaseConfig Types
// ============================================================================

// BaseConfig provides a Java-style abstract base class for configuration.
// Applications can embed this struct to inherit common configuration features
// like service name, server settings, profile, and environment.
//
// Usage Example:
//
//	package mypackage
//
//	import "github.com/fluxorio/fluxor/pkg/config"
//
//	type Config struct {
//	    // Embed BaseConfig to inherit common fields
//	    config.BaseConfig
//
//	    // Add your custom configuration fields
//	    DatabaseURL string `json:"databaseURL"`
//	    APIKey      string `json:"apiKey"`
//	}
//
//	func DefaultConfig() Config {
//	    return Config{
//	        BaseConfig: *config.NewBaseConfig(),
//	        DatabaseURL: "postgres://localhost/db",
//	        APIKey:      "",
//	    }
//	}
//
//	func (c *Config) Validate() error {
//	    // Validate BaseConfig first
//	    baseValidators := c.BaseConfig.GetValidators()
//	    for _, validator := range baseValidators {
//	        if err := validator.Validate(c); err != nil {
//	            return err
//	        }
//	    }
//
//	    // Validate your custom fields
//	    if c.DatabaseURL == "" {
//	        return fmt.Errorf("database URL is required")
//	    }
//	    return nil
//	}
//
//	// Load from file (BaseConfig supports YAML/JSON)
//	var cfg Config
//	if err := config.Load("config.yaml", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Access BaseConfig fields
//	fmt.Println(cfg.Service.Name)      // Service name
//	fmt.Println(cfg.Server.Addr)       // Server address
//	fmt.Println(cfg.Profile)           // Configuration profile
//	fmt.Println(cfg.Environment)       // Environment name
//
// See pkg/cloud/aws/config.go for a real-world example of using BaseConfig.
type BaseConfig struct {
	// Common configuration sections (can be overridden)
	Service ServiceConfig `config:"service"`
	Server  ServerConfig  `config:"server"`

	// Metadata
	Profile     string `default:"default" env:"APP_PROFILE" description:"Configuration profile (dev, staging, prod)"`
	Environment string `default:"development" env:"APP_ENV" description:"Environment name"`

	// Internal state
	loadedAt time.Time
	source   string
	cache    Cache  // Optional cache for configuration
	cacheKey string // Cache key for this configuration
	mu       sync.RWMutex
}

// ServiceConfig is part of BaseConfig (common to all applications)
type ServiceConfig struct {
	Name string `default:"app" env:"SERVICE_NAME" validate:"required,min=1,max=100" description:"Service name"`
	Port int    `default:"8080" env:"SERVICE_PORT" validate:"range:1024-65535" description:"Service port"`
}

// ServerConfig is part of BaseConfig (common to all applications)
type ServerConfig struct {
	Addr               string        `default:":8080" env:"SERVER_ADDR" validate:"required" description:"Server address"`
	MaxCCU             int           `default:"5000" env:"SERVER_MAX_CCU" validate:"range:100-100000" description:"Maximum concurrent users"`
	UtilizationPercent int           `default:"67" env:"SERVER_UTILIZATION" validate:"range:1-100" description:"Target utilization percentage"`
	ReadTimeout        time.Duration `default:"10s" env:"SERVER_READ_TIMEOUT" description:"Read timeout"`
	WriteTimeout       time.Duration `default:"10s" env:"SERVER_WRITE_TIMEOUT" description:"Write timeout"`
	MaxConns           int           `default:"100000" env:"SERVER_MAX_CONNS" validate:"range:1-1000000" description:"Maximum connections"`
	ReadBufferSize     int           `default:"8192" env:"SERVER_READ_BUFFER" validate:"range:1024-65536" description:"Read buffer size"`
	WriteBufferSize    int           `default:"8192" env:"SERVER_WRITE_BUFFER" validate:"range:1024-65536" description:"Write buffer size"`
}

// ============================================================================
// BaseConfig Methods
// ============================================================================

// NewBaseConfig creates a new BaseConfig with defaults
func NewBaseConfig() *BaseConfig {
	cfg := &BaseConfig{
		Service: ServiceConfig{
			Name: "app",
			Port: 8080,
		},
		Server: ServerConfig{
			Addr:               ":8080",
			MaxCCU:             5000,
			UtilizationPercent: 67,
			ReadTimeout:        10 * time.Second,
			WriteTimeout:       10 * time.Second,
			MaxConns:           100000,
			ReadBufferSize:     8192,
			WriteBufferSize:    8192,
		},
		Profile:     "default",
		Environment: "development",
	}
	return cfg
}

// BeforeLoad is a hook method for pre-load processing
// Override in subclasses for custom behavior
func (bc *BaseConfig) BeforeLoad() error {
	return nil
}

// AfterLoad is a hook method for post-load processing
// Override in subclasses for custom behavior
func (bc *BaseConfig) AfterLoad() error {
	return nil
}

// GetValidators returns validators for this configuration
// Override in subclasses to add custom validators
func (bc *BaseConfig) GetValidators() []Validator {
	return []Validator{
		RequiredFields("Service.Name", "Server.Addr"),
		RangeValidator("Server.MaxCCU", 100, 100000),
		RangeValidator("Server.UtilizationPercent", 1, 100),
	}
}

// GetDefaults returns custom defaults map
// Override in subclasses to provide custom defaults
func (bc *BaseConfig) GetDefaults() map[string]interface{} {
	return make(map[string]interface{})
}

// IsLoaded returns whether the configuration has been loaded
func (bc *BaseConfig) IsLoaded() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return !bc.loadedAt.IsZero()
}

// GetSource returns the source file path
func (bc *BaseConfig) GetSource() string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.source
}

// SetSource sets the source file path (internal use)
// Fail-fast: Validates source is not empty
func (bc *BaseConfig) SetSource(source string) {
	// Fail-fast: source cannot be empty
	if source == "" {
		panic(fmt.Errorf("fail-fast: source cannot be empty"))
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.source = source
}

// SetLoaded marks the configuration as loaded (internal use)
func (bc *BaseConfig) SetLoaded() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.loadedAt = time.Now()
}

// SetCache sets the cache for this configuration
// Fail-fast: Validates cache key is not empty if cache is provided
func (bc *BaseConfig) SetCache(cache Cache, cacheKey string) {
	// Fail-fast: if cache is provided, cacheKey cannot be empty
	if cache != nil && cacheKey == "" {
		panic(fmt.Errorf("fail-fast: cacheKey cannot be empty when cache is provided"))
	}

	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.cache = cache
	bc.cacheKey = cacheKey
}

// GetCache returns the cache for this configuration
func (bc *BaseConfig) GetCache() Cache {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.cache
}

// GetCacheKey returns the cache key for this configuration
func (bc *BaseConfig) GetCacheKey() string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.cacheKey
}

// ============================================================================
// Configuration Loading Functions
// ============================================================================
//
// These functions handle loading configuration from files.
// They automatically detect file format and support environment variable overrides.
//
// Flow:
//   1. Validate inputs (fail-fast)
//   2. Detect file type by extension
//   3. Load file content
//   4. Unmarshal into target struct
//
// ============================================================================

// Load loads configuration from a file (YAML or JSON).
// Automatically detects file type by extension (.yaml, .yml, .json).
// Defaults to YAML if extension is not recognized.
//
// Example:
//
//	var cfg MyConfig
//	if err := config.Load("config.yaml", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
// Fail-fast: Validates inputs before processing
func Load(path string, target interface{}) error {
	// Fail-fast: path cannot be empty
	if path == "" {
		return fmt.Errorf("fail-fast: config file path cannot be empty")
	}

	// Fail-fast: target cannot be nil
	failfast.NotNil(target, "target")

	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		return LoadYAML(path, target)
	}
	if strings.HasSuffix(path, ".json") {
		return LoadJSON(path, target)
	}
	// Default to YAML
	return LoadYAML(path, target)
}

// LoadWithEnv loads configuration from file and applies environment variable overrides.
// Environment variables use format: PREFIX_FIELD_SUBFIELD (e.g., APP_DATABASE_DSN).
//
// Example:
//
//	// Set APP_DATABASE_HOST=localhost to override Database.Host field
//	var cfg MyConfig
//	if err := config.LoadWithEnv("config.yaml", "APP", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
// Fail-fast: Validates inputs before processing
func LoadWithEnv(path string, prefix string, target interface{}) error {
	// Fail-fast: prefix cannot be empty
	if prefix == "" {
		return fmt.Errorf("fail-fast: environment variable prefix cannot be empty")
	}

	// Load from file first (validates path and target)
	if err := Load(path, target); err != nil {
		return fmt.Errorf("fail-fast: failed to load config file: %w", err)
	}

	// Apply environment variable overrides
	if err := ApplyEnvOverrides(prefix, target); err != nil {
		return fmt.Errorf("fail-fast: failed to apply env overrides: %w", err)
	}

	return nil
}

// ============================================================================
// Environment Variable Overrides
// ============================================================================
//
// These functions handle applying environment variable overrides to configuration structs.
// Environment variables are automatically converted to the appropriate field types.
//
// Naming Convention:
//   - Struct field "DatabaseHost" with prefix "APP" becomes "APP_DATABASEHOST"
//   - Nested structs: "Database.Host" becomes "APP_DATABASE_HOST"
//
// ============================================================================

// ApplyEnvOverrides applies environment variable overrides to configuration struct.
// Uses reflection to set struct fields from environment variables.
//
// The prefix is used to build environment variable names:
//   - Prefix "APP" + Field "DatabaseHost" = "APP_DATABASEHOST"
//   - Nested fields: "APP_DATABASE_HOST" maps to Database.Host
//
// Example:
//
//	// Set APP_SERVER_PORT=9090 to override Server.Port
//	var cfg MyConfig
//	if err := config.ApplyEnvOverrides("APP", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
// Fail-fast: Validates inputs before processing
func ApplyEnvOverrides(prefix string, target interface{}) error {
	// Fail-fast: prefix cannot be empty
	if prefix == "" {
		prefix = "APP" // Default prefix
	}

	// Fail-fast: target cannot be nil
	failfast.NotNil(target, "target")

	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("fail-fast: target must be a pointer to a struct, got %s", val.Kind())
	}
	if val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("fail-fast: target must be a pointer to a struct, got pointer to %s", val.Elem().Kind())
	}

	return applyEnvToStruct(prefix, val.Elem())
}

// applyEnvToStruct recursively applies environment variables to struct fields
func applyEnvToStruct(prefix string, val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Build environment variable name: PREFIX_FIELDNAME
		envKey := prefix + "_" + strings.ToUpper(fieldType.Name)
		envKey = strings.ReplaceAll(envKey, "-", "_")

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := applyEnvToStruct(envKey, field); err != nil {
				return err
			}
			continue
		}

		// Handle pointers to structs
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			if err := applyEnvToStruct(envKey, field.Elem()); err != nil {
				return err
			}
			continue
		}

		// Get environment variable value
		envValue := os.Getenv(envKey)
		if envValue == "" {
			continue // No override for this field
		}

		// Set field value based on type
		if err := setFieldFromEnv(field, envValue); err != nil {
			return fmt.Errorf("failed to set field %s from env %s: %w", fieldType.Name, envKey, err)
		}
	}

	return nil
}

// setFieldFromEnv sets a struct field value from environment variable string
func setFieldFromEnv(field reflect.Value, envValue string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(envValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intVal int64
		if _, err := fmt.Sscanf(envValue, "%d", &intVal); err != nil {
			return fmt.Errorf("invalid integer value: %s", envValue)
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uintVal uint64
		if _, err := fmt.Sscanf(envValue, "%d", &uintVal); err != nil {
			return fmt.Errorf("invalid unsigned integer value: %s", envValue)
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		var floatVal float64
		if _, err := fmt.Sscanf(envValue, "%f", &floatVal); err != nil {
			return fmt.Errorf("invalid float value: %s", envValue)
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal := strings.ToLower(envValue) == "true" || envValue == "1"
		field.SetBool(boolVal)
	case reflect.Slice:
		// For slices, split by comma
		parts := strings.Split(envValue, ",")
		sliceType := field.Type().Elem()
		slice := reflect.MakeSlice(field.Type(), len(parts), len(parts))
		for i, part := range parts {
			part = strings.TrimSpace(part)
			elem := reflect.New(sliceType).Elem()
			if err := setFieldFromEnv(elem, part); err != nil {
				return err
			}
			slice.Index(i).Set(elem)
		}
		field.Set(slice)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// ============================================================================
// Validation Functions
// ============================================================================
//
// These functions validate configuration using validators.
// Validators can be custom functions or built-in validators from validator.go.
//
// Built-in validators:
//   - RequiredFields: Check that fields are not empty
//   - RangeValidator: Check numeric values are within range
//   - StringLengthValidator: Check string length
//   - OneOfValidator: Check value is one of allowed values
//
// ============================================================================

// Validate validates configuration using registered validators.
// Runs all validators in sequence and returns the first error encountered.
//
// Example:
//
//	var cfg MyConfig
//	validators := []config.Validator{
//	    config.RequiredFields("Database.Host", "Database.Port"),
//	    config.RangeValidator("Server.Port", 1024, 65535),
//	}
//	if err := config.Validate(&cfg, validators...); err != nil {
//	    log.Fatal(err)
//	}
//
// Fail-fast: Validates inputs and fails immediately on validation errors
func Validate(config interface{}, validators ...Validator) error {
	// Fail-fast: config cannot be nil
	failfast.NotNil(config, "config")

	// Fail-fast: validators cannot be nil (but can be empty slice)
	for i, validator := range validators {
		if validator == nil {
			return fmt.Errorf("fail-fast: validator at index %d is nil", i)
		}
		if err := validator.Validate(config); err != nil {
			return fmt.Errorf("fail-fast: validation failed: %w", err)
		}
	}
	return nil
}

// ============================================================================
// Manager Type and Constructors
// ============================================================================
//
// Manager provides a high-level API for managing configuration lifecycle.
// It combines loading, validation, and file watching in one place.
//
// Typical usage:
//   1. Create manager with NewManagerFromFile or NewManagerFromFileWithEnv
//   2. Add validators with AddValidator
//   3. Validate with Validate or ValidateFromTags
//   4. Optionally watch for changes with Watch
//
// ============================================================================

// Manager manages configuration with validation, environment variable support, and file watching.
//
// Manager provides a high-level API for:
//   - Loading configuration from files
//   - Adding and running validators
//   - Watching files for changes
//   - Reloading configuration automatically
//
// Example:
//
//	var cfg MyConfig
//	mgr, err := config.NewManagerFromFile("config.yaml", &cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Add validators
//	mgr.AddValidator(config.RequiredFields("Database.Host"))
//
//	// Validate
//	if err := mgr.Validate(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Watch for changes
//	if err := mgr.Watch("", 2*time.Second, nil); err != nil {
//	    log.Fatal(err)
//	}
type Manager struct {
	config     interface{}
	validators []Validator
	watcher    *ConfigWatcher
	filePath   string
	envPrefix  string
}

// NewManager creates a new configuration manager with an existing config instance.
// Use this when you already have a loaded configuration.
//
// Example:
//
//	var cfg MyConfig
//	// ... load cfg manually ...
//	mgr := config.NewManager(&cfg)
//	mgr.AddValidator(config.RequiredFields("Database.Host"))
//
// Fail-fast: Validates config is not nil
func NewManager(config interface{}) *Manager {
	// Fail-fast: config cannot be nil
	failfast.NotNil(config, "config")

	return &Manager{
		config:     config,
		validators: make([]Validator, 0),
		watcher:    nil,
	}
}

// NewManagerFromFile creates a new configuration manager and loads config from file.
// This is the most common way to create a Manager.
//
// Example:
//
//	var cfg MyConfig
//	mgr, err := config.NewManagerFromFile("config.yaml", &cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Fail-fast: Validates inputs before processing
func NewManagerFromFile(filePath string, target interface{}) (*Manager, error) {
	// Fail-fast: filePath cannot be empty
	if filePath == "" {
		return nil, fmt.Errorf("fail-fast: config file path cannot be empty")
	}

	// Load config from file
	if err := Load(filePath, target); err != nil {
		return nil, fmt.Errorf("fail-fast: failed to load config: %w", err)
	}

	return &Manager{
		config:     target,
		validators: make([]Validator, 0),
		watcher:    nil,
		filePath:   filePath,
	}, nil
}

// NewManagerFromFileWithEnv creates a new configuration manager and loads config from file with env overrides.
// Environment variables with the given prefix will override values from the file.
//
// Example:
//
//	// Set APP_DATABASE_HOST=localhost to override Database.Host
//	var cfg MyConfig
//	mgr, err := config.NewManagerFromFileWithEnv("config.yaml", "APP", &cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Fail-fast: Validates inputs before processing
func NewManagerFromFileWithEnv(filePath string, envPrefix string, target interface{}) (*Manager, error) {
	// Fail-fast: filePath cannot be empty
	if filePath == "" {
		return nil, fmt.Errorf("fail-fast: config file path cannot be empty")
	}

	// Fail-fast: envPrefix cannot be empty
	if envPrefix == "" {
		return nil, fmt.Errorf("fail-fast: environment variable prefix cannot be empty")
	}

	// Load config from file with env overrides
	if err := LoadWithEnv(filePath, envPrefix, target); err != nil {
		return nil, fmt.Errorf("fail-fast: failed to load config: %w", err)
	}

	return &Manager{
		config:     target,
		validators: make([]Validator, 0),
		watcher:    nil,
		filePath:   filePath,
		envPrefix:  envPrefix,
	}, nil
}

// ============================================================================
// Manager Methods - Validation
// ============================================================================

// AddValidator adds a validator to the manager
// Fail-fast: Validates validator is not nil
func (m *Manager) AddValidator(validator Validator) {
	// Fail-fast: validator cannot be nil
	failfast.NotNil(validator, "validator")

	m.validators = append(m.validators, validator)
}

// Validate validates the configuration
// Fail-fast: Validates inputs and fails immediately on validation errors
func (m *Manager) Validate() error {
	return Validate(m.config, m.validators...)
}

// ValidateFromTags validates the configuration using struct tags
// Supports validate:"required,range:1-100,min:10,max:1000" format
// Fail-fast: Validates inputs before processing
func (m *Manager) ValidateFromTags() error {
	return ValidateFromTags(m.config)
}

// ============================================================================
// Manager Methods - File Watching
// ============================================================================
//
// These methods handle watching configuration files for changes and reloading.
// When a file change is detected, the configuration is automatically reloaded
// and validated. An optional callback can be called after reload.
//
// ============================================================================

// Watch starts watching the configuration file for changes.
// If filePath is empty, it uses the filePath from NewManagerFromFile.
//
// When a change is detected:
//  1. Configuration is reloaded from file
//  2. Environment variable overrides are applied (if envPrefix is set)
//  3. Configuration is validated
//  4. onReload callback is called (if provided)
//
// Example:
//
//	mgr, _ := config.NewManagerFromFile("config.yaml", &cfg)
//	err := mgr.Watch("", 2*time.Second, func() error {
//	    log.Println("Config reloaded!")
//	    return nil
//	})
//
// Parameters:
//   - filePath: Path to watch (empty string uses the file path from NewManagerFromFile)
//   - interval: How often to check for changes (e.g., 2 * time.Second)
//   - onReload: Optional callback function called after successful reload
//
// Fail-fast: Validates inputs before processing
func (m *Manager) Watch(filePath string, interval time.Duration, onReload func() error) error {
	// Use provided filePath or fall back to stored filePath
	watchPath := filePath
	if watchPath == "" {
		watchPath = m.filePath
	}

	// Fail-fast: filePath cannot be empty
	if watchPath == "" {
		return fmt.Errorf("fail-fast: config file path cannot be empty for watching")
	}

	// Fail-fast: interval must be positive
	if interval <= 0 {
		return fmt.Errorf("fail-fast: watch interval must be positive, got %v", interval)
	}

	// Create reload callback that reloads config and validates
	reloadCallback := func() error {
		if err := m.reloadConfig(watchPath); err != nil {
			return err
		}

		// Call user-provided callback if provided
		if onReload != nil {
			if err := onReload(); err != nil {
				return fmt.Errorf("onReload callback failed: %w", err)
			}
		}

		return nil
	}

	// Create watcher
	watcher, err := WatchConfig(watchPath, interval, reloadCallback)
	if err != nil {
		return fmt.Errorf("fail-fast: failed to start config watcher: %w", err)
	}

	m.watcher = watcher
	return nil
}

// StopWatching stops watching the configuration file
func (m *Manager) StopWatching() {
	if m.watcher != nil {
		m.watcher.Stop()
		m.watcher = nil
	}
}

// Reload manually reloads the configuration from file.
// This is useful when you want to reload on-demand rather than watching.
//
// The reload process:
//  1. Loads configuration from the file path used in NewManagerFromFile
//  2. Applies environment variable overrides (if envPrefix is set)
//  3. Validates the configuration
//
// Example:
//
//	mgr, _ := config.NewManagerFromFile("config.yaml", &cfg)
//	if err := mgr.Reload(); err != nil {
//	    log.Fatal(err)
//	}
//
// Fail-fast: Validates inputs before processing
func (m *Manager) Reload() error {
	// Fail-fast: filePath must be set
	if m.filePath == "" {
		return fmt.Errorf("fail-fast: cannot reload: file path not set")
	}

	return m.reloadConfig(m.filePath)
}

// reloadConfig is a helper method that reloads config from file and validates
func (m *Manager) reloadConfig(filePath string) error {
	// Reload config from file
	if m.envPrefix != "" {
		if err := LoadWithEnv(filePath, m.envPrefix, m.config); err != nil {
			return fmt.Errorf("failed to reload config: %w", err)
		}
	} else {
		if err := Load(filePath, m.config); err != nil {
			return fmt.Errorf("failed to reload config: %w", err)
		}
	}

	// Validate after reload
	if err := m.Validate(); err != nil {
		return fmt.Errorf("validation failed after reload: %w", err)
	}

	return nil
}

// ============================================================================
// Manager Methods - Accessors
// ============================================================================

// Get returns the configuration
func (m *Manager) Get() interface{} {
	return m.config
}

// ============================================================================
// Helper Functions
// ============================================================================
//
// These functions provide type-safe access to configuration values.
// Use them when you need to extract typed values from interface{}.
//
// ============================================================================

// GetTyped returns the configuration as the specified type.
// This provides type-safe access to configuration values.
//
// Example:
//
//	mgr, _ := config.NewManagerFromFile("config.yaml", &cfg)
//	typedCfg, err := config.GetTyped[MyConfig](mgr.Get())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// typedCfg is now of type MyConfig
func GetTyped[T any](config interface{}) (T, error) {
	var zero T
	val, ok := config.(T)
	if !ok {
		return zero, fmt.Errorf("configuration type mismatch: expected %T, got %T", zero, config)
	}
	return val, nil
}

// MustGetTyped returns the configuration as the specified type, panics on error.
// Use this when you're certain the type is correct.
//
// Example:
//
//	mgr, _ := config.NewManagerFromFile("config.yaml", &cfg)
//	typedCfg := config.MustGetTyped[MyConfig](mgr.Get())
//	// typedCfg is now of type MyConfig (panics if type mismatch)
func MustGetTyped[T any](config interface{}) T {
	val, err := GetTyped[T](config)
	if err != nil {
		panic(err)
	}
	return val
}
