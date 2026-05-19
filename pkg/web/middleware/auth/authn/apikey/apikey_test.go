package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
)

func TestKey_IsExpired(t *testing.T) {
	tests := []struct {
		name string
		key  *Key
		want bool
	}{
		{
			name: "not expired - no expiration",
			key: &Key{
				ExpiresAt: nil,
			},
			want: false,
		},
		{
			name: "not expired - future expiration",
			key: &Key{
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(time.Hour)
					return &t
				}(),
			},
			want: false,
		},
		{
			name: "expired",
			key: &Key{
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(-time.Hour)
					return &t
				}(),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key.IsExpired(); got != tt.want {
				t.Errorf("Key.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKey_IsValid(t *testing.T) {
	tests := []struct {
		name string
		key  *Key
		want bool
	}{
		{
			name: "valid - not revoked, not expired",
			key: &Key{
				Revoked:   false,
				ExpiresAt: nil,
			},
			want: true,
		},
		{
			name: "invalid - revoked",
			key: &Key{
				Revoked:   true,
				ExpiresAt: nil,
			},
			want: false,
		},
		{
			name: "invalid - expired",
			key: &Key{
				Revoked: false,
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(-time.Hour)
					return &t
				}(),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.key.IsValid(); got != tt.want {
				t.Errorf("Key.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKey_HasScope(t *testing.T) {
	key := &Key{
		Scopes: []string{"read", "write", "admin"},
	}

	tests := []struct {
		name  string
		scope string
		want  bool
	}{
		{
			name:  "has scope",
			scope: "read",
			want:  true,
		},
		{
			name:  "does not have scope",
			scope: "delete",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := key.HasScope(tt.scope); got != tt.want {
				t.Errorf("Key.HasScope() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_Generate(t *testing.T) {
	store := NewMemoryStore()
	hasher := NewBcryptHasher(10)
	manager := NewManager(store, hasher)

	ctx := context.Background()
	key, keyRecord, err := manager.Generate(ctx, "user123", "Test Key", []string{"read", "write"}, nil)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if key == "" {
		t.Error("Generate() key should not be empty")
	}

	if keyRecord == nil {
		t.Error("Generate() keyRecord should not be nil")
	}

	if keyRecord.PrincipalID != "user123" {
		t.Errorf("Generate() PrincipalID = %v, want %v", keyRecord.PrincipalID, "user123")
	}

	if keyRecord.Name != "Test Key" {
		t.Errorf("Generate() Name = %v, want %v", keyRecord.Name, "Test Key")
	}
}

func TestManager_Validate(t *testing.T) {
	store := NewMemoryStore()
	hasher := NewBcryptHasher(10)
	manager := NewManager(store, hasher)

	ctx := context.Background()

	// Generate a key
	key, _, err := manager.Generate(ctx, "user123", "Test Key", []string{"read"}, nil)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Validate the key
	keyRecord, err := manager.Validate(ctx, key)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if keyRecord == nil {
		t.Error("Validate() keyRecord should not be nil")
	}

	if keyRecord.PrincipalID != "user123" {
		t.Errorf("Validate() PrincipalID = %v, want %v", keyRecord.PrincipalID, "user123")
	}

	// Test invalid key
	_, err = manager.Validate(ctx, "invalid-key")
	if err == nil {
		t.Error("Validate() should return error for invalid key")
	}
	if err != authn.ErrInvalidCredential {
		t.Errorf("Validate() error = %v, want %v", err, authn.ErrInvalidCredential)
	}
}

func TestAuthenticator_Authenticate(t *testing.T) {
	store := NewMemoryStore()
	hasher := NewBcryptHasher(10)
	manager := NewManager(store, hasher)
	authenticator := NewAuthenticator(manager)

	ctx := context.Background()

	// Generate a key
	key, _, err := manager.Generate(ctx, "user123", "Test Key", []string{"read", "write"}, nil)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Authenticate
	principal, err := authenticator.Authenticate(ctx, key)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if principal == nil {
		t.Error("Authenticate() principal should not be nil")
	}

	if principal.ID != "user123" {
		t.Errorf("Authenticate() ID = %v, want %v", principal.ID, "user123")
	}

	if principal.Type != "api_key" {
		t.Errorf("Authenticate() Type = %v, want %v", principal.Type, "api_key")
	}

	// Test empty credential
	_, err = authenticator.Authenticate(ctx, "")
	if err != authn.ErrMissingCredential {
		t.Errorf("Authenticate() error = %v, want %v", err, authn.ErrMissingCredential)
	}

	// Test invalid credential
	_, err = authenticator.Authenticate(ctx, "invalid-key")
	if err == nil {
		t.Error("Authenticate() should return error for invalid key")
	}
}

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	key := &Key{
		ID:          "key1",
		PrincipalID: "user123",
		Name:        "Test Key",
		LookupHash:  "hash123",
		CreatedAt:   time.Now(),
	}

	// Test Create
	err := store.Create(ctx, key)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Test GetByID
	retrieved, err := store.GetByID(ctx, "key1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if retrieved.ID != "key1" {
		t.Errorf("GetByID() ID = %v, want %v", retrieved.ID, "key1")
	}

	// Test GetByHash
	retrieved, err = store.GetByHash(ctx, "hash123")
	if err != nil {
		t.Fatalf("GetByHash() error = %v", err)
	}
	if retrieved.LookupHash != "hash123" {
		t.Errorf("GetByHash() LookupHash = %v, want %v", retrieved.LookupHash, "hash123")
	}

	// Test ListByPrincipal
	keys, err := store.ListByPrincipal(ctx, "user123")
	if err != nil {
		t.Fatalf("ListByPrincipal() error = %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("ListByPrincipal() len = %v, want %v", len(keys), 1)
	}

	// Test Update
	key.Name = "Updated Key"
	err = store.Update(ctx, key)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Test Revoke
	err = store.Revoke(ctx, "key1")
	if err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	retrieved, _ = store.GetByID(ctx, "key1")
	if !retrieved.Revoked {
		t.Error("Revoke() should set Revoked to true")
	}

	// Test Delete
	err = store.Delete(ctx, "key1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = store.GetByID(ctx, "key1")
	if err != ErrKeyNotFound {
		t.Errorf("Delete() should remove key, error = %v", err)
	}
}

func TestBcryptHasher(t *testing.T) {
	hasher := NewBcryptHasher(10)
	key := "test-key-123"

	hash, err := hasher.Hash(key)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if hash == "" {
		t.Error("Hash() should return non-empty hash")
	}

	if !hasher.Verify(key, hash) {
		t.Error("Verify() should return true for correct key")
	}

	if hasher.Verify("wrong-key", hash) {
		t.Error("Verify() should return false for wrong key")
	}
}
