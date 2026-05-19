package core

import "fmt"

// BaseHandler provides a Java-style abstract base class for message handlers
// It provides common handler patterns and utilities
type BaseHandler struct {
	// Handler name for logging/debugging
	name string

	// Logger for handler operations
	logger Logger
}

// NewBaseHandler creates a new BaseHandler
func NewBaseHandler(name string) *BaseHandler {
	return &BaseHandler{
		name:   name,
		logger: NewDefaultLogger(),
	}
}

// SetLogger sets a custom logger for this handler
func (bh *BaseHandler) SetLogger(logger Logger) {
	bh.logger = logger
}

// Name returns the handler name
func (bh *BaseHandler) Name() string {
	return bh.name
}

// Handle is the main handler method that subclasses should implement
// This follows the template method pattern
func (bh *BaseHandler) Handle(ctx FluxorContext, msg Message) error {
	// Extract request ID for logging
	requestID := GetRequestID(ctx.Context())

	// Log incoming message
	bh.logger.WithFields(map[string]interface{}{
		"handler":    bh.name,
		"address":    msg.Headers()["address"],
		"request_id": requestID,
	}).Debug("Handling message")

	// Call hook method for subclass implementation
	return bh.doHandle(ctx, msg)
}

// doHandle is a hook method that subclasses must implement
// This is the actual handler logic
func (bh *BaseHandler) doHandle(ctx FluxorContext, msg Message) error {
	// Default implementation - subclasses should override
	return &EventBusError{Code: "NOT_IMPLEMENTED", Message: "doHandle must be implemented by subclass"}
}

// Reply is a convenience method to reply to messages
func (bh *BaseHandler) Reply(msg Message, body interface{}) error {
	// Extract request ID from headers if available
	headers := msg.Headers()
	requestID := headers["X-Request-ID"]

	bh.logger.WithFields(map[string]interface{}{
		"handler":    bh.name,
		"request_id": requestID,
	}).Debug("Replying to message")

	return msg.Reply(body)
}

// Fail is a convenience method to fail messages
func (bh *BaseHandler) Fail(msg Message, code int, message string) error {
	// Extract request ID from headers if available
	headers := msg.Headers()
	requestID := headers["X-Request-ID"]

	bh.logger.WithFields(map[string]interface{}{
		"handler":    bh.name,
		"code":       code,
		"message":    message,
		"request_id": requestID,
	}).Error(fmt.Sprintf("Handler failed: %s", message))

	return msg.Fail(code, message)
}

// DecodeBody is a convenience method to decode JSON message body
func (bh *BaseHandler) DecodeBody(msg Message, v interface{}) error {
	body := msg.Body()
	if body == nil {
		return &EventBusError{Code: "EMPTY_BODY", Message: "message body is empty"}
	}
	if v == nil {
		return &EventBusError{Code: "NIL_TARGET", Message: "decode target is nil"}
	}

	// Try JSON decode if body is []byte
	if bodyBytes, ok := body.([]byte); ok {
		return JSONDecode(bodyBytes, v)
	}

	// Body is some other type - return error
	return &EventBusError{Code: "DECODE_ERROR", Message: "failed to decode message body - expected []byte"}
}

// EncodeBody is a convenience method to encode data to JSON
func (bh *BaseHandler) EncodeBody(data interface{}) ([]byte, error) {
	if data == nil {
		return nil, &EventBusError{Code: "NIL_INPUT", Message: "value to encode is nil"}
	}
	return JSONEncode(data)
}
