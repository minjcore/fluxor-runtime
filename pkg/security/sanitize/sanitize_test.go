package sanitize

import (
	"testing"
)

func TestSanitize(t *testing.T) {
	sanitizer := DefaultSanitizer()

	input := "<script>alert('xss')</script>Hello"
	result := sanitizer.Sanitize(input)

	if result == input {
		t.Fatal("Input should be sanitized")
	}

	if !contains(result, "&lt;script&gt;") {
		t.Error("HTML should be escaped")
	}
}

func TestSanitizeFilename(t *testing.T) {
	sanitizer := DefaultSanitizer()

	tests := []struct {
		input    string
		expected string
	}{
		{"../../etc/passwd", "etcpasswd"},
		{"file.txt", "file.txt"},
		{"file..txt", "filetxt"},
		{"file/name.txt", "filename.txt"},
	}

	for _, tt := range tests {
		result := sanitizer.SanitizeFilename(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeSQL(t *testing.T) {
	sanitizer := DefaultSanitizer()

	input := "'; DROP TABLE users; --"
	result := sanitizer.SanitizeSQL(input)

	if contains(result, "DROP") {
		t.Error("SQL keywords should be removed")
	}
}

func TestSanitizeEmail(t *testing.T) {
	sanitizer := DefaultSanitizer()

	validEmail := "user@example.com"
	result := sanitizer.SanitizeEmail(validEmail)

	if result != validEmail {
		t.Errorf("Valid email should remain unchanged: got %q, want %q", result, validEmail)
	}

	invalidEmail := "not-an-email"
	result = sanitizer.SanitizeEmail(invalidEmail)
	if result != "" {
		t.Error("Invalid email should return empty string")
	}
}

func TestSanitizeString(t *testing.T) {
	input := "<script>alert('xss')</script>"
	result := SanitizeString(input)

	if result == input {
		t.Fatal("Input should be sanitized")
	}
}

func TestRemoveHTML(t *testing.T) {
	input := "<p>Hello <b>World</b></p>"
	result := RemoveHTML(input)

	if contains(result, "<") || contains(result, ">") {
		t.Error("HTML tags should be removed")
	}

	if !contains(result, "Hello") || !contains(result, "World") {
		t.Error("Text content should be preserved")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
