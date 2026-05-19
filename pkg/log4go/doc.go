// Package log4go provides a flexible, high-performance logging framework
// inspired by Log4j, Log4net, and Logrus.
//
// # Features
//
//   - Multiple log levels: TRACE, DEBUG, INFO, WARN, ERROR, FATAL
//   - Multiple appenders: Console, File, RollingFile, DailyRollingFile, Async, Multi
//   - Flexible formatters: Text (with colors), JSON, Pattern
//   - Structured logging with fields
//   - Context integration for request tracing
//   - Configuration via YAML/JSON
//   - Thread-safe operations
//   - High performance with async logging
//   - Log rotation with size and time-based policies
//
// # Quick Start
//
// Basic usage:
//
//	package main
//
//	import "github.com/fluxorio/fluxor/pkg/log4go"
//
//	func main() {
//	    // Create logger
//	    logger := log4go.GetLogger("myapp")
//
//	    // Log messages
//	    logger.Info("Application started")
//	    logger.Debug("Debug information")
//	    logger.Error("An error occurred")
//
//	    // Structured logging
//	    logger.WithField("user", "john").Info("User logged in")
//	    logger.WithFields(log4go.Fields{
//	        "user": "john",
//	        "ip": "192.168.1.1",
//	    }).Info("Login attempt")
//	}
//
// # Configuration
//
// Configure via code:
//
//	logger := log4go.NewLogger("myapp")
//	logger.SetLevel(log4go.DEBUG)
//
//	// Add console appender
//	console := log4go.NewConsoleAppender()
//	console.SetFormatter(log4go.NewJSONFormatter())
//	logger.AddAppender(console)
//
//	// Add rolling file appender
//	file, _ := log4go.NewRollingFileAppender(
//	    "file",
//	    "/var/log/app.log",
//	    log4go.NewSizeBasedRollingPolicy(10), // 10MB
//	    5, // Keep 5 backups
//	)
//	logger.AddAppender(file)
//
// Configure via YAML:
//
//	loggers:
//	  root:
//	    level: INFO
//	    appenders:
//	      - type: console
//	        name: console
//	        config:
//	          formatter: text
//	          use_colors: true
//	      - type: rolling
//	        name: file
//	        config:
//	          path: /var/log/app.log
//	          max_size: 10
//	          max_backups: 5
//	          formatter: json
//
// # Log Levels
//
// Supported log levels in order of severity:
//   - TRACE: Finest-grained informational events
//   - DEBUG: Fine-grained informational events for debugging
//   - INFO: Informational messages highlighting application progress
//   - WARN: Potentially harmful situations
//   - ERROR: Error events that might still allow the application to continue
//   - FATAL: Very severe error events that will presumably lead the application to abort
//   - OFF: Special level to disable all logging
//
// # Appenders
//
// Console Appender - writes to stdout/stderr:
//
//	appender := log4go.NewConsoleAppender()
//
// File Appender - writes to a file:
//
//	appender, _ := log4go.NewFileAppender("file", "/var/log/app.log")
//
// Rolling File Appender - writes to a file with size-based rotation:
//
//	policy := log4go.NewSizeBasedRollingPolicy(10) // 10MB
//	appender, _ := log4go.NewRollingFileAppender("rolling", "/var/log/app.log", policy, 5)
//
// Daily Rolling File Appender - writes to a file with daily rotation:
//
//	appender, _ := log4go.NewDailyRollingFileAppender("daily", "/var/log/app.log", 30)
//
// Async Appender - wraps another appender for async writing:
//
//	file, _ := log4go.NewFileAppender("file", "/var/log/app.log")
//	async := log4go.NewAsyncAppender("async", file, 1000) // Queue size 1000
//
// # Formatters
//
// Text Formatter - human-readable text with optional colors:
//
//	formatter := log4go.NewTextFormatter()
//	formatter.UseColors = true
//	formatter.ShowCaller = true
//
// JSON Formatter - structured JSON output:
//
//	formatter := log4go.NewJSONFormatter()
//	formatter.PrettyPrint = true
//
// Pattern Formatter - custom pattern-based format:
//
//	formatter := log4go.NewPatternFormatter("%d %l [%n] %m")
//	// %d=timestamp, %l=level, %n=logger name, %m=message
//
// # Structured Logging
//
// Add fields to log entries:
//
//	logger.WithField("user", "john").Info("User action")
//	logger.WithFields(log4go.Fields{
//	    "user": "john",
//	    "action": "login",
//	    "ip": "192.168.1.1",
//	}).Info("User logged in")
//
// # Context Integration
//
// Extract values from context automatically:
//
//	ctx := context.WithValue(context.Background(), "request_id", "abc123")
//	logger.WithContext(ctx).Info("Processing request")
//	// Automatically includes request_id in log
//
// # Performance
//
// For high-throughput applications, use async appender:
//
//	file, _ := log4go.NewFileAppender("file", "/var/log/app.log")
//	async := log4go.NewAsyncAppender("async", file, 10000)
//	logger.AddAppender(async)
//	defer async.Close() // Ensure all logs are flushed
//
// # Best Practices
//
//  1. Use appropriate log levels
//  2. Use structured logging with fields
//  3. Use async appenders for high-throughput scenarios
//  4. Configure log rotation to prevent disk space issues
//  5. Use context to add request/trace IDs
//  6. Close appenders properly on shutdown
//
// # Thread Safety
//
// All operations are thread-safe. Multiple goroutines can safely use the same logger.
//
// # Migration from core.Logger
//
// log4go can replace core.Logger:
//
//	// Old
//	logger := core.NewDefaultLogger()
//	logger.Info("message")
//
//	// New
//	logger := log4go.GetLogger("myapp")
//	logger.SetLevel(log4go.INFO)
//	logger.AddAppender(log4go.NewConsoleAppender())
//	logger.Info("message")
//
// Path: pkg/log4go
package log4go
