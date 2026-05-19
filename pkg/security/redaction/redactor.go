package redaction

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/security/pii"
)

// RedactionStrategy defines how data should be redacted
type RedactionStrategy string

const (
	// StrategyMask replaces with asterisks
	StrategyMask RedactionStrategy = "mask"
	// StrategyHash replaces with hash
	StrategyHash RedactionStrategy = "hash"
	// StrategyPartial shows only partial data (e.g., last 4 digits)
	StrategyPartial RedactionStrategy = "partial"
	// StrategyRemove removes the data entirely
	StrategyRemove RedactionStrategy = "remove"
	// StrategyPlaceholder replaces with placeholder text
	StrategyPlaceholder RedactionStrategy = "placeholder"
)

// RedactorConfig configures the redactor behavior
type RedactorConfig struct {
	// DefaultStrategy is the default redaction strategy
	DefaultStrategy RedactionStrategy
	// TypeStrategies maps PII types to specific strategies
	TypeStrategies map[pii.PIIType]RedactionStrategy
	// MaskChar is the character used for masking
	MaskChar rune
	// ShowLastDigits is the number of digits to show in partial strategy
	ShowLastDigits int
	// PlaceholderText is the text to use for placeholder strategy
	PlaceholderText string
	// Salt is an optional salt for hashing (for deterministic hashing)
	// If nil, a random salt is used for each hash (non-deterministic)
	Salt []byte
	// UseCryptographicHash uses SHA-256 instead of simple hash (default: true)
	UseCryptographicHash bool
}

// DefaultRedactorConfig returns a default redactor configuration
func DefaultRedactorConfig() RedactorConfig {
	return RedactorConfig{
		DefaultStrategy: StrategyMask,
		TypeStrategies: map[pii.PIIType]RedactionStrategy{
			pii.PIIEmail:       StrategyPartial,
			pii.PIICreditCard:  StrategyPartial,
			pii.PIIBankAccount: StrategyPartial,
		},
		MaskChar:              '*',
		ShowLastDigits:        4,
		PlaceholderText:       "[REDACTED]",
		UseCryptographicHash:  true,
	}
}

// Redactor redacts sensitive information from text
type Redactor struct {
	config   RedactorConfig
	detector *pii.Detector
}

// NewRedactor creates a new redactor with the given configuration
func NewRedactor(config RedactorConfig) *Redactor {
	if config.MaskChar == 0 {
		config.MaskChar = '*'
	}
	if config.ShowLastDigits == 0 {
		config.ShowLastDigits = 4
	}
	if config.PlaceholderText == "" {
		config.PlaceholderText = "[REDACTED]"
	}

	return &Redactor{
		config:   config,
		detector: pii.NewDetector(),
	}
}

// Redact redacts all PII found in the text
func (r *Redactor) Redact(text string) string {
	detections := r.detector.Detect(text)
	if len(detections) == 0 {
		return text
	}

	// Sort detections by position (reverse order for safe replacement)
	// Process from end to start to preserve positions
	result := text
	for i := len(detections) - 1; i >= 0; i-- {
		detection := detections[i]
		strategy := r.getStrategy(detection.Type)
		redacted := r.applyStrategy(detection.Value, detection.Type, strategy)
		result = result[:detection.StartPos] + redacted + result[detection.EndPos:]
	}

	return result
}

// RedactType redacts only a specific PII type
func (r *Redactor) RedactType(text string, piiType pii.PIIType) string {
	detections := r.detector.DetectType(text, piiType)
	if len(detections) == 0 {
		return text
	}

	result := text
	for i := len(detections) - 1; i >= 0; i-- {
		detection := detections[i]
		strategy := r.getStrategy(detection.Type)
		redacted := r.applyStrategy(detection.Value, detection.Type, strategy)
		result = result[:detection.StartPos] + redacted + result[detection.EndPos:]
	}

	return result
}

// RedactCustom redacts specific values in the text
func (r *Redactor) RedactCustom(text string, values []string, strategy RedactionStrategy) string {
	result := text
	for _, value := range values {
		idx := strings.Index(result, value)
		if idx >= 0 {
			redacted := r.applyStrategy(value, "", strategy)
			result = result[:idx] + redacted + result[idx+len(value):]
		}
	}
	return result
}

// getStrategy returns the redaction strategy for a PII type
func (r *Redactor) getStrategy(piiType pii.PIIType) RedactionStrategy {
	if strategy, exists := r.config.TypeStrategies[piiType]; exists {
		return strategy
	}
	return r.config.DefaultStrategy
}

// applyStrategy applies the redaction strategy to a value
func (r *Redactor) applyStrategy(value string, piiType pii.PIIType, strategy RedactionStrategy) string {
	switch strategy {
	case StrategyMask:
		return r.mask(value)
	case StrategyHash:
		return r.hash(value)
	case StrategyPartial:
		return r.partial(value, piiType)
	case StrategyRemove:
		return ""
	case StrategyPlaceholder:
		return r.config.PlaceholderText
	default:
		return r.mask(value)
	}
}

// mask replaces characters with mask character
func (r *Redactor) mask(value string) string {
	masked := make([]rune, 0, len(value))
	for _, c := range value {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			masked = append(masked, r.config.MaskChar)
		} else {
			masked = append(masked, c) // Keep formatting
		}
	}
	return string(masked)
}

// hash generates a hash representation using SHA-256 or simple hash
func (r *Redactor) hash(value string) string {
	if r.config.UseCryptographicHash {
		return r.cryptographicHash(value)
	}
	return r.simpleHash(value)
}

// cryptographicHash generates a SHA-256 hash
func (r *Redactor) cryptographicHash(value string) string {
	h := sha256.New()
	if r.config.Salt != nil {
		h.Write(r.config.Salt)
	}
	h.Write([]byte(value))
	hash := h.Sum(nil)
	return fmt.Sprintf("[HASH:%s]", hex.EncodeToString(hash[:8])) // First 8 bytes for readability
}

// simpleHash generates a simple hash (backward compatibility)
func (r *Redactor) simpleHash(value string) string {
	hash := 0
	for _, c := range value {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return fmt.Sprintf("[HASH:%x]", hash)
}

// partial shows only the last few characters
func (r *Redactor) partial(value string, piiType pii.PIIType) string {
	// Remove formatting
	cleanValue := strings.Map(func(c rune) rune {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return c
		}
		return -1
	}, value)

	if len(cleanValue) <= r.config.ShowLastDigits {
		return r.mask(value) // Too short, just mask
	}

	showCount := r.config.ShowLastDigits
	if showCount > len(cleanValue) {
		showCount = len(cleanValue)
	}

	// Determine mask pattern based on type
	switch piiType {
	case pii.PIIEmail:
		// Show first letter and domain: j***@example.com
		parts := strings.Split(value, "@")
		if len(parts) == 2 {
			emailPrefix := parts[0]
			if len(emailPrefix) > 0 {
				maskedPrefix := string(emailPrefix[0]) + strings.Repeat(string(r.config.MaskChar), len(emailPrefix)-1)
				return maskedPrefix + "@" + parts[1]
			}
		}
	case pii.PIICreditCard:
		// Show last 4 digits: ****-****-****-1234
		masked := strings.Repeat(string(r.config.MaskChar), len(cleanValue)-showCount)
		visible := cleanValue[len(cleanValue)-showCount:]
		// Try to preserve formatting
		if strings.Contains(value, "-") {
			return "****-****-****-" + visible
		}
		return masked + visible
	}

	// Default partial: show last N chars
	masked := strings.Repeat(string(r.config.MaskChar), len(cleanValue)-showCount)
	visible := cleanValue[len(cleanValue)-showCount:]
	return masked + visible
}

// SetStrategy sets the redaction strategy for a PII type
func (r *Redactor) SetStrategy(piiType pii.PIIType, strategy RedactionStrategy) {
	if r.config.TypeStrategies == nil {
		r.config.TypeStrategies = make(map[pii.PIIType]RedactionStrategy)
	}
	r.config.TypeStrategies[piiType] = strategy
}
