package vpn

import (
	"sync"

	"github.com/fluxorio/fluxor/pkg/core"
)

// VPNComponent provides VPN server integration with Fluxor
// Similar to other components, this manages VPN server lifecycle
type VPNComponent struct {
	*core.BaseComponent
	config    Config
	vpnServer VPNServer
	mu        sync.RWMutex
	started   bool
}

// NewVPNComponent creates a new VPN component
// Fail-fast: Validates configuration
func NewVPNComponent(config Config) *VPNComponent {
	return &VPNComponent{
		BaseComponent: core.NewBaseComponent("vpn"),
		config:        config,
	}
}

// Start initializes the component (overrides BaseComponent.Start to call our doStart)
func (c *VPNComponent) Start(ctx core.FluxorContext) error {
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
func (c *VPNComponent) Stop(ctx core.FluxorContext) error {
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
func (c *VPNComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// doStart initializes the VPN server
// Fail-fast: Validates state and configuration before starting
func (c *VPNComponent) doStart(ctx core.FluxorContext) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	// Validate configuration
	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	// Create VPN server
	vpnServer, err := NewVPNServer(ctx.GoCMD(), c.config)
	if err != nil {
		return &core.EventBusError{Code: "VPN_SERVER_ERROR", Message: err.Error()}
	}

	c.vpnServer = vpnServer

	// Start VPN server in goroutine (non-blocking)
	go func() {
		if err := c.vpnServer.Start(); err != nil {
			// Log error but don't fail component start
			c.Logger().Error("VPN server error", "error", err)
		}
	}()

	// Notify via EventBus (Premium Pattern integration)
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("vpn.ready", map[string]interface{}{
			"component":  "vpn",
			"listenAddr": c.config.ListenAddr,
			"protocol":   c.config.Protocol,
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// doStop stops the VPN component
func (c *VPNComponent) doStop(ctx core.FluxorContext) error {
	// Stop VPN server
	if c.vpnServer != nil {
		if err := c.vpnServer.Stop(); err != nil {
			c.Logger().Error("VPN server stop error", "error", err)
		}
		c.vpnServer = nil
	}

	// Notify via EventBus
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("vpn.stopped", map[string]interface{}{
			"component": "vpn",
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// VPNServer returns the VPN server instance
// Fail-fast: Returns error if component is not started or server is nil
func (c *VPNComponent) VPNServer() (VPNServer, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "VPN component is not started"}
	}
	if c.vpnServer == nil {
		return nil, &core.EventBusError{Code: "SERVER_NOT_INITIALIZED", Message: "VPN server is not initialized"}
	}
	return c.vpnServer, nil
}

// Metrics returns VPN server metrics
func (c *VPNComponent) Metrics() (ServerMetrics, error) {
	server, err := c.VPNServer()
	if err != nil {
		return ServerMetrics{}, err
	}
	return server.Metrics(), nil
}

// Logger returns the component logger
func (c *VPNComponent) Logger() core.Logger {
	// BaseComponent doesn't have a logger, so create a default one
	return core.NewDefaultLogger()
}
