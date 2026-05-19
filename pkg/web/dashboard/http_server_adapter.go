package dashboard

// HTTPServerAdapter adapts FastHTTPServer to HTTPServerMetricsProvider
// This is created in web package to avoid circular dependency
type HTTPServerAdapter struct {
	getMetrics func() HTTPServerMetricsData
}

// NewHTTPServerAdapter creates a new adapter
// The getMetrics function should return metrics from web.HTTPServerMetricsData
// and convert it to dashboard.HTTPServerMetricsData
func NewHTTPServerAdapter(getMetrics func() HTTPServerMetricsData) *HTTPServerAdapter {
	return &HTTPServerAdapter{getMetrics: getMetrics}
}

// Metrics implements HTTPServerMetricsProvider
func (a *HTTPServerAdapter) Metrics() HTTPServerMetricsData {
	return a.getMetrics()
}
