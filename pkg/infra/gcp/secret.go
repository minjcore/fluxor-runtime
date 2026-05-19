package gcp

import (
	"context"
	"os"
)

// SecretManager provides access to GCP Secret Manager.
type SecretManager interface {
	// AccessSecret returns the latest version payload of the secret.
	AccessSecret(ctx context.Context, secretID string) ([]byte, error)
	// AccessSecretVersion returns payload for a specific version (e.g. "latest", "1").
	AccessSecretVersion(ctx context.Context, secretID, version string) ([]byte, error)
	// AddSecretVersion adds a new version to an existing secret.
	AddSecretVersion(ctx context.Context, secretID string, payload []byte) (string, error)
	// Close releases resources.
	Close() error
}

// SecretConfig configures the Secret Manager client.
type SecretConfig struct {
	ProjectID string
}

// DefaultSecretConfig returns config with optional env-based project ID (GCP_PROJECT).
func DefaultSecretConfig(projectID string) SecretConfig {
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT")
	}
	return SecretConfig{ProjectID: projectID}
}

// NewSecretManager creates a Secret Manager client for the given project.
// Requires: cloud.google.com/go/secretmanager.
func NewSecretManager(ctx context.Context, cfg SecretConfig) (SecretManager, error) {
	return newSecretManager(ctx, cfg)
}
