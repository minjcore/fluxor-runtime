package otel

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	globalTracer trace.Tracer
	mu           sync.RWMutex
	initialized  bool
)

// Initialize initializes OpenTelemetry with the given configuration
func Initialize(ctx context.Context, config Config) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid OpenTelemetry config: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if initialized {
		return fmt.Errorf("OpenTelemetry already initialized")
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter
	var exporter sdktrace.SpanExporter
	switch config.Exporter {
	case "jaeger":
		exporter, err = newJaegerExporter(config.Endpoint)
		if err != nil {
			return fmt.Errorf("failed to create Jaeger exporter: %w", err)
		}
	case "zipkin":
		exporter, err = newZipkinExporter(config.Endpoint)
		if err != nil {
			return fmt.Errorf("failed to create Zipkin exporter: %w", err)
		}
	case "stdout":
		exporter = newStdoutExporter()
	case "none":
		exporter = newNoopExporter()
	default:
		return fmt.Errorf("unsupported exporter: %s", config.Exporter)
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create tracer
	globalTracer = tp.Tracer(config.ServiceName)

	initialized = true
	return nil
}

// Tracer returns the global tracer
func Tracer() trace.Tracer {
	mu.RLock()
	defer mu.RUnlock()
	if globalTracer == nil {
		// Return noop tracer if not initialized
		return trace.NewNoopTracerProvider().Tracer("noop")
	}
	return globalTracer
}

// StartSpan starts a new span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, opts...)
}

// SpanFromContext extracts a span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// IsInitialized returns whether OpenTelemetry has been initialized
func IsInitialized() bool {
	mu.RLock()
	defer mu.RUnlock()
	return initialized
}

// Shutdown shuts down the tracer provider
func Shutdown(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()

	if !initialized {
		return nil
	}

	if tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		return tp.Shutdown(ctx)
	}

	return nil
}
