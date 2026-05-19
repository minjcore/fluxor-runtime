package password

import (
	"testing"
)

func TestHash(t *testing.T) {
	hasher := NewHasher(DefaultHashConfig())

	password := "my-secure-password"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash is empty")
	}

	if hash == password {
		t.Fatal("Hash should not equal password")
	}
}

func TestVerify(t *testing.T) {
	hasher := NewHasher(DefaultHashConfig())

	password := "my-secure-password"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	valid, err := hasher.Verify(password, hash)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !valid {
		t.Fatal("Password verification should succeed")
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	hasher := NewHasher(DefaultHashConfig())

	password := "my-secure-password"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	wrongPassword := "wrong-password"
	valid, err := hasher.Verify(wrongPassword, hash)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if valid {
		t.Fatal("Password verification should fail for wrong password")
	}
}

func TestHashDifferentHashes(t *testing.T) {
	hasher := NewHasher(DefaultHashConfig())

	password := "my-secure-password"
	hash1, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	hash2, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	// Hashes should be different due to random salt
	if hash1 == hash2 {
		t.Fatal("Hashes should be different (different salts)")
	}

	// But both should verify
	valid1, _ := hasher.Verify(password, hash1)
	valid2, _ := hasher.Verify(password, hash2)

	if !valid1 || !valid2 {
		t.Fatal("Both hashes should verify")
	}
}

func TestMustHash(t *testing.T) {
	hasher := NewHasher(DefaultHashConfig())

	password := "my-secure-password"
	hash := hasher.MustHash(password)

	if hash == "" {
		t.Fatal("Hash is empty")
	}

	valid, err := hasher.Verify(password, hash)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !valid {
		t.Fatal("Password verification should succeed")
	}
}
