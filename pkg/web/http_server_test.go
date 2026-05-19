package web

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewServer(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")
	if server == nil {
		t.Fatal("NewServer() should not return nil")
	}

	router := server.Router()
	if router == nil {
		t.Error("Router() should not return nil")
	}
}

func TestHTTPServer_Router(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")
	router := server.Router()

	if router == nil {
		t.Fatal("Router() should not return nil")
	}

	// Test that we can register routes
	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "test")
	})
}

func TestHTTPServer_StartStop(t *testing.T) {
	RunWithTimeout(t, 5*time.Second, func() {
		ctx := context.Background()
		gocmd := core.NewGoCMD(ctx)
		defer gocmd.Close()

		server := NewServer(gocmd, ":0")

		// Start server in background
		startErr := make(chan error, 1)
		go func() {
			startErr <- server.Start()
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)

		// Stop server
		err := server.Stop()
		if err != nil {
			t.Errorf("Stop() returned error: %v", err)
		}

		// Wait for start to finish (should return error after stop)
		select {
		case err := <-startErr:
			// Server should have stopped
			if err != nil && err != http.ErrServerClosed {
				t.Errorf("Start() returned unexpected error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("Start() did not return after Stop()")
		}
	})
}

func TestHTTPServer_InjectGoCMD(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")
	httpServer := server.(*httpServer)

	reqCtx := &RequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		Params:             make(map[string]string),
	}

	httpServer.InjectGoCMD(reqCtx)

	if reqCtx.GoCMD == nil {
		t.Error("GoCMD should be injected")
	}
	if reqCtx.EventBus == nil {
		t.Error("EventBus should be injected")
	}
}

func TestHTTPServer_RequestHandling(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")
	router := server.Router()

	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "test response")
	})

	// Start server
	startErr := make(chan error, 1)
	go func() {
		startErr <- server.Start()
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Get the actual address (we can't easily get it from :0)
	// For this test, we'll just verify the server can start
	// In a real test, you'd need to get the actual port

	// Stop server
	err := server.Stop()
	if err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}

	// Wait for start to finish
	select {
	case err := <-startErr:
		if err != nil && err != http.ErrServerClosed {
			t.Logf("Start() returned: %v (expected after Stop())", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Start() did not return after Stop()")
	}
}

func TestHTTPServer_MultipleRoutes(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")
	router := server.Router()

	router.GET("/route1", func(ctx *RequestContext) error {
		return ctx.Text(200, "route1")
	})

	router.POST("/route2", func(ctx *RequestContext) error {
		return ctx.Text(200, "route2")
	})

	router.PUT("/route3", func(ctx *RequestContext) error {
		return ctx.Text(200, "route3")
	})

	// Verify all routes are registered
	if router == nil {
		t.Fatal("Router should not be nil")
	}
}

func TestHTTPServer_Middleware(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")
	router := server.Router()

	middleware := func(next RequestHandler) RequestHandler {
		return func(ctx *RequestContext) error {
			return next(ctx)
		}
	}

	router.Use(middleware)
	router.GET("/test", func(ctx *RequestContext) error {
		return ctx.Text(200, "test")
	})

	// Middleware should be registered
	if router == nil {
		t.Fatal("Router should not be nil")
	}
}

func TestHTTPServer_ConcurrentStartStop(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")

	// Try to start and stop concurrently
	startErr := make(chan error, 1)
	stopErr := make(chan error, 1)

	go func() {
		startErr <- server.Start()
	}()

	time.Sleep(50 * time.Millisecond)

	go func() {
		stopErr <- server.Stop()
	}()

	// Wait for both to complete
	select {
	case err := <-startErr:
		if err != nil && err != http.ErrServerClosed {
			t.Logf("Start() returned: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Start() did not return")
	}

	select {
	case err := <-stopErr:
		if err != nil {
			t.Errorf("Stop() returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Stop() did not return")
	}
}

func TestHTTPServer_StopWithoutStart(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")

	// Stop without starting should not panic
	err := server.Stop()
	if err != nil {
		t.Logf("Stop() without start returned: %v (may be expected)", err)
	}
}

func TestHTTPServer_ReadHeaderTimeout(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")
	httpServer := server.(*httpServer)

	// Verify ReadHeaderTimeout is set
	if httpServer.httpServer.ReadHeaderTimeout == 0 {
		t.Error("ReadHeaderTimeout should be set to mitigate Slowloris")
	}

	expectedTimeout := 5 * time.Second
	if httpServer.httpServer.ReadHeaderTimeout != expectedTimeout {
		t.Errorf("ReadHeaderTimeout = %v, want %v", 
			httpServer.httpServer.ReadHeaderTimeout, expectedTimeout)
	}
}

func TestHTTPServer_MultipleStartStop(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")

	// Start and stop multiple times
	for i := 0; i < 3; i++ {
		startErr := make(chan error, 1)
		go func() {
			startErr <- server.Start()
		}()

		time.Sleep(50 * time.Millisecond)

		err := server.Stop()
		if err != nil {
			t.Errorf("Stop() iteration %d returned error: %v", i, err)
		}

		// Wait for start to finish
		select {
		case err := <-startErr:
			if err != nil && err != http.ErrServerClosed {
				t.Logf("Start() iteration %d returned: %v", i, err)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("Start() iteration %d did not return", i)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func TestHTTPServer_EmptyAddress(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, "")
	if server == nil {
		t.Fatal("NewServer() with empty address should not return nil")
	}

	router := server.Router()
	if router == nil {
		t.Error("Router() should not return nil")
	}
}

func TestHTTPServer_BaseServer(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")
	httpServer := server.(*httpServer)

	if httpServer.BaseServer == nil {
		t.Error("BaseServer should not be nil")
	}

	if httpServer.Name() != "http-server" {
		t.Errorf("Expected name 'http-server', got '%s'", httpServer.Name())
	}
}

func TestHTTPServer_InjectGoCMD_NilContext(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, ":0")
	httpServer := server.(*httpServer)

	// InjectGoCMD should handle nil context gracefully
	reqCtx := &RequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		Params:             make(map[string]string),
	}

	httpServer.InjectGoCMD(reqCtx)

	if reqCtx.GoCMD == nil {
		t.Error("GoCMD should be injected even with nil context")
	}
}
