// Package pii provides Personally Identifiable Information (PII) detection and handling.
//
// PII detection helps identify and protect sensitive personal information in text data.
// This package supports detection of various PII types including:
//   - Email addresses
//   - Phone numbers
//   - Social Security Numbers (SSN)
//   - Credit card numbers
//   - IP addresses
//   - MAC addresses
//   - Bank account numbers
//   - Passport numbers
//   - Driver license numbers
//   - IBAN (International Bank Account Numbers)
//   - SWIFT/BIC codes
//   - Bitcoin addresses
//   - Ethereum addresses
//
// Example usage:
//
//	detector := pii.NewDetector()
//	detections := detector.Detect("Contact us at support@example.com")
//	for _, d := range detections {
//	    fmt.Printf("Found %s: %s (confidence: %.2f)\n", d.Type, d.Value, d.Confidence)
//	}
//
// The detector uses regex patterns and validation algorithms (e.g., Luhn algorithm for credit cards)
// to identify PII with confidence scores. Custom patterns can be added for specific use cases.
//
// Path: pkg/security/pii
package pii
