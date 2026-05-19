package buckey_storage

import (
	"fmt"
	"strings"
)

// NewFromConfig creates a BlobStorage from the given config.
// Supported backends: "memory", "replicated", "s3", "fs".
func NewFromConfig(cfg *Config) (BlobStorage, error) {
	if cfg == nil {
		c := DefaultConfig()
		cfg = &c
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	backend := strings.TrimSpace(cfg.Backend)
	switch backend {
	case BackendMemory:
		return NewMemory(cfg), nil
	case BackendReplicated:
		return NewReplicated(cfg), nil
	case BackendS3:
		return NewS3FromConfig(cfg)
	case BackendFS:
		if cfg.Path == "" {
			return nil, fmt.Errorf("buckey_storage: FS backend requires Config.Path")
		}
		return NewFS(cfg.Path, cfg.Prefix)
	default:
		return nil, fmt.Errorf("buckey_storage: unsupported backend %q", backend)
	}
}
