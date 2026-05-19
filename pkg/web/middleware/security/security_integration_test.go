package security_test

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware/security"
)

func TestSecurityMiddleware(t *testing.T) {
	// Create a test server context
	gocmd := core.NewGoCMD(context.Background())
	defer gocmd.Close()
	config := web.CCUBasedConfigWithUtilization(":8080", 1000, 67)
	server := web.NewFastHTTPServer(gocmd, config)
	router := server.FastRouter()

	// Test that middleware functions are created (they return FastMiddleware)
	// Note: In actual usage, these would be added via router's internal middleware
	// For now, we just verify they can be created
	headersMw := security.Headers(security.DefaultHeadersConfig())
	corsMw := security.CORS(security.CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
	})
	rateLimitMw := security.RateLimit(security.RateLimitConfig{
		RequestsPerMinute: 10,
	})

	// Verify middleware functions exist and are callable
	if headersMw == nil || corsMw == nil || rateLimitMw == nil {
		t.Error("Middleware functions should not be nil")
	}

	router.GETFast("/test", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{"status": "ok"})
	})

	// Note: Full integration test would require actual HTTP server and client
	// This test verifies the middleware can be instantiated
	_ = router
	_ = headersMw
	_ = corsMw
	_ = rateLimitMw
}
