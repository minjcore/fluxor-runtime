package otel

import (
	"strconv"

	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware creates middleware that automatically traces HTTP requests
func HTTPMiddleware() web.FastMiddleware {
	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			if !IsInitialized() {
				// If OpenTelemetry not initialized, just call next handler
				return next(ctx)
			}

			// Extract trace context from headers
			propagator := propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			)

			carrier := newHeaderCarrier(&ctx.RequestCtx.Request.Header)
			parentCtx := propagator.Extract(ctx.Context(), carrier)

			// Start span
			spanCtx, span := StartSpan(parentCtx, "http.request",
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPMethodKey.String(string(ctx.Method())),
					semconv.HTTPURLKey.String(string(ctx.Path())),
					semconv.HTTPRouteKey.String(string(ctx.Path())),
					attribute.String("http.request_id", ctx.RequestID()),
				),
			)

			defer span.End()

			// Store span context in request context
			ctx.Set("span_context", spanCtx)

			// Execute handler
			err := next(ctx)

			// Set span attributes based on response
			statusCode := ctx.RequestCtx.Response.StatusCode()
			span.SetAttributes(
				semconv.HTTPStatusCodeKey.Int(statusCode),
				attribute.Int("http.response_size", len(ctx.RequestCtx.Response.Body())),
			)

			// Set span status
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else if statusCode >= 500 {
				span.SetStatus(codes.Error, "HTTP "+strconv.Itoa(statusCode))
			} else if statusCode >= 400 {
				span.SetStatus(codes.Error, "HTTP "+strconv.Itoa(statusCode))
			} else {
				span.SetStatus(codes.Ok, "OK")
			}

			// Inject trace context into response headers
			responseCarrier := newResponseHeaderCarrier(&ctx.RequestCtx.Response.Header)
			propagator.Inject(spanCtx, responseCarrier)

			return err
		}
	}
}

// headerCarrier implements propagation.TextMapCarrier for fasthttp request headers
type headerCarrier struct {
	headers *fasthttp.RequestHeader
}

func newHeaderCarrier(headers *fasthttp.RequestHeader) *headerCarrier {
	return &headerCarrier{headers: headers}
}

func (c *headerCarrier) Get(key string) string {
	return string(c.headers.Peek(key))
}

func (c *headerCarrier) Set(key, value string) {
	c.headers.Set(key, value)
}

func (c *headerCarrier) Keys() []string {
	// Not needed for extraction
	return nil
}

// responseHeaderCarrier implements propagation.TextMapCarrier for fasthttp response headers
type responseHeaderCarrier struct {
	headers *fasthttp.ResponseHeader
}

func newResponseHeaderCarrier(headers *fasthttp.ResponseHeader) *responseHeaderCarrier {
	return &responseHeaderCarrier{headers: headers}
}

func (c *responseHeaderCarrier) Get(key string) string {
	return string(c.headers.Peek(key))
}

func (c *responseHeaderCarrier) Set(key, value string) {
	c.headers.Set(key, value)
}

func (c *responseHeaderCarrier) Keys() []string {
	// Not needed for injection
	return nil
}

// SpanFromRequest extracts the span context from a request
func SpanFromRequest(ctx *web.FastRequestContext) (trace.SpanContext, bool) {
	spanCtx, ok := ctx.Get("span_context").(contextWithSpan)
	if !ok {
		return trace.SpanContext{}, false
	}
	return spanCtx.SpanContext(), true
}

// contextWithSpan is a helper interface for extracting span context
type contextWithSpan interface {
	SpanContext() trace.SpanContext
}

// Note: The actual context.Context returned from StartSpan already contains the span,
// so we can use trace.SpanFromContext() directly. This interface is for type checking.
