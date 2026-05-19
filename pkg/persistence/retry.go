package persistence

import (
	"context"
	"database/sql"
	"time"
)

// retryableOperation executes an operation with retry logic
// Uses MaxRetries and RetryDelay from config
func (r *SQLRepository) retryableOperation(ctx context.Context, operation func(ctx context.Context) error) error {
	var lastErr error
	maxRetries := r.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1 // At least try once
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := operation(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			return err // Don't retry non-retryable errors
		}

		// Don't sleep on last attempt
		if attempt < maxRetries-1 {
			// Use configured retry delay
			retryDelay := r.config.RetryDelay
			if retryDelay <= 0 {
				retryDelay = 100 * time.Millisecond // Default
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				// Continue to next retry
			}
		}
	}

	return lastErr
}

// isRetryableError checks if an error is retryable
// Retryable errors: connection errors, timeouts, temporary failures
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for SQL connection errors
	if err == sql.ErrConnDone {
		return true
	}

	// Check for context timeout (might be retryable depending on scenario)
	// For now, we don't retry timeouts as they're usually intentional

	// Check error message for common retryable patterns
	errStr := err.Error()
	retryablePatterns := []string{
		"connection",
		"timeout",
		"temporary",
		"busy",
		"locked",
		"deadlock",
		"network",
		"dial",
	}

	for _, pattern := range retryablePatterns {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1 := s[i+j]
			c2 := substr[j]
			// Case-insensitive comparison
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 'a' - 'A'
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 'a' - 'A'
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
