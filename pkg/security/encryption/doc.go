// Package encryption provides symmetric encryption and decryption capabilities.
//
// This package supports multiple encryption algorithms for secure data storage and transmission:
//   - AES-GCM (Galois/Counter Mode): Standard authenticated encryption with AES
//   - ChaCha20-Poly1305: Modern authenticated encryption, often faster on systems without hardware AES acceleration
//
// Both algorithms provide authenticated encryption, ensuring both confidentiality and integrity of the encrypted data.
//
// Features:
//   - AES-128, AES-192, and AES-256 encryption
//   - AES-GCM authenticated encryption
//   - ChaCha20-Poly1305 authenticated encryption
//   - Password-based key derivation (SHA-256)
//   - String and byte array encryption/decryption
//   - Base64 encoding for string transport
//
// Example usage:
//
//	// Create encryptor from password
//	encryptor := encryption.NewEncryptorFromPassword("my-secret-password")
//
//	// Encrypt data
//	plaintext := "sensitive data"
//	ciphertext, err := encryptor.EncryptString(plaintext)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Decrypt data
//	decrypted, err := encryptor.DecryptString(ciphertext)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Or use raw key (32 bytes for AES-256)
//	key := make([]byte, 32)
//	// ... fill key with secure random data
//	encryptor, err := encryption.NewEncryptor(key)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	ciphertext, err := encryptor.Encrypt([]byte("data"))
//	plaintext, err := encryptor.Decrypt(ciphertext)
//
//	// Use ChaCha20-Poly1305 (requires 32-byte key)
//	key := make([]byte, 32)
//	// ... fill key with secure random data
//	chachaEncryptor, err := encryption.NewChaCha20Encryptor(key)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Or create from password
//	chachaEncryptor, err = encryption.NewChaCha20EncryptorFromPassword("my-password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	ciphertext, err = chachaEncryptor.Encrypt([]byte("data"))
//	plaintext, err = chachaEncryptor.Decrypt(ciphertext)
//
// Security considerations:
//   - Use strong, randomly generated keys
//   - For password-based encryption, use strong passwords
//   - Store keys securely (use key management systems)
//   - Rotate keys periodically
//   - Use appropriate key sizes (AES-256 recommended)
//
// Path: pkg/security/encryption
package encryption
