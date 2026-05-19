package sanitize

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

// Sanitizer provides input sanitization capabilities
type Sanitizer struct {
	// AllowHTML allows HTML tags (default: false)
	AllowHTML bool
	// MaxLength is the maximum length of input (0 = no limit)
	MaxLength int
	// RemoveControlChars removes control characters (default: true)
	RemoveControlChars bool
}

// DefaultSanitizer returns a default sanitizer
func DefaultSanitizer() *Sanitizer {
	return &Sanitizer{
		AllowHTML:          false,
		RemoveControlChars: true,
	}
}

// Sanitize sanitizes input to prevent XSS and injection attacks
func (s *Sanitizer) Sanitize(input string) string {
	// Trim whitespace
	result := strings.TrimSpace(input)

	// Apply length limit
	if s.MaxLength > 0 && len(result) > s.MaxLength {
		result = result[:s.MaxLength]
	}

	// Remove control characters
	if s.RemoveControlChars {
		result = s.removeControlChars(result)
	}

	// Escape HTML if not allowed
	if !s.AllowHTML {
		result = html.EscapeString(result)
	}

	return result
}

// removeControlChars removes control characters except newlines and tabs
func (s *Sanitizer) removeControlChars(input string) string {
	var builder strings.Builder
	for _, r := range input {
		// Allow printable characters, newlines, and tabs
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// SanitizeSQL removes SQL injection patterns
func (s *Sanitizer) SanitizeSQL(input string) string {
	// Remove SQL comment patterns
	result := regexp.MustCompile(`--.*`).ReplaceAllString(input, "")
	result = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(result, "")

	// Remove common SQL keywords that could be used in injection
	// Note: This is a basic approach - use parameterized queries in production
	sqlKeywords := []string{
		"DROP", "DELETE", "INSERT", "UPDATE", "ALTER", "CREATE",
		"EXEC", "EXECUTE", "UNION", "SELECT",
	}

	result = strings.ToUpper(result)
	for _, keyword := range sqlKeywords {
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(keyword) + `\b`)
		result = pattern.ReplaceAllString(result, "")
	}

	return strings.TrimSpace(result)
}

// SanitizeFilename sanitizes a filename to prevent path traversal
func (s *Sanitizer) SanitizeFilename(filename string) string {
	// Remove path separators
	result := strings.ReplaceAll(filename, "/", "")
	result = strings.ReplaceAll(result, "\\", "")
	result = strings.ReplaceAll(result, "..", "")

	// Remove control characters
	result = s.removeControlChars(result)

	// Remove leading/trailing dots and spaces
	result = strings.Trim(result, ". ")

	// Limit length
	if len(result) > 255 {
		result = result[:255]
	}

	return result
}

// SanitizeURL sanitizes a URL
func (s *Sanitizer) SanitizeURL(url string) string {
	// Basic URL validation - remove javascript: and data: protocols
	result := strings.TrimSpace(url)
	result = regexp.MustCompile(`(?i)^javascript:`).ReplaceAllString(result, "")
	result = regexp.MustCompile(`(?i)^data:`).ReplaceAllString(result, "")

	// Remove control characters
	result = s.removeControlChars(result)

	return result
}

// SanitizeEmail sanitizes an email address
func (s *Sanitizer) SanitizeEmail(email string) string {
	result := strings.TrimSpace(strings.ToLower(email))

	// Remove control characters
	result = s.removeControlChars(result)

	// Basic email validation
	if !strings.Contains(result, "@") {
		return ""
	}

	parts := strings.Split(result, "@")
	if len(parts) != 2 {
		return ""
	}

	// Sanitize local and domain parts
	local := s.Sanitize(parts[0])
	domain := s.Sanitize(parts[1])

	if local == "" || domain == "" {
		return ""
	}

	return local + "@" + domain
}

// RemoveHTML removes all HTML tags from input
func RemoveHTML(input string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	result := re.ReplaceAllString(input, "")

	// Decode HTML entities
	result = html.UnescapeString(result)

	return result
}

// SanitizeString is a convenience function for quick sanitization
func SanitizeString(input string) string {
	sanitizer := DefaultSanitizer()
	return sanitizer.Sanitize(input)
}
