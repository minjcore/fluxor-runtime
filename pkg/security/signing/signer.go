package signing

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
)

// SigningAlgorithm represents the signing algorithm
type SigningAlgorithm string

const (
	// AlgorithmRS256 represents RS256 (RSA with SHA-256)
	AlgorithmRS256 SigningAlgorithm = "RS256"
	// AlgorithmRS512 represents RS512 (RSA with SHA-512)
	AlgorithmRS512 SigningAlgorithm = "RS512"
	// AlgorithmES256 represents ES256 (ECDSA with SHA-256)
	AlgorithmES256 SigningAlgorithm = "ES256"
)

// Signer provides cryptographic signing capabilities
type Signer struct {
	privateKey *rsa.PrivateKey
	algorithm  SigningAlgorithm
}

// NewSigner creates a new signer with a private key
func NewSigner(privateKey *rsa.PrivateKey, algorithm SigningAlgorithm) (*Signer, error) {
	if privateKey == nil {
		return nil, errors.New("private key cannot be nil")
	}
	if algorithm == "" {
		algorithm = AlgorithmRS256
	}
	return &Signer{
		privateKey: privateKey,
		algorithm:  algorithm,
	}, nil
}

// NewSignerFromPEM creates a signer from a PEM-encoded private key
func NewSignerFromPEM(pemKey []byte, algorithm SigningAlgorithm) (*Signer, error) {
	block, _ := pem.Decode(pemKey)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	var privateKey *rsa.PrivateKey
	var err error

	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("key is not an RSA private key")
		}
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return NewSigner(privateKey, algorithm)
}

// Sign signs the given data and returns the signature
func (s *Signer) Sign(data []byte) ([]byte, error) {
	hash := s.hash(data)
	return s.signHash(hash)
}

// SignString signs a string and returns a base64-encoded signature
func (s *Signer) SignString(data string) (string, error) {
	signature, err := s.Sign([]byte(data))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

// signHash signs the hash value
func (s *Signer) signHash(hash []byte) ([]byte, error) {
	switch s.algorithm {
	case AlgorithmRS256, AlgorithmRS512:
		return rsa.SignPKCS1v15(rand.Reader, s.privateKey, s.hashFunc(), hash)
	case AlgorithmES256:
		return nil, fmt.Errorf("ES256 requires ECDSASigner, use NewECDSASigner instead")
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", s.algorithm)
	}
}

// hash computes the hash of the data based on the algorithm
func (s *Signer) hash(data []byte) []byte {
	switch s.algorithm {
	case AlgorithmRS256:
		h := sha256.Sum256(data)
		return h[:]
	case AlgorithmRS512:
		h := sha512.Sum512(data)
		return h[:]
	default:
		h := sha256.Sum256(data)
		return h[:]
	}
}

// hashFunc returns the crypto.Hash function for the algorithm
func (s *Signer) hashFunc() crypto.Hash {
	switch s.algorithm {
	case AlgorithmRS256:
		return crypto.SHA256
	case AlgorithmRS512:
		return crypto.SHA512
	default:
		return crypto.SHA256
	}
}

// PublicKey returns the public key associated with the signer
func (s *Signer) PublicKey() *rsa.PublicKey {
	return &s.privateKey.PublicKey
}

// PublicKeyPEM returns the PEM-encoded public key
func (s *Signer) PublicKeyPEM() ([]byte, error) {
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

// Verifier verifies signatures
type Verifier struct {
	publicKey *rsa.PublicKey
	algorithm SigningAlgorithm
}

// NewVerifier creates a new verifier with a public key
func NewVerifier(publicKey *rsa.PublicKey, algorithm SigningAlgorithm) (*Verifier, error) {
	if publicKey == nil {
		return nil, errors.New("public key cannot be nil")
	}
	if algorithm == "" {
		algorithm = AlgorithmRS256
	}
	return &Verifier{
		publicKey: publicKey,
		algorithm: algorithm,
	}, nil
}

// NewVerifierFromPEM creates a verifier from a PEM-encoded public key
func NewVerifierFromPEM(pemKey []byte, algorithm SigningAlgorithm) (*Verifier, error) {
	block, _ := pem.Decode(pemKey)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	var publicKey *rsa.PublicKey
	var err error

	switch block.Type {
	case "PUBLIC KEY":
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
		}
		var ok bool
		publicKey, ok = key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("key is not an RSA public key")
		}
	case "RSA PUBLIC KEY":
		publicKey, err = x509.ParsePKCS1PublicKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return NewVerifier(publicKey, algorithm)
}

// Verify verifies a signature against the data
func (v *Verifier) Verify(data []byte, signature []byte) error {
	hash := v.hash(data)
	return v.verifyHash(hash, signature)
}

// VerifyString verifies a base64-encoded signature against a string
func (v *Verifier) VerifyString(data string, signature string) error {
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}
	return v.Verify([]byte(data), sigBytes)
}

// verifyHash verifies the signature against the hash
func (v *Verifier) verifyHash(hash []byte, signature []byte) error {
	switch v.algorithm {
	case AlgorithmRS256, AlgorithmRS512:
		return rsa.VerifyPKCS1v15(v.publicKey, v.hashFunc(), hash, signature)
	case AlgorithmES256:
		return fmt.Errorf("ES256 requires ECDSAVerifier, use NewECDSAVerifier instead")
	default:
		return fmt.Errorf("unsupported algorithm: %s", v.algorithm)
	}
}

// hash computes the hash of the data
func (v *Verifier) hash(data []byte) []byte {
	switch v.algorithm {
	case AlgorithmRS256:
		h := sha256.Sum256(data)
		return h[:]
	case AlgorithmRS512:
		h := sha512.Sum512(data)
		return h[:]
	default:
		h := sha256.Sum256(data)
		return h[:]
	}
}

// hashFunc returns the crypto.Hash function for the algorithm
func (v *Verifier) hashFunc() crypto.Hash {
	switch v.algorithm {
	case AlgorithmRS256:
		return crypto.SHA256
	case AlgorithmRS512:
		return crypto.SHA512
	default:
		return crypto.SHA256
	}
}

// GenerateKeyPair generates a new RSA key pair
func GenerateKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	if bits < 2048 {
		bits = 2048 // Minimum recommended
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key pair: %w", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

// GenerateKeyPairPEM generates a new RSA key pair and returns PEM-encoded keys
func GenerateKeyPairPEM(bits int) (privateKeyPEM []byte, publicKeyPEM []byte, err error) {
	privateKey, publicKey, err := GenerateKeyPair(bits)
	if err != nil {
		return nil, nil, err
	}

	// Encode private key
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
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
