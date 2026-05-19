package gcp

import (
	"context"
	"fmt"
	"hash/crc32"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// secretManagerImpl implements SecretManager using GCP Secret Manager API.
type secretManagerImpl struct {
	client    *secretmanager.Client
	projectID string
}

// newSecretManager creates a GCP Secret Manager client.
func newSecretManager(ctx context.Context, cfg SecretConfig) (SecretManager, error) {
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("gcp: SecretConfig.ProjectID is required")
	}
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: create secretmanager client: %w", err)
	}
	return &secretManagerImpl{
		client:    client,
		projectID: cfg.ProjectID,
	}, nil
}

func (s *secretManagerImpl) secretVersionName(secretID, version string) string {
	if version == "" {
		version = "latest"
	}
	return fmt.Sprintf("projects/%s/secrets/%s/versions/%s", s.projectID, secretID, version)
}

func (s *secretManagerImpl) secretParent(secretID string) string {
	return fmt.Sprintf("projects/%s/secrets/%s", s.projectID, secretID)
}

// AccessSecret returns the latest version payload.
func (s *secretManagerImpl) AccessSecret(ctx context.Context, secretID string) ([]byte, error) {
	return s.AccessSecretVersion(ctx, secretID, "latest")
}

// AccessSecretVersion returns the payload for the given version (e.g. "latest", "1").
func (s *secretManagerImpl) AccessSecretVersion(ctx context.Context, secretID, version string) ([]byte, error) {
	name := s.secretVersionName(secretID, version)
	resp, err := s.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("gcp: access secret version %q: %w", name, err)
	}
	return resp.Payload.Data, nil
}

// AddSecretVersion adds a new version to an existing secret.
func (s *secretManagerImpl) AddSecretVersion(ctx context.Context, secretID string, payload []byte) (string, error) {
	parent := s.secretParent(secretID)
	var crc int64
	if len(payload) > 0 {
		crc = int64(crc32.Checksum(payload, crc32.MakeTable(crc32.Castagnoli)))
	}
	resp, err := s.client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: parent,
		Payload: &secretmanagerpb.SecretPayload{
			Data:       payload,
			DataCrc32C: &crc,
		},
	})
	if err != nil {
		return "", fmt.Errorf("gcp: add secret version %q: %w", parent, err)
	}
	return resp.Name, nil
}

// Close releases the client.
func (s *secretManagerImpl) Close() error {
	return s.client.Close()
}
