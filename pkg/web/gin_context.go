// Package web provides Gin HTTP server support for Fluxor.
// GinRequestContext wraps gin.Context with Fluxor context (GoCMD, EventBus, routing).

package web

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/gin-gonic/gin"
)

// DefaultEventBusPublishTimeout bounds PublishWithRouting when the EventBus call
// does not take an explicit timeout (tracing, cancellation, avoids hangs).
const DefaultEventBusPublishTimeout = 3 * time.Second

// APIError is a stable JSON shape for clients and observability (codes, not raw strings only).
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GinRequestContext wraps gin.Context with Fluxor context (GoCMD, EventBus, request ID, params).
// It provides the same API surface as FastRequestContext so handlers can use Fluxor features
// (EventBus, routing headers, Context()) while using Gin's router and middleware.
// Scope is optional; set by UnitOfWorkMiddleware when you need per-request lifecycle (tracing, logger, metrics).
type GinRequestContext struct {
	*core.BaseRequestContext
	GinCtx    *gin.Context
	GoCMD     core.GoCMD
	EventBus  core.EventBus
	Params    map[string]string
	Scope     core.Scope // optional; set by UnitOfWorkMiddleware
	requestID string
}

// GinRequestHandler handles HTTP requests with GinRequestContext (Fluxor-aware).
type GinRequestHandler func(ctx *GinRequestContext) error

// JSON writes JSON response.
func (c *GinRequestContext) JSON(statusCode int, data interface{}) error {
	if statusCode < 100 || statusCode > 599 {
		return fmt.Errorf("invalid status code: %d", statusCode)
	}
	if c.GinCtx == nil {
		return fmt.Errorf("GinCtx is nil")
	}
	c.GinCtx.JSON(statusCode, data)
	return nil
}

// BindJSON binds JSON request body to v.
func (c *GinRequestContext) BindJSON(v interface{}) error {
	if v == nil {
		return fmt.Errorf("cannot bind to nil value")
	}
	if c.GinCtx == nil {
		return fmt.Errorf("GinCtx is nil")
	}
	return c.GinCtx.ShouldBindJSON(v)
}

// Text writes text response.
func (c *GinRequestContext) Text(statusCode int, text string) error {
	if c.GinCtx == nil {
		return fmt.Errorf("GinCtx is nil")
	}
	c.GinCtx.String(statusCode, text)
	return nil
}

// Query returns query parameter value.
func (c *GinRequestContext) Query(key string) string {
	if c.GinCtx == nil {
		return ""
	}
	return c.GinCtx.Query(key)
}

// Param returns path parameter value (from Gin route params).
func (c *GinRequestContext) Param(key string) string {
	if c.Params != nil {
		if v, ok := c.Params[key]; ok {
			return v
		}
	}
	if c.GinCtx != nil {
		return c.GinCtx.Param(key)
	}
	return ""
}

// Method returns HTTP method as bytes (for compatibility with FastRequestContext).
func (c *GinRequestContext) Method() []byte {
	if c.GinCtx == nil || c.GinCtx.Request == nil {
		return nil
	}
	return []byte(c.GinCtx.Request.Method)
}

// Path returns request path as bytes (for compatibility with FastRequestContext).
func (c *GinRequestContext) Path() []byte {
	if c.GinCtx == nil || c.GinCtx.Request == nil || c.GinCtx.Request.URL == nil {
		return nil
	}
	return []byte(c.GinCtx.Request.URL.Path)
}

// Error writes an API error response and aborts the request (code "error").
func (c *GinRequestContext) Error(msg string, statusCode int) {
	c.AbortWithAPIError(statusCode, "error", msg)
}

// AbortWithAPIError writes a structured API error and aborts the request.
func (c *GinRequestContext) AbortWithAPIError(statusCode int, code, message string) {
	if c.GinCtx != nil {
		c.GinCtx.AbortWithStatusJSON(statusCode, APIError{Code: code, Message: message})
	}
}

// AbortWithError writes err as the message and aborts (default code "internal_error" when code is empty).
func (c *GinRequestContext) AbortWithError(statusCode int, err error, code string) {
	if err == nil {
		return
	}
	if code == "" {
		code = "internal_error"
	}
	c.AbortWithAPIError(statusCode, code, err.Error())
}

// RequestID returns the request ID for this request.
func (c *GinRequestContext) RequestID() string {
	return c.requestID
}

// RealIP returns the client IP (X-Forwarded-For / X-Real-IP when behind proxy).
func (c *GinRequestContext) RealIP() string {
	if v := c.Get("real_ip"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if c.GinCtx != nil {
		return c.GinCtx.ClientIP()
	}
	return ""
}

// UserID resolves subject id: typed JWTClaims (Gin + Fluxor store) → legacy claim maps → user_id → X-User-ID.
func (c *GinRequestContext) UserID() string {
	if id := userIDFromTypedClaims(c.Get(ContextKeyJWTClaims)); id != "" {
		return id
	}
	if c.GinCtx != nil {
		if v, ok := c.GinCtx.Get(ContextKeyJWTClaims); ok {
			if id := userIDFromTypedClaims(v); id != "" {
				return id
			}
		}
	}
	legacyKeys := []string{"user", "jwt", "claims"}
	for _, key := range legacyKeys {
		if userID := c.getUserIDFromJWTClaimsMap(key); userID != "" {
			return userID
		}
	}
	if userID, ok := c.Get("user_id").(string); ok && userID != "" {
		return userID
	}
	if c.GinCtx != nil && c.GinCtx.Request != nil {
		return c.GinCtx.GetHeader("X-User-ID")
	}
	return ""
}

func userIDFromTypedClaims(v interface{}) string {
	switch x := v.(type) {
	case *JWTClaims:
		if x != nil && x.UserID != "" {
			return x.UserID
		}
	case JWTClaims:
		if x.UserID != "" {
			return x.UserID
		}
	}
	return ""
}

func (c *GinRequestContext) getUserIDFromJWTClaimsMap(key string) string {
	claimsInterface := c.Get(key)
	if claimsInterface == nil && c.GinCtx != nil {
		if v, ok := c.GinCtx.Get(key); ok {
			claimsInterface = v
		}
	}
	if claimsInterface == nil {
		return ""
	}
	if claimsMap, ok := claimsInterface.(map[string]interface{}); ok {
		for _, k := range []string{"user_id", "sub", "id"} {
			if userID, ok := claimsMap[k].(string); ok && userID != "" {
				return userID
			}
		}
	}
	return ""
}

// ResourceID returns a generic resource identifier from common path param names (domain routing hint).
// Prefer setting ContextKeyFloxID or X-Flox-ID when that id should drive EventBus / Flox routing.
func (c *GinRequestContext) ResourceID() string {
	paramNames := []string{"streamId", "orderId", "aggregateId", "entityId", "id"}
	for _, name := range paramNames {
		if val := c.Param(name); val != "" {
			return val
		}
	}
	return ""
}

// FloxID returns the Fluxor routing id: header X-Flox-ID, explicit ContextKeyFloxID, then GoCMD context.
// It does not infer from path params; use ResourceID or middleware that sets ContextKeyFloxID.
func (c *GinRequestContext) FloxID() string {
	if c.GinCtx != nil && c.GinCtx.Request != nil {
		if floxid := c.GinCtx.GetHeader("X-Flox-ID"); floxid != "" {
			return floxid
		}
	}
	if v := c.Get(ContextKeyFloxID); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	if c.GinCtx != nil {
		if v, ok := c.GinCtx.Get(ContextKeyFloxID); ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	if c.GoCMD != nil {
		if floxid := core.GetFloxID(c.GoCMD.Context()); floxid != "" {
			return floxid
		}
	}
	return ""
}

// Context returns the request's context (deadline/cancel/trace propagation) with RequestID and FloxID values.
func (c *GinRequestContext) Context() context.Context {
	if c.GinCtx == nil || c.GinCtx.Request == nil {
		return context.Background()
	}

	ctx := c.GinCtx.Request.Context()

	if c.requestID != "" {
		ctx = core.WithRequestID(ctx, c.requestID)
	}
	if floxid := c.FloxID(); floxid != "" {
		ctx = core.WithFloxID(ctx, floxid)
	}

	return ctx
}

// GetRoutingHeaders returns routing headers for EventBus (X-Flox-ID, X-User-ID, etc.).
func (c *GinRequestContext) GetRoutingHeaders() map[string]string {
	headers := make(map[string]string)
	if floxid := c.FloxID(); floxid != "" {
		headers["X-Flox-ID"] = floxid
	}
	if userID := c.UserID(); userID != "" {
		headers["X-User-ID"] = userID
	}
	if sessionID, ok := c.Get("session_id").(string); ok && sessionID != "" {
		headers["X-Session-ID"] = sessionID
	} else if c.GinCtx != nil && c.GinCtx.Request != nil {
		if s := c.GinCtx.GetHeader("X-Session-ID"); s != "" {
			headers["X-Session-ID"] = s
		}
	}
	if c.requestID != "" {
		headers["X-Request-ID"] = c.requestID
	}
	if c.GinCtx != nil && c.GinCtx.Request != nil {
		if rk := c.GinCtx.GetHeader("X-Route-Key"); rk != "" {
			headers["X-Route-Key"] = rk
		}
	}
	return headers
}

// PublishWithRouting publishes to EventBus with context routing (FloxID, etc.) and a default deadline.
func (c *GinRequestContext) PublishWithRouting(address string, body interface{}) error {
	if c.EventBus == nil {
		return fmt.Errorf("EventBus is not available")
	}
	ctx, cancel := context.WithTimeout(c.Context(), DefaultEventBusPublishTimeout)
	defer cancel()
	return c.EventBus.PublishWithContext(ctx, address, body)
}

// SendWithRouting sends request-reply to EventBus with context routing and a deadline aligned with timeout.
func (c *GinRequestContext) SendWithRouting(address string, body interface{}, timeout time.Duration) (core.Message, error) {
	if c.EventBus == nil {
		return nil, fmt.Errorf("EventBus is not available")
	}
	if timeout <= 0 {
		timeout = DefaultEventBusPublishTimeout
	}
	ctx, cancel := context.WithTimeout(c.Context(), timeout)
	defer cancel()
	return c.EventBus.SendWithContext(ctx, address, body, timeout)
}
