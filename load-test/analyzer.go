package main

import (
	"fmt"
	"math"
)

// MetricsAnalyzer analyzes metrics snapshots and detects anomalies
type MetricsAnalyzer struct{}

// NewMetricsAnalyzer creates a new metrics analyzer
func NewMetricsAnalyzer() *MetricsAnalyzer {
	return &MetricsAnalyzer{}
}

// CalculatePhaseStats calculates statistics for a phase from snapshots
func (ma *MetricsAnalyzer) CalculatePhaseStats(snapshots []MetricsSnapshot) PhaseStats {
	if len(snapshots) == 0 {
		return PhaseStats{}
	}

	stats := PhaseStats{
		Samples:        len(snapshots),
		StartTime:      snapshots[0].Timestamp,
		EndTime:        snapshots[len(snapshots)-1].Timestamp,
		MinGoroutines:  snapshots[0].Goroutines,
		MinMemoryAlloc: snapshots[0].MemoryAlloc,
	}

	var sumGoroutines, sumMemoryAlloc, sumQueueUtil, sumAllocRate float64

	for _, s := range snapshots {
		sumGoroutines += float64(s.Goroutines)
		sumMemoryAlloc += float64(s.MemoryAlloc)
		sumQueueUtil += s.QueueUtil
		sumAllocRate += s.AllocRate

		if s.Goroutines > stats.MaxGoroutines {
			stats.MaxGoroutines = s.Goroutines
		}
		if s.Goroutines < stats.MinGoroutines {
			stats.MinGoroutines = s.Goroutines
		}
		if s.MemoryAlloc > stats.MaxMemoryAlloc {
			stats.MaxMemoryAlloc = s.MemoryAlloc
		}
		if s.MemoryAlloc < stats.MinMemoryAlloc {
			stats.MinMemoryAlloc = s.MemoryAlloc
		}
		if s.QueueUtil > stats.MaxQueueUtil {
			stats.MaxQueueUtil = s.QueueUtil
		}
	}

	n := float64(len(snapshots))
	stats.AvgGoroutines = sumGoroutines / n
	stats.AvgMemoryAlloc = sumMemoryAlloc / n
	stats.AvgQueueUtil = sumQueueUtil / n
	stats.AllocRateAvg = sumAllocRate / n

	last := snapshots[len(snapshots)-1]
	stats.FinalGoroutines = last.Goroutines
	stats.FinalMemoryAlloc = last.MemoryAlloc
	stats.FinalQueueUtil = last.QueueUtil
	stats.FinalCCU = last.CCU
	stats.FinalCCUUtil = last.CCUUtil
	stats.Duration = stats.EndTime.Sub(stats.StartTime).String()

	// GC cycles that ran during this phase (cumulative counter delta)
	if len(snapshots) > 1 {
		stats.GCCount = last.NumGC - snapshots[0].NumGC
	}

	// Memory trend: linear regression slope (bytes/sample).
	// Negative → releasing, positive → growing.
	stats.MemoryTrend = calcTrend(snapshots)

	return stats
}

// calcTrend returns the least-squares slope of MemoryAlloc over sample index (bytes/sample).
func calcTrend(snapshots []MetricsSnapshot) float64 {
	n := float64(len(snapshots))
	if n < 2 {
		return 0
	}
	var sumX, sumY, sumXY, sumX2 float64
	for i, s := range snapshots {
		x := float64(i)
		y := float64(s.MemoryAlloc)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-9 {
		return 0
	}
	return (n*sumXY - sumX*sumY) / denom
}

// AnalyzeMetrics creates a monitoring report from phase snapshot files.
func (ma *MetricsAnalyzer) AnalyzeMetrics(files map[string]string) (*MonitoringReport, error) {
	report := &MonitoringReport{Anomalies: []Anomaly{}}

	var baselineSnaps, recoverySnaps []MetricsSnapshot

	if f, ok := files["baseline"]; ok {
		snaps, err := LoadSnapshots(f)
		if err != nil {
			return nil, fmt.Errorf("error loading baseline: %w", err)
		}
		baselineSnaps = snaps
		report.Baseline = ma.CalculatePhaseStats(snaps)
	}

	if f, ok := files["load"]; ok {
		snaps, err := LoadSnapshots(f)
		if err != nil {
			return nil, fmt.Errorf("error loading load phase: %w", err)
		}
		report.Load = ma.CalculatePhaseStats(snaps)
	}

	if f, ok := files["recovery"]; ok {
		snaps, err := LoadSnapshots(f)
		if err != nil {
			return nil, fmt.Errorf("error loading recovery: %w", err)
		}
		recoverySnaps = snaps
		report.Recovery = ma.CalculatePhaseStats(snaps)
	}

	ma.detectAnomalies(report, baselineSnaps, recoverySnaps)
	ma.generateSummary(report)
	return report, nil
}

// detectAnomalies compares baseline and recovery with GC-aware memory classification.
func (ma *MetricsAnalyzer) detectAnomalies(report *MonitoringReport, baselineSnaps, recoverySnaps []MetricsSnapshot) {
	baseline := report.Baseline
	recovery := report.Recovery

	if baseline.Samples == 0 || recovery.Samples == 0 {
		return
	}

	// --- Goroutine leak ---
	// Compare recovery final vs baseline avg. Goroutines are cheap to reason about:
	// a 10% increase that persists is a real concern.
	goroutineThreshold := baseline.AvgGoroutines * 1.10
	if float64(recovery.FinalGoroutines) > goroutineThreshold {
		diff := float64(recovery.FinalGoroutines) - baseline.AvgGoroutines
		pct := diff / baseline.AvgGoroutines * 100
		sev := "low"
		if pct > 20 {
			sev = "high"
		} else if pct > 10 {
			sev = "medium"
		}
		report.Anomalies = append(report.Anomalies, Anomaly{
			Type:        "goroutine_leak",
			Severity:    sev,
			Description: fmt.Sprintf("Goroutines +%.0f (+%.1f%%) above baseline after recovery", diff, pct),
			Baseline:    baseline.AvgGoroutines,
			Recovery:    float64(recovery.FinalGoroutines),
			Threshold:   goroutineThreshold,
			Difference:  diff,
			PercentDiff: pct,
		})
	}

	// --- Memory: GC-aware classification ---
	// Reference points chosen to be conservative:
	//   baseline ref = MaxMemoryAlloc (worst idle point — server already had a chance to GC)
	//   recovery best = MinMemoryAlloc (best point after GC ran during recovery)
	// This avoids false positives from cold-start vs warm-server comparisons.
	baselineRef := float64(baseline.MaxMemoryAlloc)
	recoveryBest := float64(recovery.MinMemoryAlloc)

	if baselineRef > 0 {
		ratio := recoveryBest / baselineRef
		trend := recovery.MemoryTrend // bytes/sample
		gcRan := recovery.GCCount > 0

		switch {
		case trend < -100_000:
			// Memory decreasing fast during recovery: GC actively releasing.
			// Informational only — not an error.
			report.Anomalies = append(report.Anomalies, Anomaly{
				Type:        "gc_releasing",
				Severity:    "info",
				Description: fmt.Sprintf("Memory releasing during recovery (%.0f bytes/sample) — GC working normally", trend),
				Baseline:    baselineRef,
				Recovery:    recoveryBest,
				PercentDiff: (recoveryBest - baselineRef) / baselineRef * 100,
			})

		case ratio <= 1.5:
			// Within 1.5× baseline max — normal working-set variation. No anomaly.

		case !gcRan:
			// Recovery phase ended before GC had a chance to sweep.
			// Flag as low — extend recovery window before concluding.
			report.Anomalies = append(report.Anomalies, Anomaly{
				Type:        "memory_gc_pending",
				Severity:    "low",
				Description: fmt.Sprintf("Heap %.1f× baseline max, but GC has not run during recovery — extend observation window", ratio),
				Baseline:    baselineRef,
				Recovery:    recoveryBest,
				Threshold:   baselineRef * 1.5,
				PercentDiff: (recoveryBest - baselineRef) / baselineRef * 100,
			})

		case ratio > 3.0:
			// GC ran but heap is still >3× baseline max — strong signal of retained heap.
			report.Anomalies = append(report.Anomalies, Anomaly{
				Type:        "memory_retained",
				Severity:    "high",
				Description: fmt.Sprintf("Heap %.1f× baseline max after %d GC cycles — likely working-set growth or cache retained (%.1f MB above baseline)", ratio, recovery.GCCount, (recoveryBest-baselineRef)/1024/1024),
				Baseline:    baselineRef,
				Recovery:    recoveryBest,
				Threshold:   baselineRef * 3.0,
				Difference:  recoveryBest - baselineRef,
				PercentDiff: (recoveryBest - baselineRef) / baselineRef * 100,
			})

		default:
			// 1.5×–3× baseline max after GC: likely connection-pool warm-up or
			// request-scoped caches (DB prepared statements, HTTP keep-alive buffers).
			// Medium severity — worth investigating but not alarming.
			report.Anomalies = append(report.Anomalies, Anomaly{
				Type:        "memory_retained",
				Severity:    "medium",
				Description: fmt.Sprintf("Heap %.1f× baseline max after %d GC cycles — likely warm working set (pool buffers, caches)", ratio, recovery.GCCount),
				Baseline:    baselineRef,
				Recovery:    recoveryBest,
				Threshold:   baselineRef * 1.5,
				Difference:  recoveryBest - baselineRef,
				PercentDiff: (recoveryBest - baselineRef) / baselineRef * 100,
			})
		}
	}

	// --- Queue not idle ---
	if recovery.FinalQueueUtil > 0.1 {
		report.Anomalies = append(report.Anomalies, Anomaly{
			Type:        "queue_not_idle",
			Severity:    "medium",
			Description: fmt.Sprintf("Queue utilization not idle: %.2f%%", recovery.FinalQueueUtil),
			Recovery:    recovery.FinalQueueUtil,
		})
	}

	// --- CCU not idle ---
	if recovery.FinalCCU > 0 {
		report.Anomalies = append(report.Anomalies, Anomaly{
			Type:        "ccu_not_idle",
			Severity:    "low",
			Description: fmt.Sprintf("CCU not idle: %d active connections", recovery.FinalCCU),
			Recovery:    float64(recovery.FinalCCU),
		})
	}
}

// generateSummary generates summary statistics and recommendations.
func (ma *MetricsAnalyzer) generateSummary(report *MonitoringReport) {
	summary := &report.Summary
	// Count only non-info anomalies as real issues
	for _, a := range report.Anomalies {
		if a.Severity != "info" {
			summary.Issues++
		}
	}

	if summary.Issues == 0 {
		summary.Status = "healthy"
		summary.Recommendations = []string{"No anomalies detected. System recovered to baseline state."}
	} else {
		critical, high := 0, 0
		for _, a := range report.Anomalies {
			switch a.Severity {
			case "critical":
				critical++
			case "high":
				high++
			}
		}
		switch {
		case critical > 0:
			summary.Status = "critical"
		case high > 0:
			summary.Status = "warning"
		default:
			summary.Status = "info"
		}

		for _, a := range report.Anomalies {
			if a.Severity == "info" {
				continue
			}
			switch a.Type {
			case "goroutine_leak":
				summary.Recommendations = append(summary.Recommendations,
					fmt.Sprintf("Goroutines leaked: %.0f not released — check for unclosed goroutines in handlers", a.Difference))
			case "memory_retained":
				summary.Recommendations = append(summary.Recommendations,
					fmt.Sprintf("Heap %.1f× baseline after GC — inspect DB pool buffers, HTTP keep-alive caches, and prepared-statement caches", a.Recovery/a.Baseline))
			case "memory_gc_pending":
				summary.Recommendations = append(summary.Recommendations,
					"Extend recovery window to ≥60s to give GC time to sweep before concluding")
			case "queue_not_idle":
				summary.Recommendations = append(summary.Recommendations,
					"Queue not returning to idle — check for stuck requests or undersized worker pool")
			case "ccu_not_idle":
				summary.Recommendations = append(summary.Recommendations,
					"CCU not zero — check for connection leaks in handlers")
			}
		}
	}
}
