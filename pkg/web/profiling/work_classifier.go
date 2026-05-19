package profiling

import (
	"sync"
)

// WorkType represents the type of work being performed
type WorkType string

const (
	WorkTypeIOBound  WorkType = "io-bound"
	WorkTypeCPUBound WorkType = "cpu-bound"
	WorkTypeMixed    WorkType = "mixed"
	WorkTypeUnknown  WorkType = "unknown"
)

// WorkPattern tracks execution patterns for classification
type WorkPattern struct {
	IOCount    int
	CPUCount   int
	TotalCount int
}

// WorkClassifier classifies work types based on stack traces and execution patterns
type WorkClassifier struct {
	mu                sync.RWMutex
	goroutineWork     map[int]WorkType // Track work type per goroutine
	executionPatterns map[string]*WorkPattern
	stackAnalyzer     *StackAnalyzer
}

// NewWorkClassifier creates a new work classifier
func NewWorkClassifier() *WorkClassifier {
	return &WorkClassifier{
		goroutineWork:     make(map[int]WorkType),
		executionPatterns: make(map[string]*WorkPattern),
		stackAnalyzer:     NewStackAnalyzer(),
	}
}

// Classify classifies work type based on stack trace
func (wc *WorkClassifier) Classify(goroutineID int, stackTrace []string) WorkType {
	if len(stackTrace) == 0 {
		return WorkTypeUnknown
	}

	// Use stack analyzer to determine work type
	workType := wc.stackAnalyzer.Analyze(stackTrace)

	// Update goroutine work tracking
	wc.mu.Lock()
	prevType, exists := wc.goroutineWork[goroutineID]
	
	// Detect mixed work: if previous type was different, mark as mixed
	if exists && prevType != workType && prevType != WorkTypeMixed && workType != WorkTypeUnknown {
		workType = WorkTypeMixed
	}
	
	wc.goroutineWork[goroutineID] = workType
	wc.mu.Unlock()

	return workType
}

// GetWorkType returns the current work type for a goroutine
func (wc *WorkClassifier) GetWorkType(goroutineID int) WorkType {
	wc.mu.RLock()
	defer wc.mu.RUnlock()
	
	workType, exists := wc.goroutineWork[goroutineID]
	if !exists {
		return WorkTypeUnknown
	}
	return workType
}

// GetWorkTypeStats returns statistics about work types
func (wc *WorkClassifier) GetWorkTypeStats() map[WorkType]int {
	wc.mu.RLock()
	defer wc.mu.RUnlock()

	stats := make(map[WorkType]int)
	for _, workType := range wc.goroutineWork {
		stats[workType]++
	}
	return stats
}

// Clear removes all tracked goroutines (useful for cleanup)
func (wc *WorkClassifier) Clear() {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	
	wc.goroutineWork = make(map[int]WorkType)
	wc.executionPatterns = make(map[string]*WorkPattern)
}
