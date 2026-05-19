package core

import "context"

// Scope represents a bounded lifecycle (e.g. per-request): Start at init (e.g. in middleware), Stop on cleanup (defer).
// Use Context() to get the scoped context for tracing, logger, metrics, etc.
type Scope interface {
	Start() error              // called at init (typically in middleware)
	Stop()                     // cleanup (call in defer)
	Context() context.Context
}

// requestScope is a simple per-request scope holding a context.
type requestScope struct {
	ctx context.Context
}

// NewRequestScope creates a Scope that wraps ctx. Call Start() in middleware, defer Stop().
func NewRequestScope(ctx context.Context) Scope {
	return &requestScope{ctx: ctx}
}

// Start initializes the scope (e.g. tracing, scoped logger, metrics). Override in composed types as needed.
func (s *requestScope) Start() error {
	return nil
}

// Stop performs cleanup (end span, flush buffer, release resources). Override in composed types as needed.
func (s *requestScope) Stop() {}

// Context returns the scoped context.
func (s *requestScope) Context() context.Context {
	return s.ctx
}
