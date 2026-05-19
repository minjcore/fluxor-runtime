package web

import (
	"net/http"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/gin-gonic/gin"
)

// fluxorScopeKey is the key used to store Scope in gin.Context (set by UnitOfWorkMiddleware).
const fluxorScopeKey = "fluxor_scope"

// UnitOfWorkMiddleware creates a per-request Scope (Start) and attaches it to the request.
// Register with Engine().Use(web.UnitOfWorkMiddleware()) so that handlers receive GinRequestContext.Scope.
// Stop is invoked from ginRouter.wrap (defer) after the Fluxor handler returns, so cleanup is unified for all Fluxor routes.
func UnitOfWorkMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		scope := core.NewRequestScope(c.Request.Context())
		if err := scope.Start(); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "khởi tạo thất bại"})
			return
		}
		c.Set(fluxorScopeKey, scope)
		c.Next()
	}
}
