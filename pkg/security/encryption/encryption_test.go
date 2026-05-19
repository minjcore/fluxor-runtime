package encryption

import (
	"crypto/rand"
	"testing"
)

func TestNewEncryptor(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	encryptor, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	if encryptor == nil {
		t.Fatal("Encryptor is nil")
	}
}

func TestNewEncryptorInvalidKeyLength(t *testing.T) {
	key := make([]byte, 15) // Invalid length

	_, err := NewEncryptor(key)
	if err == nil {
		t.Fatal("Expected error for invalid key length")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	encryptor, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor failed: %v", err)
	}

	plaintext := []byte("sensitive data")
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Fatal("Ciphertext is empty")
	}

	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted data doesn't match: got %s, want %s", string(decrypted), string(plaintext))
	}
}

func TestEncryptDecryptString(t *testing.T) {
	encryptor := NewEncryptorFromPassword("my-secret-password")

	plaintext := "sensitive data"
	ciphertext, err := encryptor.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("EncryptString failed: %v", err)
	}

	if ciphertext == "" {
		t.Fatal("Ciphertext is empty")
	}

	decrypted, err := encryptor.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("DecryptString failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted data doesn't match: got %s, want %s", decrypted, plaintext)
	}
}

func TestEncryptDecryptDifferentKeys(t *testing.T) {
	key1 := make([]byte, 32)
	rand.Read(key1)

	key2 := make([]byte, 32)
	rand.Read(key2)

	encryptor1, _ := NewEncryptor(key1)
	encryptor2, _ := NewEncryptor(key2)

	plaintext := []byte("sensitive data")
	ciphertext, err := encryptor1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Try to decrypt with different key - should fail
	_, err = encryptor2.Decrypt(ciphertext)
	if err == nil {
		t.Fatal("Expected error when decrypting with different key")
	}
}

func TestNewEncryptorFromPassword(t *testing.T) {
	encryptor := NewEncryptorFromPassword("my-password")

	if encryptor == nil {
		t.Fatal("Encryptor is nil")
	}

	plaintext := []byte("test")
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted data doesn't match")
	}
}
