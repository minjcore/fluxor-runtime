package apikey

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
)

// Key represents an API key
type Key struct {
	// ID is the unique identifier of the key
	ID string

	// KeyHash is the bcrypt hashed value of the key (for verification)
	KeyHash string

	// LookupHash is a deterministic, *peppered* KDF output for fast lookup.
	// It is NOT the stored verifier (KeyHash is bcrypt for verification).
	LookupHash string

	// Prefix is the prefix shown to users (e.g., "sk_live_")
	Prefix string

	// Name is a human-readable name for the key
	Name string

	// PrincipalID is the ID of the principal this key belongs to
	PrincipalID string

	// Scopes are the permissions/scopes granted to this key
	Scopes []string

	// CreatedAt is when the key was created
	CreatedAt time.Time

	// ExpiresAt is when the key expires (nil means never expires)
	ExpiresAt *time.Time

	// LastUsedAt is when the key was last used
	LastUsedAt *time.Time

	// Revoked indicates if the key has been revoked
	Revoked bool
}

// IsExpired checks if the key has expired
func (k *Key) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsValid checks if the key is valid (not revoked and not expired)
func (k *Key) IsValid() bool {
	return !k.Revoked && !k.IsExpired()
}

// HasScope checks if the key has a specific scope
func (k *Key) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// Store is the interface for storing and retrieving API keys
type Store interface {
	// Create creates a new API key
	Create(ctx context.Context, key *Key) error

	// GetByID retrieves a key by ID
	GetByID(ctx context.Context, id string) (*Key, error)

	// GetByHash retrieves a key by its hash
	GetByHash(ctx context.Context, hash string) (*Key, error)

	// ListByPrincipal lists all keys for a principal
	ListByPrincipal(ctx context.Context, principalID string) ([]*Key, error)

	// Update updates an existing key
	Update(ctx context.Context, key *Key) error

	// Delete deletes a key
	Delete(ctx context.Context, id string) error

	// Revoke revokes a key
	Revoke(ctx context.Context, id string) error
}

// Manager manages API keys
type Manager struct {
	store    Store
	hasher   Hasher
	prefix   string
	keyLen   int
	pepper   []byte
}

// NewManager creates a new API key manager
func NewManager(store Store, hasher Hasher, opts ...ManagerOption) *Manager {
	m := &Manager{
		store:  store,
		hasher: hasher,
		prefix: "sk_",
		keyLen: 32,
		pepper: []byte(os.Getenv("FLUXOR_APIKEY_LOOKUP_PEPPER")),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// ManagerOption configures a Manager
type ManagerOption func(*Manager)

// WithPrefix sets the key prefix
func WithPrefix(prefix string) ManagerOption {
	return func(m *Manager) {
		m.prefix = prefix
	}
}

// WithKeyLength sets the key length
func WithKeyLength(length int) ManagerOption {
	return func(m *Manager) {
		m.keyLen = length
	}
}

// WithLookupPepper sets the pepper used for deterministic lookup hashing.
//
// This pepper MUST remain stable for the lifetime of stored keys, otherwise
// previously issued keys cannot be looked up.
//
// Recommended: set from a secret manager or env var.
func WithLookupPepper(pepper string) ManagerOption {
	return func(m *Manager) {
		m.pepper = []byte(pepper)
	}
}

// Generate generates a new API key
func (m *Manager) Generate(ctx context.Context, principalID, name string, scopes []string, expiresIn *time.Duration) (string, *Key, error) {
	// Generate random key material
	keyBytes := make([]byte, m.keyLen)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Encode to base64
	keyMaterial := base64.URLEncoding.EncodeToString(keyBytes)
	fullKey := m.prefix + keyMaterial

	// Create deterministic lookup hash for fast retrieval
	lookupHash := m.createLookupHash(fullKey)

	// Hash the key with bcrypt for secure storage
	keyHash, err := m.hasher.Hash(fullKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to hash key: %w", err)
	}

	// Create key record
	key := &Key{
		ID:         generateID(),
		KeyHash:    keyHash,
		LookupHash: lookupHash,
		Prefix:     m.prefix,
		Name:       name,
		PrincipalID: principalID,
		Scopes:     scopes,
		CreatedAt:  time.Now(),
	}

	if expiresIn != nil {
		expiresAt := time.Now().Add(*expiresIn)
		key.ExpiresAt = &expiresAt
	}

	// Store the key
	if err := m.store.Create(ctx, key); err != nil {
		return "", nil, fmt.Errorf("failed to store key: %w", err)
	}

	return fullKey, key, nil
}

// Validate validates an API key and returns the key record
func (m *Manager) Validate(ctx context.Context, apiKey string) (*Key, error) {
	if apiKey == "" {
		return nil, authn.ErrMissingCredential
	}

	// Create lookup hash for fast retrieval
	lookupHash := m.createLookupHash(apiKey)

	// Look up the key using the deterministic hash
	key, err := m.store.GetByHash(ctx, lookupHash)
	if err != nil {
		return nil, authn.ErrInvalidCredential
	}

	// Verify the key against the stored bcrypt hash
	if !m.hasher.Verify(apiKey, key.KeyHash) {
		return nil, authn.ErrInvalidCredential
	}

	// Check if key is valid
	if !key.IsValid() {
		if key.IsExpired() {
			return nil, authn.ErrExpiredCredential
		}
		return nil, authn.ErrInvalidCredential
	}

	// Update last used timestamp
	now := time.Now()
	key.LastUsedAt = &now
	if err := m.store.Update(ctx, key); err != nil {
		// Log error but don't fail authentication
		_ = err
	}

	return key, nil
}

// createLookupHash creates a deterministic, peppered lookup value for key retrieval.
//
// Security rationale:
// - The verifier is stored as bcrypt (KeyHash).
// - The lookup hash exists only to index the key quickly; it must be deterministic.
// - Using a slow KDF (scrypt) + server-side pepper reduces offline brute-force risk
//   in case the lookup index is leaked.
func (m *Manager) createLookupHash(key string) string {
	pepper := m.pepper
	if len(pepper) == 0 {
		// Backward compatible default for dev/test. Production should set
		// FLUXOR_APIKEY_LOOKUP_PEPPER or pass WithLookupPepper.
		pepper = []byte("fluxor-dev-insecure-default-pepper-change-me")
	}

	// N=32768, r=8, p=1 is a moderate cost for online authentication lookups.
	derived, err := scrypt.Key([]byte(key), pepper, 32768, 8, 1, 32)
	if err != nil {
		// Fail closed: do not allow lookups without a hash.
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(derived)
}

// Revoke revokes an API key
func (m *Manager) Revoke(ctx context.Context, keyID string) error {
	return m.store.Revoke(ctx, keyID)
}

// List lists all keys for a principal
func (m *Manager) List(ctx context.Context, principalID string) ([]*Key, error) {
	return m.store.ListByPrincipal(ctx, principalID)
}

// Delete deletes an API key
func (m *Manager) Delete(ctx context.Context, keyID string) error {
	return m.store.Delete(ctx, keyID)
}

// Hasher hashes API keys for storage
type Hasher interface {
	// Hash hashes a key
	Hash(key string) (string, error)

	// Verify verifies a key against a hash
	Verify(key, hash string) bool
}

// BcryptHasher uses bcrypt for hashing
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher creates a new bcrypt hasher
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// Hash hashes a key using bcrypt
func (h *BcryptHasher) Hash(key string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(key), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash key: %w", err)
	}
	return string(hash), nil
}

// Verify verifies a key against a hash
func (h *BcryptHasher) Verify(key, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(key))
	return err == nil
}

// generateID generates a unique ID for a key
func generateID() string {
	// In production, use a proper ID generator (UUID, etc.)
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
