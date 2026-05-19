package core

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// LogLevel represents logging severity
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// ============================================================================
// HybridLogger - Text log với optional JSON append
// ============================================================================

// HybridLogger logs text format, optionally appends JSON
type HybridLogger struct {
	service    string
	jsonAppend bool // append JSON sau text log
	mu         sync.Mutex
	output     *os.File
	location   *time.Location // Timezone location for timestamps
}

// NewHybridLogger creates a hybrid logger
// jsonAppend=true: "2024-01-05 10:00:00 [INFO] message {json}"
// jsonAppend=false: "2024-01-05 10:00:00 [INFO] message"
// timezone is optional, defaults to UTC if empty
func NewHybridLogger(service string, jsonAppend bool, timezone ...string) *HybridLogger {
	tz := "UTC"
	if len(timezone) > 0 && timezone[0] != "" {
		tz = timezone[0]
	}
	// ParseTimezone is exported from logger.go
	location := ParseTimezone(tz)
	return &HybridLogger{
		service:    service,
		jsonAppend: jsonAppend,
		output:     os.Stdout,
		location:   location,
	}
}

// log writes hybrid log entry
func (l *HybridLogger) log(level LogLevel, msg string, fields map[string]interface{}) {
	now := time.Now().In(l.location)
	timestamp := now.Format("2006-01-02 15:04:05")

	// Build text part
	text := fmt.Sprintf("%s [%s] %s", timestamp, level, msg)

	// Append JSON if enabled
	if l.jsonAppend {
		entry := LogEntry{
			Timestamp: now.Format(time.RFC3339Nano),
			Level:     level,
			Service:   l.service,
			Message:   msg,
			Fields:    fields,
		}
		if data, err := json.Marshal(entry); err == nil {
			text = fmt.Sprintf("%s %s", text, string(data))
		}
	}

	l.mu.Lock()
	fmt.Fprintln(l.output, text)
	l.mu.Unlock()
}

// Info logs info level message
func (l *HybridLogger) Info(msg string, fields ...map[string]interface{}) {
	l.log(LevelInfo, msg, mergeFields(fields))
}

// Error logs error level message
func (l *HybridLogger) Error(msg string, fields ...map[string]interface{}) {
	l.log(LevelError, msg, mergeFields(fields))
}

// Debug logs debug level message
func (l *HybridLogger) Debug(msg string, fields ...map[string]interface{}) {
	l.log(LevelDebug, msg, mergeFields(fields))
}

// Warn logs warning level message
func (l *HybridLogger) Warn(msg string, fields ...map[string]interface{}) {
	l.log(LevelWarn, msg, mergeFields(fields))
}

// ============================================================================
// StructuredLogger - Pure JSON logging
// ============================================================================

// StructuredLogger provides JSON structured logging
type StructuredLogger struct {
	service  string
	mu       sync.Mutex
	output   *os.File
	location *time.Location // Timezone location for timestamps
}

// NewStructuredLogger creates a new structured logger
// timezone is optional, defaults to UTC if empty
func NewStructuredLogger(service string, timezone ...string) *StructuredLogger {
	tz := "UTC"
	if len(timezone) > 0 && timezone[0] != "" {
		tz = timezone[0]
	}
	location := ParseTimezone(tz)
	return &StructuredLogger{
		service:  service,
		output:   os.Stdout,
		location: location,
	}
}

// log writes a structured log entry
func (l *StructuredLogger) log(level LogLevel, msg string, fields map[string]interface{}) {
	now := time.Now().In(l.location)
	entry := LogEntry{
		Timestamp: now.Format(time.RFC3339Nano),
		Level:     level,
		Service:   l.service,
		Message:   msg,
		Fields:    fields,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "log marshal error: %v\n", err)
		return
	}

	l.mu.Lock()
	l.output.Write(data)
	l.output.Write([]byte("\n"))
	l.mu.Unlock()
}

// Debug logs debug level message
func (l *StructuredLogger) Debug(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(LevelDebug, msg, f)
}

// Info logs info level message
func (l *StructuredLogger) Info(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(LevelInfo, msg, f)
}

// Warn logs warning level message
func (l *StructuredLogger) Warn(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(LevelWarn, msg, f)
}

// Error logs error level message
func (l *StructuredLogger) Error(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(LevelError, msg, f)
}

// WithFields returns logger with preset fields
func (l *StructuredLogger) WithFields(fields map[string]interface{}) *LogContext {
	return &LogContext{
		logger: l,
		fields: fields,
	}
}

// LogContext holds context for structured logging
type LogContext struct {
	logger *StructuredLogger
	fields map[string]interface{}
}

// Info logs with context fields
func (c *LogContext) Info(msg string, fields ...map[string]interface{}) {
	merged := make(map[string]interface{})
	for k, v := range c.fields {
		merged[k] = v
	}
	for k, v := range mergeFields(fields) {
		merged[k] = v
	}
	c.logger.log(LevelInfo, msg, merged)
}

// Error logs error with context fields
func (c *LogContext) Error(msg string, fields ...map[string]interface{}) {
	merged := make(map[string]interface{})
	for k, v := range c.fields {
		merged[k] = v
	}
	for k, v := range mergeFields(fields) {
		merged[k] = v
	}
	c.logger.log(LevelError, msg, merged)
}

// Helper functions

func mergeFields(fields []map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}
	result := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			result[k] = v
		}
	}
	return result
}

// F is a shorthand for creating field maps
func F(keyvals ...interface{}) map[string]interface{} {
	fields := make(map[string]interface{})
	for i := 0; i < len(keyvals)-1; i += 2 {
		if key, ok := keyvals[i].(string); ok {
			fields[key] = keyvals[i+1]
		}
	}
	return fields
}

// ============================================================================
// FileAppendLogger - Append logs to file
// ============================================================================

// FileAppendLogger appends logs to a file with optional JSON format
type FileAppendLogger struct {
	service    string
	filePath   string
	jsonFormat bool
	file       *os.File
	mu         sync.Mutex
}

// FileLoggerConfig configuration for FileAppendLogger
type FileLoggerConfig struct {
	Service    string // Service name for log entries
	FilePath   string // Path to log file (e.g., "app.log")
	JSONFormat bool   // true = JSON lines, false = text format
}

// NewFileAppendLogger creates a new file append logger
// Example:
//
//	logger, err := NewFileAppendLogger(FileLoggerConfig{
//	    Service:    "my-service",
//	    FilePath:   "app.log",
//	    JSONFormat: true,
//	})
func NewFileAppendLogger(cfg FileLoggerConfig) (*FileAppendLogger, error) {
	if cfg.FilePath == "" {
		cfg.FilePath = "app.log"
	}
	if cfg.Service == "" {
		cfg.Service = "default"
	}

	// Open file in append mode, create if not exists
	// Use restrictive permissions (0600) to prevent unauthorized access to log files
	f, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", cfg.FilePath, err)
	}

	return &FileAppendLogger{
		service:    cfg.Service,
		filePath:   cfg.FilePath,
		jsonFormat: cfg.JSONFormat,
		file:       f,
	}, nil
}

// log writes a log entry to file
func (l *FileAppendLogger) log(level LogLevel, msg string, fields map[string]interface{}) {
	now := time.Now()
	var line string

	if l.jsonFormat {
		entry := LogEntry{
			Timestamp: now.UTC().Format(time.RFC3339Nano),
			Level:     level,
			Service:   l.service,
			Message:   msg,
			Fields:    fields,
		}
		if data, err := json.Marshal(entry); err == nil {
			line = string(data)
		} else {
			line = fmt.Sprintf("%s [%s] %s (marshal error: %v)",
				now.Format("2006-01-02 15:04:05"), level, msg, err)
		}
	} else {
		// Text format: 2024-01-05 10:00:00 [INFO] service: message key=value
		line = fmt.Sprintf("%s [%s] %s: %s",
			now.Format("2006-01-02 15:04:05"), level, l.service, msg)
		if len(fields) > 0 {
			for k, v := range fields {
				line += fmt.Sprintf(" %s=%v", k, v)
			}
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.WriteString(line + "\n")
	}
}

// Debug logs debug level message
func (l *FileAppendLogger) Debug(msg string, fields ...map[string]interface{}) {
	l.log(LevelDebug, msg, mergeFields(fields))
}

// Info logs info level message
func (l *FileAppendLogger) Info(msg string, fields ...map[string]interface{}) {
	l.log(LevelInfo, msg, mergeFields(fields))
}

// Warn logs warning level message
func (l *FileAppendLogger) Warn(msg string, fields ...map[string]interface{}) {
	l.log(LevelWarn, msg, mergeFields(fields))
}

// Error logs error level message
func (l *FileAppendLogger) Error(msg string, fields ...map[string]interface{}) {
	l.log(LevelError, msg, mergeFields(fields))
}

// Sync flushes buffered data to disk
func (l *FileAppendLogger) Sync() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Sync()
	}
	return nil
}

// Close closes the log file
func (l *FileAppendLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

// FilePath returns the path to the log file
func (l *FileAppendLogger) FilePath() string {
	return l.filePath
}

// ============================================================================
// MultiLogger - Write to multiple destinations (console + file)
// ============================================================================

// FieldLogger interface for loggers with field support
type FieldLogger interface {
	Debug(msg string, fields ...map[string]interface{})
	Info(msg string, fields ...map[string]interface{})
	Warn(msg string, fields ...map[string]interface{})
	Error(msg string, fields ...map[string]interface{})
}

// MultiLogger writes to multiple loggers simultaneously
type MultiLogger struct {
	loggers []FieldLogger
}

// NewMultiLogger creates a logger that writes to multiple destinations
// Example:
//
//	consoleLog := core.NewHybridLogger("svc", false)
//	fileLog, _ := core.NewFileAppendLogger(cfg)
//	multi := core.NewMultiLogger(consoleLog, fileLog)
func NewMultiLogger(loggers ...FieldLogger) *MultiLogger {
	return &MultiLogger{loggers: loggers}
}

// Debug logs to all destinations
func (m *MultiLogger) Debug(msg string, fields ...map[string]interface{}) {
	for _, l := range m.loggers {
		l.Debug(msg, fields...)
	}
}

// Info logs to all destinations
func (m *MultiLogger) Info(msg string, fields ...map[string]interface{}) {
	for _, l := range m.loggers {
		l.Info(msg, fields...)
	}
}

// Warn logs to all destinations
func (m *MultiLogger) Warn(msg string, fields ...map[string]interface{}) {
	for _, l := range m.loggers {
		l.Warn(msg, fields...)
	}
}

// Error logs to all destinations
func (m *MultiLogger) Error(msg string, fields ...map[string]interface{}) {
	for _, l := range m.loggers {
		l.Error(msg, fields...)
	}
}
