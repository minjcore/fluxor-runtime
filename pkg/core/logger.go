package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/appendlog"
)

// Logger provides structured logging capabilities
// This abstraction allows swapping logging implementations
type Logger interface {
	// Error logs an error message
	Error(args ...interface{})

	// Info logs an informational message
	Info(args ...interface{})

	// Debug logs a debug message
	Debug(args ...interface{})

	// WithFields returns a new logger with structured fields
	// This enables structured logging with key-value pairs
	WithFields(fields map[string]interface{}) Logger

	// WithContext returns a new logger with context values
	// Extracts request ID and other context values automatically
	WithContext(ctx context.Context) Logger
}

// LoggerConfig configures logger behavior
type LoggerConfig struct {
	// JSONOutput enables JSON structured output
	JSONOutput bool
	// Level sets the minimum log level (DEBUG, INFO, ERROR)
	Level string
	// AppendLogStore enables persistent logging to append-only log store
	// If nil, logs are only written to console
	AppendLogStore appendlog.Store
	// AppendLogEnabled enables/disables append log writing
	AppendLogEnabled bool
	// Timezone sets the timezone for log timestamps (e.g., "UTC+7", "UTC-5", "UTC+0")
	// Defaults to UTC if empty or invalid
	Timezone string
}

// defaultLogger implements Logger using Go's standard log package
// Can be swapped with other logging implementations (e.g., structured loggers)
// Now supports optional append-only log persistence
type defaultLogger struct {
	errorLogger *log.Logger
	infoLogger  *log.Logger
	debugLogger *log.Logger
	config      LoggerConfig
	fields      map[string]interface{} // Structured fields
	mu          sync.RWMutex           // Protects appendLogStore access
	location    *time.Location         // Timezone location for timestamps
}

// fileLogger wraps a Logger to include file name in log messages
type fileLogger struct {
	Logger
	fileName string
}

// ParseTimezone parses a timezone string supporting both IANA timezone names (e.g., "Asia/Ho_Chi_Minh")
// and UTC offset format (e.g., "UTC+7", "UTC-5", "UTC+0")
// Returns a *time.Location or UTC if parsing fails
// This function is exported so it can be used to set timezone in global context
func ParseTimezone(tz string) *time.Location {
	return parseTimezone(tz)
}

// parseTimezone parses a timezone string supporting both IANA timezone names (e.g., "Asia/Ho_Chi_Minh")
// and UTC offset format (e.g., "UTC+7", "UTC-5", "UTC+0")
// Returns a *time.Location or UTC if parsing fails
func parseTimezone(tz string) *time.Location {
	if tz == "" {
		return time.UTC
	}

	// Handle "UTC" or "UTC+0" as UTC
	if tz == "UTC" || tz == "UTC+0" || tz == "UTC-0" {
		return time.UTC
	}

	// First, try to load as IANA timezone name (e.g., "Asia/Ho_Chi_Minh", "America/New_York")
	location, err := time.LoadLocation(tz)
	if err == nil {
		return location
	}

	// If IANA timezone loading fails, try UTC offset format: UTC+7, UTC-5, etc.
	// Regex to match UTC followed by optional +/- and number
	re := regexp.MustCompile(`^UTC([+-])(\d+)$`)
	matches := re.FindStringSubmatch(strings.ToUpper(tz))
	if len(matches) == 3 {
		sign := matches[1]
		offsetStr := matches[2]
		offset, err := strconv.Atoi(offsetStr)
		if err == nil {
			// Convert to seconds
			seconds := offset * 3600
			if sign == "-" {
				seconds = -seconds
			}
			return time.FixedZone(fmt.Sprintf("UTC%s%d", sign, offset), seconds)
		}
	}

	// If all parsing fails, default to UTC
	return time.UTC
}

// NewDefaultLogger creates a new default logger implementation
func NewDefaultLogger() Logger {
	return NewLogger(LoggerConfig{
		JSONOutput:       false,
		Level:            "DEBUG",
		AppendLogEnabled: false,
		Timezone:         "UTC",
	})
}

// NewLogger creates a new logger with configuration
func NewLogger(config LoggerConfig) Logger {
	location := parseTimezone(config.Timezone)
	return &defaultLogger{
		errorLogger: log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
		infoLogger:  log.New(os.Stdout, "[INFO] ", log.LstdFlags|log.Lshortfile),
		debugLogger: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
		config:      config,
		fields:      make(map[string]interface{}),
		location:    location,
	}
}

// NewLoggerWithAppendLog creates a logger with append-only log persistence
func NewLoggerWithAppendLog(config LoggerConfig, appendLogStore appendlog.Store) Logger {
	config.AppendLogStore = appendLogStore
	config.AppendLogEnabled = true
	return NewLogger(config)
}

// NewJSONLogger creates a logger with JSON output enabled
func NewJSONLogger() Logger {
	return NewLogger(LoggerConfig{
		JSONOutput:       true,
		Level:            "DEBUG",
		AppendLogEnabled: false,
		Timezone:         "UTC",
	})
}

// NewLoggerWithFileName creates a logger that includes the file name in log messages
// The file name is automatically detected from the caller's location using runtime.Caller
func NewLoggerWithFileName() Logger {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
	}
	fileName := filepath.Base(file)
	return &fileLogger{
		Logger:   NewDefaultLogger(),
		fileName: fileName,
	}
}

// Error logs an error message with file name prefix
func (fl *fileLogger) Error(args ...interface{}) {
	fl.Logger.Error(fmt.Sprintf("[%s] %s", fl.fileName, fmt.Sprint(args...)))
}

// Info logs an informational message with file name prefix
func (fl *fileLogger) Info(args ...interface{}) {
	fl.Logger.Info(fmt.Sprintf("[%s] %s", fl.fileName, fmt.Sprint(args...)))
}

// Debug logs a debug message with file name prefix
func (fl *fileLogger) Debug(args ...interface{}) {
	fl.Logger.Debug(fmt.Sprintf("[%s] %s", fl.fileName, fmt.Sprint(args...)))
}

// WithFields returns a new logger with structured fields and preserves file name
func (fl *fileLogger) WithFields(fields map[string]interface{}) Logger {
	return &fileLogger{
		Logger:   fl.Logger.WithFields(fields),
		fileName: fl.fileName,
	}
}

// WithContext returns a new logger with context values and preserves file name
func (fl *fileLogger) WithContext(ctx context.Context) Logger {
	return &fileLogger{
		Logger:   fl.Logger.WithContext(ctx),
		fileName: fl.fileName,
	}
}

// logEntry represents a structured log entry (Cloud Logging compatible: severity, msg, optional trace)
type logEntry struct {
	Timestamp string                 `json:"timestamp,omitempty"`
	Level     string                 `json:"level"`
	Severity  string                 `json:"severity,omitempty"` // Cloud Logging: INFO, ERROR, DEBUG
	Message   string                 `json:"message"`
	Msg       string                 `json:"msg,omitempty"` // Cloud Logging alias for message
	Trace     string                 `json:"trace,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// log writes a log entry with structured fields
// Also writes to append-only log store if enabled
func (l *defaultLogger) log(level string, logger *log.Logger, message string) {
	// Check log level
	if !l.shouldLog(level) {
		return
	}

	// Create log entry with configured timezone
	now := time.Now().In(l.location)
	entry := logEntry{
		Timestamp: now.Format(time.RFC3339),
		Level:     level,
		Severity:  level, // Cloud Logging expects "severity"
		Message:   message,
		Msg:       message,
	}
	if len(l.fields) > 0 {
		entry.Fields = make(map[string]interface{})
		for k, v := range l.fields {
			entry.Fields[k] = v
			if k == "trace" {
				if s, ok := v.(string); ok && s != "" {
					entry.Trace = s
				}
			}
		}
	}

	// Write to console (stdout/stderr)
	// Use depth 2 to skip log() and the level-specific method (Error/Info/etc)
	if l.config.JSONOutput {
		jsonData, err := json.Marshal(entry)
		if err == nil {
			logger.Output(2, string(jsonData))
		} else {
			// Fallback to plain text if JSON marshal fails
			logger.Output(2, fmt.Sprintf("[%s] %s %v", level, message, l.fields))
		}
	} else {
		// Plain text output with fields appended
		if len(l.fields) > 0 {
			logger.Output(2, fmt.Sprintf("%s %v", message, l.fields))
		} else {
			logger.Output(2, message)
		}
	}

	// Write to append-only log store if enabled
	if l.config.AppendLogEnabled && l.config.AppendLogStore != nil {
		l.writeToAppendLog(entry)
	}
}

// writeToAppendLog writes log entry to append-only log store
// This is best-effort and non-blocking
func (l *defaultLogger) writeToAppendLog(entry logEntry) {
	l.mu.RLock()
	store := l.config.AppendLogStore
	l.mu.RUnlock()

	if store == nil {
		return
	}

	// Marshal entry to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		// Best-effort: if marshaling fails, try plain text
		jsonData = []byte(fmt.Sprintf("[%s] %s", entry.Level, entry.Message))
	}

	// Append to log store (non-blocking, fail-fast on backpressure)
	// Use goroutine to avoid blocking the main logging path
	go func() {
		_, err := store.Append(jsonData)
		if err != nil {
			// Best-effort: silently ignore append errors to avoid log loops
			// Could optionally log to stderr, but that might cause recursion
		}
	}()
}

// shouldLog checks if the log level should be logged based on config
func (l *defaultLogger) shouldLog(level string) bool {
	levels := map[string]int{
		"DEBUG": 0,
		"INFO":  1,
		"ERROR": 2,
	}

	configLevel, ok := levels[l.config.Level]
	if !ok {
		configLevel = 0 // Default to DEBUG if invalid
	}

	logLevel, ok := levels[level]
	if !ok {
		return true // Unknown levels are always logged
	}

	return logLevel >= configLevel
}

// Error logs an error message
func (l *defaultLogger) Error(args ...interface{}) {
	l.log("ERROR", l.errorLogger, fmt.Sprint(args...))
}

// Info logs an informational message
func (l *defaultLogger) Info(args ...interface{}) {
	l.log("INFO", l.infoLogger, fmt.Sprint(args...))
}

// Debug logs a debug message
func (l *defaultLogger) Debug(args ...interface{}) {
	l.log("DEBUG", l.debugLogger, fmt.Sprint(args...))
}

// WithFields returns a new logger with structured fields
// Fields are included in all subsequent log entries
func (l *defaultLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	// Copy existing fields
	for k, v := range l.fields {
		newFields[k] = v
	}
	// Merge new fields (new fields override existing ones)
	for k, v := range fields {
		newFields[k] = v
	}
	return &defaultLogger{
		errorLogger: l.errorLogger,
		infoLogger:  l.infoLogger,
		debugLogger: l.debugLogger,
		config:      l.config,
		fields:      newFields,
		location:    l.location,
	}
}

// WithContext returns a new logger with context values
// Automatically extracts request ID and other context values
func (l *defaultLogger) WithContext(ctx context.Context) Logger {
	fields := make(map[string]interface{})

	// Extract request ID from context
	if requestID := GetRequestID(ctx); requestID != "" {
		fields["request_id"] = requestID
	}
	// Extract trace ID from context (e.g. OpenTelemetry, GCP trace for Cloud Logging)
	if traceID := GetTrace(ctx); traceID != "" {
		fields["trace"] = traceID
	}

	// Copy existing fields
	for k, v := range l.fields {
		fields[k] = v
	}

	return &defaultLogger{
		errorLogger: l.errorLogger,
		infoLogger:  l.infoLogger,
		debugLogger: l.debugLogger,
		config:      l.config,
		fields:      fields,
		location:    l.location,
	}
}

// SetAppendLogStore updates the append log store for an existing logger
// This allows enabling/disabling append log after logger creation
func (l *defaultLogger) SetAppendLogStore(store appendlog.Store) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.AppendLogStore = store
	if store != nil {
		l.config.AppendLogEnabled = true
	} else {
		l.config.AppendLogEnabled = false
	}
}

// Package-level logger instance for convenience functions
var (
	defaultLoggerInstance Logger
	defaultLoggerOnce     sync.Once
)

// initDefaultLogger initializes the default logger instance
func initDefaultLogger() {
	defaultLoggerInstance = NewDefaultLogger()
}

// hasFormatSpecifiers checks if string contains format specifiers like %s, %d, %v, etc.
func hasFormatSpecifiers(s string) bool {
	// Simple check: look for % followed by a letter or digit
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '%' {
			next := s[i+1]
			// Check for format specifiers: %s, %d, %v, %f, %t, %x, %X, %o, %b, %c, %q, %p, %e, %E, %g, %G, %U
			if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || (next >= '0' && next <= '9') || next == '.' || next == '+' || next == '-' || next == '#' {
				return true
			}
		}
	}
	return false
}

// Error logs an error message with format support
// Supports both: core.Error("message") and core.Error("format %s", arg)
func Error(args ...interface{}) {
	defaultLoggerOnce.Do(initDefaultLogger)
	if len(args) == 0 {
		return
	}

	// Smart detection: if first arg is string with format specifiers and has more args, use Sprintf
	if len(args) > 1 {
		if format, ok := args[0].(string); ok && hasFormatSpecifiers(format) {
			defaultLoggerInstance.Error(fmt.Sprintf(format, args[1:]...))
			return
		}
	}

	// Otherwise, use Sprint (works for plain messages and non-format cases)
	defaultLoggerInstance.Error(fmt.Sprint(args...))
}

// Info logs an informational message with format support
// Supports both: core.Info("message") and core.Info("format %s", arg)
func Info(args ...interface{}) {
	defaultLoggerOnce.Do(initDefaultLogger)
	if len(args) == 0 {
		return
	}

	// Smart detection: if first arg is string with format specifiers and has more args, use Sprintf
	if len(args) > 1 {
		if format, ok := args[0].(string); ok && hasFormatSpecifiers(format) {
			defaultLoggerInstance.Info(fmt.Sprintf(format, args[1:]...))
			return
		}
	}

	// Otherwise, use Sprint (works for plain messages and non-format cases)
	defaultLoggerInstance.Info(fmt.Sprint(args...))
}

// Debug logs a debug message with format support
// Supports both: core.Debug("message") and core.Debug("format %s", arg)
func Debug(args ...interface{}) {
	defaultLoggerOnce.Do(initDefaultLogger)
	if len(args) == 0 {
		return
	}

	// Smart detection: if first arg is string with format specifiers and has more args, use Sprintf
	if len(args) > 1 {
		if format, ok := args[0].(string); ok && hasFormatSpecifiers(format) {
			defaultLoggerInstance.Debug(fmt.Sprintf(format, args[1:]...))
			return
		}
	}

	// Otherwise, use Sprint (works for plain messages and non-format cases)
	defaultLoggerInstance.Debug(fmt.Sprint(args...))
}
