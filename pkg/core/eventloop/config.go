package eventloop

import "time"

// BackpressurePolicy defines how to handle messages when queue is full
type BackpressurePolicy int

const (
	// BackpressureBlock waits when queue is full (default)
	BackpressureBlock BackpressurePolicy = iota
	// BackpressureDrop drops message when queue is full
	BackpressureDrop
	// BackpressureDropByTopic drops messages by topic priority (future)
	BackpressureDropByTopic
)

// EventLoopConfig configures an EventLoopGroup
type EventLoopConfig struct {
	// NumLoops is the number of event loops to create
	// Default: runtime.GOMAXPROCS(0)
	NumLoops int

	// QueueSize is the maximum queue size per loop (bounded for backpressure)
	// Default: 4096
	QueueSize int

	// CPUAffinity pins each loop goroutine to a CPU core
	// Default: false (let Go scheduler manage)
	CPUAffinity bool

	// Backpressure defines behavior when queue is full
	// Default: BackpressureBlock
	Backpressure BackpressurePolicy

	// Metrics enables per-loop metrics collection
	// Default: true
	Metrics bool
}

// DefaultEventLoopConfig returns default configuration
func DefaultEventLoopConfig() EventLoopConfig {
	return EventLoopConfig{
		NumLoops:     0, // Will be set to GOMAXPROCS(0) if 0
		QueueSize:    4096,
		CPUAffinity:  false,
		Backpressure: BackpressureBlock,
		Metrics:      true,
	}
}

// Validate validates the configuration
func (c *EventLoopConfig) Validate() error {
	if c.NumLoops < 0 {
		return &EventLoopError{Code: "INVALID_CONFIG", Message: "NumLoops cannot be negative"}
	}
	if c.QueueSize < 1 {
		return &EventLoopError{Code: "INVALID_CONFIG", Message: "QueueSize must be at least 1"}
	}
	return nil
}

// EventLoopError represents an event loop error
type EventLoopError struct {
	Code    string
	Message string
}

func (e *EventLoopError) Error() string {
	return e.Code + ": " + e.Message
}

// LoopMetrics provides metrics for a single event loop
type LoopMetrics struct {
	LoopID            int
	QueueLength       int64
	DroppedMessages   int64
	ProcessedMessages int64
	AvgLatency        time.Duration
	P50Latency        time.Duration
	P95Latency        time.Duration
	P99Latency        time.Duration
	Throughput        float64 // messages per second
}
