package profiling

import (
	"strings"
)

// StackAnalyzer analyzes stack traces to determine work type
type StackAnalyzer struct {
	ioPatterns  []string
	cpuPatterns []string
}

// NewStackAnalyzer creates a new stack analyzer
func NewStackAnalyzer() *StackAnalyzer {
	return &StackAnalyzer{
		ioPatterns: []string{
			"net.",
			"syscall.",
			"io.",
			"os.",
			"Read",
			"Write",
			"Accept",
			"Listen",
			"Dial",
			"Connect",
			"Recv",
			"Send",
			"fasthttp.",
			"http.",
			"tls.",
			"crypto/tls.",
			"database/sql.",
			"gorm.io",
			"gorm.io",
			"redis.",
			"context.",
			"time.Sleep",
			"select",
			"chan",
			"<-",
		},
		cpuPatterns: []string{
			"crypto/",
			"encoding/",
			"compress/",
			"hash.",
			"sha256.",
			"md5.",
			"aes.",
			"rsa.",
			"json.Marshal",
			"json.Unmarshal",
			"xml.",
			"gzip.",
			"zlib.",
			"image.",
			"image/jpeg",
			"image/png",
			"math.",
			"sort.",
			"strings.",
			"bytes.",
			"strconv.",
			"regexp.",
			"reflect.",
			"runtime.",
			"sync.",
			"atomic.",
		},
	}
}

// Analyze analyzes a stack trace and returns the work type
func (sa *StackAnalyzer) Analyze(stackTrace []string) WorkType {
	if len(stackTrace) == 0 {
		return WorkTypeUnknown
	}

	hasIO := false
	hasCPU := false

	// Check each frame in the stack trace
	for _, frame := range stackTrace {
		frameLower := strings.ToLower(frame)
		
		// Check for IO patterns
		for _, pattern := range sa.ioPatterns {
			if strings.Contains(frameLower, strings.ToLower(pattern)) {
				hasIO = true
				break
			}
		}

		// Check for CPU patterns
		for _, pattern := range sa.cpuPatterns {
			if strings.Contains(frameLower, strings.ToLower(pattern)) {
				hasCPU = true
				break
			}
		}

		// Early exit if we found both
		if hasIO && hasCPU {
			return WorkTypeMixed
		}
	}

	// Determine work type
	if hasIO && hasCPU {
		return WorkTypeMixed
	}
	if hasIO {
		return WorkTypeIOBound
	}
	if hasCPU {
		return WorkTypeCPUBound
	}

	// Default to IO-bound for HTTP server operations
	// Most HTTP operations are IO-bound unless proven otherwise
	return WorkTypeIOBound
}

// IsIOFrame checks if a frame indicates IO-bound work
func (sa *StackAnalyzer) IsIOFrame(frame string) bool {
	frameLower := strings.ToLower(frame)
	for _, pattern := range sa.ioPatterns {
		if strings.Contains(frameLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// IsCPUFrame checks if a frame indicates CPU-bound work
func (sa *StackAnalyzer) IsCPUFrame(frame string) bool {
	frameLower := strings.ToLower(frame)
	for _, pattern := range sa.cpuPatterns {
		if strings.Contains(frameLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
