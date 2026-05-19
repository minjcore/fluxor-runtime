// Package signing provides cryptographic signing and verification capabilities.
//
// This package supports digital signatures for data integrity and authenticity verification.
// It supports multiple algorithms including:
//   - RS256 (RSA with SHA-256)
//   - RS512 (RSA with SHA-512)
//   - ES256 (ECDSA with SHA-256, P-256 curve)
//
// Features:
//   - Generate RSA key pairs (2048-bit and higher)
//   - Generate ECDSA key pairs (P-256 curve for ES256)
//   - Sign data with private keys
//   - Verify signatures with public keys
//   - PEM encoding/decoding for key storage and exchange
//   - String-based signing for convenience
//
// Example usage:
//
//	// Generate key pair
//	privateKey, publicKey, err := signing.GenerateKeyPair(2048)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create signer
//	signer, err := signing.NewSigner(privateKey, signing.AlgorithmRS256)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Sign data
//	data := []byte("important message")
//	signature, err := signer.Sign(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Verify signature
//	verifier, err := signing.NewVerifier(publicKey, signing.AlgorithmRS256)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	err = verifier.Verify(data, signature)
//	if err != nil {
//	    log.Fatal("Signature verification failed")
//	}
//
// This package is useful for:
//   - Software artifact signing and verification
//   - Message integrity verification
//   - API request signing
//   - Supply chain security (SBOM signing)
//
// Path: pkg/security/signing
package signing
