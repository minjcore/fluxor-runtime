package core

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
	"github.com/gorilla/websocket"
)

// WebSocketEventBusBridge bridges WebSocket connections to EventBus
type WebSocketEventBusBridge struct {
	eventBus EventBus
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]*wsClient
	mu       sync.RWMutex
	logger   Logger
}

// wsClient represents a WebSocket client connection
type wsClient struct {
	conn           *websocket.Conn
	bridge         *WebSocketEventBusBridge
	subscriptions  map[string]Consumer // address -> consumer
	mu             sync.RWMutex
	requestID      int64
	requestMu      sync.Mutex
	pendingReplies map[string]chan *wsMessage // requestID -> reply channel
	replyMu        sync.Mutex
}

// wsMessage represents a WebSocket message
type wsMessage struct {
	Op      string            `json:"op"`      // publish, send, request, subscribe, unsubscribe
	Address string            `json:"address"` // EventBus address
	Body    interface{}       `json:"body"`    // Message body
	ID      string            `json:"id"`      // Request ID for request/reply
	Timeout int64             `json:"timeout"` // Timeout in milliseconds
	Error   string            `json:"error,omitempty"`
	Result  interface{}       `json:"result,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// NewWebSocketEventBusBridge creates a new WebSocket bridge
func NewWebSocketEventBusBridge(eventBus EventBus) *WebSocketEventBusBridge {
	// Fail-fast: eventBus cannot be nil
	failfast.NotNil(eventBus, "eventBus")
	return &WebSocketEventBusBridge{
		eventBus: eventBus,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		clients: make(map[*websocket.Conn]*wsClient),
		logger:  NewDefaultLogger(),
	}
}

// HandleWebSocket handles WebSocket upgrade and connection
func (b *WebSocketEventBusBridge) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := b.upgrader.Upgrade(w, r, nil)
	if err != nil {
		b.logger.Error("WebSocket upgrade failed", "error", err)
		return
	}

	client := &wsClient{
		conn:           conn,
		bridge:         b,
		subscriptions:  make(map[string]Consumer),
		pendingReplies: make(map[string]chan *wsMessage),
	}

	b.mu.Lock()
	b.clients[conn] = client
	b.mu.Unlock()

	// Handle client messages
	go client.handleMessages()
}

// removeClient removes a client and cleans up subscriptions
func (b *WebSocketEventBusBridge) removeClient(conn *websocket.Conn) {
	b.mu.Lock()
	client, ok := b.clients[conn]
	delete(b.clients, conn)
	b.mu.Unlock()

	if ok {
		client.cleanup()
		conn.Close()
	}
}

// handleMessages handles incoming WebSocket messages
func (c *wsClient) handleMessages() {
	defer func() {
		c.cleanup()
		c.bridge.removeClient(c.conn)
	}()

	for {
		var msg wsMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.bridge.logger.Error("WebSocket read error", "error", err)
			}
			break
		}

		// Handle message based on operation
		switch msg.Op {
		case "publish":
			c.handlePublish(&msg)
		case "send":
			c.handleSend(&msg)
		case "request":
			c.handleRequest(&msg)
		case "subscribe":
			c.handleSubscribe(&msg)
		case "unsubscribe":
			c.handleUnsubscribe(&msg)
		default:
			c.sendError(&msg, fmt.Sprintf("unknown operation: %s", msg.Op))
		}
	}
}

// handlePublish handles publish operation
func (c *wsClient) handlePublish(msg *wsMessage) {
	// Fail-fast: validate address
	if err := ValidateAddress(msg.Address); err != nil {
		c.sendError(msg, err.Error())
		return
	}
	if err := c.bridge.eventBus.Publish(msg.Address, msg.Body); err != nil {
		c.sendError(msg, err.Error())
		return
	}
	c.sendResult(msg, map[string]interface{}{"status": "ok"})
}

// handleSend handles send operation
func (c *wsClient) handleSend(msg *wsMessage) {
	// Fail-fast: validate address
	if err := ValidateAddress(msg.Address); err != nil {
		c.sendError(msg, err.Error())
		return
	}
	if err := c.bridge.eventBus.Send(msg.Address, msg.Body); err != nil {
		c.sendError(msg, err.Error())
		return
	}
	c.sendResult(msg, map[string]interface{}{"status": "ok"})
}

// handleRequest handles request operation
func (c *wsClient) handleRequest(msg *wsMessage) {
	// Fail-fast: validate address
	if err := ValidateAddress(msg.Address); err != nil {
		c.sendError(msg, err.Error())
		return
	}
	timeout := 5 * time.Second
	if msg.Timeout > 0 {
		timeout = time.Duration(msg.Timeout) * time.Millisecond
	}

	reply, err := c.bridge.eventBus.Request(msg.Address, msg.Body, timeout)
	if err != nil {
		c.sendError(msg, err.Error())
		return
	}

	var replyBody interface{}
	if err := reply.DecodeBody(&replyBody); err != nil {
		c.sendError(msg, fmt.Sprintf("failed to decode reply: %v", err))
		return
	}

	c.sendResult(msg, replyBody)
}

// handleSubscribe handles subscribe operation
func (c *wsClient) handleSubscribe(msg *wsMessage) {
	// Fail-fast: validate address
	if err := ValidateAddress(msg.Address); err != nil {
		c.sendError(msg, err.Error())
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.subscriptions[msg.Address]; exists {
		c.sendError(msg, "already subscribed")
		return
	}

	consumer := c.bridge.eventBus.Consumer(msg.Address)
	consumer.Handler(func(ctx FluxorContext, eventMsg Message) error {
		var body interface{}
		if err := eventMsg.DecodeBody(&body); err != nil {
			return err
		}

		// Send message to WebSocket client
		wsMsg := &wsMessage{
			Op:      "message",
			Address: msg.Address,
			Body:    body,
			Headers: eventMsg.Headers(),
		}

		c.mu.RLock()
		err := c.conn.WriteJSON(wsMsg)
		c.mu.RUnlock()

		if err != nil {
			c.bridge.logger.Error("failed to send message to client", "error", err)
		}

		return nil
	})

	c.subscriptions[msg.Address] = consumer
	c.sendResult(msg, map[string]interface{}{"status": "subscribed", "address": msg.Address})
}

// handleUnsubscribe handles unsubscribe operation
func (c *wsClient) handleUnsubscribe(msg *wsMessage) {
	// Fail-fast: validate address
	if err := ValidateAddress(msg.Address); err != nil {
		c.sendError(msg, err.Error())
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	consumer, exists := c.subscriptions[msg.Address]
	if !exists {
		c.sendError(msg, "not subscribed")
		return
	}

	if err := consumer.Unregister(); err != nil {
		c.sendError(msg, err.Error())
		return
	}

	delete(c.subscriptions, msg.Address)
	c.sendResult(msg, map[string]interface{}{"status": "unsubscribed", "address": msg.Address})
}

// sendError sends an error response
func (c *wsClient) sendError(msg *wsMessage, errorMsg string) {
	response := &wsMessage{
		Op:    msg.Op,
		ID:    msg.ID,
		Error: errorMsg,
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	c.conn.WriteJSON(response)
}

// sendResult sends a success response
func (c *wsClient) sendResult(msg *wsMessage, result interface{}) {
	response := &wsMessage{
		Op:     msg.Op,
		ID:     msg.ID,
		Result: result,
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	c.conn.WriteJSON(response)
}

// cleanup cleans up client resources
func (c *wsClient) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Unregister all subscriptions
	for address, consumer := range c.subscriptions {
		consumer.Unregister()
		delete(c.subscriptions, address)
	}
}
