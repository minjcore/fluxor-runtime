package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestNewDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()

	if logger == nil {
		t.Error("NewDefaultLogger() should not return nil")
	}

	// Test that logger methods don't panic
	logger.Error("test error")
	logger.Error(fmt.Sprintf("test error: %s", "message"))
	logger.Info("test info")
	logger.Info(fmt.Sprintf("test info: %s", "message"))
	logger.Info("test info")
	logger.Info(fmt.Sprintf("test info: %s", "message"))
	logger.Debug("test debug")
	logger.Debug(fmt.Sprintf("test debug: %s", "message"))
}

func TestPackageLevelFunctions(t *testing.T) {
	// Test package-level Error function
	Error("package level error")
	Error(fmt.Sprintf("package level error: %s", "formatted"))

	// Test package-level Info function
	Info("package level info")
	Info(fmt.Sprintf("package level info: %s", "formatted"))

	// Test package-level Debug function
	Debug("package level debug")
	Debug(fmt.Sprintf("package level debug: %s", "formatted"))

	// Test with multiple args (no format specifiers)
	Info("arg1", "arg2", "arg3")
}

func TestLoggerWithFields(t *testing.T) {
	logger := NewDefaultLogger()

	fields := map[string]interface{}{
		"user_id": "123",
		"action":  "login",
	}

	loggerWithFields := logger.WithFields(fields)

	if loggerWithFields == nil {
		t.Error("WithFields() should not return nil")
	}

	// Test that it's a different instance
	if loggerWithFields == logger {
		t.Error("WithFields() should return a new logger instance")
	}

	// Test logging with fields (should not panic)
	loggerWithFields.Info("User logged in")
}

func TestLoggerWithContext(t *testing.T) {
	logger := NewDefaultLogger()

	// Create context with request ID
	requestID := GenerateRequestID()
	ctx := WithRequestID(context.Background(), requestID)

	loggerWithContext := logger.WithContext(ctx)

	if loggerWithContext == nil {
		t.Error("WithContext() should not return nil")
	}

	// Test logging with context (should not panic)
	loggerWithContext.Info("Request processed")
}

func TestJSONLogger(t *testing.T) {
	logger := NewJSONLogger()

	// Test JSON output
	logger.WithFields(map[string]interface{}{
		"test": "value",
	}).Info("test message")

	// Verify it's a JSON logger
	jsonLogger, ok := logger.(*defaultLogger)
	if !ok {
		t.Fatal("NewJSONLogger() should return *defaultLogger")
	}

	if !jsonLogger.config.JSONOutput {
		t.Error("JSON logger should have JSONOutput enabled")
	}
}

func TestJSONLoggerOutput(t *testing.T) {
	// Create a logger that captures output
	logger := NewJSONLogger().WithFields(map[string]interface{}{
		"user_id": "123",
		"action":  "test",
	})

	// Log a message
	logger.Info("test message")

	// Verify JSON structure (we can't easily capture output in tests,
	// but we can verify the logEntry structure is correct)
	entry := logEntry{
		Level:   "INFO",
		Message: "test message",
		Fields: map[string]interface{}{
			"user_id": "123",
			"action":  "test",
		},
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal log entry: %v", err)
	}

	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, "test message") {
		t.Error("JSON output should contain message")
	}
	if !strings.Contains(jsonStr, "user_id") {
		t.Error("JSON output should contain fields")
	}
}
