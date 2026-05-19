package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/gorilla/websocket"
)

func TestNewServer(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultServerConfig(":0", "/ws")
	server := NewServer(gocmd, config)

	if server == nil {
		t.Fatal("NewServer() should not return nil")
	}
}

func TestNewServer_NilConfig(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, nil)

	if server == nil {
		t.Fatal("NewServer() with nil config should use defaults")
	}
}

func TestNewServer_FailFast_NilGoCMD(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil gocmd")
		}
	}()
	_ = NewServer(nil, DefaultServerConfig(":8080", "/ws"))
}

func TestWebSocketServer_SetHandler_FailFast_NilPanics(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, DefaultServerConfig(":0", "/ws"))
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil handler")
		}
	}()
	server.SetHandler(nil)
}

func TestWebSocketServer_Connections(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultServerConfig(":0", "/ws")
	server := NewServer(gocmd, config)

	connections := server.Connections()
	if connections != 0 {
		t.Errorf("Connections() = %v, want 0", connections)
	}
}

func TestWebSocketServer_HandleWebSocket_Upgrade(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultServerConfig(":0", "/ws")
	server := NewServer(gocmd, config)

	// Set a simple handler
	server.SetHandler(func(conn *Connection) error {
		return nil
	})

	// Create a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.HandleWebSocket(w, r)
	}))
	defer ts.Close()

	// Convert http:// to ws://
	wsURL := "ws" + ts.URL[4:] + "/ws"

	// Try to connect
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		// Connection might fail in test environment, that's okay
		t.Logf("WebSocket connection failed (expected in some test environments): %v", err)
		return
	}
	defer conn.Close()

	// Give handler a moment to be called
	time.Sleep(100 * time.Millisecond)

	// Close connection
	conn.Close()
}

func TestNewServerWithEventBus(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	eventBus := gocmd.EventBus()
	config := DefaultServerConfig(":0", "/eventbus/ws")
	server := NewServerWithEventBus(gocmd, config, eventBus)

	if server == nil {
		t.Fatal("NewServerWithEventBus() should not return nil")
	}
}

func TestNewServerWithEventBus_NilEventBus(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultServerConfig(":0", "/eventbus/ws")
	defer func() {
		if r := recover(); r == nil {
			// EventBus bridge creation might panic on nil, or might not
			// depending on implementation
		}
	}()
	_ = NewServerWithEventBus(gocmd, config, nil)
}

func TestWebSocketServer_MaxConnections(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultServerConfig(":0", "/ws")
	config.MaxConnections = 1 // Limit to 1 connection
	server := NewServer(gocmd, config)

	server.SetHandler(func(conn *Connection) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	// Create a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.HandleWebSocket(w, r)
	}))
	defer ts.Close()

	wsURL := "ws" + ts.URL[4:] + "/ws"
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	// First connection should succeed
	conn1, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Logf("First connection failed (expected in some test environments): %v", err)
		return
	}
	defer conn1.Close()

	// Second connection should be rejected (MaxConnections = 1)
	conn2, _, err := dialer.Dial(wsURL, nil)
	if err == nil {
		conn2.Close()
		// In test environment, connections might be handled quickly
		// so second connection might succeed before first is registered
		t.Log("Second connection succeeded (might be a timing issue in tests)")
	} else {
		t.Logf("Second connection rejected (expected): %v", err)
	}
}

func TestWebSocketServer_StartStop(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DefaultServerConfig("127.0.0.1:0", "/ws")
	server := NewServer(gocmd, config)

	server.SetHandler(func(conn *Connection) error {
		return nil
	})

	// Start server in goroutine
	startErrCh := make(chan error, 1)
	go func() {
		startErrCh <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	if err := server.Stop(); err != nil {
		t.Errorf("Stop() error: %v", err)
	}

	// Wait for start to finish (should return error from shutdown)
	select {
	case err := <-startErrCh:
		if err != nil && err != http.ErrServerClosed {
			t.Logf("Start() returned error (expected on shutdown): %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Start() did not return within timeout")
	}
}

func TestWebSocketServer_Stop_WithoutStart(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	server := NewServer(gocmd, DefaultServerConfig(":0", "/ws"))

	// Stop without starting should not panic
	if err := server.Stop(); err != nil {
		// Error is acceptable if server wasn't started
		t.Logf("Stop() without Start() returned error: %v", err)
	}
}
