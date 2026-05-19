package errorhandler

import (
	"fmt"
	"log"
)

// Handler is an error handler that processes errors
type Handler interface {
	Handle(err error) error
}

// HandlerFunc is a function that implements Handler
type HandlerFunc func(err error) error

// Handle implements the Handler interface
func (f HandlerFunc) Handle(err error) error {
	return f(err)
}

// Chain chains multiple handlers together
func Chain(handlers ...Handler) Handler {
	return HandlerFunc(func(err error) error {
		currentErr := err
		for _, handler := range handlers {
			if handler == nil {
				continue
			}
			currentErr = handler.Handle(currentErr)
			if currentErr == nil {
				return nil
			}
		}
		return currentErr
	})
}

// LogHandler logs errors
type LogHandler struct {
	Logger func(format string, args ...interface{})
}

// NewLogHandler creates a new LogHandler
func NewLogHandler() *LogHandler {
	return &LogHandler{
		Logger: log.Printf,
	}
}

// Handle implements the Handler interface
func (h *LogHandler) Handle(err error) error {
	if err == nil {
		return nil
	}
	
	fluxorErr, ok := As(err)
	if ok {
		if h.Logger != nil {
			h.Logger("Error [%s]: %s (severity: %s)", 
				fluxorErr.Code, 
				fluxorErr.Message, 
				fluxorErr.Severity)
		}
	} else {
		if h.Logger != nil {
			h.Logger("Error: %v", err)
		}
	}
	
	return err
}

// TransformHandler transforms errors using a transformation function
type TransformHandler struct {
	Transform func(error) error
}

// NewTransformHandler creates a new TransformHandler
func NewTransformHandler(transform func(error) error) *TransformHandler {
	return &TransformHandler{
		Transform: transform,
	}
}

// Handle implements the Handler interface
func (h *TransformHandler) Handle(err error) error {
	if err == nil || h.Transform == nil {
		return err
	}
	return h.Transform(err)
}

// FilterHandler filters errors based on a predicate
type FilterHandler struct {
	Filter func(error) bool
	Handler Handler
}

// NewFilterHandler creates a new FilterHandler
func NewFilterHandler(filter func(error) bool, handler Handler) *FilterHandler {
	return &FilterHandler{
		Filter:  filter,
		Handler: handler,
	}
}

// Handle implements the Handler interface
func (h *FilterHandler) Handle(err error) error {
	if err == nil || h.Filter == nil {
		return err
	}
	
	if h.Filter(err) {
		if h.Handler != nil {
			return h.Handler.Handle(err)
		}
	}
	
	return err
}

// RetryHandler handles errors by retrying
type RetryHandler struct {
	MaxRetries int
	RetryFunc  func(error) bool
	Handler    Handler
}

// NewRetryHandler creates a new RetryHandler
func NewRetryHandler(maxRetries int, retryFunc func(error) bool, handler Handler) *RetryHandler {
	return &RetryHandler{
		MaxRetries: maxRetries,
		RetryFunc:  retryFunc,
		Handler:    handler,
	}
}

// Handle implements the Handler interface
func (h *RetryHandler) Handle(err error) error {
	if err == nil {
		return nil
	}
	
	if h.RetryFunc != nil && h.RetryFunc(err) && h.MaxRetries > 0 {
		// Retry logic would be implemented here
		// This is a placeholder for retry functionality
		return fmt.Errorf("retry not implemented: %w", err)
	}
	
	if h.Handler != nil {
		return h.Handler.Handle(err)
	}
	
	return err
}

// SuppressHandler suppresses certain errors
type SuppressHandler struct {
	Suppress func(error) bool
}

// NewSuppressHandler creates a new SuppressHandler
func NewSuppressHandler(suppress func(error) bool) *SuppressHandler {
	return &SuppressHandler{
		Suppress: suppress,
	}
}

// Handle implements the Handler interface
func (h *SuppressHandler) Handle(err error) error {
	if err == nil || h.Suppress == nil {
		return err
	}
	
	if h.Suppress(err) {
		return nil
	}
	
	return err
}
