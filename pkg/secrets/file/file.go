package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
	"gopkg.in/yaml.v3"
)

// FileProvider is a file-based secret provider that reads secrets from YAML, JSON, or properties files
type FileProvider struct {
	config     *Config
	secrets    map[string]interface{}
	mu         sync.RWMutex
	format     string
	watcher    *fileWatcher
	ctx        context.Context
	cancel     context.CancelFunc
}

// New creates a new file-based secret provider
// Fail-fast: Validates inputs before processing
func New(config *Config) (*FileProvider, error) {
	return NewWithContext(context.Background(), config)
}

// NewWithContext creates a new file-based secret provider with a context
// Fail-fast: Validates inputs before processing
func NewWithContext(ctx context.Context, config *Config) (*FileProvider, error) {
	// Fail-fast: config cannot be nil
	failfast.NotNil(config, "config")
	// Fail-fast: context cannot be nil
	failfast.NotNil(ctx, "context")

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("fail-fast: invalid config: %w", err)
	}

	providerCtx, cancel := context.WithCancel(ctx)
	provider := &FileProvider{
		config:  config,
		secrets: make(map[string]interface{}),
		ctx:     providerCtx,
		cancel:  cancel,
	}

	// Detect format if not specified
	if config.Format == "" {
		provider.format = DetectFormat(config.Path)
	} else {
		provider.format = config.Format
	}

	// Load secrets from file
	if err := provider.Reload(); err != nil {
		cancel()
		return nil, fmt.Errorf("fail-fast: failed to load secrets: %w", err)
	}

	// Start file watcher if enabled
	if config.WatchInterval > 0 {
		watcher, err := newFileWatcher(providerCtx, config.Path, config.WatchInterval, provider.Reload)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("fail-fast: failed to start file watcher: %w", err)
		}
		provider.watcher = watcher
	}

	return provider, nil
}

// NewFromPath creates a new file-based secret provider from a file path
// Format is auto-detected from file extension
// Fail-fast: Validates inputs before processing
func NewFromPath(path string) (*FileProvider, error) {
	return NewFromPathWithContext(context.Background(), path)
}

// NewFromPathWithContext creates a new file-based secret provider from a file path with a context
// Format is auto-detected from file extension
// Fail-fast: Validates inputs before processing
func NewFromPathWithContext(ctx context.Context, path string) (*FileProvider, error) {
	// Fail-fast: path cannot be empty
	if path == "" {
		return nil, fmt.Errorf("fail-fast: file path cannot be empty")
	}

	config := DefaultConfig()
	config.Path = path
	return NewWithContext(ctx, config)
}

// Reload reloads secrets from the file
// Fail-fast: Validates inputs and fails immediately on errors
func (p *FileProvider) Reload() error {
	return p.ReloadWithContext(p.ctx)
}

// ReloadWithContext reloads secrets from the file with a context
// Fail-fast: Validates inputs and fails immediately on errors
func (p *FileProvider) ReloadWithContext(ctx context.Context) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("fail-fast: context cancelled: %w", ctx.Err())
	default:
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if file exists
	if _, err := os.Stat(p.config.Path); os.IsNotExist(err) {
		return &ErrFileNotReadable{Path: p.config.Path, Err: err}
	}

	// Read file
	// #nosec G304 -- path is provided by the caller (library function); callers should validate/lock down inputs if untrusted.
	data, err := os.ReadFile(p.config.Path)
	if err != nil {
		return &ErrFileNotReadable{Path: p.config.Path, Err: err}
	}

	// Fail-fast: file cannot be empty
	if len(data) == 0 {
		return fmt.Errorf("fail-fast: secrets file %s is empty", p.config.Path)
	}

	// Parse based on format
	var secrets map[string]interface{}
	switch p.format {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &secrets); err != nil {
			return &ErrInvalidFileFormat{Path: p.config.Path, Format: p.format, Err: err}
		}
	case "json":
		if err := json.Unmarshal(data, &secrets); err != nil {
			return &ErrInvalidFileFormat{Path: p.config.Path, Format: p.format, Err: err}
		}
	case "properties":
		secrets, err = parseProperties(data)
		if err != nil {
			return &ErrInvalidFileFormat{Path: p.config.Path, Format: p.format, Err: err}
		}
	default:
		return &ErrInvalidFileFormat{
			Path:   p.config.Path,
			Format: p.format,
			Err:    fmt.Errorf("unsupported format: %s", p.format),
		}
	}

	// Flatten nested maps for easier access
	p.secrets = flattenMap(secrets, p.config.KeyPrefix)
	return nil
}

// GetSecret retrieves a secret by key from the file
// Fail-fast: Validates inputs before processing
func (p *FileProvider) GetSecret(key string) (string, error) {
	return p.GetSecretWithContext(p.ctx, key)
}

// GetSecretWithContext retrieves a secret by key from the file with a context
// Fail-fast: Validates inputs before processing
func (p *FileProvider) GetSecretWithContext(ctx context.Context, key string) (string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("fail-fast: context cancelled: %w", ctx.Err())
	default:
	}

	// Fail-fast: key cannot be empty
	if key == "" {
		return "", fmt.Errorf("fail-fast: secret key cannot be empty")
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Build full key with prefix
	fullKey := key
	if p.config.KeyPrefix != "" {
		fullKey = p.config.KeyPrefix + key
	}

	value, ok := p.secrets[fullKey]
	if !ok {
		// Try without prefix as fallback
		value, ok = p.secrets[key]
		if !ok {
			return "", &ErrSecretNotFound{Key: key}
		}
	}

	// Convert to string
	return convertToString(value), nil
}

// GetSecretBytes retrieves a secret by key as bytes from the file
// Fail-fast: Validates inputs before processing
func (p *FileProvider) GetSecretBytes(key string) ([]byte, error) {
	return p.GetSecretBytesWithContext(p.ctx, key)
}

// GetSecretBytesWithContext retrieves a secret by key as bytes from the file with a context
// Fail-fast: Validates inputs before processing
func (p *FileProvider) GetSecretBytesWithContext(ctx context.Context, key string) ([]byte, error) {
	// Fail-fast: key cannot be empty
	if key == "" {
		return nil, fmt.Errorf("fail-fast: secret key cannot be empty")
	}

	str, err := p.GetSecretWithContext(ctx, key)
	if err != nil {
		return nil, err
	}

	return []byte(str), nil
}

// ListSecrets returns all secret keys available in the file
func (p *FileProvider) ListSecrets() ([]string, error) {
	return p.ListSecretsWithContext(p.ctx)
}

// ListSecretsWithContext returns all secret keys available in the file with a context
func (p *FileProvider) ListSecretsWithContext(ctx context.Context) ([]string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("fail-fast: context cancelled: %w", ctx.Err())
	default:
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	keys := make([]string, 0, len(p.secrets))
	for key := range p.secrets {
		// Remove prefix if present
		if p.config.KeyPrefix != "" && strings.HasPrefix(key, p.config.KeyPrefix) {
			key = strings.TrimPrefix(key, p.config.KeyPrefix)
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// Close stops the file watcher and releases resources
func (p *FileProvider) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.watcher != nil {
		p.watcher.stop()
	}
	return nil
}

// parseProperties parses a properties file into a map
func parseProperties(data []byte) (map[string]interface{}, error) {
	props := make(map[string]interface{})
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle key=value pairs
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			props[key] = value
		}
	}

	return props, nil
}

// flattenMap flattens a nested map structure into a single-level map with dot-notation keys
// Example: {"db": {"host": "localhost"}} becomes {"db.host": "localhost"}
func flattenMap(m map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	flattenMapRecursive(m, prefix, "", result)
	return result
}

// flattenMapRecursive recursively flattens nested maps
func flattenMapRecursive(m map[string]interface{}, keyPrefix, currentPrefix string, result map[string]interface{}) {
	for key, value := range m {
		var fullKey string
		if currentPrefix == "" {
			fullKey = key
		} else {
			fullKey = currentPrefix + "." + key
		}

		// If value is a nested map, recurse (don't add prefix yet)
		if nestedMap, ok := value.(map[string]interface{}); ok {
			flattenMapRecursive(nestedMap, keyPrefix, fullKey, result)
		} else {
			// Add key prefix only when storing the final value
			if keyPrefix != "" {
				fullKey = keyPrefix + fullKey
			}
			result[fullKey] = value
		}
	}
}

// convertToString converts a value to string
func convertToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// fileWatcher watches a file for changes and triggers reload callbacks
type fileWatcher struct {
	filePath  string
	interval  time.Duration
	callback  func() error
	mu        sync.RWMutex
	lastMod   time.Time
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// newFileWatcher creates a new file watcher
func newFileWatcher(ctx context.Context, filePath string, interval time.Duration, callback func() error) (*fileWatcher, error) {
	// Get initial file modification time
	var lastMod time.Time
	if info, err := os.Stat(filePath); err == nil {
		lastMod = info.ModTime()
	}

	watcherCtx, cancel := context.WithCancel(ctx)
	watcher := &fileWatcher{
		filePath: filePath,
		interval: interval,
		callback: callback,
		ctx:      watcherCtx,
		cancel:   cancel,
		lastMod:  lastMod,
	}

	// Start watching
	watcher.wg.Add(1)
	go watcher.watch()

	return watcher, nil
}

// watch periodically checks for file changes
func (fw *fileWatcher) watch() {
	defer fw.wg.Done()

	ticker := time.NewTicker(fw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-fw.ctx.Done():
			return
		case <-ticker.C:
			fw.checkForChanges()
		}
	}
}

// checkForChanges checks if the file has changed
func (fw *fileWatcher) checkForChanges() {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Get file modification time
	info, err := os.Stat(fw.filePath)
	if err != nil {
		// File doesn't exist or can't be accessed - skip this check
		return
	}

	modTime := info.ModTime()

	// Check if file has changed
	if modTime.After(fw.lastMod) {
		fw.lastMod = modTime

		// Call reload callback
		if fw.callback != nil {
			if err := fw.callback(); err != nil {
				// Log error but continue watching
				// In production, you might want to use a logger here
				_ = err
			}
		}
	}
}

// stop stops watching for changes
func (fw *fileWatcher) stop() {
	fw.cancel()
	fw.wg.Wait()
}

// ============================================================================
// Helper Functions
// ============================================================================

// DetectFormat detects the file format from the file extension
func DetectFormat(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".properties", ".props", ".conf":
		return "properties"
	default:
		return "yaml" // Default to YAML
	}
}

// ValidateFileExists checks if a file exists and is readable
func ValidateFileExists(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("fail-fast: file path cannot be empty")
	}

	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return &ErrFileNotReadable{Path: filePath, Err: err}
	}
	if err != nil {
		return &ErrFileNotReadable{Path: filePath, Err: err}
	}

	if info.IsDir() {
		return &ErrFileNotReadable{Path: filePath, Err: fmt.Errorf("path is a directory, not a file")}
	}

	return nil
}

// GetSecretOrDefault retrieves a secret by key, returning a default value if not found
func (p *FileProvider) GetSecretOrDefault(key string, defaultValue string) string {
	value, err := p.GetSecret(key)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetSecretOrDefaultWithContext retrieves a secret by key with context, returning a default value if not found
func (p *FileProvider) GetSecretOrDefaultWithContext(ctx context.Context, key string, defaultValue string) string {
	value, err := p.GetSecretWithContext(ctx, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// HasSecret checks if a secret key exists
func (p *FileProvider) HasSecret(key string) bool {
	_, err := p.GetSecret(key)
	return err == nil
}

// HasSecretWithContext checks if a secret key exists with context
func (p *FileProvider) HasSecretWithContext(ctx context.Context, key string) bool {
	_, err := p.GetSecretWithContext(ctx, key)
	return err == nil
}
