package signing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestGenerateECDSAKeyPair(t *testing.T) {
	privateKey, publicKey, err := GenerateECDSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateECDSAKeyPair failed: %v", err)
	}

	if privateKey == nil {
		t.Fatal("Private key is nil")
	}

	if publicKey == nil {
		t.Fatal("Public key is nil")
	}

	if privateKey.Curve != elliptic.P256() {
		t.Errorf("Expected P-256 curve, got %v", privateKey.Curve)
	}
}

func TestECDSASignerAndVerifier(t *testing.T) {
	privateKey, publicKey, err := GenerateECDSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateECDSAKeyPair failed: %v", err)
	}

	signer, err := NewECDSASigner(privateKey, AlgorithmES256)
	if err != nil {
		t.Fatalf("NewECDSASigner failed: %v", err)
	}

	verifier, err := NewECDSAVerifier(publicKey, AlgorithmES256)
	if err != nil {
		t.Fatalf("NewECDSAVerifier failed: %v", err)
	}

	data := []byte("test data")
	signature, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if len(signature) == 0 {
		t.Fatal("Signature is empty")
	}

	err = verifier.Verify(data, signature)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
}

func TestECDSASignerAndVerifierString(t *testing.T) {
	privateKey, publicKey, err := GenerateECDSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateECDSAKeyPair failed: %v", err)
	}

	signer, err := NewECDSASigner(privateKey, AlgorithmES256)
	if err != nil {
		t.Fatalf("NewECDSASigner failed: %v", err)
	}

	verifier, err := NewECDSAVerifier(publicKey, AlgorithmES256)
	if err != nil {
		t.Fatalf("NewECDSAVerifier failed: %v", err)
	}

	data := "test string"
	signature, err := signer.SignString(data)
	if err != nil {
		t.Fatalf("SignString failed: %v", err)
	}

	if signature == "" {
		t.Fatal("Signature is empty")
	}

	err = verifier.VerifyString(data, signature)
	if err != nil {
		t.Fatalf("VerifyString failed: %v", err)
	}
}

func TestECDSASignerInvalidSignature(t *testing.T) {
	privateKey, publicKey, err := GenerateECDSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateECDSAKeyPair failed: %v", err)
	}

	signer, err := NewECDSASigner(privateKey, AlgorithmES256)
	if err != nil {
		t.Fatalf("NewECDSASigner failed: %v", err)
	}

	verifier, err := NewECDSAVerifier(publicKey, AlgorithmES256)
	if err != nil {
		t.Fatalf("NewECDSAVerifier failed: %v", err)
	}

	data := []byte("test data")
	signature, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Tamper with data
	tamperedData := []byte("tampered data")
	err = verifier.Verify(tamperedData, signature)
	if err == nil {
		t.Fatal("Expected verification to fail for tampered data")
	}
}

func TestGenerateECDSAKeyPairPEM(t *testing.T) {
	privateKeyPEM, publicKeyPEM, err := GenerateECDSAKeyPairPEM()
	if err != nil {
		t.Fatalf("GenerateECDSAKeyPairPEM failed: %v", err)
	}

	if len(privateKeyPEM) == 0 {
		t.Fatal("Private key PEM is empty")
	}

	if len(publicKeyPEM) == 0 {
		t.Fatal("Public key PEM is empty")
	}

	// Test loading from PEM
	signer, err := NewECDSASignerFromPEM(privateKeyPEM, AlgorithmES256)
	if err != nil {
		t.Fatalf("NewECDSASignerFromPEM failed: %v", err)
	}

	verifier, err := NewECDSAVerifierFromPEM(publicKeyPEM, AlgorithmES256)
	if err != nil {
		t.Fatalf("NewECDSAVerifierFromPEM failed: %v", err)
	}

	data := []byte("test")
	signature, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	err = verifier.Verify(data, signature)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
}

func TestECDSASignerUnsupportedCurve(t *testing.T) {
	// Generate key with unsupported curve (P-384 instead of P-256)
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate P-384 key: %v", err)
	}

	_, err = NewECDSASigner(privateKey, AlgorithmES256)
	if err == nil {
		t.Fatal("Expected error for unsupported curve")
	}
}
