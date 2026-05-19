package log4go_test

import (
	"context"
	"fmt"

	"github.com/fluxorio/fluxor/pkg/log4go"
)

// Example demonstrates basic usage
func Example() {
	// Create logger
	logger := log4go.GetLogger("myapp")

	// Log messages at different levels
	logger.Info("Application started")
	logger.Debug("Debug information")
	logger.Warn("Warning message")
	logger.Error("Error occurred")

	// Output will vary based on timestamp
}

// ExampleLogger_WithFields demonstrates structured logging
func ExampleLogger_WithFields() {
	logger := log4go.GetLogger("myapp")
	logger.AddAppender(log4go.NewConsoleAppender())

	// Add structured fields
	logger.WithFields(log4go.Fields{
		"user":   "john",
		"action": "login",
		"ip":     "192.168.1.1",
	}).Info("User logged in")

	// Chain field additions
	logger.
		WithField("user", "jane").
		WithField("role", "admin").
		Info("Admin action")
}

// ExampleLogger_WithContext demonstrates context integration
func ExampleLogger_WithContext() {
	logger := log4go.GetLogger("myapp")
	logger.AddAppender(log4go.NewConsoleAppender())

	// Create context with request ID
	ctx := context.WithValue(context.Background(), "request_id", "req-12345")

	// Log with context - automatically includes request_id
	logger.WithContext(ctx).Info("Processing request")
}

// ExampleNewRollingFileAppender demonstrates file rotation
func ExampleNewRollingFileAppender() {
	logger := log4go.NewLogger("myapp")

	// Create rolling file appender (10MB per file, keep 5 backups)
	policy := log4go.NewSizeBasedRollingPolicy(10)
	appender, err := log4go.NewRollingFileAppender(
		"file",
		"/tmp/app.log",
		policy,
		5,
	)
	if err != nil {
		panic(err)
	}
	defer appender.Close()

	logger.AddAppender(appender)
	logger.Info("This will be written to a rotating file")
}

// ExampleNewAsyncAppender demonstrates async logging for performance
func ExampleNewAsyncAppender() {
	logger := log4go.NewLogger("myapp")

	// Create file appender
	file, _ := log4go.NewFileAppender("file", "/tmp/app.log")

	// Wrap with async appender for non-blocking writes
	async := log4go.NewAsyncAppender("async", file, 1000)
	defer async.Close() // Important: flush pending logs

	logger.AddAppender(async)

	// High-throughput logging
	for i := 0; i < 10000; i++ {
		logger.Infof("Processing item %d", i)
	}
}

// ExampleJSONFormatter demonstrates JSON output
func ExampleJSONFormatter() {
	logger := log4go.NewLogger("myapp")

	// Create console appender with JSON formatter
	appender := log4go.NewConsoleAppender()
	formatter := log4go.NewJSONFormatter()
	formatter.PrettyPrint = true
	appender.SetFormatter(formatter)

	logger.AddAppender(appender)

	logger.WithFields(log4go.Fields{
		"user": "john",
		"age":  30,
	}).Info("User data")
}

// ExamplePatternFormatter demonstrates custom patterns
func ExamplePatternFormatter() {
	logger := log4go.NewLogger("myapp")

	// Create console appender with pattern formatter
	appender := log4go.NewConsoleAppender()
	formatter := log4go.NewPatternFormatter("[%d] %l - %n - %m")
	appender.SetFormatter(formatter)

	logger.AddAppender(appender)
	logger.Info("Custom format message")
}

// ExampleMultiAppender demonstrates writing to multiple destinations
func ExampleMultiAppender() {
	logger := log4go.NewLogger("myapp")

	// Create multiple appenders
	console := log4go.NewConsoleAppender()
	file, _ := log4go.NewFileAppender("file", "/tmp/app.log")

	// Combine into multi appender
	multi := log4go.NewMultiAppender("multi", console, file)
	logger.AddAppender(multi)

	// This will be written to both console and file
	logger.Info("Message to multiple destinations")
}

// ExampleLoadConfigFromFile demonstrates YAML configuration
func ExampleLoadConfigFromFile() {
	// Load configuration from YAML file
	config, err := log4go.LoadConfigFromFile("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	// Apply to logger
	logger := log4go.NewLogger("myapp")
	if loggerConfig, ok := config.Loggers["myapp"]; ok {
		log4go.ApplyConfig(logger, loggerConfig)
	}

	logger.Info("Configured from YAML")
}

// ExampleDailyRollingFileAppender demonstrates daily log rotation
func ExampleDailyRollingFileAppender() {
	logger := log4go.NewLogger("myapp")

	// Create daily rolling appender (keep 30 days)
	appender, err := log4go.NewDailyRollingFileAppender(
		"daily",
		"/var/log/app.log",
		30,
	)
	if err != nil {
		panic(err)
	}
	defer appender.Close()

	logger.AddAppender(appender)
	logger.Info("This will be in a daily rotated file")
}

// ExampleLevel demonstrates log level filtering
func ExampleLevel() {
	logger := log4go.NewLogger("myapp")
	logger.AddAppender(log4go.NewConsoleAppender())

	// Set to WARN - only WARN, ERROR, FATAL will be logged
	logger.SetLevel(log4go.WARN)

	logger.Debug("This won't be logged")
	logger.Info("This won't be logged either")
	logger.Warn("This will be logged")
	logger.Error("This will be logged too")
}
