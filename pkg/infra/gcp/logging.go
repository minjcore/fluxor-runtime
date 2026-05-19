package gcp

import (
	"context"
	"os"
	"time"
)

// LogEntry represents a single log entry for Cloud Logging.
type LogEntry struct {
	Timestamp time.Time
	Severity  string // DEBUG, INFO, WARNING, ERROR, CRITICAL
	Message   string
	Payload   map[string]any
	Trace     string
	SpanID    string
	Resource  map[string]string
	Labels    map[string]string
}

// LoggingClient writes logs to GCP Cloud Logging.
type LoggingClient interface {
	// Write sends log entries to Cloud Logging.
	Write(ctx context.Context, entries []LogEntry) error
	// Close releases resources.
	Close() error
}

// LoggingConfig configures the Cloud Logging client.
type LoggingConfig struct {
	ProjectID string
	LogName   string // e.g. "myapp" -> projects/PROJECT/logs/myapp
}

// DefaultLoggingConfig returns config with optional env-based project ID (GCP_PROJECT, GCP_LOG_NAME).
func DefaultLoggingConfig(projectID string) LoggingConfig {
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT")
	}
	logName := os.Getenv("GCP_LOG_NAME")
	if logName == "" {
		logName = "default"
	}
	return LoggingConfig{
		ProjectID: projectID,
		LogName:   logName,
	}
}

// NewLoggingClient creates a Cloud Logging client for the given project.
// Requires LoggingConfig.ProjectID (or set GCP_PROJECT). Uses cloud.google.com/go/logging.
func NewLoggingClient(ctx context.Context, cfg LoggingConfig) (LoggingClient, error) {
	return newLoggingClient(ctx, cfg)
}
