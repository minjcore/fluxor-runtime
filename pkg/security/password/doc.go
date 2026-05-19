// Package password provides secure password hashing and verification.
//
// This package uses Argon2id, the winner of the Password Hashing Competition,
// which provides resistance against GPU-based attacks and side-channel attacks.
//
// Features:
//   - Argon2id password hashing
//   - Configurable memory, iterations, and parallelism
//   - Constant-time password verification
//   - Standard encoding format for hash storage
//
// Example usage:
//
//	hasher := password.NewHasher(password.DefaultHashConfig())
//
//	// Hash a password
//	hash, err := hasher.Hash("my-secure-password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Verify a password
//	valid, err := hasher.Verify("my-secure-password", hash)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if valid {
//	    fmt.Println("Password is correct")
//	}
//
// Security considerations:
//   - Use strong, unique passwords
//   - Adjust memory/iterations based on your security requirements
//   - Store hashes securely
//   - Never store plaintext passwords
//   - Use constant-time comparison (built-in)
//
// Path: pkg/security/password
package password
