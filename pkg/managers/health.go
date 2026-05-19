package managers

import (
	"context"
	"sync"
	"time"
)

// HealthCheck represents a health check function
type HealthCheck func(ctx context.Context) error

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Name      string        // Check name
	Healthy   bool          // Whether check passed
	Error     string        // Error message if unhealthy
	Duration  time.Duration // How long check took
	Timestamp time.Time     // When check was performed
}

// HealthChecker manages health checks for components
type HealthChecker struct {
	checks map[string]HealthCheck
	mu     sync.RWMutex
}

// newHealthChecker creates a new health checker
func newHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]HealthCheck),
	}
}

// Register registers a health check
func (hc *HealthChecker) Register(name string, check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = check
}

// Unregister removes a health check
func (hc *HealthChecker) Unregister(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	delete(hc.checks, name)
}

// CheckAll runs all registered health checks
func (hc *HealthChecker) CheckAll(ctx context.Context) []HealthCheckResult {
	hc.mu.RLock()
	checksToRun := make(map[string]HealthCheck, len(hc.checks))
	for name, check := range hc.checks {
		checksToRun[name] = check
	}
	hc.mu.RUnlock()

	results := make([]HealthCheckResult, 0, len(checksToRun))
	for name, check := range checksToRun {
		result := hc.runCheck(ctx, name, check)
		results = append(results, result)
	}
	return results
}

// runCheck runs a single health check
func (hc *HealthChecker) runCheck(ctx context.Context, name string, check HealthCheck) HealthCheckResult {
	start := time.Now()
	result := HealthCheckResult{
		Name:      name,
		Timestamp: start,
	}

	// Create timeout context
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Run check
	err := check(checkCtx)
	result.Duration = time.Since(start)

	if err != nil {
		result.Healthy = false
		result.Error = err.Error()
	} else {
		result.Healthy = true
	}

	return result
}

// IsHealthy returns true if all checks pass
func (hc *HealthChecker) IsHealthy(ctx context.Context) bool {
	results := hc.CheckAll(ctx)
	for _, result := range results {
		if !result.Healthy {
			return false
		}
	}
	return true
}

// Managers methods for health checking

// RegisterHealthCheck registers a health check with Managers
func (m *Managers) RegisterHealthCheck(name string, check HealthCheck) {
	if m.healthChecker == nil {
		m.mu.Lock()
		if m.healthChecker == nil {
			m.healthChecker = newHealthChecker()
		}
		m.mu.Unlock()
	}
	m.healthChecker.Register(name, check)
}

// UnregisterHealthCheck removes a health check
func (m *Managers) UnregisterHealthCheck(name string) {
	if m.healthChecker != nil {
		m.healthChecker.Unregister(name)
	}
}

// CheckHealth runs all registered health checks
func (m *Managers) CheckHealth(ctx context.Context) []HealthCheckResult {
	if m.healthChecker == nil {
		return []HealthCheckResult{}
	}
	return m.healthChecker.CheckAll(ctx)
}

// IsHealthy returns true if all health checks pass
func (m *Managers) IsHealthy(ctx context.Context) bool {
	if m.healthChecker == nil {
		return true
	}
	return m.healthChecker.IsHealthy(ctx)
}

// DefaultHealthChecks registers default health checks for Managers components
func (m *Managers) DefaultHealthChecks() {
	// HTTP Server health check
	m.RegisterHealthCheck("http-server", func(ctx context.Context) error {
		server := m.HTTPServer()
		if server == nil {
			return nil // No HTTP server registered
		}
		// Check if server is running (implementation depends on server type)
		return nil
	})

	// Cache health check
	m.RegisterHealthCheck("cache", func(ctx context.Context) error {
		cache := m.Cache()
		if cache == nil {
			return nil // No cache registered
		}
		// Try a simple operation
		key := "managers-health-check"
		value := []byte("ok")
		if err := cache.Set(ctx, key, value, 1*time.Second); err != nil {
			return err
		}
		_, err := cache.Get(ctx, key)
		return err
	})

	// EventBus health check
	m.RegisterHealthCheck("eventbus", func(ctx context.Context) error {
		eventBus := m.EventBus()
		if eventBus == nil {
			return nil // No EventBus
		}
		// EventBus is available
		return nil
	})
}
