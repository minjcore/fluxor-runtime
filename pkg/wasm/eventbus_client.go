package wasm

import (
	"fmt"
	"sync"
	"time"
)

// EventBusClient provides EventBus interface for WASM clients
type EventBusClient interface {
	Publish(address string, body interface{}) error
	Send(address string, body interface{}) error
	Request(address string, body interface{}, timeout time.Duration) (Message, error)
	Consumer(address string) Consumer
	Close() error
}

// Consumer represents a message consumer
type Consumer interface {
	Handler(handler MessageHandler) Consumer
	Unregister() error
}

// MessageHandler handles incoming messages
type MessageHandler func(msg Message) error

// wsEventBusClient implements EventBusClient using WebSocket
type wsEventBusClient struct {
	wsURL          string
	conn           interface{} // Will be *js.Value in WASM
	mu             sync.RWMutex
	connected      bool
	subscriptions  map[string]*wsConsumer
	subMu          sync.Mutex
	requestID      int64
	requestMu      sync.Mutex
	pendingReplies map[string]chan *WSMessage
	replyMu        sync.Mutex
	messageQueue   []*WSMessage
	queueMu        sync.Mutex
}

// wsConsumer implements Consumer
type wsConsumer struct {
	address string
	client  *wsEventBusClient
	handler MessageHandler
}

// NewEventBusClient creates a new EventBus client
// In WASM, this will connect via WebSocket
func NewEventBusClient(wsURL string) (EventBusClient, error) {
	client := &wsEventBusClient{
		wsURL:          wsURL,
		subscriptions:  make(map[string]*wsConsumer),
		pendingReplies: make(map[string]chan *WSMessage),
		messageQueue:   make([]*WSMessage, 0),
	}

	// Connection will be established in WASM via syscall/js
	// For now, we'll mark as not connected
	client.connected = false

	return client, nil
}

// Publish publishes a message
func (c *wsEventBusClient) Publish(address string, body interface{}) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	msg := &WSMessage{
		Op:      "publish",
		Address: address,
		Body:    body,
	}

	return c.sendMessage(msg)
}

// Send sends a point-to-point message
func (c *wsEventBusClient) Send(address string, body interface{}) error {
	if !c.connected {
		return fmt.Errorf("not connected")
	}

	msg := &WSMessage{
		Op:      "send",
		Address: address,
		Body:    body,
	}

	return c.sendMessage(msg)
}

// Request sends a request and waits for reply
func (c *wsEventBusClient) Request(address string, body interface{}, timeout time.Duration) (Message, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Generate request ID
	c.requestMu.Lock()
	c.requestID++
	requestID := fmt.Sprintf("req-%d", c.requestID)
	c.requestMu.Unlock()

	// Create reply channel
	replyChan := make(chan *WSMessage, 1)
	c.replyMu.Lock()
	c.pendingReplies[requestID] = replyChan
	c.replyMu.Unlock()

	// Send request
	msg := &WSMessage{
		Op:      "request",
		Address: address,
		Body:    body,
		ID:      requestID,
		Timeout: int64(timeout.Milliseconds()),
	}

	if err := c.sendMessage(msg); err != nil {
		c.replyMu.Lock()
		delete(c.pendingReplies, requestID)
		c.replyMu.Unlock()
		return nil, err
	}

	// Wait for reply
	select {
	case reply := <-replyChan:
		c.replyMu.Lock()
		delete(c.pendingReplies, requestID)
		c.replyMu.Unlock()

		if reply.Error != "" {
			return nil, fmt.Errorf("%s", reply.Error)
		}

		return &messageImpl{
			body:    reply.Result,
			headers: reply.Headers,
		}, nil
	case <-time.After(timeout):
		c.replyMu.Lock()
		delete(c.pendingReplies, requestID)
		c.replyMu.Unlock()
		return nil, fmt.Errorf("request timeout")
	}
}

// Consumer creates a consumer for the given address
func (c *wsEventBusClient) Consumer(address string) Consumer {
	return &wsConsumer{
		address: address,
		client:  c,
	}
}

// Handler sets the message handler
func (c *wsConsumer) Handler(handler MessageHandler) Consumer {
	c.handler = handler

	// Subscribe to address
	c.client.subMu.Lock()
	c.client.subscriptions[c.address] = c
	c.client.subMu.Unlock()

	// Send subscribe message
	msg := &WSMessage{
		Op:      "subscribe",
		Address: c.address,
	}

	c.client.sendMessage(msg)

	return c
}

// Unregister unregisters the consumer
func (c *wsConsumer) Unregister() error {
	c.client.subMu.Lock()
	delete(c.client.subscriptions, c.address)
	c.client.subMu.Unlock()

	// Send unsubscribe message
	msg := &WSMessage{
		Op:      "unsubscribe",
		Address: c.address,
	}

	return c.client.sendMessage(msg)
}

// Close closes the client
func (c *wsEventBusClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Unregister all subscriptions
	c.subMu.Lock()
	for address := range c.subscriptions {
		msg := &WSMessage{
			Op:      "unsubscribe",
			Address: address,
		}
		c.sendMessage(msg)
	}
	c.subscriptions = make(map[string]*wsConsumer)
	c.subMu.Unlock()

	c.connected = false
	return nil
}

// sendMessage sends a message via WebSocket
// In WASM, this will use syscall/js to call JavaScript WebSocket API
func (c *wsEventBusClient) sendMessage(msg *WSMessage) error {
	// This will be implemented in WASM using syscall/js
	// For now, just queue the message
	c.queueMu.Lock()
	c.messageQueue = append(c.messageQueue, msg)
	c.queueMu.Unlock()
	return nil
}

// handleMessage handles incoming WebSocket messages
func (c *wsEventBusClient) handleMessage(msg *WSMessage) {
	// Handle replies to requests
	if msg.ID != "" {
		c.replyMu.Lock()
		if replyChan, ok := c.pendingReplies[msg.ID]; ok {
			select {
			case replyChan <- msg:
			default:
			}
		}
		c.replyMu.Unlock()
		return
	}

	// Handle subscription messages
	if msg.Op == "message" {
		c.subMu.Lock()
		consumer, ok := c.subscriptions[msg.Address]
		c.subMu.Unlock()

		if ok && consumer.handler != nil {
			message := &messageImpl{
				body:    msg.Body,
				headers: msg.Headers,
			}
			consumer.handler(message)
		}
	}
}

// connect establishes WebSocket connection
// This will be implemented in WASM
func (c *wsEventBusClient) connect() error {
	// Implementation in WASM will use syscall/js
	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()
	return nil
}
