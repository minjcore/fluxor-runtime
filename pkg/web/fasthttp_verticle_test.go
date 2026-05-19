package web

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewFastHTTPVerticle(t *testing.T) {
	config := DefaultFastHTTPServerConfig(":8080")
	verticle := NewFastHTTPVerticle(config)

	if verticle == nil {
		t.Fatal("NewFastHTTPVerticle() should not return nil")
	}

	if verticle.config != config {
		t.Error("Config should be set correctly")
	}

	if verticle.Name() != "fasthttp-verticle" {
		t.Errorf("Expected name 'fasthttp-verticle', got '%s'", verticle.Name())
	}
}

func TestNewFastHTTPVerticle_NilConfig(t *testing.T) {
	verticle := NewFastHTTPVerticle(nil)

	if verticle == nil {
		t.Fatal("NewFastHTTPVerticle() should not return nil")
	}

	if verticle.config != nil {
		t.Error("Config should be nil when not provided")
	}
}

func TestFastHTTPVerticle_doStart(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	verticle := NewFastHTTPVerticle(config)
	fluxorCtx := newTestFluxorContext(gocmd)

	err := verticle.doStart(fluxorCtx)
	if err != nil {
		t.Fatalf("doStart() returned error: %v", err)
	}

	// Verify server was created
	if verticle.server == nil {
		t.Error("Server should be created on doStart()")
	}
}

func TestFastHTTPVerticle_doStart_NilConfig(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	verticle := NewFastHTTPVerticle(nil)
	fluxorCtx := newTestFluxorContext(gocmd)

	err := verticle.doStart(fluxorCtx)
	if err != nil {
		t.Fatalf("doStart() returned error: %v", err)
	}

	// Should create default config
	if verticle.config == nil {
		t.Error("Config should be created with default values")
	}

	if verticle.config.Addr != ":8080" {
		t.Errorf("Expected default address ':8080', got '%s'", verticle.config.Addr)
	}
}

func TestFastHTTPVerticle_doStop(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	verticle := NewFastHTTPVerticle(config)
	fluxorCtx := newTestFluxorContext(gocmd)

	// Start first
	err := verticle.doStart(fluxorCtx)
	if err != nil {
		t.Fatalf("doStart() returned error: %v", err)
	}

	// Stop
	err = verticle.doStop(fluxorCtx)
	if err != nil {
		t.Errorf("doStop() returned error: %v", err)
	}
}

func TestFastHTTPVerticle_doStop_WithoutStart(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":8080")
	verticle := NewFastHTTPVerticle(config)
	fluxorCtx := newTestFluxorContext(gocmd)

	// Stop without starting should not panic
	err := verticle.doStop(fluxorCtx)
	if err != nil {
		t.Logf("doStop() without start returned: %v (may be expected)", err)
	}
}

func TestFastHTTPVerticle_AsyncStart(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	verticle := NewFastHTTPVerticle(config)
	fluxorCtx := newTestFluxorContext(gocmd)

	// Start first
	err := verticle.doStart(fluxorCtx)
	if err != nil {
		t.Fatalf("doStart() returned error: %v", err)
	}

	// AsyncStart
	resultChan := make(chan error, 1)
	verticle.AsyncStart(fluxorCtx, func(err error) {
		resultChan <- err
	})

	// Wait for result (timeout protection: fail after 1s if callback not invoked)
	select {
	case err := <-resultChan:
		if err != nil {
			t.Errorf("AsyncStart() returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("AsyncStart() did not call resultHandler")
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	verticle.AsyncStop(fluxorCtx, func(err error) {
		if err != nil {
			t.Logf("AsyncStop() returned error: %v", err)
		}
	})
}

func TestFastHTTPVerticle_AsyncStop(t *testing.T) {
	RunWithTimeout(t, 5*time.Second, func() {
		ctx := context.Background()
		gocmd := core.NewGoCMD(ctx)
		defer gocmd.Close()

		config := DefaultFastHTTPServerConfig(":0")
		verticle := NewFastHTTPVerticle(config)
		fluxorCtx := newTestFluxorContext(gocmd)

		// Start first
		err := verticle.doStart(fluxorCtx)
		if err != nil {
			t.Fatalf("doStart() returned error: %v", err)
		}

		// AsyncStop
		resultChan := make(chan error, 1)
		verticle.AsyncStop(fluxorCtx, func(err error) {
			resultChan <- err
		})

		// Wait for result
		select {
		case err := <-resultChan:
			if err != nil {
				t.Logf("AsyncStop() returned error: %v (may be expected if server not started)", err)
			}
		case <-time.After(1 * time.Second):
			t.Error("AsyncStop() did not call resultHandler")
		}
	})
}

func TestFastHTTPVerticle_Server(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	verticle := NewFastHTTPVerticle(config)
	fluxorCtx := newTestFluxorContext(gocmd)

	// Before start, server should be nil
	if verticle.Server() != nil {
		t.Error("Server() should return nil before doStart()")
	}

	// Start
	err := verticle.doStart(fluxorCtx)
	if err != nil {
		t.Fatalf("doStart() returned error: %v", err)
	}

	// After start, server should be available
	server := verticle.Server()
	if server == nil {
		t.Error("Server() should not return nil after doStart()")
	}
}

func TestFastHTTPVerticle_FastRouter(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultFastHTTPServerConfig(":0")
	verticle := NewFastHTTPVerticle(config)
	fluxorCtx := newTestFluxorContext(gocmd)

	// Before start, router should be nil
	if verticle.FastRouter() != nil {
		t.Error("FastRouter() should return nil before doStart()")
	}

	// Start
	err := verticle.doStart(fluxorCtx)
	if err != nil {
		t.Fatalf("doStart() returned error: %v", err)
	}

	// After start, router should be available
	router := verticle.FastRouter()
	if router == nil {
		t.Error("FastRouter() should not return nil after doStart()")
	}

	// Test that we can register routes
	router.GETFast("/test", func(ctx *FastRequestContext) error {
		return ctx.JSON(200, map[string]string{"message": "test"})
	})
}

func TestFastHTTPVerticle_BaseVerticle(t *testing.T) {
	config := DefaultFastHTTPServerConfig(":8080")
	verticle := NewFastHTTPVerticle(config)

	if verticle.BaseVerticle == nil {
		t.Error("BaseVerticle should not be nil")
	}
}
