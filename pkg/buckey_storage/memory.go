package buckey_storage

import (
	"context"
	"sync"
)

// MemoryStorage is an in-memory implementation of BlobStorage.
// It is safe for concurrent use.
type MemoryStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
	cfg  *Config
}

// NewMemoryStorage creates an in-memory blob storage with default config.
func NewMemoryStorage() *MemoryStorage {
	cfg := DefaultConfig()
	return NewMemory(&cfg)
}

// NewMemory creates an in-memory blob storage with the given config.
// Prefix from config is applied to all keys.
func NewMemory(cfg *Config) *MemoryStorage {
	if cfg == nil {
		c := DefaultConfig()
		cfg = &c
	}
	return &MemoryStorage{
		data: make(map[string][]byte),
		cfg:  cfg,
	}
}

func (m *MemoryStorage) fullKey(key string) string {
	if m.cfg != nil && m.cfg.Prefix != "" {
		return m.cfg.Prefix + key
	}
	return key
}

// Put stores data under key.
func (m *MemoryStorage) Put(ctx context.Context, key string, data []byte) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	if data == nil {
		data = []byte{}
	}
	full := m.fullKey(key)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[full] = append([]byte(nil), data...)
	return nil
}

// Get retrieves data by key.
func (m *MemoryStorage) Get(ctx context.Context, key string) ([]byte, error) {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return nil, err
	}
	full := m.fullKey(key)
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.data == nil {
		return nil, ErrNotFound
	}
	b, ok := m.data[full]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte(nil), b...), nil
}

// Delete removes the blob at key.
func (m *MemoryStorage) Delete(ctx context.Context, key string) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	full := m.fullKey(key)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, full)
	return nil
}

// List returns keys with the given prefix (without the config prefix).
// Returned keys are logical keys (without config prefix).
func (m *MemoryStorage) List(ctx context.Context, prefix string) ([]string, error) {
	ValidateContext(ctx)
	fullPrefix := m.fullKey(prefix)
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.data == nil {
		return nil, nil
	}
	var out []string
	cfgPrefix := ""
	if m.cfg != nil && m.cfg.Prefix != "" {
		cfgPrefix = m.cfg.Prefix
	}
	for k := range m.data {
		if !hasPrefix(k, fullPrefix) {
			continue
		}
		logical := k
		if cfgPrefix != "" && len(k) >= len(cfgPrefix) {
			logical = k[len(cfgPrefix):]
		}
		out = append(out, logical)
	}
	return out, nil
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
