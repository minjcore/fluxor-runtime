package dashboard

import (
	"github.com/fluxorio/fluxor/pkg/web"
)

func init() {
	// Register function to auto-register FastHTTPServer instances
	web.SetHTTPServerRegistrar(func(server *web.FastHTTPServer) {
		collector := GetMetricsCollector()
		// Use server address as name, or generate unique name
		name := "http-server"
		if len(collector.httpServers) > 0 {
			name = "http-server-" + string(rune(len(collector.httpServers)+1))
		}
		// Create adapter to implement HTTPServerMetricsProvider
		// Convert web.HTTPServerMetricsData to dashboard.HTTPServerMetricsData
		adapter := NewHTTPServerAdapter(func() HTTPServerMetricsData {
			webMetrics := server.GetHTTPServerMetrics()
			return HTTPServerMetricsData{
				QueuedRequests:        webMetrics.QueuedRequests,
				RejectedRequests:      webMetrics.RejectedRequests,
				TotalRequests:         webMetrics.TotalRequests,
				SuccessfulRequests:    webMetrics.SuccessfulRequests,
				ErrorRequests:         webMetrics.ErrorRequests,
				QueueCapacity:         webMetrics.QueueCapacity,
				QueueUtilization:      webMetrics.QueueUtilization,
				Workers:               webMetrics.Workers,
				CurrentCCU:            webMetrics.CurrentCCU,
				CCUUtilization:        webMetrics.CCUUtilization,
				BytesSent:             webMetrics.BytesSent,
				BytesReceived:         webMetrics.BytesReceived,
				AverageLatencyMs:      webMetrics.AverageLatencyMs,
				ArrivalRate:           webMetrics.ArrivalRate,
				ExpectedQueueLength:   webMetrics.ExpectedQueueLength,
				LittlesLawValidation:  webMetrics.LittlesLawValidation,
			}
		})
		collector.RegisterHTTPServer(name, adapter)
		// Start profiling if not already started
		// Pass nil - StartProfiling will use context.Background() internally
		collector.StartProfiling(nil)
	})
}

// RegisterHTTPServerForMetrics registers an HTTP server with the metrics collector
// This can be called manually if auto-registration is not desired
func RegisterHTTPServerForMetrics(name string, server HTTPServerMetricsProvider) {
	collector := GetMetricsCollector()
	collector.RegisterHTTPServer(name, server)
	// Start profiling if not already started
	// Pass nil - StartProfiling will use context.Background() internally
	collector.StartProfiling(nil)
}
