package gcp

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// cloudTeeLogger wraps a core.Logger and tees log entries to GCP Cloud Logging (async, best-effort).
type cloudTeeLogger struct {
	inner  core.Logger
	client LoggingClient
	fields map[string]any
	trace  string
}

// NewLoggerWithCloud returns a Logger that forwards to inner and also sends entries to GCP Cloud Logging.
// When profile is "gke-autopilot" (or similar) and GCP_PROJECT is set, use this so logs go to Cloud instead of (or in addition to) file.
// closeFn must be called on shutdown to flush and close the logging client; if Cloud client failed to create, closeFn is a no-op.
func NewLoggerWithCloud(ctx context.Context, inner core.Logger, cfg LoggingConfig) (core.Logger, func() error, error) {
	client, err := NewLoggingClient(ctx, cfg)
	if err != nil {
		return inner, func() error { return nil }, nil
	}
	w := &cloudTeeLogger{
		inner:  inner,
		client: client,
		fields: make(map[string]any),
	}
	return w, client.Close, nil
}

func (w *cloudTeeLogger) tee(severity, message string) {
	payload := make(map[string]any)
	for k, v := range w.fields {
		payload[k] = v
	}
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Severity:  severity,
		Message:   message,
		Payload:   payload,
		Trace:     w.trace,
	}
	go func() {
		_ = w.client.Write(context.Background(), []LogEntry{entry})
	}()
}

func (w *cloudTeeLogger) Error(args ...interface{}) {
	w.inner.Error(args...)
	w.tee("ERROR", fmt.Sprint(args...))
}

func (w *cloudTeeLogger) Info(args ...interface{}) {
	w.inner.Info(args...)
	w.tee("INFO", fmt.Sprint(args...))
}

func (w *cloudTeeLogger) Debug(args ...interface{}) {
	w.inner.Debug(args...)
	w.tee("DEBUG", fmt.Sprint(args...))
}

func (w *cloudTeeLogger) WithFields(fields map[string]interface{}) core.Logger {
	merged := make(map[string]any)
	for k, v := range w.fields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	return &cloudTeeLogger{
		inner:  w.inner.WithFields(fields),
		client: w.client,
		fields: merged,
		trace:  w.trace,
	}
}

func (w *cloudTeeLogger) WithContext(ctx context.Context) core.Logger {
	trace := core.GetTrace(ctx)
	return &cloudTeeLogger{
		inner:  w.inner.WithContext(ctx),
		client: w.client,
		fields: w.fields,
		trace:  trace,
	}
}
