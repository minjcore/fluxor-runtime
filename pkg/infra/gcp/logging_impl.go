package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/logging"
)

// loggingClientImpl implements LoggingClient using cloud.google.com/go/logging.
type loggingClientImpl struct {
	client *logging.Client
	logger *logging.Logger
}

// newLoggingClient creates a GCP Cloud Logging client.
func newLoggingClient(ctx context.Context, cfg LoggingConfig) (LoggingClient, error) {
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("gcp: LoggingConfig.ProjectID is required")
	}
	client, err := logging.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("gcp: create logging client: %w", err)
	}
	logName := cfg.LogName
	if logName == "" {
		logName = "default"
	}
	logger := client.Logger(logName)
	return &loggingClientImpl{
		client: client,
		logger: logger,
	}, nil
}

// severityFromString maps our severity string to logging.Severity.
func severityFromString(s string) logging.Severity {
	switch s {
	case "DEBUG":
		return logging.Debug
	case "INFO":
		return logging.Info
	case "WARNING":
		return logging.Warning
	case "ERROR":
		return logging.Error
	case "CRITICAL":
		return logging.Critical
	case "ALERT":
		return logging.Alert
	case "EMERGENCY":
		return logging.Emergency
	case "NOTICE":
		return logging.Notice
	default:
		return logging.ParseSeverity(s)
	}
}

// Write sends log entries to Cloud Logging. Best-effort, non-blocking (Logger.Log is async).
func (c *loggingClientImpl) Write(ctx context.Context, entries []LogEntry) error {
	for _, e := range entries {
		payload := make(map[string]any)
		payload["message"] = e.Message
		for k, v := range e.Payload {
			payload[k] = v
		}
		sev := severityFromString(e.Severity)
		ent := logging.Entry{
			Payload:   payload,
			Severity:  sev,
			Timestamp: e.Timestamp,
		}
		if e.Trace != "" {
			ent.Trace = e.Trace
		}
		if e.SpanID != "" {
			ent.SpanID = e.SpanID
		}
		if len(e.Labels) > 0 {
			ent.Labels = e.Labels
		}
		c.logger.Log(ent)
	}
	return nil
}

// Close flushes buffered entries and closes the client.
func (c *loggingClientImpl) Close() error {
	_ = c.logger.Flush()
	return c.client.Close()
}
