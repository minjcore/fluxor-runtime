package kernel

import (
	"context"
	"strings"
)

// MessageHeaders is implemented by event bus messages (and similar transports).
type MessageHeaders interface {
	Headers() map[string]string
}

// MetaFromHeaders maps X-* routing headers into Meta.
func MetaFromHeaders(h map[string]string) Meta {
	if h == nil {
		return Meta{}
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			for mk, mv := range h {
				if strings.EqualFold(mk, k) {
					return mv
				}
			}
		}
		return ""
	}
	return Meta{
		RequestID: get("X-Request-ID", "x-request-id"),
		UserID:    get("X-User-ID", "x-user-id"),
		FloxID:    get("X-Flox-ID", "x-flox-id"),
	}
}

// AdaptMessageHandler turns a kernel Handler into a consumer callback (base ctx is usually context.Background() or kernel.Context()).
func AdaptMessageHandler(base context.Context, h Handler) func(MessageHeaders) error {
	if base == nil {
		base = context.Background()
	}
	return func(m MessageHeaders) error {
		meta := MetaFromHeaders(m.Headers())
		ctx := NewAppContext(base, meta)
		return h(ctx)
	}
}
