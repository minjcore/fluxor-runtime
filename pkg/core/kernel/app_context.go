package kernel

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core"
)

// Meta carries cross-cutting identifiers (HTTP headers, EventBus metadata, CLI flags).
type Meta struct {
	RequestID string
	UserID    string
	FloxID    string
}

// AppContext is a context.Context plus Fluxor routing identity (HTTP, EventBus, workers share the same contract).
type AppContext interface {
	context.Context
	Meta() Meta
	RequestID() string
	UserID() string
	FloxID() string
}

type appContext struct {
	context.Context
	meta Meta
}

// NewAppContext wraps ctx with meta (for tests, EventBus, CLI). HTTP adapters should pass c.Request.Context().
func NewAppContext(ctx context.Context, meta Meta) AppContext {
	if ctx == nil {
		ctx = context.Background()
	}
	return &appContext{Context: ctx, meta: meta}
}

// WithMeta returns a copy of ctx as AppContext with replaced meta (keeps cancel/deadline from ctx).
func WithMeta(ctx context.Context, meta Meta) AppContext {
	if ctx == nil {
		ctx = context.Background()
	}
	return &appContext{Context: ctx, meta: meta}
}

func (a *appContext) Meta() Meta   { return a.meta }
func (a *appContext) RequestID() string {
	if a.meta.RequestID != "" {
		return a.meta.RequestID
	}
	return core.GetRequestID(a.Context)
}
func (a *appContext) UserID() string { return a.meta.UserID }
func (a *appContext) FloxID() string {
	if a.meta.FloxID != "" {
		return a.meta.FloxID
	}
	return core.GetFloxID(a.Context)
}
