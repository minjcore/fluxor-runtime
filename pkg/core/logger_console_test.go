package core

import (
	"fmt"
	"testing"
)

// TestLoggerConsoleOutput tests that logs are written to console
func TestLoggerConsoleOutput(t *testing.T) {
	logger := NewDefaultLogger()

	t.Log("Testing console output...")

	// Test all log levels
	logger.Debug("Debug message")
	logger.Debug(fmt.Sprintf("Debug formatted: %s", "test"))

	logger.Info("Info message")
	logger.Info(fmt.Sprintf("Info formatted: %s", "test"))

	logger.Info("Info message")
	logger.Info(fmt.Sprintf("Info formatted: %s", "test"))

	logger.Error("Error message")
	logger.Error(fmt.Sprintf("Error formatted: %s", "test"))

	// Test with fields
	loggerWithFields := logger.WithFields(map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	})
	loggerWithFields.Info("Message with fields")

	// Test JSON logger
	jsonLogger := NewJSONLogger()
	jsonLogger.Info("JSON logger test")
	jsonLogger.WithFields(map[string]interface{}{
		"component": "test",
		"action":    "console_output",
	}).Info("JSON logger with fields")

	t.Log("Console output test completed - check output above")
}
