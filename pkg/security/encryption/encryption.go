package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// Encryptor provides encryption and decryption capabilities
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor with a key
// Key should be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256
func NewEncryptor(key []byte) (*Encryptor, error) {
	keyLen := len(key)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return nil, fmt.Errorf("invalid key length: %d (must be 16, 24, or 32 bytes)", keyLen)
	}

	return &Encryptor{
		key: key,
	}, nil
}

// NewEncryptorFromPassword creates an encryptor from a password using SHA-256
// This derives a 32-byte key (AES-256) from the password
func NewEncryptorFromPassword(password string) *Encryptor {
	hash := sha256.Sum256([]byte(password))
	return &Encryptor{
		key: hash[:],
	}
}

// Encrypt encrypts data using AES-GCM
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-GCM
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return encodeBase64(ciphertext), nil
}

// DecryptString decrypts a base64-encoded ciphertext string
func (e *Encryptor) DecryptString(ciphertext string) (string, error) {
	ciphertextBytes, err := decodeBase64(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	plaintext, err := e.Decrypt(ciphertextBytes)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// ChaCha20Encryptor provides ChaCha20-Poly1305 encryption and decryption capabilities
// ChaCha20-Poly1305 is an authenticated encryption algorithm that provides both
// confidentiality and integrity. It's generally faster than AES on systems without
// hardware AES acceleration.
type ChaCha20Encryptor struct {
	key []byte
	aead cipher.AEAD
}

// NewChaCha20Encryptor creates a new ChaCha20 encryptor with a key
// Key must be exactly 32 bytes (256 bits) for ChaCha20-Poly1305
func NewChaCha20Encryptor(key []byte) (*ChaCha20Encryptor, error) {
	keyLen := len(key)
	if keyLen != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key length: %d (must be %d bytes for ChaCha20-Poly1305)", keyLen, chacha20poly1305.KeySize)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20-Poly1305: %w", err)
	}

	return &ChaCha20Encryptor{
		key:  key,
		aead: aead,
	}, nil
}

// NewChaCha20EncryptorFromPassword creates a ChaCha20 encryptor from a password using SHA-256
// This derives a 32-byte key from the password
func NewChaCha20EncryptorFromPassword(password string) (*ChaCha20Encryptor, error) {
	hash := sha256.Sum256([]byte(password))
	return NewChaCha20Encryptor(hash[:])
}

// Encrypt encrypts data using ChaCha20-Poly1305
func (e *ChaCha20Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := e.aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using ChaCha20-Poly1305
func (e *ChaCha20Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := e.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext
func (e *ChaCha20Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return encodeBase64(ciphertext), nil
}

// DecryptString decrypts a base64-encoded ciphertext string
func (e *ChaCha20Encryptor) DecryptString(ciphertext string) (string, error) {
	ciphertextBytes, err := decodeBase64(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	plaintext, err := e.Decrypt(ciphertextBytes)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
