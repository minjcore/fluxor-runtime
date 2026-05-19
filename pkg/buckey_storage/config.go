package buckey_storage

import (
	"fmt"
	"strings"
)

// Backend type for blob storage.
const (
	BackendMemory     = "memory"
	BackendReplicated = "replicated"
	BackendS3         = "s3"
	BackendFS         = "fs"
)

// Config configures blob storage. It can be loaded from JSON (e.g. storage.json).
type Config struct {
	// Backend selects the implementation: "memory" (required for now).
	Backend string `json:"backend"`

	// Prefix is an optional key prefix applied to all keys (e.g. "app/").
	Prefix string `json:"prefix,omitempty"`

	// Bucket is reserved for S3-style backends (future).
	Bucket string `json:"bucket,omitempty"`

	// Path is reserved for local filesystem backends (future).
	Path string `json:"path,omitempty"`

	// Replicas is the number of copies per key on one server (for backend "replicated"). Default 3.
	Replicas int `json:"replicas,omitempty"`

	// Region, AccessKeyID, SecretAccessKey for S3 backend (or use env AWS_*).
	Region       string `json:"region,omitempty"`
	AccessKeyID  string `json:"accessKeyID,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`

	// DefaultTTL is optional TTL for Put (0 = no expiry). Supported by memory/replicated and TTL wrapper.
	DefaultTTLSeconds int `json:"defaultTTLSeconds,omitempty"`
}

// DefaultConfig returns a default in-memory storage configuration.
func DefaultConfig() Config {
	return Config{
		Backend: BackendMemory,
		Prefix:  "",
	}
}

// Validate checks that config is valid. Returns an error if backend is empty or unsupported.
func (c *Config) Validate() error {
	backend := strings.TrimSpace(c.Backend)
	if backend == "" {
		return fmt.Errorf("fail-fast: buckey_storage backend cannot be empty")
	}
	switch backend {
	case BackendMemory, BackendReplicated, BackendS3, BackendFS:
		return nil
	default:
		return fmt.Errorf("fail-fast: unsupported buckey_storage backend %q", backend)
	}
}
