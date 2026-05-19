package zalo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// tokenFileSchema is the JSON structure persisted on disk.
type tokenFileSchema struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"` // RFC3339
}

// FileTokenStore implements TokenStore by reading/writing a single JSON file.
// Safe for concurrent use; uses mutex and atomic write (temp file + rename).
type FileTokenStore struct {
	path string
	mu   sync.Mutex
}

// NewFileTokenStore returns a new file-based token store. The file is created
// on first StoreTokens call; GetAccessToken returns an error if the file does not exist.
func NewFileTokenStore(path string) *FileTokenStore {
	return &FileTokenStore{path: path}
}

// GetAccessToken reads the token file and returns stored tokens and expiry.
// Returns an error if the file is missing or unreadable.
func (f *FileTokenStore) GetAccessToken(ctx context.Context) (accessToken, refreshToken string, expiresAt time.Time, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := os.ReadFile(f.path)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("read token file: %w", err)
	}

	var s tokenFileSchema
	if err := json.Unmarshal(data, &s); err != nil {
		return "", "", time.Time{}, fmt.Errorf("parse token file: %w", err)
	}
	if s.AccessToken == "" {
		return "", "", time.Time{}, fmt.Errorf("token file: empty access_token")
	}

	expiresAt, err = time.Parse(time.RFC3339, s.ExpiresAt)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("token file: invalid expires_at: %w", err)
	}

	return s.AccessToken, s.RefreshToken, expiresAt, nil
}

// StoreTokens writes access token, refresh token, and expiry to the file.
// expiresIn is in seconds; expires_at is computed as now + expiresIn.
// Uses atomic write (temp file in same dir then rename). File mode 0600.
func (f *FileTokenStore) StoreTokens(ctx context.Context, accessToken, refreshToken string, expiresIn int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	s := tokenFileSchema{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt.UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal token file: %w", err)
	}

	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create token file dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(f.path)+".*")
	if err != nil {
		return fmt.Errorf("create temp token file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if err := os.Chmod(tmpPath, 0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod token file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write token file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync token file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp token file: %w", err)
	}
	if err := os.Rename(tmpPath, f.path); err != nil {
		return fmt.Errorf("rename token file: %w", err)
	}
	return nil
}

// Bootstrap creates or overwrites the token file with the given tokens.
// Use this on first deploy when token file does not exist; expiresInSeconds
// can come from env (e.g. ZALO_ZNS_TOKEN_EXPIRES_IN) or a default (e.g. 3600).
func (f *FileTokenStore) Bootstrap(accessToken, refreshToken string, expiresInSeconds int) error {
	if accessToken == "" || refreshToken == "" {
		return fmt.Errorf("bootstrap: access_token and refresh_token required")
	}
	if expiresInSeconds <= 0 {
		expiresInSeconds = 3600
	}
	return f.StoreTokens(context.Background(), accessToken, refreshToken, expiresInSeconds)
}
