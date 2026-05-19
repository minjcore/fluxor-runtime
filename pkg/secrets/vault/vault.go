package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// VaultProvider is a HashiCorp Vault KV v2 secret provider
type VaultProvider struct {
	client    *vaultapi.Client
	kv        *vaultapi.KVv2
	config    *Config
	ctx       context.Context
	cancel    context.CancelFunc
}

// New creates a new Vault secret provider
func New(config *Config) (*VaultProvider, error) {
	return NewWithContext(context.Background(), config)
}

// NewWithContext creates a new Vault secret provider with a context.
func NewWithContext(ctx context.Context, config *Config) (*VaultProvider, error) {
	failfast.NotNil(config, "config")
	failfast.NotNil(ctx, "context")

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("fail-fast: invalid config: %w", err)
	}

	apiConfig := vaultapi.DefaultConfig()
	apiConfig.Address = config.Address
	if config.Timeout > 0 {
		apiConfig.Timeout = config.Timeout
	}

	client, err := vaultapi.NewClient(apiConfig)
	if err != nil {
		return nil, &ErrVaultConnection{Err: err}
	}

	client.SetToken(config.Token)
	if config.Namespace != "" {
		client.SetNamespace(config.Namespace)
	}

	providerCtx, cancel := context.WithCancel(ctx)
	p := &VaultProvider{
		client: client,
		kv:     client.KVv2(config.MountPath),
		config: config,
		ctx:    providerCtx,
		cancel: cancel,
	}
	return p, nil
}

// GetSecret retrieves a secret by path. The key is the path in Vault (e.g. "myapp/db").
// If the secret has a single data key, its value is returned; otherwise JSON-encoded data is returned.
func (p *VaultProvider) GetSecret(key string) (string, error) {
	return p.GetSecretWithContext(p.ctx, key)
}

// GetSecretWithContext retrieves a secret by path with context
func (p *VaultProvider) GetSecretWithContext(ctx context.Context, key string) (string, error) {
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("fail-fast: context cancelled: %w", ctx.Err())
	default:
	}

	if key == "" {
		return "", fmt.Errorf("fail-fast: secret path cannot be empty")
	}

	secret, err := p.kv.Get(ctx, key)
	if err != nil {
		if isNotFound(err) {
			return "", &ErrSecretNotFound{Path: key}
		}
		return "", &ErrVaultConnection{Err: err}
	}

	if secret == nil || secret.Data == nil {
		return "", &ErrSecretNotFound{Path: key}
	}

	return dataToString(secret.Data), nil
}

// GetSecretBytes retrieves a secret by path as bytes
func (p *VaultProvider) GetSecretBytes(key string) ([]byte, error) {
	return p.GetSecretBytesWithContext(p.ctx, key)
}

// GetSecretBytesWithContext retrieves a secret by path as bytes with context
func (p *VaultProvider) GetSecretBytesWithContext(ctx context.Context, key string) ([]byte, error) {
	s, err := p.GetSecretWithContext(ctx, key)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// ListSecrets lists secret paths under the mount (top-level only)
func (p *VaultProvider) ListSecrets() ([]string, error) {
	return p.ListSecretsWithContext(p.ctx)
}

// ListSecretsWithContext lists secret paths with context
func (p *VaultProvider) ListSecretsWithContext(ctx context.Context) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("fail-fast: context cancelled: %w", ctx.Err())
	default:
	}

	// KV v2 list: GET mount_path/metadata/[path]
	path := p.config.MountPath + "/metadata"
	secret, err := p.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, &ErrVaultConnection{Err: err}
	}
	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	keysRaw, ok := secret.Data["keys"]
	if !ok {
		return []string{}, nil
	}

	keysSlice, ok := keysRaw.([]interface{})
	if !ok {
		return []string{}, nil
	}

	out := make([]string, 0, len(keysSlice))
	for _, k := range keysSlice {
		if s, ok := k.(string); ok {
			out = append(out, strings.TrimSuffix(s, "/"))
		}
	}
	return out, nil
}

// Reload is a no-op for Vault (secrets are read on demand)
func (p *VaultProvider) Reload() error {
	return p.ReloadWithContext(p.ctx)
}

// ReloadWithContext is a no-op for Vault
func (p *VaultProvider) ReloadWithContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("fail-fast: context cancelled: %w", ctx.Err())
	default:
	}
	return nil
}

// Close releases resources
func (p *VaultProvider) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

// GetSecretOrDefault returns the secret or a default value if not found
func (p *VaultProvider) GetSecretOrDefault(key string, defaultValue string) string {
	v, err := p.GetSecret(key)
	if err != nil {
		return defaultValue
	}
	return v
}

// GetSecretOrDefaultWithContext returns the secret or default with context
func (p *VaultProvider) GetSecretOrDefaultWithContext(ctx context.Context, key string, defaultValue string) string {
	v, err := p.GetSecretWithContext(ctx, key)
	if err != nil {
		return defaultValue
	}
	return v
}

// HasSecret returns true if the path exists
func (p *VaultProvider) HasSecret(key string) bool {
	_, err := p.GetSecret(key)
	return err == nil
}

// HasSecretWithContext returns true if the path exists (with context)
func (p *VaultProvider) HasSecretWithContext(ctx context.Context, key string) bool {
	_, err := p.GetSecretWithContext(ctx, key)
	return err == nil
}

func dataToString(data map[string]interface{}) string {
	if len(data) == 0 {
		return ""
	}
	if len(data) == 1 {
		for _, v := range data {
			return fmt.Sprintf("%v", v)
		}
	}
	b, _ := json.Marshal(data)
	return string(b)
}

func isNotFound(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "secret not found") ||
		strings.Contains(err.Error(), "404") ||
		strings.Contains(err.Error(), "no value found"))
}
