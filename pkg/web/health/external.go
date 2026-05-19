package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPCheck creates a health check for an external HTTP service
func HTTPCheck(url string, timeout time.Duration) Checker {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return func(ctx context.Context) error {
		client := &http.Client{
			Timeout: timeout,
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return &Error{Message: fmt.Sprintf("failed to create request: %v", err)}
		}

		resp, err := client.Do(req)
		if err != nil {
			return &Error{Message: fmt.Sprintf("HTTP request failed: %v", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return &Error{Message: fmt.Sprintf("HTTP status %d", resp.StatusCode)}
		}

		return nil
	}
}

// HTTPCheckWithHeaders creates a health check for an external HTTP service with custom headers
func HTTPCheckWithHeaders(url string, timeout time.Duration, headers map[string]string) Checker {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	return func(ctx context.Context) error {
		client := &http.Client{
			Timeout: timeout,
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return &Error{Message: fmt.Sprintf("failed to create request: %v", err)}
		}

		// Add custom headers
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		if err != nil {
			return &Error{Message: fmt.Sprintf("HTTP request failed: %v", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return &Error{Message: fmt.Sprintf("HTTP status %d", resp.StatusCode)}
		}

		return nil
	}
}
