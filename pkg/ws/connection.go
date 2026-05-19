package ws

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ErrConnectionClosed is returned when attempting to use a closed connection
var ErrConnectionClosed = errors.New("websocket connection is closed")

// Connection represents a WebSocket connection
type Connection struct {
	conn      *websocket.Conn
	id        string
	mu        sync.RWMutex
	closed    bool
	onClose   func(*Connection)
	onMessage func(*Connection, []byte)
	ctx       context.Context
	cancel    context.CancelFunc
}

// newConnection creates a new connection wrapper
func newConnection(conn *websocket.Conn, id string, onClose func(*Connection), onMessage func(*Connection, []byte)) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		conn:      conn,
		id:        id,
		onClose:   onClose,
		onMessage: onMessage,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// ID returns the connection ID
func (c *Connection) ID() string {
	return c.id
}

// WriteMessage sends a text or binary message
func (c *Connection) WriteMessage(messageType int, data []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return ErrConnectionClosed
	}
	return c.conn.WriteMessage(messageType, data)
}

// WriteJSON sends a JSON message
func (c *Connection) WriteJSON(v interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return ErrConnectionClosed
	}
	return c.conn.WriteJSON(v)
}

// ReadMessage reads a message from the connection
func (c *Connection) ReadMessage() (messageType int, p []byte, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return 0, nil, ErrConnectionClosed
	}
	return c.conn.ReadMessage()
}

// Close closes the connection
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	c.cancel()
	if c.onClose != nil {
		go c.onClose(c) // Call onClose in goroutine to avoid deadlock
	}
	return c.conn.Close()
}

// IsClosed returns whether the connection is closed
func (c *Connection) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// Context returns the connection's context
func (c *Connection) Context() context.Context {
	return c.ctx
}

// SetReadDeadline sets the read deadline
func (c *Connection) SetReadDeadline(t time.Time) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return ErrConnectionClosed
	}
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline
func (c *Connection) SetWriteDeadline(t time.Time) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return ErrConnectionClosed
	}
	return c.conn.SetWriteDeadline(t)
}

// SetPongHandler sets the pong handler
func (c *Connection) SetPongHandler(h func(string) error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.closed {
		c.conn.SetPongHandler(h)
	}
}
