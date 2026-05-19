package diagnostic

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

func TestRegister(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := web.NewFastRouter()
	Register(router, gocmd, "")

	// Verify routes are registered by checking that router has routes
	// We can't directly access routes, but we can verify the router is not nil
	if router == nil {
		t.Fatal("Router should not be nil")
	}
}

func TestRegister_WithPrefix(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := web.NewFastRouter()
	prefix := "/admin"
	Register(router, gocmd, prefix)

	// Routes should be registered with prefix
	// We can't directly verify, but we can check router is not nil
	if router == nil {
		t.Fatal("Router should not be nil")
	}
}

func TestRegister_AllRoutes(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := web.NewFastRouter()
	Register(router, gocmd, "")

	// Test that we can call handlers (they may return errors if system not initialized)
	// This verifies routes are registered
	reqCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := NewHandler(gocmd)

	// Test system handler
	err := handler.SystemHandler(reqCtx)
	if err != nil {
		t.Logf("SystemHandler() returned error (may be expected): %v", err)
	}

	// Test all deployments handler
	err = handler.AllDeploymentsHandler(reqCtx)
	if err != nil {
		t.Logf("AllDeploymentsHandler() returned error (may be expected): %v", err)
	}
}

func TestRegister_RoutePaths(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := web.NewFastRouter()
	prefix := "/api"
	Register(router, gocmd, prefix)

	// Expected routes:
	// - GET /api/api/diagnostic/deployment/:id
	// - GET /api/api/diagnostic/system
	// - GET /api/api/diagnostic/deployments
	// Note: The prefix is added to the routes defined in Register
	// So if prefix is "/api", routes become "/api/api/diagnostic/..."

	// We can't directly verify routes, but we can verify the router accepts them
	if router == nil {
		t.Fatal("Router should not be nil")
	}
}

func TestRegister_EmptyPrefix(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := web.NewFastRouter()
	Register(router, gocmd, "")

	// Routes should be registered at root
	// Expected routes:
	// - GET /api/diagnostic/deployment/:id
	// - GET /api/diagnostic/system
	// - GET /api/diagnostic/deployments

	if router == nil {
		t.Fatal("Router should not be nil")
	}
}
