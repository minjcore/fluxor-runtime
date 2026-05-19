//go:build !windows
// +build !windows

package dashboard

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// getOSThreadCount returns the actual number of OS threads for the current process
// This is more accurate than GOMAXPROCS which only shows the max threads for Go code
func getOSThreadCount() int {
	pid := os.Getpid()
	
	// Method 1: Try ps -p <pid> -o thcount= (works on macOS and Linux)
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "thcount=")
	output, err := cmd.Output()
	if err == nil {
		line := strings.TrimSpace(string(output))
		if count, err := strconv.Atoi(line); err == nil && count > 0 {
			return count
		}
	}
	
	// Method 2: Try ps -M <pid> and count lines (macOS specific)
	// ps -M lists all threads, so we count the lines
	cmd = exec.Command("ps", "-M", strconv.Itoa(pid))
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		// First line is header, rest are threads
		// Subtract 1 for header, but also check if we have valid output
		if len(lines) > 1 {
			// Count non-empty lines (excluding header)
			count := 0
			for i := 1; i < len(lines); i++ {
				line := strings.TrimSpace(lines[i])
				if line != "" && !strings.HasPrefix(line, "USER") {
					count++
				}
			}
			if count > 0 {
				return count
			}
		}
	}
	
	// Method 3: Try Linux /proc/self/status (Linux specific)
	if runtime.GOOS == "linux" {
		cmd = exec.Command("sh", "-c", "grep Threads /proc/self/status | awk '{print $2}'")
		output, err = cmd.Output()
		if err == nil {
			line := strings.TrimSpace(string(output))
			if count, err := strconv.Atoi(line); err == nil && count > 0 {
				return count
			}
		}
	}
	
	// Final fallback: use GOMAXPROCS as approximation
	// This is less accurate but better than nothing
	return runtime.GOMAXPROCS(0)
}
