package profiling

import (
	"testing"
	"time"
)

func TestWorkClassifier(t *testing.T) {
	classifier := NewWorkClassifier()

	// Test IO-bound classification
	ioStack := []string{
		"net/http.(*conn).serve",
		"net.(*conn).Read",
		"syscall.Read",
	}
	workType := classifier.Classify(1, ioStack)
	if workType != WorkTypeIOBound {
		t.Errorf("Expected IO-bound, got %s", workType)
	}

	// Test CPU-bound classification
	cpuStack := []string{
		"crypto/sha256.Sum256",
		"encoding/json.Marshal",
		"compress/gzip.Compress",
	}
	workType = classifier.Classify(2, cpuStack)
	if workType != WorkTypeCPUBound {
		t.Errorf("Expected CPU-bound, got %s", workType)
	}

	// Test mixed classification
	mixedStack := []string{
		"net/http.(*conn).serve",
		"crypto/sha256.Sum256",
		"encoding/json.Marshal",
	}
	workType = classifier.Classify(3, mixedStack)
	if workType != WorkTypeMixed {
		t.Errorf("Expected mixed, got %s", workType)
	}
}

func TestStackAnalyzer(t *testing.T) {
	analyzer := NewStackAnalyzer()

	// Test IO frame detection
	if !analyzer.IsIOFrame("net/http.(*conn).serve") {
		t.Error("Expected IO frame to be detected")
	}

	// Test CPU frame detection
	if !analyzer.IsCPUFrame("crypto/sha256.Sum256") {
		t.Error("Expected CPU frame to be detected")
	}

	// Test mixed stack
	mixedStack := []string{
		"net/http.(*conn).serve",
		"crypto/sha256.Sum256",
	}
	workType := analyzer.Analyze(mixedStack)
	if workType != WorkTypeMixed {
		t.Errorf("Expected mixed, got %s", workType)
	}
}

func TestGoroutineProfiler(t *testing.T) {
	classifier := NewWorkClassifier()
	profiler := NewGoroutineProfiler(classifier)

	stackTrace := []string{
		"net/http.(*conn).serve",
		"net.(*conn).Read",
	}
	profiler.Update(1, stackTrace, WorkTypeIOBound)

	profile := profiler.GetProfile(1)
	if profile == nil {
		t.Fatal("Expected profile to exist")
	}
	if profile.WorkType != WorkTypeIOBound {
		t.Errorf("Expected IO-bound, got %s", profile.WorkType)
	}

	stats := profiler.GetStats()
	if stats.Total != 1 {
		t.Errorf("Expected 1 goroutine, got %d", stats.Total)
	}
}

func TestBottleneckDetector(t *testing.T) {
	detector := NewBottleneckDetector()

	// Test queue full bottleneck
	metrics := &ServerMetricsForProfiling{
		QueuedRequests:   9500,
		QueueCapacity:      10000,
		QueueUtilization:   95.0,
		Workers:            100,
		CCUUtilization:     50.0,
		IOBoundWorkersActive: 50,
		CPUBoundWorkersActive: 0,
		MixedWorkersActive:   0,
	}

	goroutineStats := &GoroutineStats{
		Total:      100,
		MixedWork:  0,
	}

	bottlenecks := detector.Detect(metrics, goroutineStats)
	if len(bottlenecks) == 0 {
		t.Error("Expected bottleneck to be detected")
	}

	foundQueueFull := false
	for _, b := range bottlenecks {
		if b.Type == BottleneckQueueFull {
			foundQueueFull = true
			break
		}
	}
	if !foundQueueFull {
		t.Error("Expected queue full bottleneck")
	}
}

func TestRuntimeProfiler(t *testing.T) {
	classifier := NewWorkClassifier()
	profiler := NewGoroutineProfiler(classifier)
	runtimeProfiler := NewRuntimeProfiler(1*time.Second, classifier, profiler)

	if runtimeProfiler.IsRunning() {
		t.Error("Expected profiler to not be running initially")
	}

	// Test profiling
	err := runtimeProfiler.Profile()
	if err != nil {
		t.Errorf("Profile() returned error: %v", err)
	}

	// Test getting profiling data
	data := runtimeProfiler.GetProfilingData()
	if data == nil {
		t.Fatal("Expected profiling data")
	}
}
