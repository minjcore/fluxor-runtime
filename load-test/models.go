// Package main provides type definitions for monitoring metrics
package main

import "time"

// MetricsSnapshot represents a single metrics snapshot
type MetricsSnapshot struct {
	Timestamp       time.Time   `json:"timestamp"`
	Phase           string      `json:"phase"`
	Goroutines      int         `json:"goroutines"`
	MemoryAlloc     uint64      `json:"memoryAlloc"`
	MemoryTotalAlloc uint64     `json:"memoryTotalAlloc"`
	MemorySys       uint64      `json:"memorySys"`
	QueueUtil       float64     `json:"queueUtil"`
	QueueCapacity   int         `json:"queueCapacity"`
	QueuedRequests  int64       `json:"queuedRequests"`
	CCU             int         `json:"ccu"`
	CCUUtil         float64     `json:"ccuUtil"`
	IOActive        int         `json:"ioActive"`
	CPUActive       int         `json:"cpuActive"`
	MixedActive     int         `json:"mixedActive"`
	Bottlenecks     []Bottleneck `json:"bottlenecks"`
	AllocRate       float64     `json:"allocRate"`
	GCRate          float64     `json:"gcRate"`
	NumGC           uint32      `json:"numGC"`
}

// Bottleneck represents a detected bottleneck
type Bottleneck struct {
	Type          string `json:"type"`
	Severity      string `json:"severity"`
	Description   string `json:"description"`
	Recommendation string `json:"recommendation,omitempty"`
}

// PhaseStats represents statistics for a monitoring phase
type PhaseStats struct {
	Duration       string    `json:"duration"`
	Samples        int       `json:"samples"`
	StartTime      time.Time `json:"startTime"`
	EndTime        time.Time `json:"endTime"`
	AvgGoroutines  float64   `json:"avgGoroutines"`
	MaxGoroutines  int       `json:"maxGoroutines"`
	MinGoroutines  int       `json:"minGoroutines"`
	AvgMemoryAlloc float64   `json:"avgMemoryAlloc"`
	MaxMemoryAlloc uint64    `json:"maxMemoryAlloc"`
	MinMemoryAlloc uint64    `json:"minMemoryAlloc"`
	AvgQueueUtil   float64   `json:"avgQueueUtil"`
	MaxQueueUtil   float64   `json:"maxQueueUtil"`
	FinalGoroutines int      `json:"finalGoroutines"`
	FinalMemoryAlloc uint64  `json:"finalMemoryAlloc"`
	FinalQueueUtil  float64  `json:"finalQueueUtil"`
	FinalCCU        int      `json:"finalCCU"`
	FinalCCUUtil    float64  `json:"finalCCUUtil"`

	// GC-aware fields
	GCCount     uint32  `json:"gcCount"`     // GC cycles that ran during this phase
	MemoryTrend float64 `json:"memoryTrend"` // bytes/sample: negative = releasing, positive = growing
	AllocRateAvg float64 `json:"allocRateAvg"` // average alloc rate (bytes/s)
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
	Baseline    float64 `json:"baseline,omitempty"`
	Recovery    float64 `json:"recovery,omitempty"`
	Threshold   float64 `json:"threshold,omitempty"`
	Difference  float64 `json:"difference,omitempty"`
	PercentDiff float64 `json:"percentDiff,omitempty"`
}

// MonitoringReport represents the complete monitoring report
type MonitoringReport struct {
	Baseline  PhaseStats   `json:"baseline"`
	Load      PhaseStats   `json:"load"`
	Recovery  PhaseStats   `json:"recovery"`
	Anomalies []Anomaly    `json:"anomalies"`
	Summary   SummaryStats `json:"summary"`
}

// SummaryStats represents summary statistics
type SummaryStats struct {
	Status         string   `json:"status"`
	Issues         int      `json:"issues"`
	Recommendations []string `json:"recommendations"`
}

// MonitorConfig holds monitoring configuration
type MonitorConfig struct {
	ServerURL      string
	Phase          string
	Duration       time.Duration
	Interval       time.Duration
	OutputFile     string
	AnalyzeFiles   []string
	SkipProfiling  bool
}
