package core

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/fluxorio/fluxor/pkg/core/eventloop"
	"github.com/fluxorio/fluxor/pkg/core/failfast"
	"github.com/fluxorio/fluxor/pkg/protobuf"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

// eventBus implements EventBus
//
// Ownership and circular reference:
//   - eventBus is OWNED BY GoCMD (created in NewGoCMD, closed in GoCMD.Close)
//   - eventBus REFERENCES GoCMD (to create FluxorContext for message handlers)
//   - This circular reference (gocmd → eventBus → gocmd) is intentional
//   - Both are cleaned up together in GoCMD.Close(), no memory leak
//
// Thread-safety:
//   - mu protects the consumers map
//   - Individual consumer has its own mutex for handler field
//   - Publish/Send/Request use RLock (concurrent reads)
//   - Consumer registration uses Lock (exclusive writes)
//
// eventBus implements both EventBus and Producer.
var _ Producer = (*eventBus)(nil)

type eventBus struct {
	consumers   map[string][]*consumer
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	gocmd       GoCMD
	executor    concurrency.Executor
	logger      Logger
	loopGroup   *eventloop.EventLoopGroup
	sendCounter uint64 // atomic; round-robin index for Send/SendWithContext
}

// buildHeaders constructs a routing-headers map lazily — allocates only when
// FloxID or RequestID are actually present in ctx, saving 1 alloc per message
// in the common case where neither is set.
func buildHeaders(ctx context.Context) map[string]string {
	var h map[string]string
	if id := GetFloxID(ctx); id != "" {
		h = map[string]string{"X-Flox-ID": id}
	}
	if id := GetRequestID(ctx); id != "" {
		if h == nil {
			h = make(map[string]string, 1)
		}
		h["X-Request-ID"] = id
	}
	return h // nil when neither is set; newMessage handles nil headers
}

// EventBusConfig configures an EventBus
type EventBusConfig struct {
	// UseEventLoopGroup enables EventLoopGroup for CPU-based routing
	// Default: false (backward compatible)
	UseEventLoopGroup bool

	// EventLoopConfig configures EventLoopGroup (only used if UseEventLoopGroup is true)
	EventLoopConfig eventloop.EventLoopConfig
}

// NewEventBus creates a new event bus
func NewEventBus(ctx context.Context, gocmd GoCMD) EventBus {
	return NewEventBusWithConfig(ctx, gocmd, EventBusConfig{})
}

// NewEventBusWithConfig creates a new event bus with configuration
func NewEventBusWithConfig(ctx context.Context, gocmd GoCMD, config EventBusConfig) EventBus {
	ctx, cancel := context.WithCancel(ctx)

	// Create logger
	logger := NewDefaultLogger()

	// Create Executor for message processing
	// 256 workers: each permanent consumer occupies one worker indefinitely;
	// 10 was too few once several verticles + SSE connections coexist.
	executorConfig := concurrency.DefaultExecutorConfig()
	executorConfig.Workers = 256
	executorConfig.QueueSize = 4096
	executor := concurrency.NewExecutor(ctx, executorConfig)

	eb := &eventBus{
		consumers: make(map[string][]*consumer),
		ctx:       ctx,
		cancel:    cancel,
		gocmd:     gocmd,
		executor:  executor,
		logger:    logger,
	}

	// Create EventLoopGroup if enabled
	if config.UseEventLoopGroup {
		loopConfig := config.EventLoopConfig
		if loopConfig.NumLoops == 0 && loopConfig.QueueSize == 0 {
			loopConfig = eventloop.DefaultEventLoopConfig()
		}
		loopGroup, err := eventloop.NewEventLoopGroup(ctx, loopConfig)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to create EventLoopGroup: %v, falling back to Executor-only", err))
		} else {
			eb.loopGroup = loopGroup
		}
	}

	return eb
}

func (eb *eventBus) Publish(address string, body interface{}) error {
	// Fail-fast: validate inputs immediately
	if err := ValidateAddress(address); err != nil {
		return err
	}
	if err := ValidateBody(body); err != nil {
		return err
	}

	// Auto-encode to protobuf (if proto.Message) or JSON if not already []byte
	jsonBody, err := eb.encodeBody(body)
	if err != nil {
		return fmt.Errorf("encode body failed: %w", err)
	}

	eb.mu.RLock()
	consumers := eb.consumers[address]
	eb.mu.RUnlock()

	msg := newMessage(jsonBody, buildHeaders(eb.ctx), "", eb)

	// If EventLoopGroup is enabled, dispatch through it
	if eb.loopGroup != nil {
		return eb.publishViaEventLoop(address, msg, consumers)
	}

	// Original flow: direct mailbox send
	for _, c := range consumers {
		// Use Mailbox abstraction (hides channel operations)
		if err := c.mailbox.Send(msg); err != nil {
			if err == concurrency.ErrMailboxFull {
				// Non-blocking: if handler is busy, skip
				continue
			}
			if err == concurrency.ErrMailboxClosed {
				return eb.ctx.Err()
			}
			return err
		}
	}

	return nil
}

// PublishWithContext publishes a message using the provided context for routing
func (eb *eventBus) PublishWithContext(ctx context.Context, address string, body interface{}) error {
	// Fail-fast: validate inputs immediately
	if err := ValidateAddress(address); err != nil {
		return err
	}
	if err := ValidateBody(body); err != nil {
		return err
	}
	if ctx == nil {
		ctx = eb.ctx // Fallback to EventBus context
	}

	// Auto-encode to protobuf (if proto.Message) or JSON if not already []byte
	jsonBody, err := eb.encodeBody(body)
	if err != nil {
		return fmt.Errorf("encode body failed: %w", err)
	}

	eb.mu.RLock()
	consumers := eb.consumers[address]
	eb.mu.RUnlock()

	msg := newMessage(jsonBody, buildHeaders(ctx), "", eb)

	// If EventLoopGroup is enabled, dispatch through it
	if eb.loopGroup != nil {
		return eb.publishViaEventLoopWithContext(ctx, address, msg, consumers)
	}

	// Original flow: direct mailbox send
	for _, c := range consumers {
		// Use Mailbox abstraction (hides channel operations)
		if err := c.mailbox.Send(msg); err != nil {
			if err == concurrency.ErrMailboxFull {
				// Non-blocking: if handler is busy, skip
				continue
			}
			if err == concurrency.ErrMailboxClosed {
				return ctx.Err()
			}
			return err
		}
	}

	return nil
}

// publishViaEventLoop publishes message via EventLoopGroup
// FloxID from context has highest priority and is set in Event.Key
func (eb *eventBus) publishViaEventLoop(address string, msg Message, consumers []*consumer) error {
	return eb.publishViaEventLoopWithContext(eb.ctx, address, msg, consumers)
}

// publishViaEventLoopWithContext publishes message via EventLoopGroup using provided context
// FloxID from context has highest priority and is set in Event.Key
func (eb *eventBus) publishViaEventLoopWithContext(ctx context.Context, address string, msg Message, consumers []*consumer) error {
	// Priority 1: FloxID from context (highest priority - higher than headers)
	var routingKey string
	if floxid := GetFloxID(ctx); floxid != "" {
		routingKey = floxid
	}

	// Create event for each consumer
	for _, c := range consumers {
		// Merge headers from message with context FloxID (if not already in headers)
		headers := make(map[string]string)
		for k, v := range msg.Headers() {
			headers[k] = v
		}
		// Set FloxID in headers if from context and not already set
		if routingKey != "" && headers["X-Flox-ID"] == "" {
			headers["X-Flox-ID"] = routingKey
		}

		event := &eventloop.Event{
			Key:     routingKey, // FloxID from context (highest priority)
			Address: address,
			Body:    msg.Body(),
			Headers: headers,
			Handler: func(ctx context.Context, ev *eventloop.Event) error {
				// Forward to consumer mailbox
				return c.mailbox.Send(msg)
			},
		}

		// Dispatch to EventLoopGroup using provided context
		if err := eb.loopGroup.Dispatch(ctx, event); err != nil {
			if err == concurrency.ErrMailboxFull {
				// Non-blocking: if handler is busy, skip
				continue
			}
			return err
		}
	}
	return nil
}

func (eb *eventBus) Send(address string, body interface{}) error {
	// Fail-fast: validate inputs immediately
	if err := ValidateAddress(address); err != nil {
		return err
	}
	if err := ValidateBody(body); err != nil {
		return err
	}

	// Auto-encode to protobuf (if proto.Message) or JSON if not already []byte
	jsonBody, err := eb.encodeBody(body)
	if err != nil {
		return fmt.Errorf("encode body failed: %w", err)
	}

	eb.mu.RLock()
	consumers := eb.consumers[address]
	eb.mu.RUnlock()

	// Fail-fast: no handlers registered
	if len(consumers) == 0 {
		return &EventBusError{Code: "NO_HANDLERS", Message: "No handlers registered for address: " + address}
	}

	consumer := consumers[atomic.AddUint64(&eb.sendCounter, 1)%uint64(len(consumers))]
	msg := newMessage(jsonBody, buildHeaders(eb.ctx), "", eb)

	// If EventLoopGroup is enabled, dispatch through it
	if eb.loopGroup != nil {
		return eb.sendViaEventLoop(address, msg, consumer)
	}

	// Original flow: direct mailbox send
	// Use Mailbox abstraction (hides select statement)
	// Note: Mailbox.Send() is non-blocking, so timeout is handled by backpressure
	if err := consumer.mailbox.Send(msg); err != nil {
		if err == concurrency.ErrMailboxFull {
			return ErrTimeout
		}
		if err == concurrency.ErrMailboxClosed {
			return eb.ctx.Err()
		}
		return err
	}
	return nil
}

// sendViaEventLoop sends message via EventLoopGroup
// FloxID from context has highest priority and is set in Event.Key
func (eb *eventBus) sendViaEventLoop(address string, msg Message, consumer *consumer) error {
	_, err := eb.sendViaEventLoopWithContext(eb.ctx, address, msg, consumer, 0)
	return err
}

// sendViaEventLoopWithContext sends message via EventLoopGroup using provided context
// FloxID from context has highest priority and is set in Event.Key
func (eb *eventBus) sendViaEventLoopWithContext(ctx context.Context, address string, msg Message, consumer *consumer, timeout time.Duration) (Message, error) {
	// Priority 1: FloxID from context (highest priority - higher than headers)
	var routingKey string
	if floxid := GetFloxID(ctx); floxid != "" {
		routingKey = floxid
	}

	// Merge headers from message with context FloxID (if not already in headers)
	headers := make(map[string]string)
	for k, v := range msg.Headers() {
		headers[k] = v
	}
	// Set FloxID in headers if from context and not already set
	if routingKey != "" && headers["X-Flox-ID"] == "" {
		headers["X-Flox-ID"] = routingKey
	}

	event := &eventloop.Event{
		Key:     routingKey, // FloxID from context (highest priority)
		Address: address,
		Body:    msg.Body(),
		Headers: headers,
		Handler: func(ctx context.Context, ev *eventloop.Event) error {
			// Forward to consumer mailbox
			return consumer.mailbox.Send(msg)
		},
	}

	// Dispatch to EventLoopGroup using provided context
	if err := eb.loopGroup.Dispatch(ctx, event); err != nil {
		if err == concurrency.ErrMailboxFull {
			return nil, ErrTimeout
		}
		return nil, err
	}
	// Note: Request-reply pattern requires correlation ID tracking
	// For now, return error indicating it's not fully implemented
	return nil, fmt.Errorf("SendWithContext: request-reply pattern requires correlation ID tracking (not yet implemented)")
}

// SendWithContext sends a point-to-point message using the provided context for routing
func (eb *eventBus) SendWithContext(ctx context.Context, address string, body interface{}, timeout time.Duration) (Message, error) {
	// Fail-fast: validate inputs immediately
	if err := ValidateAddress(address); err != nil {
		return nil, err
	}
	if err := ValidateBody(body); err != nil {
		return nil, err
	}
	if err := ValidateTimeout(timeout); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = eb.ctx // Fallback to EventBus context
	}

	// Auto-encode to protobuf (if proto.Message) or JSON if not already []byte
	jsonBody, err := eb.encodeBody(body)
	if err != nil {
		return nil, fmt.Errorf("encode body failed: %w", err)
	}

	eb.mu.RLock()
	consumers := eb.consumers[address]
	eb.mu.RUnlock()

	// Fail-fast: no handlers registered
	if len(consumers) == 0 {
		return nil, &EventBusError{Code: "NO_HANDLERS", Message: "No handlers registered for address: " + address}
	}

	consumer := consumers[atomic.AddUint64(&eb.sendCounter, 1)%uint64(len(consumers))]
	msg := newMessage(jsonBody, buildHeaders(ctx), "", eb)

	// If EventLoopGroup is enabled, dispatch through it
	if eb.loopGroup != nil {
		return eb.sendViaEventLoopWithContext(ctx, address, msg, consumer, timeout)
	}

	// Original flow: direct mailbox send
	// Use Mailbox abstraction (hides select statement)
	// Note: Mailbox.Send() is non-blocking, so timeout is handled by backpressure
	if err := consumer.mailbox.Send(msg); err != nil {
		if err == concurrency.ErrMailboxFull {
			return nil, ErrTimeout
		}
		if err == concurrency.ErrMailboxClosed {
			return nil, ctx.Err()
		}
		return nil, err
	}
	// Note: Request-reply pattern requires correlation ID tracking
	// For now, return error indicating it's not fully implemented
	return nil, fmt.Errorf("SendWithContext: request-reply pattern requires correlation ID tracking (not yet implemented)")
}

func (eb *eventBus) Request(address string, body interface{}, timeout time.Duration) (Message, error) {
	// Fail-fast: validate inputs immediately
	if err := ValidateAddress(address); err != nil {
		return nil, err
	}
	if err := ValidateBody(body); err != nil {
		return nil, err
	}
	if err := ValidateTimeout(timeout); err != nil {
		return nil, err
	}

	// Auto-encode to protobuf (if proto.Message) or JSON if not already []byte
	jsonBody, err := eb.encodeBody(body)
	if err != nil {
		return nil, fmt.Errorf("encode body failed: %w", err)
	}

	replyAddress := generateReplyAddress()
	replyMailbox := concurrency.NewBoundedMailbox(1) // Hidden: channel creation

	// Register temporary reply handler
	replyConsumer := eb.Consumer(replyAddress)
	replyConsumer.Handler(func(ctx FluxorContext, msg Message) error {
		// Use Mailbox abstraction (hides channel send)
		if err := replyMailbox.Send(msg); err != nil {
			// Log if mailbox full (non-blocking) - this indicates reply arrived but couldn't be delivered
			eb.logger.Info(fmt.Sprintf("reply mailbox full for address %s: %v", replyAddress, err))
		}
		return nil
	})
	// Unregister is called after timeout/response, error is intentionally ignored
	// as the consumer cleanup is best-effort during request completion
	defer func() { _ = replyConsumer.Unregister() }()

	// Send request with reply address
	headers := map[string]string{"replyAddress": replyAddress}
	// Extract request ID from context if available
	if requestID := GetRequestID(eb.ctx); requestID != "" {
		headers["X-Request-ID"] = requestID
	}
	msg := newMessage(jsonBody, headers, replyAddress, eb)

	eb.mu.RLock()
	consumers := eb.consumers[address]
	eb.mu.RUnlock()

	// Fail-fast: no handlers registered
	if len(consumers) == 0 {
		return nil, &EventBusError{Code: "NO_HANDLERS", Message: "No handlers registered for address: " + address}
	}

	consumer := consumers[0]

	// Use Mailbox abstraction (hides select statement)
	// Note: Mailbox.Send() is non-blocking, timeout handled by backpressure
	if err := consumer.mailbox.Send(msg); err != nil {
		if err == concurrency.ErrMailboxFull {
			return nil, ErrTimeout
		}
		if err == concurrency.ErrMailboxClosed {
			return nil, eb.ctx.Err()
		}
		return nil, err
	}

	// Wait for reply using Mailbox abstraction (hides select statement)
	replyCtx, replyCancel := context.WithTimeout(eb.ctx, timeout)
	defer replyCancel()

	reply, err := replyMailbox.Receive(replyCtx)
	if err != nil {
		if err == context.DeadlineExceeded {
			return nil, ErrTimeout
		}
		return nil, err
	}

	if msg, ok := reply.(Message); ok {
		return msg, nil
	}
	return nil, fmt.Errorf("invalid reply message type")
}

func (eb *eventBus) Consumer(address string) Consumer {
	// Fail-fast: validate address immediately
	if err := ValidateAddress(address); err != nil {
		failfast.Err(err)
	}

	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Initialize fluxorCtx when creating consumer
	// Create FluxorContext for the consumer using eventBus's GoCMD reference
	var fluxorCtx FluxorContext
	if eb.gocmd != nil {
		fluxorCtx = newFluxorContext(eb.ctx, eb.gocmd)
	}

	c := &consumer{
		address:  address,
		mailbox:  concurrency.NewBoundedMailbox(100), // Hidden: channel creation
		eventBus: eb,
		ctx:      fluxorCtx,           // Initialize ctx to prevent nil pointer
		done:     make(chan struct{}), // Channel for Completion() notification (closed when mailbox processing stops)
	}

	eb.consumers[address] = append(eb.consumers[address], c)
	return c
}

func (eb *eventBus) Close() error {
	eb.cancel()

	// Close EventLoopGroup if enabled
	if eb.loopGroup != nil {
		if err := eb.loopGroup.Close(); err != nil {
			eb.logger.Info(fmt.Sprintf("EventLoopGroup close error: %v", err))
		}
	}

	// Shutdown executor gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := eb.executor.Shutdown(shutdownCtx); err != nil {
		eb.logger.Info(fmt.Sprintf("EventBus executor shutdown timeout: %v", err))
	}

	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, consumers := range eb.consumers {
		for _, c := range consumers {
			// Close mailbox (hides channel close operation)
			c.mailbox.Close()
		}
	}
	eb.consumers = make(map[string][]*consumer)
	return nil
}

// consumer implements Consumer
// Uses Mailbox abstraction to hide channel operations
type consumer struct {
	address  string
	mailbox  concurrency.Mailbox // Abstracted: hides chan Message
	handler  MessageHandler
	eventBus *eventBus
	ctx      FluxorContext
	mu       sync.RWMutex
	done     chan struct{} // Channel for Completion() notification (closed when mailbox closes)
}

func (c *consumer) Handler(handler MessageHandler) Consumer {
	// Fail-fast: handler cannot be nil
	failfast.NotNil(handler, "handler")

	c.mu.Lock()
	c.handler = handler
	c.mu.Unlock()

	// Start processing messages using Executor (hides go func() call)
	task := concurrency.NewNamedTask(
		fmt.Sprintf("eventbus-consumer-%s", c.address),
		func(ctx context.Context) error {
			return c.processMessages(ctx)
		},
	)
	if err := c.eventBus.executor.Submit(task); err != nil {
		c.eventBus.logger.Error(fmt.Sprintf("Failed to submit consumer task for address %s: %v", c.address, err))
		// Close done channel since processing won't start
		close(c.done)
		// Still return consumer - handler can be set later
	}
	return c
}

func (c *consumer) processMessages(ctx context.Context) error {
	// Panic isolation - recover from panics without re-panicking
	// This allows the message processing loop to continue even if one message handler panics
	defer func() {
		if r := recover(); r != nil {
			// Log panic but don't re-panic - maintain panic isolation
			c.eventBus.logger.Error(fmt.Sprintf("panic in message processing loop for address %s (isolated): %v", c.address, r))
		}
		// Close done channel to notify Completion() when mailbox processing stops
		close(c.done)
	}()

	// Use Mailbox abstraction (hides select statement and channel operations)
	for {
		// Receive message using Mailbox (hides channel receive and select)
		msg, err := c.mailbox.Receive(ctx)
		if err != nil {
			// Mailbox closed or context cancelled
			return err
		}

		// Type assert to Message
		imsg, ok := msg.(Message)
		if !ok {
			c.eventBus.logger.Info(fmt.Sprintf("Invalid message type received for address %s", c.address))
			continue
		}

		if c.handler != nil {
			fluxorCtx := c.ctx
			if fluxorCtx == nil {
				if c.eventBus.gocmd != nil {
					fluxorCtx = newFluxorContext(c.eventBus.ctx, c.eventBus.gocmd)
				}
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						c.eventBus.logger.Error(fmt.Sprintf("handler panic for address %s (isolated): %v", c.address, r))
					}
				}()
				if err := c.handler(fluxorCtx, imsg); err != nil {
					requestID := ""
					if headers := imsg.Headers(); headers != nil {
						requestID = headers["X-Request-ID"]
					}
					if requestID != "" {
						c.eventBus.logger.Error(fmt.Sprintf("handler error for address %s (request_id=%s): %v", c.address, requestID, err))
					} else {
						c.eventBus.logger.Error(fmt.Sprintf("handler error for address %s: %v", c.address, err))
					}
				}
			}()
		} else {
			c.eventBus.logger.Info(fmt.Sprintf("handler is nil for address %s", c.address))
		}
	}
}

func (c *consumer) Completion() <-chan struct{} {
	// Return the done channel that will be closed when mailbox processing stops
	// This is efficient - no polling, just channel notification
	return c.done
}

func (c *consumer) Unregister() error {
	c.eventBus.mu.Lock()
	defer c.eventBus.mu.Unlock()

	consumers := c.eventBus.consumers[c.address]
	for i, cons := range consumers {
		if cons == c {
			c.eventBus.consumers[c.address] = append(consumers[:i], consumers[i+1:]...)
			break
		}
	}

	// Close mailbox (hides channel close operation)
	c.mailbox.Close()
	return nil
}

func generateReplyAddress() string {
	return "reply." + uuid.New().String()
}

// encodeBody encodes body to protobuf (if proto.Message) or JSON - fail-fast
func (eb *eventBus) encodeBody(body interface{}) (interface{}, error) {
	// Fail-fast: validate body
	if err := ValidateBody(body); err != nil {
		return nil, err
	}

	// If already []byte, return as-is
	if data, ok := body.([]byte); ok {
		return data, nil
	}

	// Auto-detect: if body is proto.Message, encode as protobuf
	if _, ok := body.(proto.Message); ok {
		data, err := protobuf.ProtobufEncode(body)
		if err != nil {
			return nil, fmt.Errorf("protobuf encode failed: %w", err)
		}
		return data, nil
	}

	// Otherwise, encode to JSON - errors are propagated immediately
	return JSONEncode(body)
}
