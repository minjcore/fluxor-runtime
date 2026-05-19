package proxy

import (
	"sync"

	"github.com/fluxorio/fluxor/pkg/core"
)

// ProxyComponent provides proxy server integration with Fluxor
// Similar to other components, this manages proxy server lifecycle
type ProxyComponent struct {
	*core.BaseComponent
	config      Config
	proxyServer ProxyServer
	mu          sync.RWMutex
	started     bool
}

// NewProxyComponent creates a new proxy component
// Fail-fast: Validates configuration
func NewProxyComponent(config Config) *ProxyComponent {
	return &ProxyComponent{
		BaseComponent: core.NewBaseComponent("proxy"),
		config:        config,
	}
}

// Start initializes the component (overrides BaseComponent.Start to call our doStart)
func (c *ProxyComponent) Start(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return &core.EventBusError{Code: "ALREADY_STARTED", Message: "component already started"}
	}

	// Call our custom doStart
	if err := c.doStart(ctx); err != nil {
		return err
	}

	c.started = true
	return nil
}

// Stop stops the component (overrides BaseComponent.Stop to call our doStop)
func (c *ProxyComponent) Stop(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	// Call our custom doStop
	if err := c.doStop(ctx); err != nil {
		return err
	}

	c.started = false
	return nil
}

// IsStarted returns whether the component is started (overrides BaseComponent.IsStarted)
func (c *ProxyComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// doStart initializes the proxy server
// Fail-fast: Validates state and configuration before starting
func (c *ProxyComponent) doStart(ctx core.FluxorContext) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	// Validate configuration
	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	// Create proxy server
	proxyServer, err := NewProxyServer(ctx.GoCMD(), c.config)
	if err != nil {
		return &core.EventBusError{Code: "PROXY_SERVER_ERROR", Message: err.Error()}
	}

	c.proxyServer = proxyServer

	// Start proxy server in goroutine (non-blocking)
	go func() {
		if err := c.proxyServer.Start(); err != nil {
			// Log error but don't fail component start
			c.Logger().WithFields(map[string]interface{}{"error": err.Error()}).Error("Proxy server error")
		}
	}()

	// Notify via EventBus (Premium Pattern integration)
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("proxy.ready", map[string]interface{}{
			"component":  "proxy",
			"listenAddr": c.config.ListenAddr,
			"protocol":   c.config.Protocol,
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// doStop stops the proxy component
func (c *ProxyComponent) doStop(ctx core.FluxorContext) error {
	// Stop proxy server
	if c.proxyServer != nil {
		if err := c.proxyServer.Stop(); err != nil {
			c.Logger().WithFields(map[string]interface{}{"error": err.Error()}).Error("Proxy server stop error")
		}
		c.proxyServer = nil
	}

	// Notify via EventBus
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("proxy.stopped", map[string]interface{}{
			"component": "proxy",
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// ProxyServer returns the proxy server instance
// Fail-fast: Returns error if component is not started or server is nil
func (c *ProxyComponent) ProxyServer() (ProxyServer, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "Proxy component is not started"}
	}
	if c.proxyServer == nil {
		return nil, &core.EventBusError{Code: "SERVER_NOT_INITIALIZED", Message: "Proxy server is not initialized"}
	}
	return c.proxyServer, nil
}

// Metrics returns proxy server metrics
func (c *ProxyComponent) Metrics() (ServerMetrics, error) {
	server, err := c.ProxyServer()
	if err != nil {
		return ServerMetrics{}, err
	}
	return server.Metrics(), nil
}

// Logger returns the component logger
func (c *ProxyComponent) Logger() core.Logger {
	// BaseComponent doesn't have a logger, so create a default one
	return core.NewDefaultLogger()
}
