package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
)

// RPCRequest represents an RPC request
type RPCRequest struct {
	// CacheKey is the cache key to retrieve info
	CacheKey string `json:"cache_key"`

	// Additional request data
	Data map[string]interface{} `json:"data,omitempty"`
}

// RPCResponse represents an RPC response
type RPCResponse struct {
	// Success indicates if the request was successful
	Success bool `json:"success"`

	// Data contains the response data (from cache)
	Data []byte `json:"data,omitempty"`

	// Error message if request failed
	Error string `json:"error,omitempty"`

	// CorrelationID matches the request
	CorrelationID string `json:"correlation_id"`
}

// RPCClient provides RPC functionality
type RPCClient struct {
	publisher    Publisher
	consumer     Consumer
	conn         *Connection
	cache        cache.Cache
	replyHandler ReplyHandler // Optional custom reply handler

	// Pending requests
	pending    map[string]chan *RPCResponse
	pendingMu  sync.RWMutex
	replyQueue string
	timeout    time.Duration
}

// NewRPCClient creates a new RPC client
// Fail-fast: Validates inputs
func NewRPCClient(conn *Connection, cache cache.Cache, replyQueue string, timeout time.Duration) (*RPCClient, error) {
	if conn == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "connection cannot be nil"}
	}
	if cache == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "cache cannot be nil"}
	}
	if replyQueue == "" {
		return nil, &Error{Code: "INVALID_INPUT", Message: "reply queue cannot be empty"}
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Create publisher
	publisher, err := NewPublisher(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	// Create consumer for replies
	consumer, err := NewConsumer(conn)
	if err != nil {
		// Clean up publisher on error
		if closeErr := publisher.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to create consumer: %w (also failed to close publisher: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	client := &RPCClient{
		publisher:    publisher,
		consumer:     consumer,
		conn:         conn,
		cache:        cache,
		pending:      make(map[string]chan *RPCResponse),
		replyQueue:   replyQueue,
		timeout:      timeout,
		replyHandler: nil, // Will use default handler
	}

	// Start consuming replies with default handler
	ctx := context.Background()
	if err := consumer.Consume(ctx, replyQueue, client.handleReply); err != nil {
		// Clean up resources on error
		var closeErrs []error
		if closeErr := publisher.Close(); closeErr != nil {
			closeErrs = append(closeErrs, fmt.Errorf("failed to close publisher: %w", closeErr))
		}
		if closeErr := consumer.Close(); closeErr != nil {
			closeErrs = append(closeErrs, fmt.Errorf("failed to close consumer: %w", closeErr))
		}
		if len(closeErrs) > 0 {
			return nil, fmt.Errorf("failed to start consuming replies: %w (cleanup errors: %v)", err, closeErrs)
		}
		return nil, fmt.Errorf("failed to start consuming replies: %w", err)
	}

	return client, nil
}

// SetReplyHandler sets a custom reply handler (similar to EventBus Consumer.Handler)
// If set, this handler will be called for all reply messages before default handling
func (c *RPCClient) SetReplyHandler(handler ReplyHandler) {
	c.replyHandler = handler
}

// Call sends an RPC request and waits for response
// Fail-fast: Validates inputs
func (c *RPCClient) Call(ctx context.Context, requestQueue string, req RPCRequest) (*RPCResponse, error) {
	// Fail-fast: Validate context
	if ctx == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}

	// Fail-fast: Validate queue name
	if err := ValidateQueueName(requestQueue); err != nil {
		return nil, err
	}

	// Fail-fast: Validate cache key
	if req.CacheKey == "" {
		return nil, &Error{Code: "INVALID_INPUT", Message: "cache key cannot be empty"}
	}

	// Generate correlation ID
	correlationID := generateCorrelationID()

	// Create response channel
	responseChan := make(chan *RPCResponse, 1)

	// Register pending request
	c.pendingMu.Lock()
	c.pending[correlationID] = responseChan
	c.pendingMu.Unlock()

	// Cleanup on return
	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, correlationID)
		close(responseChan)
		c.pendingMu.Unlock()
	}()

	// Create request message
	msg := Message{
		Body:          req,
		ReplyTo:       c.replyQueue,
		CorrelationID: correlationID,
		DeliveryMode:  2, // Persistent
	}

	// Publish request
	if err := c.publisher.Publish(ctx, "", requestQueue, msg); err != nil {
		return nil, fmt.Errorf("failed to publish request: %w", err)
	}

	// Wait for response with timeout
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
	case <-time.After(c.timeout):
		return nil, &Error{Code: "TIMEOUT", Message: fmt.Sprintf("RPC call timeout after %v", c.timeout)}
	case response := <-responseChan:
		return response, nil
	}
}

// handleReply handles incoming reply messages
// Similar to EventBus MessageHandler pattern
func (c *RPCClient) handleReply(ctx context.Context, delivery *Delivery) error {
	// If custom handler is set, call it first (similar to EventBus Consumer.Handler)
	if c.replyHandler != nil {
		if err := c.replyHandler(ctx, delivery); err != nil {
			// Handler can return error to nack message
			return err
		}
		// If handler succeeds, continue with default processing
	}

	// Default handling: match correlation ID and send to pending request
	var response RPCResponse
	if err := delivery.DecodeBody(&response); err != nil {
		// Log error but don't fail (invalid response format)
		return nil
	}

	// Find pending request
	c.pendingMu.RLock()
	responseChan, exists := c.pending[response.CorrelationID]
	c.pendingMu.RUnlock()

	if !exists {
		// No pending request for this correlation ID
		// This is normal if handler already processed it
		return nil
	}

	// Send response (non-blocking)
	select {
	case responseChan <- &response:
	default:
		// Channel already closed or full
	}

	return nil
}

// Close closes the RPC client
func (c *RPCClient) Close() error {
	var errs []error

	if c.publisher != nil {
		if err := c.publisher.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.consumer != nil {
		if err := c.consumer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing RPC client: %v", errs)
	}

	return nil
}

// RPCServer provides RPC server functionality
type RPCServer struct {
	consumer     Consumer
	cache        cache.Cache
	requestQueue string
	publisher    Publisher // Reused publisher for replies
	conn         *Connection
}

// NewRPCServer creates a new RPC server
// Fail-fast: Validates inputs
func NewRPCServer(conn *Connection, cache cache.Cache, requestQueue string) (*RPCServer, error) {
	if conn == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "connection cannot be nil"}
	}
	if cache == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "cache cannot be nil"}
	}
	if err := ValidateQueueName(requestQueue); err != nil {
		return nil, err
	}

	consumer, err := NewConsumer(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Create publisher for reuse (more efficient than creating new one per reply)
	publisher, err := NewPublisher(conn)
	if err != nil {
		// Clean up consumer on error
		if closeErr := consumer.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to create publisher: %w (also failed to close consumer: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	return &RPCServer{
		consumer:     consumer,
		cache:        cache,
		requestQueue: requestQueue,
		publisher:    publisher,
		conn:         conn,
	}, nil
}

// Start starts the RPC server
// Fail-fast: Validates state
func (s *RPCServer) Start(ctx context.Context) error {
	if ctx == nil {
		return &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}

	return s.consumer.Consume(ctx, s.requestQueue, s.handleRequest)
}

// handleRequest handles incoming RPC requests
func (s *RPCServer) handleRequest(ctx context.Context, delivery *Delivery) error {
	// Decode request
	var req RPCRequest
	if err := delivery.DecodeBody(&req); err != nil {
		// Send error response
		return s.sendErrorResponse(ctx, delivery, fmt.Sprintf("invalid request: %v", err))
	}

	// Get info from cache
	cacheData, err := s.cache.Get(ctx, req.CacheKey)
	if err != nil {
		// Cache miss or error
		return s.sendErrorResponse(ctx, delivery, fmt.Sprintf("cache error for key %s: %v", req.CacheKey, err))
	}

	// Send success response
	return s.sendSuccessResponse(ctx, delivery, cacheData)
}

// sendSuccessResponse sends a success response
func (s *RPCServer) sendSuccessResponse(ctx context.Context, delivery *Delivery, data []byte) error {
	response := RPCResponse{
		Success:       true,
		Data:          data,
		CorrelationID: delivery.CorrelationID,
	}

	// Use cached publisher (reused for efficiency)
	if s.publisher == nil {
		return &Error{Code: "INVALID_STATE", Message: "publisher not available"}
	}

	// Publish reply
	msg := Message{
		Body:          response,
		CorrelationID: delivery.CorrelationID,
		DeliveryMode:  2, // Persistent
	}

	// Use ReplyTo from original message
	if delivery.ReplyTo == "" {
		return &Error{Code: "INVALID_STATE", Message: "no reply-to address in request"}
	}

	return s.publisher.Publish(ctx, "", delivery.ReplyTo, msg)
}

// sendErrorResponse sends an error response
func (s *RPCServer) sendErrorResponse(ctx context.Context, delivery *Delivery, errorMsg string) error {
	response := RPCResponse{
		Success:       false,
		Error:         errorMsg,
		CorrelationID: delivery.CorrelationID,
	}

	// Use cached publisher (reused for efficiency)
	if s.publisher == nil {
		return &Error{Code: "INVALID_STATE", Message: "publisher not available"}
	}

	// Publish reply
	msg := Message{
		Body:          response,
		CorrelationID: delivery.CorrelationID,
		DeliveryMode:  2, // Persistent
	}

	// Use ReplyTo from original message
	if delivery.ReplyTo == "" {
		return &Error{Code: "INVALID_STATE", Message: "no reply-to address in request"}
	}

	return s.publisher.Publish(ctx, "", delivery.ReplyTo, msg)
}

// Close closes the RPC server
func (s *RPCServer) Close() error {
	var errs []error

	if s.consumer != nil {
		if err := s.consumer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if s.publisher != nil {
		if err := s.publisher.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing RPC server: %v", errs)
	}

	return nil
}

// generateCorrelationID generates a unique correlation ID
func generateCorrelationID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// This should never happen in practice, but if crypto/rand fails,
		// we cannot safely generate a correlation ID. Panic to fail fast.
		panic(fmt.Sprintf("failed to generate correlation ID: %v", err))
	}
	return hex.EncodeToString(b)
}
