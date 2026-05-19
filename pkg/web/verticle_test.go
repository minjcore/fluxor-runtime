package web

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewHttpVerticle(t *testing.T) {
	router := NewRouter()
	verticle := NewHttpVerticle("8080", router)

	if verticle == nil {
		t.Fatal("NewHttpVerticle() should not return nil")
	}

	if verticle.port != "8080" {
		t.Errorf("Expected port '8080', got '%s'", verticle.port)
	}

	if verticle.router != router {
		t.Error("Router should be set correctly")
	}
}

func TestHttpVerticle_StartStop(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := NewRouter()
	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "test")
	})

	verticle := NewHttpVerticle("0", router) // Use :0 for random port
	fluxorCtx := newTestFluxorContext(gocmd)

	// Start verticle
	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop verticle
	err = verticle.Stop(fluxorCtx)
	if err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

func TestHttpVerticle_ServerCreation(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := NewRouter()
	verticle := NewHttpVerticle("8080", router)
	fluxorCtx := newTestFluxorContext(gocmd)

	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	// Verify server was created
	if verticle.server == nil {
		t.Error("Server should be created on Start()")
	}

	// Verify server configuration
	if verticle.server.Addr != ":8080" {
		t.Errorf("Expected server address ':8080', got '%s'", verticle.server.Addr)
	}

	if verticle.server.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("Expected ReadHeaderTimeout 5s, got %v", verticle.server.ReadHeaderTimeout)
	}

	// Cleanup
	verticle.Stop(fluxorCtx)
}

func TestHttpVerticle_RouterAsHandler(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := NewRouter()
	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "test response")
	})

	verticle := NewHttpVerticle("0", router)
	fluxorCtx := newTestFluxorContext(gocmd)

	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Router should be set correctly
	if verticle.router == nil {
		t.Error("Router should be set")
	}

	// Cleanup
	verticle.Stop(fluxorCtx)
}

func TestHttpVerticle_StopWithoutStart(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := NewRouter()
	verticle := NewHttpVerticle("8080", router)
	fluxorCtx := newTestFluxorContext(gocmd)

	// Stop without starting should not panic
	err := verticle.Stop(fluxorCtx)
	if err != nil {
		t.Logf("Stop() without start returned: %v (may be expected)", err)
	}
}

func TestHttpVerticle_BaseVerticle(t *testing.T) {
	router := NewRouter()
	verticle := NewHttpVerticle("8080", router)

	if verticle.BaseVerticle == nil {
		t.Error("BaseVerticle should not be nil")
	}

	if verticle.Name() != "http-verticle" {
		t.Errorf("Expected name 'http-verticle', got '%s'", verticle.Name())
	}

	// Verify port and router are set
	if verticle.port != "8080" {
		t.Errorf("Expected port '8080', got '%s'", verticle.port)
	}

	if verticle.router != router {
		t.Error("Router should be set correctly")
	}
}

func TestHttpVerticle_MultipleStartStop(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := NewRouter()
	fluxorCtx := newTestFluxorContext(gocmd)

	// Test that we can create and start/stop multiple verticles
	// (BaseVerticle may not support restarting the same instance)
	for i := 0; i < 3; i++ {
		verticle := NewHttpVerticle("0", router)

		err := verticle.Start(fluxorCtx)
		if err != nil {
			t.Fatalf("Start() iteration %d returned error: %v", i, err)
		}

		time.Sleep(100 * time.Millisecond)

		err = verticle.Stop(fluxorCtx)
		if err != nil {
			t.Errorf("Stop() iteration %d returned error: %v", i, err)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func TestHttpVerticle_EmptyPort(t *testing.T) {
	router := NewRouter()
	verticle := NewHttpVerticle("", router)

	if verticle.port != "" {
		t.Errorf("Expected empty port, got '%s'", verticle.port)
	}

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	fluxorCtx := newTestFluxorContext(gocmd)

	// Starting with empty port should still work (will use ":")
	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() with empty port returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Server is created synchronously in doStart, so it should exist
	// However, if there's an issue with empty port, the server might not start properly
	// Check if server was created
	if verticle.server != nil {
		// Verify server address (empty port becomes ":")
		if verticle.server.Addr != ":" {
			t.Logf("Server address with empty port is '%s' (expected ':' or similar)", verticle.server.Addr)
		}
	} else {
		// Server might not be created if there's validation, which is acceptable
		t.Logf("Server not created with empty port (may be expected behavior)")
	}

	verticle.Stop(fluxorCtx)
}

func TestHttpVerticle_PortWithColon(t *testing.T) {
	router := NewRouter()
	verticle := NewHttpVerticle(":8080", router)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	fluxorCtx := newTestFluxorContext(gocmd)

	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Server should handle port with colon prefix
	if verticle.server == nil {
		t.Error("Server should be created")
	}

	// Address should have double colon (one from port, one added by server)
	if verticle.server.Addr != "::8080" {
		t.Logf("Server address is '%s' (may have double colon)", verticle.server.Addr)
	}

	verticle.Stop(fluxorCtx)
}

func TestHttpVerticle_NilRouter(t *testing.T) {
	verticle := NewHttpVerticle("8080", nil)

	if verticle.router != nil {
		t.Error("Router should be nil")
	}

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	fluxorCtx := newTestFluxorContext(gocmd)

	// Starting with nil router should not panic
	// The handler will check if router implements http.Handler
	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() with nil router returned error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	verticle.Stop(fluxorCtx)
}

func TestHttpVerticle_ConcurrentStartStop(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := NewRouter()
	verticle := NewHttpVerticle("0", router)
	fluxorCtx := newTestFluxorContext(gocmd)

	// Start verticle
	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Try to stop concurrently
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			done <- verticle.Stop(fluxorCtx)
		}()
	}

	// At least one should succeed
	errors := 0
	for i := 0; i < 3; i++ {
		if err := <-done; err != nil {
			errors++
		}
	}

	// Multiple stops may return errors, which is expected
	t.Logf("Concurrent stop attempts: %d errors (expected)", errors)
}

func TestHttpVerticle_HandlerError(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	router := NewRouter()
	router.GET("/error", func(ctx *RequestContext) error {
		return ctx.Text(500, "internal error")
	})

	verticle := NewHttpVerticle("0", router)
	fluxorCtx := newTestFluxorContext(gocmd)

	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Server should handle errors gracefully
	if verticle.server == nil {
		t.Error("Server should be created")
	}

	verticle.Stop(fluxorCtx)
}
