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

func setupTestConnection(t *testing.T) (*Connection, func()) {
	t.Helper()

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)

	var wsConn *Connection
	var closeFunc func()

	// Create a test HTTP server with WebSocket handler
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		wsConn = newConnection(conn, "test-conn", func(c *Connection) {
			// onClose callback
		}, nil)
	}))
	defer ts.Close()

	// Connect to WebSocket
	wsURL := "ws" + ts.URL[4:]
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	closeFunc = func() {
		conn.Close()
		gocmd.Close()
		ts.Close()
	}

	// Wait a bit for connection to be established
	time.Sleep(50 * time.Millisecond)

	if wsConn == nil {
		closeFunc()
		t.Fatal("Connection was not created")
	}

	return wsConn, closeFunc
}

func TestConnection_ID(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	id := conn.ID()
	if id == "" {
		t.Error("ID() should not return empty string")
	}
}

func TestConnection_IsClosed(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	if conn.IsClosed() {
		t.Error("New connection should not be closed")
	}

	conn.Close()
	time.Sleep(10 * time.Millisecond)

	if !conn.IsClosed() {
		t.Error("Connection should be closed after Close()")
	}
}

func TestConnection_Close(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	if err := conn.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}

	// Closing twice should not panic
	if err := conn.Close(); err != nil {
		t.Logf("Close() second time returned error: %v", err)
	}
}

func TestConnection_WriteMessage_AfterClose(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	conn.Close()
	time.Sleep(10 * time.Millisecond)

	err := conn.WriteMessage(websocket.TextMessage, []byte("test"))
	if err == nil {
		t.Error("WriteMessage() after Close() should return error")
	}
	if err != ErrConnectionClosed {
		t.Errorf("WriteMessage() error = %v, want ErrConnectionClosed", err)
	}
}

func TestConnection_WriteJSON_AfterClose(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	conn.Close()
	time.Sleep(10 * time.Millisecond)

	err := conn.WriteJSON(map[string]string{"test": "value"})
	if err == nil {
		t.Error("WriteJSON() after Close() should return error")
	}
	if err != ErrConnectionClosed {
		t.Errorf("WriteJSON() error = %v, want ErrConnectionClosed", err)
	}
}

func TestConnection_ReadMessage_AfterClose(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	conn.Close()
	time.Sleep(10 * time.Millisecond)

	_, _, err := conn.ReadMessage()
	if err == nil {
		t.Error("ReadMessage() after Close() should return error")
	}
	if err != ErrConnectionClosed {
		t.Errorf("ReadMessage() error = %v, want ErrConnectionClosed", err)
	}
}

func TestConnection_Context(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	ctx := conn.Context()
	if ctx == nil {
		t.Error("Context() should not return nil")
	}

	// Context should be cancelled when connection is closed
	conn.Close()
	time.Sleep(10 * time.Millisecond)

	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled after Close()")
	}
}

func TestConnection_SetReadDeadline(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	deadline := time.Now().Add(1 * time.Second)
	err := conn.SetReadDeadline(deadline)
	if err != nil {
		t.Errorf("SetReadDeadline() error: %v", err)
	}
}

func TestConnection_SetWriteDeadline(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	deadline := time.Now().Add(1 * time.Second)
	err := conn.SetWriteDeadline(deadline)
	if err != nil {
		t.Errorf("SetWriteDeadline() error: %v", err)
	}
}

func TestConnection_SetReadDeadline_AfterClose(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	conn.Close()
	time.Sleep(10 * time.Millisecond)

	deadline := time.Now().Add(1 * time.Second)
	err := conn.SetReadDeadline(deadline)
	if err == nil {
		t.Error("SetReadDeadline() after Close() should return error")
	}
}

func TestConnection_SetWriteDeadline_AfterClose(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	conn.Close()
	time.Sleep(10 * time.Millisecond)

	deadline := time.Now().Add(1 * time.Second)
	err := conn.SetWriteDeadline(deadline)
	if err == nil {
		t.Error("SetWriteDeadline() after Close() should return error")
	}
}

func TestConnection_SetPongHandler(t *testing.T) {
	conn, cleanup := setupTestConnection(t)
	defer cleanup()

	called := false
	conn.SetPongHandler(func(string) error {
		called = true
		return nil
	})

	// Setting pong handler should not panic
	if called {
		t.Error("Pong handler should not be called immediately")
	}

	cleanup()
}
