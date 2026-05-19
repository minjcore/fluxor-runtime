// Package apikey provides API key authentication functionality.
//
// This package includes:
//   - API key generation and management
//   - Key validation and verification
//   - Key storage interfaces
//   - Key rotation and revocation support
//
// Example usage:
//
//	// Create a key manager
//	store := &MyKeyStore{} // implement apikey.Store
//	hasher := apikey.NewBcryptHasher(10)
//	manager := apikey.NewManager(store, hasher, apikey.WithPrefix("sk_live_"))
//
//	// Generate a new API key
//	key, keyRecord, err := manager.Generate(ctx, "user123", "My API Key", []string{"read", "write"}, nil)
//
//	// Validate an API key
//	keyRecord, err := manager.Validate(ctx, key)
//
//	// Create an authenticator
//	authenticator := apikey.NewAuthenticator(manager)
//	principal, err := authenticator.Authenticate(ctx, key)
package apikey
