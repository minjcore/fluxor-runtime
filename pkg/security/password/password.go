package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// HashConfig configures password hashing parameters
type HashConfig struct {
	// Memory is the amount of memory to use in KiB (default: 64*1024 = 64MB)
	Memory uint32
	// Iterations is the number of iterations (default: 3)
	Iterations uint32
	// Parallelism is the number of threads (default: 2)
	Parallelism uint8
	// SaltLength is the length of the salt in bytes (default: 16)
	SaltLength uint32
	// KeyLength is the length of the derived key in bytes (default: 32)
	KeyLength uint32
}

// DefaultHashConfig returns a default hash configuration
func DefaultHashConfig() HashConfig {
	return HashConfig{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// Hasher provides password hashing and verification
type Hasher struct {
	config HashConfig
}

// NewHasher creates a new password hasher with the given configuration
func NewHasher(config HashConfig) *Hasher {
	if config.Memory == 0 {
		config.Memory = 64 * 1024
	}
	if config.Iterations == 0 {
		config.Iterations = 3
	}
	if config.Parallelism == 0 {
		config.Parallelism = 2
	}
	if config.SaltLength == 0 {
		config.SaltLength = 16
	}
	if config.KeyLength == 0 {
		config.KeyLength = 32
	}

	return &Hasher{
		config: config,
	}
}

// Hash hashes a password using Argon2id
func (h *Hasher) Hash(password string) (string, error) {
	// Generate salt
	salt := make([]byte, h.config.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash password
	hash := argon2.IDKey([]byte(password), salt, h.config.Iterations, h.config.Memory, h.config.Parallelism, h.config.KeyLength)

	// Encode: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		h.config.Memory,
		h.config.Iterations,
		h.config.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encoded, nil
}

// Verify verifies a password against a hash
func (h *Hasher) Verify(password, encodedHash string) (bool, error) {
	// Parse encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return false, errors.New("unsupported hash algorithm")
	}

	// Parse version
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("failed to parse version: %w", err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("unsupported version: %d", version)
	}

	// Parse parameters
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Compute hash of provided password
	otherHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(hash)))

	// Constant-time comparison
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}

// MustHash hashes a password and panics on error (for convenience)
func (h *Hasher) MustHash(password string) string {
	hash, err := h.Hash(password)
	if err != nil {
		panic(fmt.Sprintf("password hashing failed: %v", err))
	}
	return hash
}
