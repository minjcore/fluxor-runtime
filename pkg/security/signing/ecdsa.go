package signing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
)

// ECDSASigner provides ECDSA-based signing capabilities
type ECDSASigner struct {
	privateKey *ecdsa.PrivateKey
	algorithm  SigningAlgorithm
	curve      elliptic.Curve
}

// NewECDSASigner creates a new ECDSA signer with a private key
func NewECDSASigner(privateKey *ecdsa.PrivateKey, algorithm SigningAlgorithm) (*ECDSASigner, error) {
	if privateKey == nil {
		return nil, errors.New("private key cannot be nil")
	}
	if algorithm != AlgorithmES256 {
		return nil, fmt.Errorf("unsupported ECDSA algorithm: %s (only ES256 supported)", algorithm)
	}

	curve := privateKey.Curve
	if curve != elliptic.P256() {
		return nil, fmt.Errorf("unsupported curve (only P-256 supported for ES256)")
	}

	return &ECDSASigner{
		privateKey: privateKey,
		algorithm:  algorithm,
		curve:      curve,
	}, nil
}

// NewECDSASignerFromPEM creates an ECDSA signer from a PEM-encoded private key
func NewECDSASignerFromPEM(pemKey []byte, algorithm SigningAlgorithm) (*ECDSASigner, error) {
	block, _ := pem.Decode(pemKey)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	var privateKey *ecdsa.PrivateKey
	var err error

	switch block.Type {
	case "EC PRIVATE KEY":
		privateKey, err = x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("key is not an ECDSA private key")
		}
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return NewECDSASigner(privateKey, algorithm)
}

// Sign signs the given data and returns the signature
func (s *ECDSASigner) Sign(data []byte) ([]byte, error) {
	hash := s.hash(data)
	r, sig, err := ecdsa.Sign(rand.Reader, s.privateKey, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Encode r and s as ASN.1 DER format
	signature := encodeECDSASignature(r, sig)
	return signature, nil
}

// SignString signs a string and returns a base64-encoded signature
func (s *ECDSASigner) SignString(data string) (string, error) {
	signature, err := s.Sign([]byte(data))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// hash computes the hash of the data
func (s *ECDSASigner) hash(data []byte) []byte {
	switch s.algorithm {
	case AlgorithmES256:
		h := sha256.Sum256(data)
		return h[:]
	default:
		h := sha256.Sum256(data)
		return h[:]
	}
}

// PublicKey returns the public key associated with the signer
func (s *ECDSASigner) PublicKey() *ecdsa.PublicKey {
	return &s.privateKey.PublicKey
}

// PublicKeyPEM returns the PEM-encoded public key
func (s *ECDSASigner) PublicKeyPEM() ([]byte, error) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(s.PublicKey())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return pubKeyPEM, nil
}

// ECDSAVerifier verifies ECDSA signatures
type ECDSAVerifier struct {
	publicKey *ecdsa.PublicKey
	algorithm SigningAlgorithm
	curve     elliptic.Curve
}

// NewECDSAVerifier creates a new ECDSA verifier with a public key
func NewECDSAVerifier(publicKey *ecdsa.PublicKey, algorithm SigningAlgorithm) (*ECDSAVerifier, error) {
	if publicKey == nil {
		return nil, errors.New("public key cannot be nil")
	}
	if algorithm != AlgorithmES256 {
		return nil, fmt.Errorf("unsupported ECDSA algorithm: %s (only ES256 supported)", algorithm)
	}

	curve := publicKey.Curve
	if curve != elliptic.P256() {
		return nil, fmt.Errorf("unsupported curve (only P-256 supported for ES256)")
	}

	return &ECDSAVerifier{
		publicKey: publicKey,
		algorithm: algorithm,
		curve:     curve,
	}, nil
}

// NewECDSAVerifierFromPEM creates an ECDSA verifier from a PEM-encoded public key
func NewECDSAVerifierFromPEM(pemKey []byte, algorithm SigningAlgorithm) (*ECDSAVerifier, error) {
	block, _ := pem.Decode(pemKey)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	var publicKey *ecdsa.PublicKey
	var err error

	switch block.Type {
	case "PUBLIC KEY":
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
		}
		var ok bool
		publicKey, ok = key.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("key is not an ECDSA public key")
		}
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return NewECDSAVerifier(publicKey, algorithm)
}

// Verify verifies a signature against the data
func (v *ECDSAVerifier) Verify(data []byte, signature []byte) error {
	hash := v.hash(data)
	r, s, err := decodeECDSASignature(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	if !ecdsa.Verify(v.publicKey, hash, r, s) {
		return errors.New("signature verification failed")
	}

	return nil
}

// VerifyString verifies a base64-encoded signature against a string
func (v *ECDSAVerifier) VerifyString(data string, signature string) error {
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}
	return v.Verify([]byte(data), sigBytes)
}

// hash computes the hash of the data
func (v *ECDSAVerifier) hash(data []byte) []byte {
	switch v.algorithm {
	case AlgorithmES256:
		h := sha256.Sum256(data)
		return h[:]
	default:
		h := sha256.Sum256(data)
		return h[:]
	}
}

// GenerateECDSAKeyPair generates a new ECDSA key pair (P-256 curve for ES256)
func GenerateECDSAKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ECDSA key pair: %w", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

// GenerateECDSAKeyPairPEM generates a new ECDSA key pair and returns PEM-encoded keys
func GenerateECDSAKeyPairPEM() (privateKeyPEM []byte, publicKeyPEM []byte, err error) {
	privateKey, publicKey, err := GenerateECDSAKeyPair()
	if err != nil {
		return nil, nil, err
	}

	// Encode private key
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	privateKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Encode public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return privateKeyPEM, publicKeyPEM, nil
}

// encodeECDSASignature encodes r and s as ASN.1 DER format
func encodeECDSASignature(r, s *big.Int) []byte {
	// Simple encoding: r and s as 32-byte big-endian integers concatenated
	// For production, use proper ASN.1 DER encoding
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	
	// Pad to 32 bytes
	rPadded := make([]byte, 32)
	sPadded := make([]byte, 32)
	copy(rPadded[32-len(rBytes):], rBytes)
	copy(sPadded[32-len(sBytes):], sBytes)
	
	return append(rPadded, sPadded...)
}

// decodeECDSASignature decodes ASN.1 DER format signature
func decodeECDSASignature(signature []byte) (*big.Int, *big.Int, error) {
	if len(signature) != 64 {
		return nil, nil, fmt.Errorf("invalid signature length: expected 64 bytes, got %d", len(signature))
	}
	
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])
	
	return r, s, nil
}
