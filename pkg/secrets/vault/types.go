package vault

import "context"

// Provider is the interface for Vault-based secret providers
type Provider interface {
	GetSecret(key string) (string, error)
	GetSecretWithContext(ctx context.Context, key string) (string, error)
	GetSecretBytes(key string) ([]byte, error)
	GetSecretBytesWithContext(ctx context.Context, key string) ([]byte, error)
	ListSecrets() ([]string, error)
	ListSecretsWithContext(ctx context.Context) ([]string, error)
	Reload() error
	ReloadWithContext(ctx context.Context) error
	Close() error
}
