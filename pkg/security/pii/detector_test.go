package pii

import (
	"regexp"
	"testing"
)

func TestDetector_Detect(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name       string
		text       string
		wantTypes  []PIIType
		wantCount  int
	}{
		{
			name:      "Email detection",
			text:      "Contact us at support@example.com for help",
			wantTypes: []PIIType{PIIEmail},
			wantCount: 1,
		},
		{
			name:      "Phone number detection",
			text:      "Call us at 555-123-4567",
			wantTypes: []PIIType{PIIPhone},
			wantCount: 1,
		},
		{
			name:      "SSN detection",
			text:      "SSN: 123-45-6789",
			wantTypes: []PIIType{PIISSN},
			wantCount: 1,
		},
		{
			name:      "Credit card detection",
			text:      "Card: 4532-1234-5678-9010",
			wantTypes: []PIIType{PIICreditCard},
			wantCount: 1,
		},
		{
			name:      "IP address detection",
			text:      "IP: 192.168.1.1",
			wantTypes: []PIIType{PIIIPAddress},
			wantCount: 1,
		},
		{
			name:      "Multiple PII types",
			text:      "Email: test@example.com, Phone: 555-123-4567",
			wantTypes: []PIIType{PIIEmail, PIIPhone},
			wantCount: 2,
		},
		{
			name:      "No PII",
			text:      "This is just regular text with no sensitive data",
			wantTypes: []PIIType{},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detections := detector.Detect(tt.text)
			if len(detections) != tt.wantCount {
				t.Errorf("Detect() got %d detections, want %d", len(detections), tt.wantCount)
			}

			detectedTypes := make(map[PIIType]bool)
			for _, d := range detections {
				detectedTypes[d.Type] = true
			}

			for _, wantType := range tt.wantTypes {
				if !detectedTypes[wantType] {
					t.Errorf("Detect() missing type %s", wantType)
				}
			}
		})
	}
}

func TestDetector_HasPII(t *testing.T) {
	detector := NewDetector()

	tests := []struct {
		name string
		text string
		want bool
	}{
		{"Has email", "Contact test@example.com", true},
		{"Has SSN", "SSN: 123-45-6789", true},
		{"No PII", "Regular text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detector.HasPII(tt.text); got != tt.want {
				t.Errorf("HasPII() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetector_DetectType(t *testing.T) {
	detector := NewDetector()

	text := "Email: test@example.com, Phone: 555-123-4567"

	emailDetections := detector.DetectType(text, PIIEmail)
	if len(emailDetections) != 1 {
		t.Errorf("DetectType(email) got %d, want 1", len(emailDetections))
	}
	if emailDetections[0].Type != PIIEmail {
		t.Errorf("DetectType() got type %s, want %s", emailDetections[0].Type, PIIEmail)
	}
}

func TestDetector_AddPattern(t *testing.T) {
	detector := NewDetector()

	customType := PIIType("custom_id")
	pattern := regexp.MustCompile(`CUST-\d{6}`)
	detector.AddPattern(customType, pattern)

	text := "Customer ID: CUST-123456"
	detections := detector.DetectType(text, customType)
	if len(detections) != 1 {
		t.Errorf("DetectType(custom) got %d, want 1", len(detections))
	}
}

func TestDetector_validateLuhn(t *testing.T) {
	detector := NewDetector()

	validCards := []string{
		"4532015112830366",
		"4532-0151-1283-0366",
		"4532 0151 1283 0366",
	}

	invalidCards := []string{
		"1234567890123456",
		"4532015112830367",
	}

	for _, card := range validCards {
		if !detector.validateLuhn(card) {
			t.Errorf("validateLuhn(%s) = false, want true", card)
		}
	}

	for _, card := range invalidCards {
		if detector.validateLuhn(card) {
			t.Errorf("validateLuhn(%s) = true, want false", card)
		}
	}
}
