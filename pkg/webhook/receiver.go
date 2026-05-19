package webhook

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// Receiver receives and processes webhook requests
type Receiver struct {
	*core.BaseComponent
	config    *ReceiverConfig
	eventBus  core.EventBus
	endpoints map[string]*endpointHandler
	mu        sync.RWMutex
}

// endpointHandler handles a single webhook endpoint
type endpointHandler struct {
	config    EndpointConfig
	validator SignatureValidator
}

// NewReceiver creates a new webhook receiver
func NewReceiver(config *ReceiverConfig) *Receiver {
	failfast.NotNil(config, "config")
	
	return &Receiver{
		BaseComponent: core.NewBaseComponent("webhook-receiver"),
		config:        config,
		endpoints:     make(map[string]*endpointHandler),
	}
}

// doStart initializes the receiver
func (r *Receiver) doStart(ctx core.FluxorContext) error {
	if err := r.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	r.eventBus = ctx.EventBus()
	if r.eventBus == nil {
		return &core.EventBusError{Code: "EVENTBUS_NOT_AVAILABLE", Message: "EventBus is not available"}
	}

	// Initialize endpoints
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, endpointConfig := range r.config.Endpoints {
		endpointConfig.SetDefaults()

		handler := &endpointHandler{
			config:    endpointConfig,
			validator: endpointConfig.GetValidator(),
		}

		// Full path includes prefix
		fullPath := strings.TrimSuffix(r.config.Prefix, "/") + "/" + strings.TrimPrefix(endpointConfig.Path, "/")
		r.endpoints[fullPath] = handler
	}

	return nil
}

// doStop cleans up the receiver
func (r *Receiver) doStop(ctx core.FluxorContext) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.endpoints = make(map[string]*endpointHandler)
	r.eventBus = nil

	return nil
}

// HandleRequest handles a webhook request
func (r *Receiver) HandleRequest(req *WebhookRequest) error {
	failfast.NotNil(req, "req")

	r.mu.RLock()
	handler, ok := r.endpoints[req.Path]
	r.mu.RUnlock()

	if !ok {
		return fmt.Errorf("webhook endpoint not found: %s", req.Path)
	}

	// Validate signature if validator is configured
	if handler.validator != nil {
		if err := handler.validator.Validate(req.Payload, "", req.Headers); err != nil {
			if r.config.OnError != nil {
				if handleErr := r.config.OnError(req.Path, err); handleErr != nil {
					return handleErr
				}
			}
			return fmt.Errorf("signature validation failed: %w", err)
		}
	}

	// Publish to EventBus
	if r.eventBus != nil {
		event := map[string]interface{}{
			"path":        req.Path,
			"payload":     req.Payload,
			"headers":     req.Headers,
			"queryParams": req.QueryParams,
			"method":      req.Method,
		}

		if err := r.eventBus.Publish(handler.config.EventBusAddress, event); err != nil {
			if r.config.OnError != nil {
				if handleErr := r.config.OnError(req.Path, err); handleErr != nil {
					return handleErr
				}
			}
			return fmt.Errorf("failed to publish webhook event: %w", err)
		}
	}

	return nil
}

// GetEndpoint returns the endpoint configuration for a path
func (r *Receiver) GetEndpoint(path string) (*EndpointConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, ok := r.endpoints[path]
	if !ok {
		return nil, false
	}

	return &handler.config, true
}

// GetEndpoints returns all registered endpoint paths
func (r *Receiver) GetEndpoints() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	paths := make([]string, 0, len(r.endpoints))
	for path := range r.endpoints {
		paths = append(paths, path)
	}

	return paths
}