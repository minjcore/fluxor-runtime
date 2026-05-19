package pii

import (
	"regexp"
	"strings"
)

// PIIType represents the type of PII detected
type PIIType string

const (
	// PIIEmail represents email addresses
	PIIEmail PIIType = "email"
	// PIIPhone represents phone numbers
	PIIPhone PIIType = "phone"
	// PIISSN represents Social Security Numbers (US)
	PIISSN PIIType = "ssn"
	// PIICreditCard represents credit card numbers
	PIICreditCard PIIType = "credit_card"
	// PIIIPAddress represents IP addresses
	PIIIPAddress PIIType = "ip_address"
	// PIIMACAddress represents MAC addresses
	PIIMACAddress PIIType = "mac_address"
	// PIIBankAccount represents bank account numbers
	PIIBankAccount PIIType = "bank_account"
	// PIIPassport represents passport numbers
	PIIPassport PIIType = "passport"
	// PIIDriverLicense represents driver license numbers
	PIIDriverLicense PIIType = "driver_license"
	// PIIIBAN represents International Bank Account Numbers
	PIIIBAN PIIType = "iban"
	// PIISWIFT represents SWIFT/BIC codes
	PIISWIFT PIIType = "swift"
	// PIIBitcoin represents Bitcoin addresses
	PIIBitcoin PIIType = "bitcoin"
	// PIIEthereum represents Ethereum addresses
	PIIEthereum PIIType = "ethereum"
)

// Detection represents a detected PII instance
type Detection struct {
	Type      PIIType
	Value     string
	StartPos  int
	EndPos    int
	Confidence float64 // 0.0 to 1.0
}

// Detector detects PII in text
type Detector struct {
	patterns map[PIIType]*regexp.Regexp
}

// NewDetector creates a new PII detector with default patterns
func NewDetector() *Detector {
	d := &Detector{
		patterns: make(map[PIIType]*regexp.Regexp),
	}
	d.initPatterns()
	return d
}

// initPatterns initializes regex patterns for PII detection
func (d *Detector) initPatterns() {
	// Email pattern
	d.patterns[PIIEmail] = regexp.MustCompile(`(?i)\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)

	// Phone number patterns (various formats)
	d.patterns[PIIPhone] = regexp.MustCompile(`(?i)(?:\+?1[-.\s]?)?\(?([0-9]{3})\)?[-.\s]?([0-9]{3})[-.\s]?([0-9]{4})`)

	// SSN pattern (XXX-XX-XXXX)
	d.patterns[PIISSN] = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

	// Credit card pattern (Luhn algorithm not validated here, but format checked)
	d.patterns[PIICreditCard] = regexp.MustCompile(`\b(?:\d{4}[-.\s]?){3}\d{4}\b`)

	// IP address pattern
	d.patterns[PIIIPAddress] = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)

	// MAC address pattern
	d.patterns[PIIMACAddress] = regexp.MustCompile(`(?i)\b(?:[0-9A-F]{2}[:-]){5}(?:[0-9A-F]{2})\b`)

	// Bank account (basic pattern - very generic)
	d.patterns[PIIBankAccount] = regexp.MustCompile(`\b\d{8,17}\b`) // 8-17 digits

	// Passport number (varies by country, using common patterns)
	d.patterns[PIIPassport] = regexp.MustCompile(`(?i)\b[A-Z]{1,2}\d{6,9}\b`)

	// Driver license (varies by state/country)
	d.patterns[PIIDriverLicense] = regexp.MustCompile(`(?i)\b[A-Z]{1,2}\d{6,12}\b`)

	// IBAN (International Bank Account Number)
	// Format: 2 letters (country) + 2 digits (check) + up to 30 alphanumeric
	d.patterns[PIIIBAN] = regexp.MustCompile(`(?i)\b[A-Z]{2}\d{2}[A-Z0-9]{4,30}\b`)

	// SWIFT/BIC code (8 or 11 characters)
	d.patterns[PIISWIFT] = regexp.MustCompile(`(?i)\b[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?\b`)

	// Bitcoin address (Base58, starts with 1, 3, or bc1)
	d.patterns[PIIBitcoin] = regexp.MustCompile(`\b[13][a-km-zA-HJ-NP-Z1-9]{25,34}\b|\bbc1[a-z0-9]{39,59}\b`)

	// Ethereum address (0x followed by 40 hex characters)
	d.patterns[PIIEthereum] = regexp.MustCompile(`(?i)\b0x[a-f0-9]{40}\b`)
}

// Detect finds all PII instances in the given text
func (d *Detector) Detect(text string) []Detection {
	var detections []Detection

	for piiType, pattern := range d.patterns {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				value := text[match[0]:match[1]]
				confidence := d.calculateConfidence(piiType, value)
				
				detections = append(detections, Detection{
					Type:       piiType,
					Value:      value,
					StartPos:   match[0],
					EndPos:     match[1],
					Confidence: confidence,
				})
			}
		}
	}

	return detections
}

// DetectType finds PII instances of a specific type
func (d *Detector) DetectType(text string, piiType PIIType) []Detection {
	var detections []Detection
	pattern, exists := d.patterns[piiType]
	if !exists {
		return detections
	}

	matches := pattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			value := text[match[0]:match[1]]
			confidence := d.calculateConfidence(piiType, value)

			detections = append(detections, Detection{
				Type:       piiType,
				Value:      value,
				StartPos:   match[0],
				EndPos:     match[1],
				Confidence: confidence,
			})
		}
	}

	return detections
}

// HasPII checks if the text contains any PII
func (d *Detector) HasPII(text string) bool {
	detections := d.Detect(text)
	return len(detections) > 0
}

// calculateConfidence calculates confidence score for a detected PII
func (d *Detector) calculateConfidence(piiType PIIType, value string) float64 {
	// Base confidence
	confidence := 0.8

	// Additional validation based on type
	switch piiType {
	case PIICreditCard:
		// Validate using Luhn algorithm for better confidence
		if d.validateLuhn(value) {
			confidence = 0.95
		} else {
			confidence = 0.5
		}
	case PIISSN:
		// SSN validation: first 3 digits should not be 000, 666, or 900-999
		// Middle 2 digits should not be 00
		// Last 4 digits should not be 0000
		cleanSSN := strings.ReplaceAll(value, "-", "")
		if len(cleanSSN) == 9 {
			first := cleanSSN[0:3]
			middle := cleanSSN[3:5]
			last := cleanSSN[5:9]
			
			if first == "000" || first == "666" || (first[0] == '9' && first[1] >= '0') {
				confidence = 0.3
			} else if middle == "00" {
				confidence = 0.5
			} else if last == "0000" {
				confidence = 0.5
			} else {
				confidence = 0.9
			}
		} else {
			confidence = 0.5
		}
	case PIIEmail:
		// Email validation
		if strings.Contains(value, "@") && strings.Contains(value, ".") {
			confidence = 0.9
		}
	case PIIPhone:
		// Phone number validation
		if len(value) >= 10 {
			confidence = 0.85
		}
	case PIIIBAN:
		// IBAN validation: check length and format
		if len(value) >= 15 && len(value) <= 34 {
			confidence = 0.9
		} else {
			confidence = 0.5
		}
	case PIISWIFT:
		// SWIFT validation: should be 8 or 11 characters
		clean := strings.ReplaceAll(value, " ", "")
		if len(clean) == 8 || len(clean) == 11 {
			confidence = 0.95
		} else {
			confidence = 0.5
		}
	case PIIBitcoin:
		// Bitcoin address validation
		if strings.HasPrefix(value, "1") || strings.HasPrefix(value, "3") || strings.HasPrefix(value, "bc1") {
			confidence = 0.9
		}
	case PIIEthereum:
		// Ethereum address validation
		if strings.HasPrefix(strings.ToLower(value), "0x") && len(value) == 42 {
			confidence = 0.95
		}
	}

	return confidence
}

// validateLuhn validates credit card number using Luhn algorithm
func (d *Detector) validateLuhn(number string) bool {
	// Remove non-digits
	digits := regexp.MustCompile(`\D`).ReplaceAllString(number, "")
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}

	sum := 0
	alternate := false

	// Process digits from right to left
	for i := len(digits) - 1; i >= 0; i-- {
		digit := int(digits[i] - '0')
		if alternate {
			digit *= 2
			if digit > 9 {
				digit = (digit % 10) + 1
			}
		}
		sum += digit
		alternate = !alternate
	}

	return sum%10 == 0
}

// AddPattern allows adding custom patterns for PII detection
func (d *Detector) AddPattern(piiType PIIType, pattern *regexp.Regexp) {
	d.patterns[piiType] = pattern
}

// GetPattern returns the regex pattern for a PII type
func (d *Detector) GetPattern(piiType PIIType) (*regexp.Regexp, bool) {
	pattern, exists := d.patterns[piiType]
	return pattern, exists
}
