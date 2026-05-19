package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/zipkin"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// newJaegerExporter creates a Jaeger exporter
func newJaegerExporter(endpoint string) (sdktrace.SpanExporter, error) {
	if endpoint == "" {
		endpoint = "http://localhost:14268/api/traces"
	}

	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	return exporter, nil
}

// newZipkinExporter creates a Zipkin exporter
func newZipkinExporter(endpoint string) (sdktrace.SpanExporter, error) {
	if endpoint == "" {
		endpoint = "http://localhost:9411/api/v2/spans"
	}

	exporter, err := zipkin.New(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create Zipkin exporter: %w", err)
	}

	return exporter, nil
}

// newStdoutExporter creates a stdout exporter (for debugging)
func newStdoutExporter() sdktrace.SpanExporter {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		// Fallback to noop if stdout fails
		return newNoopExporter()
	}
	return exporter
}

// newNoopExporter creates a noop exporter (no tracing)
func newNoopExporter() sdktrace.SpanExporter {
	return &noopExporter{}
}

// noopExporter is a noop span exporter
type noopExporter struct{}

func (e *noopExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *noopExporter) Shutdown(ctx context.Context) error {
	return nil
}
