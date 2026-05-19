package log4go

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RollingPolicy defines when to roll over log files
type RollingPolicy interface {
	ShouldRoll(file *os.File) bool
	GetRolledFileName(basePath string) string
}

// SizeBasedRollingPolicy rolls over when file reaches a certain size
type SizeBasedRollingPolicy struct {
	MaxSize int64 // in bytes
}

// NewSizeBasedRollingPolicy creates a size-based rolling policy
func NewSizeBasedRollingPolicy(maxSizeMB int) *SizeBasedRollingPolicy {
	return &SizeBasedRollingPolicy{
		MaxSize: int64(maxSizeMB) * 1024 * 1024,
	}
}

// ShouldRoll checks if the file should be rolled
func (p *SizeBasedRollingPolicy) ShouldRoll(file *os.File) bool {
	if file == nil {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}

	return info.Size() >= p.MaxSize
}

// GetRolledFileName returns the name for the rolled file
func (p *SizeBasedRollingPolicy) GetRolledFileName(basePath string) string {
	timestamp := time.Now().Format("20060102-150405")
	ext := filepath.Ext(basePath)
	base := basePath[:len(basePath)-len(ext)]
	return fmt.Sprintf("%s.%s%s", base, timestamp, ext)
}

// TimeBasedRollingPolicy rolls over at specified time intervals
type TimeBasedRollingPolicy struct {
	Interval time.Duration
	lastRoll time.Time
}

// NewTimeBasedRollingPolicy creates a time-based rolling policy
func NewTimeBasedRollingPolicy(interval time.Duration) *TimeBasedRollingPolicy {
	return &TimeBasedRollingPolicy{
		Interval: interval,
		lastRoll: time.Now(),
	}
}

// ShouldRoll checks if enough time has passed
func (p *TimeBasedRollingPolicy) ShouldRoll(file *os.File) bool {
	return time.Since(p.lastRoll) >= p.Interval
}

// GetRolledFileName returns the name for the rolled file
func (p *TimeBasedRollingPolicy) GetRolledFileName(basePath string) string {
	p.lastRoll = time.Now()
	timestamp := p.lastRoll.Format("20060102-150405")
	ext := filepath.Ext(basePath)
	base := basePath[:len(basePath)-len(ext)]
	return fmt.Sprintf("%s.%s%s", base, timestamp, ext)
}

// RollingFileAppender writes to a file with automatic rollover
type RollingFileAppender struct {
	*baseAppender
	filePath      string
	file          *os.File
	policy        RollingPolicy
	maxBackups    int
	mu            sync.Mutex
	rolledFiles   []string
}

// NewRollingFileAppender creates a rolling file appender
func NewRollingFileAppender(name string, filePath string, policy RollingPolicy, maxBackups int) (Appender, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &RollingFileAppender{
		baseAppender: &baseAppender{
			name:      name,
			formatter: DefaultFormatter(),
		},
		filePath:    filePath,
		file:        file,
		policy:      policy,
		maxBackups:  maxBackups,
		rolledFiles: make([]string, 0),
	}, nil
}

// Append writes the log entry and checks for rollover
func (a *RollingFileAppender) Append(entry *LogEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if we should roll
	if a.policy.ShouldRoll(a.file) {
		a.rollOver()
	}

	// Write log entry
	if a.file != nil {
		formatter := a.GetFormatter()
		formatted := formatter.Format(entry)
		fmt.Fprintln(a.file, formatted)
	}
}

// rollOver performs the file rollover
func (a *RollingFileAppender) rollOver() {
	// Close current file
	if a.file != nil {
		a.file.Close()
	}

	// Rename current file
	rolledFileName := a.policy.GetRolledFileName(a.filePath)
	if err := os.Rename(a.filePath, rolledFileName); err == nil {
		a.rolledFiles = append(a.rolledFiles, rolledFileName)

		// Remove old backups if exceeding maxBackups
		if len(a.rolledFiles) > a.maxBackups {
			toRemove := a.rolledFiles[0]
			os.Remove(toRemove)
			a.rolledFiles = a.rolledFiles[1:]
		}
	}

	// Open new file
	file, err := os.OpenFile(a.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Failed to open new file, try to reopen old one
		file, _ = os.OpenFile(rolledFileName, os.O_WRONLY|os.O_APPEND, 0644)
	}
	a.file = file
}

// Close closes the rolling file appender
func (a *RollingFileAppender) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.file != nil {
		err := a.file.Close()
		a.file = nil
		return err
	}
	return nil
}

// DailyRollingFileAppender rolls over daily
type DailyRollingFileAppender struct {
	*RollingFileAppender
}

// NewDailyRollingFileAppender creates a daily rolling file appender
func NewDailyRollingFileAppender(name string, filePath string, maxBackups int) (Appender, error) {
	policy := NewTimeBasedRollingPolicy(24 * time.Hour)
	return NewRollingFileAppender(name, filePath, policy, maxBackups)
}
