package managers

import (
	"github.com/fluxorio/fluxor/pkg/observability/prometheus"
)

// CreateMetrics creates a metrics collector based on Managers configuration
func (m *Managers) CreateMetrics() (*prometheus.Metrics, error) {
	if !m.config.EnableMetrics {
		return nil, nil // Metrics disabled
	}

	// Use the global metrics instance from prometheus package
	// This ensures consistent metrics across the application
	return prometheus.GetMetrics(), nil
}

// Metrics returns the registered metrics instance
func (m *Managers) Metrics() *prometheus.Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}
