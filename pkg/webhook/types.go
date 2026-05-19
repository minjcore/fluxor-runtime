package webhook

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// SignatureAlgorithm represents a signature validation algorithm
type SignatureAlgorithm string

const (
	// SignatureAlgorithmHMACSHA1 uses HMAC-SHA1 for signature validation
	SignatureAlgorithmHMACSHA1 SignatureAlgorithm = "hmac-sha1"
	// SignatureAlgorithmHMACSHA256 uses HMAC-SHA256 for signature validation
	SignatureAlgorithmHMACSHA256 SignatureAlgorithm = "hmac-sha256"
	// SignatureAlgorithmHMACSHA512 uses HMAC-SHA512 for signature validation
	SignatureAlgorithmHMACSHA512 SignatureAlgorithm = "hmac-sha512"
)

// SignatureValidator validates webhook signatures
type SignatureValidator interface {
	// Validate validates the webhook signature
	Validate(payload []byte, signature string, headers map[string]string) error
}

// HMACSignatureValidator implements HMAC-based signature validation
type HMACSignatureValidator struct {
	secret     []byte
	algorithm  SignatureAlgorithm
	headerName string // Header name containing the signature (e.g., "X-Hub-Signature-256")
	prefix     string // Signature prefix (e.g., "sha256=")
}

// NewHMACSignatureValidator creates a new HMAC signature validator
func NewHMACSignatureValidator(secret string, algorithm SignatureAlgorithm, headerName string, prefix string) *HMACSignatureValidator {
	failfast.If(secret != "", "secret cannot be empty")
	failfast.If(headerName != "", "headerName cannot be empty")
	
	return &HMACSignatureValidator{
		secret:     []byte(secret),
		algorithm:  algorithm,
		headerName: headerName,
		prefix:     prefix,
	}
}

// Validate validates the webhook signature using HMAC
func (v *HMACSignatureValidator) Validate(payload []byte, signature string, headers map[string]string) error {
	// Get signature from headers if not provided
	if signature == "" {
		sig, ok := headers[v.headerName]
		if !ok {
			return fmt.Errorf("signature header %s not found", v.headerName)
		}
		signature = sig
	}

	// Remove prefix if present
	if v.prefix != "" && strings.HasPrefix(signature, v.prefix) {
		signature = strings.TrimPrefix(signature, v.prefix)
	}

	// Compute expected signature
	var h hash.Hash
	switch v.algorithm {
	case SignatureAlgorithmHMACSHA1:
		h = hmac.New(sha1.New, v.secret)
	case SignatureAlgorithmHMACSHA256:
		h = hmac.New(sha256.New, v.secret)
	case SignatureAlgorithmHMACSHA512:
		h = hmac.New(sha512.New, v.secret)
	default:
		return fmt.Errorf("unsupported algorithm: %s", v.algorithm)
	}

	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Compare signatures (constant-time comparison)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// GitHubSignatureValidator validates GitHub webhook signatures
// GitHub uses HMAC-SHA256 with header "X-Hub-Signature-256" and prefix "sha256="
func NewGitHubSignatureValidator(secret string) *HMACSignatureValidator {
	return NewHMACSignatureValidator(secret, SignatureAlgorithmHMACSHA256, "X-Hub-Signature-256", "sha256=")
}

// StripeSignatureValidator validates Stripe webhook signatures
// Stripe uses HMAC-SHA256 with header "Stripe-Signature" (timestamp,signature format)
// For simplicity, we provide basic HMAC validation - full Stripe validation requires timestamp verification
func NewStripeSignatureValidator(secret string) *HMACSignatureValidator {
	// Note: Stripe signature format is more complex (includes timestamp)
	// This is a simplified version - for production, use Stripe's official SDK
	return NewHMACSignatureValidator(secret, SignatureAlgorithmHMACSHA256, "Stripe-Signature", "")
}

// WebhookRequest represents a webhook request
type WebhookRequest struct {
	// Path is the webhook path (e.g., "/webhooks/github")
	Path string
	// Payload is the raw request body
	Payload []byte
	// Headers contains request headers
	Headers map[string]string
	// QueryParams contains query parameters
	QueryParams map[string]string
	// Method is the HTTP method (usually POST)
	Method string
}

// WebhookHandler handles webhook requests
type WebhookHandler func(req *WebhookRequest) error