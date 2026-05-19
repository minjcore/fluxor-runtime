// Package redaction provides utilities for redacting sensitive information from text.
//
// Redaction helps protect sensitive data by replacing it with masked, hashed, or placeholder values.
// This package integrates with the PII detection package to automatically identify and redact
// Personally Identifiable Information.
//
// Redaction strategies:
//   - Mask: Replace characters with asterisks or custom characters
//   - Hash: Replace with a cryptographic hash (SHA-256) or simple hash
//   - Partial: Show only last N digits/characters (e.g., last 4 digits of credit card)
//   - Remove: Completely remove the sensitive data
//   - Placeholder: Replace with placeholder text like "[REDACTED]"
//
// Security features:
//   - Cryptographic hashing (SHA-256) with optional salt support
//   - Deterministic hashing when salt is provided
//   - Non-deterministic hashing when salt is not provided
//
// Example usage:
//
//	config := redaction.DefaultRedactorConfig()
//	config.DefaultStrategy = redaction.StrategyPartial
//	redactor := redaction.NewRedactor(config)
//
//	text := "Credit card: 4532-1234-5678-9010"
//	redacted := redactor.Redact(text)
//	// Result: "Credit card: ****-****-****-9010"
//
// Different strategies can be configured for different PII types. This is useful for compliance
// with regulations like GDPR, HIPAA, or PCI-DSS where different data requires different handling.
//
// Path: pkg/security/redaction
package redaction
