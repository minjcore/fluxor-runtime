package otel

import (
	"context"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// PublishWithSpan publishes a message with span propagation
func PublishWithSpan(ctx context.Context, eventBus core.EventBus, address string, body interface{}) error {
	if !IsInitialized() {
		return eventBus.Publish(address, body)
	}

	_, span := StartSpan(ctx, "eventbus.publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("fluxor"),
			semconv.MessagingDestinationKey.String(address),
			semconv.MessagingOperationKey.String("publish"),
		),
	)
	defer span.End()

	// Add request ID to span if available
	if requestID := core.GetRequestID(ctx); requestID != "" {
		span.SetAttributes(attribute.String("request_id", requestID))
	}

	err := eventBus.Publish(address, body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "OK")
	}

	return err
}

// SendWithSpan sends a message with span propagation
func SendWithSpan(ctx context.Context, eventBus core.EventBus, address string, body interface{}) error {
	if !IsInitialized() {
		return eventBus.Send(address, body)
	}

	_, span := StartSpan(ctx, "eventbus.send",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("fluxor"),
			semconv.MessagingDestinationKey.String(address),
			semconv.MessagingOperationKey.String("send"),
		),
	)
	defer span.End()

	// Add request ID to span if available
	if requestID := core.GetRequestID(ctx); requestID != "" {
		span.SetAttributes(attribute.String("request_id", requestID))
	}

	err := eventBus.Send(address, body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "OK")
	}

	return err
}

// RequestWithSpan sends a request with span propagation
func RequestWithSpan(ctx context.Context, eventBus core.EventBus, address string, body interface{}, timeout time.Duration) (core.Message, error) {
	if !IsInitialized() {
		return eventBus.Request(address, body, timeout)
	}

	_, span := StartSpan(ctx, "eventbus.request",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("fluxor"),
			semconv.MessagingDestinationKey.String(address),
			semconv.MessagingOperationKey.String("request"),
		),
	)
	defer span.End()

	// Add request ID to span if available
	if requestID := core.GetRequestID(ctx); requestID != "" {
		span.SetAttributes(attribute.String("request_id", requestID))
	}

	msg, err := eventBus.Request(address, body, timeout)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "OK")
	}

	return msg, err
}

// WrapConsumerHandler wraps a consumer handler with span creation
func WrapConsumerHandler(address string, handler core.MessageHandler) core.MessageHandler {
	if !IsInitialized() {
		return handler
	}

	return func(ctx core.FluxorContext, msg core.Message) error {
		// Extract span context from message headers if available
		// For now, create a new span for each message
		_, span := StartSpan(ctx.Context(), "eventbus.consume",
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(
				semconv.MessagingSystemKey.String("fluxor"),
				semconv.MessagingDestinationKey.String(address),
				semconv.MessagingOperationKey.String("consume"),
			),
		)
		defer span.End()

		// Add request ID to span if available
		if requestID := core.GetRequestID(ctx.Context()); requestID != "" {
			span.SetAttributes(attribute.String("request_id", requestID))
		}

		// Execute handler
		err := handler(ctx, msg)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "OK")
		}

		return err
	}
}
