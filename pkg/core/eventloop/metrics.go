package eventloop

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Metrics registry
	metricsOnce sync.Once
	metrics     *EventLoopMetrics

	// Metric descriptors
	queueLengthDesc = prometheus.NewDesc(
		"fluxor_eventloop_queue_length",
		"Current queue length per event loop",
		[]string{"loop_id"},
		nil,
	)

	droppedMessagesDesc = prometheus.NewDesc(
		"fluxor_eventloop_dropped_messages_total",
		"Total dropped messages per event loop",
		[]string{"loop_id"},
		nil,
	)

	processedMessagesDesc = prometheus.NewDesc(
		"fluxor_eventloop_processed_messages_total",
		"Total processed messages per event loop",
		[]string{"loop_id"},
		nil,
	)

	latencyDesc = prometheus.NewDesc(
		"fluxor_eventloop_latency_seconds",
		"Average latency per event loop (seconds)",
		[]string{"loop_id"},
		nil,
	)
)

// EventLoopMetrics provides Prometheus metrics for event loops
type EventLoopMetrics struct {
	mu sync.RWMutex
}

// GetMetrics returns the global metrics instance
func GetMetrics() *EventLoopMetrics {
	metricsOnce.Do(func() {
		metrics = &EventLoopMetrics{}
	})
	return metrics
}

// Describe implements prometheus.Collector
func (m *EventLoopMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- queueLengthDesc
	ch <- droppedMessagesDesc
	ch <- processedMessagesDesc
	ch <- latencyDesc
}

// Collect implements prometheus.Collector
func (m *EventLoopMetrics) Collect(ch chan<- prometheus.Metric) {
	// This will be populated by EventLoopGroup when metrics are enabled
	// For now, return empty metrics
}

// UpdateMetrics updates metrics from loop statistics
func (m *EventLoopMetrics) UpdateMetrics(stats []LoopMetrics) {
	// This is called by EventLoopGroup to update metrics
	// Implementation can be enhanced to track histograms, etc.
}

// RegisterMetrics registers event loop metrics with a Prometheus registry
func RegisterMetrics(registry prometheus.Registerer, group *EventLoopGroup) {
	if group == nil || !group.config.Metrics {
		return
	}

	collector := &eventLoopCollector{group: group}
	registry.MustRegister(collector)
}

// eventLoopCollector collects metrics from EventLoopGroup
type eventLoopCollector struct {
	group *EventLoopGroup
}

func (c *eventLoopCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- queueLengthDesc
	ch <- droppedMessagesDesc
	ch <- processedMessagesDesc
	ch <- latencyDesc
}

func (c *eventLoopCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.group.Stats()
	for _, stat := range stats {
		loopID := fmt.Sprintf("%d", stat.LoopID)

		ch <- prometheus.MustNewConstMetric(
			queueLengthDesc,
			prometheus.GaugeValue,
			float64(stat.QueueLength),
			loopID,
		)

		ch <- prometheus.MustNewConstMetric(
			droppedMessagesDesc,
			prometheus.CounterValue,
			float64(stat.DroppedMessages),
			loopID,
		)

		ch <- prometheus.MustNewConstMetric(
			processedMessagesDesc,
			prometheus.CounterValue,
			float64(stat.ProcessedMessages),
			loopID,
		)

		ch <- prometheus.MustNewConstMetric(
			latencyDesc,
			prometheus.GaugeValue,
			stat.AvgLatency.Seconds(),
			loopID,
		)
	}
}
