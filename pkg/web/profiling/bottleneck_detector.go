package profiling

import (
	"fmt"
)

// BottleneckType represents the type of bottleneck
type BottleneckType string

const (
	BottleneckQueueFull BottleneckType = "queue_full"
	BottleneckIOBound   BottleneckType = "io_bound"
	BottleneckCPUBound  BottleneckType = "cpu_bound"
	BottleneckMixedWork BottleneckType = "mixed_work"
	BottleneckNone      BottleneckType = "none"
)

// Severity represents the severity of a bottleneck
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Bottleneck represents a detected bottleneck
type Bottleneck struct {
	Type          BottleneckType `json:"type"`
	Severity      Severity       `json:"severity"`
	Description   string         `json:"description"`
	Recommendation string        `json:"recommendation"`
}

// ServerMetricsForProfiling extends ServerMetrics with profiling-specific data
type ServerMetricsForProfiling struct {
	// Basic metrics
	QueuedRequests   int64
	RejectedRequests int64
	QueueCapacity    int
	Workers          int
	QueueUtilization float64
	CurrentCCU       int
	CCUUtilization   float64
	
	// Profiling metrics
	IOBoundWorkersActive  int
	CPUBoundWorkersActive int
	MixedWorkersActive    int
	IOBoundQueueLength    int64
	CPUBoundQueueLength   int64
}

// BottleneckDetector detects bottlenecks in the system
type BottleneckDetector struct{}

// NewBottleneckDetector creates a new bottleneck detector
func NewBottleneckDetector() *BottleneckDetector {
	return &BottleneckDetector{}
}

// Detect detects bottlenecks based on metrics
func (bd *BottleneckDetector) Detect(metrics *ServerMetricsForProfiling, goroutineStats *GoroutineStats) []Bottleneck {
	var bottlenecks []Bottleneck

	// 1. Check for queue full bottleneck
	if metrics.QueueUtilization > 90.0 {
		severity := SeverityMedium
		if metrics.QueueUtilization > 95.0 {
			severity = SeverityHigh
		}
		if metrics.QueueUtilization >= 100.0 {
			severity = SeverityCritical
		}
		
		bottlenecks = append(bottlenecks, Bottleneck{
			Type:          BottleneckQueueFull,
			Severity:      severity,
			Description:   fmt.Sprintf("Queue utilization at %.2f%%, queue is nearly full", metrics.QueueUtilization),
			Recommendation: "Increase queue size or reduce incoming request rate",
		})
	}

	// 2. Check for mixed work anti-pattern
	if metrics.MixedWorkersActive > 0 || goroutineStats.MixedWork > 0 {
		severity := SeverityLow
		if metrics.MixedWorkersActive > 5 || goroutineStats.MixedWork > 5 {
			severity = SeverityMedium
		}
		if metrics.MixedWorkersActive > 10 || goroutineStats.MixedWork > 10 {
			severity = SeverityHigh
		}
		
		bottlenecks = append(bottlenecks, Bottleneck{
			Type:          BottleneckMixedWork,
			Severity:      severity,
			Description:   fmt.Sprintf("%d workers detected doing both IO and CPU work (anti-pattern)", metrics.MixedWorkersActive),
			Recommendation: "Offload CPU work to dedicated CPU-bound worker pool",
		})
	}

	// 3. Check for IO-bound bottleneck
	var ioUtilization float64
	if metrics.Workers > 0 {
		ioUtilization = float64(metrics.IOBoundWorkersActive) / float64(metrics.Workers) * 100.0
	}
	if ioUtilization > 80.0 && metrics.QueueUtilization > 50.0 {
		severity := SeverityMedium
		if ioUtilization > 90.0 {
			severity = SeverityHigh
		}
		
		bottlenecks = append(bottlenecks, Bottleneck{
			Type:          BottleneckIOBound,
			Severity:      severity,
			Description:   fmt.Sprintf("IO-bound workers saturated (%.1f%% utilization), queue filling up", ioUtilization),
			Recommendation: "Increase IO-bound workers or optimize IO operations",
		})
	}

	// 4. Check for CPU-bound bottleneck
	var cpuUtilization float64
	if metrics.Workers > 0 {
		cpuUtilization = float64(metrics.CPUBoundWorkersActive) / float64(metrics.Workers) * 100.0
	}
	if cpuUtilization > 80.0 && metrics.CCUUtilization > 70.0 {
		severity := SeverityMedium
		if cpuUtilization > 90.0 {
			severity = SeverityHigh
		}
		
		bottlenecks = append(bottlenecks, Bottleneck{
			Type:          BottleneckCPUBound,
			Severity:      severity,
			Description:   fmt.Sprintf("CPU-bound workers saturated (%.1f%% utilization)", cpuUtilization),
			Recommendation: "Offload CPU work to dedicated CPU-bound worker pool or optimize CPU operations",
		})
	}

	// 5. Check for queue imbalance
	if metrics.IOBoundQueueLength > 0 && metrics.CPUBoundQueueLength == 0 {
		// IO queue has work but CPU queue is empty - might indicate mixed work
		if metrics.MixedWorkersActive > 0 {
			bottlenecks = append(bottlenecks, Bottleneck{
				Type:          BottleneckMixedWork,
				Severity:      SeverityLow,
				Description:   "IO queue has work but CPU queue is empty - workers may be doing CPU work in IO handlers",
				Recommendation: "Separate IO and CPU work into different worker pools",
			})
		}
	}

	// If no bottlenecks detected, return empty slice
	if len(bottlenecks) == 0 {
		return []Bottleneck{}
	}

	return bottlenecks
}
