package signing

import (
	"testing"
)

func TestSigner_Verify(t *testing.T) {
	// Generate key pair
	privateKey, publicKey, err := GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create signer and verifier
	signer, err := NewSigner(privateKey, AlgorithmRS256)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	verifier, err := NewVerifier(publicKey, AlgorithmRS256)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}

	// Test data
	data := []byte("test message")
	data2 := []byte("different message")

	// Sign
	signature, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	// Verify correct signature
	err = verifier.Verify(data, signature)
	if err != nil {
		t.Errorf("Verify() failed for correct signature: %v", err)
	}

	// Verify wrong data
	err = verifier.Verify(data2, signature)
	if err == nil {
		t.Errorf("Verify() should fail for wrong data")
	}

	// Verify wrong signature
	wrongSig := make([]byte, len(signature))
	copy(wrongSig, signature)
	wrongSig[0] ^= 0xFF
	err = verifier.Verify(data, wrongSig)
	if err == nil {
		t.Errorf("Verify() should fail for wrong signature")
	}
}

func TestSigner_SignString(t *testing.T) {
	privateKey, publicKey, err := GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signer, err := NewSigner(privateKey, AlgorithmRS256)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	verifier, err := NewVerifier(publicKey, AlgorithmRS256)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}

	data := "test message"
	signature, err := signer.SignString(data)
	if err != nil {
		t.Fatalf("Failed to sign string: %v", err)
	}

	err = verifier.VerifyString(data, signature)
	if err != nil {
		t.Errorf("VerifyString() failed: %v", err)
	}
}

func TestSigner_FromPEM(t *testing.T) {
	// Generate PEM keys
	privatePEM, publicPEM, err := GenerateKeyPairPEM(2048)
	if err != nil {
		t.Fatalf("Failed to generate key pair PEM: %v", err)
	}

	// Create signer and verifier from PEM
	signer, err := NewSignerFromPEM(privatePEM, AlgorithmRS256)
	if err != nil {
		t.Fatalf("Failed to create signer from PEM: %v", err)
	}

	verifier, err := NewVerifierFromPEM(publicPEM, AlgorithmRS256)
	if err != nil {
		t.Fatalf("Failed to create verifier from PEM: %v", err)
	}

	// Test signing and verification
	data := []byte("test message")
	signature, err := signer.Sign(data)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	err = verifier.Verify(data, signature)
	if err != nil {
		t.Errorf("Verify() failed: %v", err)
	}
}

func TestSigner_PublicKeyPEM(t *testing.T) {
	privateKey, publicKey, err := GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	signer, err := NewSigner(privateKey, AlgorithmRS256)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	publicPEM, err := signer.PublicKeyPEM()
	if err != nil {
		t.Fatalf("Failed to get public key PEM: %v", err)
	}

	// Verify we can parse it back
	verifier, err := NewVerifierFromPEM(publicPEM, AlgorithmRS256)
	if err != nil {
		t.Fatalf("Failed to create verifier from extracted PEM: %v", err)
	}

	// Verify the public key matches
	extractedPublicKey := verifier.publicKey
	if extractedPublicKey.N.Cmp(publicKey.N) != 0 {
		t.Error("Extracted public key does not match original")
	}
}

func TestSigner_DifferentAlgorithms(t *testing.T) {
	privateKey, publicKey, err := GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	algorithms := []SigningAlgorithm{AlgorithmRS256, AlgorithmRS512}

	for _, alg := range algorithms {
		t.Run(string(alg), func(t *testing.T) {
			signer, err := NewSigner(privateKey, alg)
			if err != nil {
				t.Fatalf("Failed to create signer: %v", err)
			}

			verifier, err := NewVerifier(publicKey, alg)
			if err != nil {
				t.Fatalf("Failed to create verifier: %v", err)
			}

			data := []byte("test message")
			signature, err := signer.Sign(data)
			if err != nil {
				t.Fatalf("Failed to sign: %v", err)
			}

			err = verifier.Verify(data, signature)
			if err != nil {
				t.Errorf("Verify() failed: %v", err)
			}
		})
	}
}
