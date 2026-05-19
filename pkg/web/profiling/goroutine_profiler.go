package profiling

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GoroutineState represents the state of a goroutine
type GoroutineState string

const (
	GoroutineStateRunning GoroutineState = "running"
	GoroutineStateWaiting GoroutineState = "waiting"
	GoroutineStateBlocked GoroutineState = "blocked"
	GoroutineStateUnknown GoroutineState = "unknown"
)

// GoroutineProfile represents a profile of a single goroutine
type GoroutineProfile struct {
	ID         int
	State      GoroutineState
	StackTrace []string
	WorkType   WorkType
	Duration   time.Duration
	LastSeen   time.Time
}

// GoroutineProfiler tracks goroutine profiles
type GoroutineProfiler struct {
	mu       sync.RWMutex
	profiles map[int]*GoroutineProfile
	classifier *WorkClassifier
}

// NewGoroutineProfiler creates a new goroutine profiler
func NewGoroutineProfiler(classifier *WorkClassifier) *GoroutineProfiler {
	return &GoroutineProfiler{
		profiles:   make(map[int]*GoroutineProfile),
		classifier: classifier,
	}
}

// Update updates the profile for a goroutine
func (gp *GoroutineProfiler) Update(goroutineID int, stackTrace []string, workType WorkType) {
	gp.mu.Lock()
	defer gp.mu.Unlock()

	profile, exists := gp.profiles[goroutineID]
	if !exists {
		profile = &GoroutineProfile{
			ID:       goroutineID,
			State:    GoroutineStateUnknown,
			WorkType: WorkTypeUnknown,
			LastSeen: time.Now(),
		}
		gp.profiles[goroutineID] = profile
	}

	// Update profile
	profile.StackTrace = stackTrace
	profile.WorkType = workType
	profile.State = gp.determineState(stackTrace)
	profile.LastSeen = time.Now()
}

// determineState determines the goroutine state from stack trace
func (gp *GoroutineProfiler) determineState(stackTrace []string) GoroutineState {
	if len(stackTrace) == 0 {
		return GoroutineStateUnknown
	}

	// Check for blocking operations
	for _, frame := range stackTrace {
		frameLower := strings.ToLower(frame)
		
		// Waiting on channel
		if strings.Contains(frameLower, "chan") || strings.Contains(frameLower, "<-") {
			return GoroutineStateWaiting
		}
		
		// Blocked on syscall
		if strings.Contains(frameLower, "syscall") {
			return GoroutineStateBlocked
		}
		
		// Blocked on network I/O
		if strings.Contains(frameLower, "net.") && (strings.Contains(frameLower, "read") || strings.Contains(frameLower, "write")) {
			return GoroutineStateBlocked
		}
		
		// Blocked on sleep
		if strings.Contains(frameLower, "sleep") {
			return GoroutineStateWaiting
		}
	}

	// Default to running if no blocking operations detected
	return GoroutineStateRunning
}

// GetProfile returns the profile for a goroutine
func (gp *GoroutineProfiler) GetProfile(goroutineID int) *GoroutineProfile {
	gp.mu.RLock()
	defer gp.mu.RUnlock()
	
	return gp.profiles[goroutineID]
}

// GetAllProfiles returns all goroutine profiles
func (gp *GoroutineProfiler) GetAllProfiles() map[int]*GoroutineProfile {
	gp.mu.RLock()
	defer gp.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	result := make(map[int]*GoroutineProfile)
	for id, profile := range gp.profiles {
		result[id] = &GoroutineProfile{
			ID:         profile.ID,
			State:       profile.State,
			StackTrace:   append([]string{}, profile.StackTrace...),
			WorkType:    profile.WorkType,
			Duration:    profile.Duration,
			LastSeen:    profile.LastSeen,
		}
	}
	return result
}

// GetStats returns statistics about goroutines
func (gp *GoroutineProfiler) GetStats() GoroutineStats {
	gp.mu.RLock()
	defer gp.mu.RUnlock()

	stats := GoroutineStats{
		Total:      len(gp.profiles),
		ByState:    make(map[GoroutineState]int),
		ByWorkType: make(map[WorkType]int),
		MixedWork:  0,
	}

	for _, profile := range gp.profiles {
		stats.ByState[profile.State]++
		stats.ByWorkType[profile.WorkType]++
		if profile.WorkType == WorkTypeMixed {
			stats.MixedWork++
		}
	}

	return stats
}

// GoroutineStats provides statistics about goroutines
type GoroutineStats struct {
	Total      int
	ByState    map[GoroutineState]int
	ByWorkType map[WorkType]int
	MixedWork  int
}

// ParseStackTraces parses stack traces from runtime.Stack output
func ParseStackTraces(stackData []byte) map[int][]string {
	result := make(map[int][]string)
	
	lines := strings.Split(string(stackData), "\n")
	var currentID int
	var currentStack []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check if this is a goroutine header (e.g., "goroutine 1 [running]:")
		if strings.HasPrefix(line, "goroutine ") {
			// Save previous goroutine if exists
			if currentID > 0 && len(currentStack) > 0 {
				result[currentID] = currentStack
			}
			
			// Parse goroutine ID
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				idStr := parts[1]
				if id, err := strconv.Atoi(idStr); err == nil {
					currentID = id
					currentStack = []string{}
				}
			}
			continue
		}
		
		// Add stack frame
		if currentID > 0 && strings.HasPrefix(line, "	") {
			currentStack = append(currentStack, line)
		}
	}
	
	// Save last goroutine
	if currentID > 0 && len(currentStack) > 0 {
		result[currentID] = currentStack
	}
	
	return result
}

// CaptureAllGoroutines captures all goroutine stack traces
func CaptureAllGoroutines() map[int][]string {
	buf := make([]byte, 64*1024)
	n := runtime.Stack(buf, true)
	return ParseStackTraces(buf[:n])
}
