package file

import "context"

// Provider is the interface for file-based secret providers
type Provider interface {
	// GetSecret retrieves a secret by key from the file
	GetSecret(key string) (string, error)

	// GetSecretWithContext retrieves a secret by key from the file with a context
	GetSecretWithContext(ctx context.Context, key string) (string, error)

	// GetSecretBytes retrieves a secret by key as bytes from the file
	GetSecretBytes(key string) ([]byte, error)

	// GetSecretBytesWithContext retrieves a secret by key as bytes from the file with a context
	GetSecretBytesWithContext(ctx context.Context, key string) ([]byte, error)

	// ListSecrets returns all secret keys available in the file
	ListSecrets() ([]string, error)

	// ListSecretsWithContext returns all secret keys available in the file with a context
	ListSecretsWithContext(ctx context.Context) ([]string, error)

	// Reload reloads secrets from the file
	Reload() error

	// ReloadWithContext reloads secrets from the file with a context
	ReloadWithContext(ctx context.Context) error

	// Close stops the file watcher and releases resources
	Close() error
}
