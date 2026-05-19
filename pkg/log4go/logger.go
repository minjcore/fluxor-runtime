// Package log4go provides a flexible, high-performance logging framework
// inspired by Log4j, Log4net, and Logrus.
//
// Features:
//   - Multiple log levels (TRACE, DEBUG, INFO, WARN, ERROR, FATAL)
//   - Multiple appenders (Console, File, RollingFile, Async)
//   - Flexible formatters (Text, JSON, Pattern)
//   - Structured logging with fields
//   - Context integration
//   - High performance with async logging
//   - Thread-safe
//
// Example:
//
//	logger := log4go.NewLogger("myapp")
//	logger.SetLevel(log4go.INFO)
//	logger.AddAppender(log4go.NewConsoleAppender())
//	logger.Info("Application started")
//	logger.WithFields(log4go.Fields{"user": "john"}).Info("User logged in")
package log4go

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Level represents the log level
type Level int

const (
	TRACE Level = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	OFF // Special level to disable logging
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case TRACE:
		return "TRACE"
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	case OFF:
		return "OFF"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel parses a string into a Level
func ParseLevel(level string) Level {
	switch level {
	case "TRACE":
		return TRACE
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	case "OFF":
		return OFF
	default:
		return INFO // Default to INFO
	}
}

// Fields represents structured logging fields
type Fields map[string]interface{}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time
	Level     Level
	Logger    string
	Message   string
	Fields    Fields
	File      string
	Line      int
	Function  string
}

// Logger is the main logging interface
type Logger interface {
	// Level methods
	SetLevel(level Level)
	GetLevel() Level
	IsLevelEnabled(level Level) bool

	// Logging methods
	Trace(args ...interface{})
	Tracef(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})

	// Structured logging
	WithField(key string, value interface{}) Logger
	WithFields(fields Fields) Logger
	WithContext(ctx context.Context) Logger

	// Appender management
	AddAppender(appender Appender)
	RemoveAppender(name string)
	GetAppenders() []Appender

	// Name
	Name() string
}

// logger implements the Logger interface
type logger struct {
	name      string
	level     Level
	appenders []Appender
	fields    Fields
	mu        sync.RWMutex
}

// NewLogger creates a new logger with the given name
func NewLogger(name string) Logger {
	return &logger{
		name:      name,
		level:     INFO,
		appenders: make([]Appender, 0),
		fields:    make(Fields),
	}
}

// Name returns the logger name
func (l *logger) Name() string {
	return l.name
}

// SetLevel sets the minimum log level
func (l *logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level
func (l *logger) GetLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// IsLevelEnabled checks if a level is enabled
func (l *logger) IsLevelEnabled(level Level) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level && l.level != OFF
}

// AddAppender adds an appender to the logger
func (l *logger) AddAppender(appender Appender) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.appenders = append(l.appenders, appender)
}

// RemoveAppender removes an appender by name
func (l *logger) RemoveAppender(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, appender := range l.appenders {
		if appender.Name() == name {
			l.appenders = append(l.appenders[:i], l.appenders[i+1:]...)
			break
		}
	}
}

// GetAppenders returns all appenders
func (l *logger) GetAppenders() []Appender {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]Appender, len(l.appenders))
	copy(result, l.appenders)
	return result
}

// log is the internal logging method
func (l *logger) log(level Level, message string) {
	if !l.IsLevelEnabled(level) {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	function := "unknown"
	if ok {
		// Get function name
		pc, _, _, ok := runtime.Caller(2)
		if ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				function = fn.Name()
			}
		}
	} else {
		file = "unknown"
		line = 0
	}

	// Create log entry
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Logger:    l.name,
		Message:   message,
		Fields:    l.copyFields(),
		File:      file,
		Line:      line,
		Function:  function,
	}

	// Send to all appenders
	l.mu.RLock()
	appenders := make([]Appender, len(l.appenders))
	copy(appenders, l.appenders)
	l.mu.RUnlock()

	for _, appender := range appenders {
		appender.Append(entry)
	}
}

// copyFields creates a copy of the fields map
func (l *logger) copyFields() Fields {
	l.mu.RLock()
	defer l.mu.RUnlock()
	fields := make(Fields, len(l.fields))
	for k, v := range l.fields {
		fields[k] = v
	}
	return fields
}

// Trace logs a trace message
func (l *logger) Trace(args ...interface{}) {
	l.log(TRACE, fmt.Sprint(args...))
}

// Tracef logs a formatted trace message
func (l *logger) Tracef(format string, args ...interface{}) {
	l.log(TRACE, fmt.Sprintf(format, args...))
}

// Debug logs a debug message
func (l *logger) Debug(args ...interface{}) {
	l.log(DEBUG, fmt.Sprint(args...))
}

// Debugf logs a formatted debug message
func (l *logger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, fmt.Sprintf(format, args...))
}

// Info logs an info message
func (l *logger) Info(args ...interface{}) {
	l.log(INFO, fmt.Sprint(args...))
}

// Infof logs a formatted info message
func (l *logger) Infof(format string, args ...interface{}) {
	l.log(INFO, fmt.Sprintf(format, args...))
}

// Warn logs a warning message
func (l *logger) Warn(args ...interface{}) {
	l.log(WARN, fmt.Sprint(args...))
}

// Warnf logs a formatted warning message
func (l *logger) Warnf(format string, args ...interface{}) {
	l.log(WARN, fmt.Sprintf(format, args...))
}

// Error logs an error message
func (l *logger) Error(args ...interface{}) {
	l.log(ERROR, fmt.Sprint(args...))
}

// Errorf logs a formatted error message
func (l *logger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, fmt.Sprintf(format, args...))
}

// Fatal logs a fatal message and panics
func (l *logger) Fatal(args ...interface{}) {
	message := fmt.Sprint(args...)
	l.log(FATAL, message)
	panic(message)
}

// Fatalf logs a formatted fatal message and panics
func (l *logger) Fatalf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.log(FATAL, message)
	panic(message)
}

// WithField returns a new logger with an additional field
func (l *logger) WithField(key string, value interface{}) Logger {
	newFields := l.copyFields()
	newFields[key] = value
	return &logger{
		name:      l.name,
		level:     l.level,
		appenders: l.GetAppenders(),
		fields:    newFields,
	}
}

// WithFields returns a new logger with additional fields
func (l *logger) WithFields(fields Fields) Logger {
	newFields := l.copyFields()
	for k, v := range fields {
		newFields[k] = v
	}
	return &logger{
		name:      l.name,
		level:     l.level,
		appenders: l.GetAppenders(),
		fields:    newFields,
	}
}

// WithContext returns a new logger with context values
func (l *logger) WithContext(ctx context.Context) Logger {
	newFields := l.copyFields()

	// Extract common context values
	if requestID := ctx.Value("request_id"); requestID != nil {
		newFields["request_id"] = requestID
	}
	if traceID := ctx.Value("trace_id"); traceID != nil {
		newFields["trace_id"] = traceID
	}
	if userID := ctx.Value("user_id"); userID != nil {
		newFields["user_id"] = userID
	}

	return &logger{
		name:      l.name,
		level:     l.level,
		appenders: l.GetAppenders(),
		fields:    newFields,
	}
}

// Global logger instance
var (
	globalLogger     Logger
	globalLoggerOnce sync.Once
)

// GetLogger returns a logger with the given name
// If name is empty, returns the global logger
func GetLogger(name string) Logger {
	if name == "" {
		globalLoggerOnce.Do(func() {
			globalLogger = NewLogger("root")
			globalLogger.SetLevel(INFO)
			globalLogger.AddAppender(NewConsoleAppender())
		})
		return globalLogger
	}
	return NewLogger(name)
}

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger Logger) {
	globalLogger = logger
}
