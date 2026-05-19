package encryption

import (
	"crypto/rand"
	"testing"
)

func TestNewChaCha20Encryptor(t *testing.T) {
	// Valid 32-byte key
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encryptor, err := NewChaCha20Encryptor(key)
	if err != nil {
		t.Fatalf("Failed to create ChaCha20 encryptor: %v", err)
	}

	if encryptor == nil {
		t.Fatal("Encryptor is nil")
	}

	if encryptor.aead == nil {
		t.Fatal("AEAD is nil")
	}
}

func TestNewChaCha20Encryptor_InvalidKeySize(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
	}{
		{"16 bytes", 16},
		{"24 bytes", 24},
		{"0 bytes", 0},
		{"31 bytes", 31},
		{"33 bytes", 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			_, err := rand.Read(key)
			if err != nil {
				t.Fatalf("Failed to generate key: %v", err)
			}

			_, err = NewChaCha20Encryptor(key)
			if err == nil {
				t.Errorf("Expected error for key size %d, got nil", tt.keySize)
			}
		})
	}
}

func TestChaCha20Encryptor_EncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encryptor, err := NewChaCha20Encryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	plaintext := []byte("Hello, ChaCha20-Poly1305!")
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if len(ciphertext) <= len(plaintext) {
		t.Errorf("Ciphertext should be longer than plaintext (includes nonce and tag)")
	}

	// Decrypt
	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match. Expected: %s, Got: %s", plaintext, decrypted)
	}
}

func TestChaCha20Encryptor_EncryptDecrypt_EmptyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encryptor, err := NewChaCha20Encryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	plaintext := []byte{}
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt empty plaintext: %v", err)
	}

	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt empty plaintext: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("Expected empty decrypted text, got %d bytes", len(decrypted))
	}
}

func TestChaCha20Encryptor_EncryptDecrypt_LargePlaintext(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encryptor, err := NewChaCha20Encryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	// Create a large plaintext (1MB)
	plaintext := make([]byte, 1024*1024)
	_, err = rand.Read(plaintext)
	if err != nil {
		t.Fatalf("Failed to generate plaintext: %v", err)
	}

	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt large plaintext: %v", err)
	}

	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt large plaintext: %v", err)
	}

	if len(decrypted) != len(plaintext) {
		t.Errorf("Decrypted text length doesn't match. Expected: %d, Got: %d", len(plaintext), len(decrypted))
	}

	// Verify content
	for i := range plaintext {
		if plaintext[i] != decrypted[i] {
			t.Errorf("Byte mismatch at position %d", i)
			break
		}
	}
}

func TestChaCha20Encryptor_Encrypt_NonceUniqueness(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encryptor, err := NewChaCha20Encryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	plaintext := []byte("Test message")
	ciphertext1, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	ciphertext2, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Ciphertexts should be different due to random nonces
	if string(ciphertext1) == string(ciphertext2) {
		t.Error("Ciphertexts should be different (different nonces)")
	}

	// Both should decrypt to the same plaintext
	decrypted1, err := encryptor.Decrypt(ciphertext1)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	decrypted2, err := encryptor.Decrypt(ciphertext2)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted1) != string(plaintext) || string(decrypted2) != string(plaintext) {
		t.Error("Both decryptions should yield the original plaintext")
	}
}

func TestChaCha20Encryptor_Decrypt_InvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encryptor, err := NewChaCha20Encryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte{1, 2, 3}},
		{"corrupted", []byte("invalid ciphertext data that's too short")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptor.Decrypt(tt.ciphertext)
			if err == nil {
				t.Errorf("Expected error for %s ciphertext, got nil", tt.name)
			}
		})
	}
}

func TestChaCha20Encryptor_Decrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	_, err := rand.Read(key1)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	key2 := make([]byte, 32)
	_, err = rand.Read(key2)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encryptor1, err := NewChaCha20Encryptor(key1)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	encryptor2, err := NewChaCha20Encryptor(key2)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	plaintext := []byte("Secret message")
	ciphertext, err := encryptor1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Try to decrypt with wrong key
	_, err = encryptor2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Expected error when decrypting with wrong key, got nil")
	}
}

func TestNewChaCha20EncryptorFromPassword(t *testing.T) {
	password := "my-secret-password"
	encryptor, err := NewChaCha20EncryptorFromPassword(password)
	if err != nil {
		t.Fatalf("Failed to create encryptor from password: %v", err)
	}

	if encryptor == nil {
		t.Fatal("Encryptor is nil")
	}

	// Test encryption/decryption
	plaintext := []byte("Test message")
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match. Expected: %s, Got: %s", plaintext, decrypted)
	}
}

func TestChaCha20Encryptor_EncryptStringDecryptString(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encryptor, err := NewChaCha20Encryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	plaintext := "Hello, ChaCha20-Poly1305!"
	ciphertext, err := encryptor.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt string: %v", err)
	}

	if ciphertext == "" {
		t.Error("Ciphertext should not be empty")
	}

	decrypted, err := encryptor.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt string: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted text doesn't match. Expected: %s, Got: %s", plaintext, decrypted)
	}
}

func TestChaCha20Encryptor_DecryptString_InvalidBase64(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	encryptor, err := NewChaCha20Encryptor(key)
	if err != nil {
		t.Fatalf("Failed to create encryptor: %v", err)
	}

	_, err = encryptor.DecryptString("invalid-base64!!!")
	if err == nil {
		t.Error("Expected error for invalid base64, got nil")
	}
}