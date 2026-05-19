package core

import (
	"context"
)

// FloxIDKey is the context key for FloxID
// FloxID is a universal routing identifier (aggregate ID, stream ID, entity ID, etc.)
var FloxIDKey = struct{ name string }{"floxid"}

// WithFloxID adds a FloxID to the context
// FloxID is a universal routing identifier used for EventLoop routing
// Examples: aggregate ID (DDD), stream ID, entity ID, etc.
func WithFloxID(ctx context.Context, floxid string) context.Context {
	return context.WithValue(ctx, FloxIDKey, floxid)
}

// GetFloxID retrieves the FloxID from context
// Returns empty string if FloxID is not set
func GetFloxID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if floxid, ok := ctx.Value(FloxIDKey).(string); ok {
		return floxid
	}
	return ""
}

// GenerateFloxID generates a new FloxID
// Uses the same format as GenerateRequestID for consistency
func GenerateFloxID() string {
	return GenerateRequestID() // Reuse request ID generator for consistency
}

// WithNewFloxID adds a new FloxID to the context
func WithNewFloxID(ctx context.Context) context.Context {
	return WithFloxID(ctx, GenerateFloxID())
}
