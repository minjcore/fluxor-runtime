// Gin adapters for pkg/core/kernel (v2 runtime). Gin is transport-only; kernel.Handler stays generic.

package web

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/kernel"
	"github.com/gin-gonic/gin"
)

type ginBridgeKey struct{}

// GinFromContext returns the *gin.Context stored by GinKernelHandler (for writing JSON, etc.).
func GinFromContext(ctx context.Context) (*gin.Context, bool) {
	v := ctx.Value(ginBridgeKey{})
	gc, ok := v.(*gin.Context)
	return gc, ok
}

// GinKernelHandler adapts a kernel.Handler to gin.HandlerFunc.
// It builds the same GinRequestContext wiring as ginRouter.wrap, then maps to kernel.AppContext
// (Request.Context + meta) so handlers share identity with EventBus / workers.
func GinKernelHandler(gocmd core.GoCMD, h kernel.Handler) gin.HandlerFunc {
	stack := kernel.Chain(kernel.Recovery())
	wrapped := stack(h)
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = core.GenerateRequestID()
		}
		c.Header("X-Request-ID", requestID)

		params := make(map[string]string)
		for _, p := range c.Params {
			params[p.Key] = p.Value
		}

		fluxorCtx := &GinRequestContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			GinCtx:             c,
			GoCMD:              gocmd,
			EventBus:           gocmd.EventBus(),
			Params:             params,
			requestID:          requestID,
		}
		if v, ok := c.Get(fluxorScopeKey); ok {
			if sc, ok := v.(core.Scope); ok {
				fluxorCtx.Scope = sc
			}
		}
		fluxorCtx.Set("real_ip", c.ClientIP())

		if fluxorCtx.Scope != nil {
			defer fluxorCtx.Scope.Stop()
		}

		base := context.WithValue(fluxorCtx.Context(), ginBridgeKey{}, c)
		appCtx := kernel.NewAppContext(base, kernel.Meta{
			RequestID: fluxorCtx.RequestID(),
			UserID:    fluxorCtx.UserID(),
			FloxID:    fluxorCtx.FloxID(),
		})

		if err := wrapped(appCtx); err != nil {
			status, code, msg := ginHandlerHTTPError(err)
			c.AbortWithStatusJSON(status, APIError{Code: code, Message: msg})
		}
	}
}
