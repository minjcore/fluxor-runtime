// Package sanitize provides input sanitization and validation utilities.
//
// This package helps prevent common security vulnerabilities including:
//   - Cross-Site Scripting (XSS)
//   - SQL Injection
//   - Path Traversal
//   - Command Injection
//
// Features:
//   - HTML escaping and sanitization
//   - SQL injection pattern removal
//   - Filename sanitization
//   - URL sanitization
//   - Email sanitization
//   - Control character removal
//
// Example usage:
//
//	sanitizer := sanitize.DefaultSanitizer()
//
//	// Sanitize user input
//	userInput := "<script>alert('xss')</script>Hello"
//	safe := sanitizer.Sanitize(userInput)
//	// Result: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;Hello"
//
//	// Sanitize filename
//	filename := "../../etc/passwd"
//	safe := sanitizer.SanitizeFilename(filename)
//	// Result: "etcpasswd"
//
//	// Sanitize SQL input (use parameterized queries in production!)
//	sqlInput := "'; DROP TABLE users; --"
//	safe := sanitizer.SanitizeSQL(sqlInput)
//
//	// Quick sanitization
//	safe := sanitize.SanitizeString(userInput)
//
// Security considerations:
//   - Always use parameterized queries for SQL (SanitizeSQL is a last resort)
//   - Validate input before sanitizing
//   - Use appropriate sanitization for each context
//   - Consider using a dedicated HTML sanitizer library for complex HTML
//
// Path: pkg/security/sanitize
package sanitize
