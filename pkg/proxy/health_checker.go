package proxy

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// HealthChecker performs health checks on backends
type HealthChecker struct {
	timeout time.Duration
	client  *http.Client
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		timeout: timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// CheckHealth checks the health of a backend
func (hc *HealthChecker) CheckHealth(backend Backend) (bool, time.Duration, error) {
	// Use custom health check URL if provided (HTTP only)
	if backend.HealthCheckURL != "" {
		return hc.checkHTTPHealth(backend.HealthCheckURL)
	}

	// Default: TCP connection check
	return hc.checkTCPHealth(backend.URL)
}

// checkHTTPHealth performs HTTP health check
func (hc *HealthChecker) checkHTTPHealth(url string) (bool, time.Duration, error) {
	start := time.Now()

	resp, err := hc.client.Get(url)
	latency := time.Since(start)

	if err != nil {
		return false, latency, err
	}
	defer resp.Body.Close()

	// Consider healthy if status code is 2xx or 3xx
	healthy := resp.StatusCode >= 200 && resp.StatusCode < 400
	return healthy, latency, nil
}

// checkTCPHealth performs TCP connection health check
func (hc *HealthChecker) checkTCPHealth(url string) (bool, time.Duration, error) {
	start := time.Now()

	addr, err := parseBackendAddr(url)
	if err != nil {
		return false, time.Since(start), fmt.Errorf("invalid backend URL: %w", err)
	}

	conn, err := net.DialTimeout("tcp", addr, hc.timeout)
	latency := time.Since(start)

	if err != nil {
		return false, latency, err
	}
	conn.Close()

	return true, latency, nil
}
