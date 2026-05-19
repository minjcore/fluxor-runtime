package eureka

import (
	"context"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/failfast"
	"github.com/fluxorio/fluxor/pkg/web"
)

// EurekaVerticleConfig configures the EurekaVerticle
type EurekaVerticleConfig struct {
	// Address is the HTTP server address (e.g., ":8761")
	// If empty, will use "eureka_addr" from context config or default to ":8761"
	Address string

	// Prefix is the route prefix for Eureka routes (e.g., "" for root, "/eureka" for /eureka prefix)
	// If empty, routes are registered at root
	Prefix string

	// Registry configuration
	RegistryConfig *RegistryConfig

	// Enable eviction of expired instances
	EnableEviction bool
}

// DefaultEurekaVerticleConfig returns default Eureka verticle configuration
func DefaultEurekaVerticleConfig() EurekaVerticleConfig {
	return EurekaVerticleConfig{
		Address:         ":8761",
		Prefix:          "",
		RegistryConfig:  DefaultRegistryConfig(),
		EnableEviction:  true,
	}
}

// EurekaVerticle is a reusable verticle that provides Eureka service registry functionality
// It can be deployed into any application to add Eureka service registry endpoints
type EurekaVerticle struct {
	*core.BaseVerticle
	server        *web.FastHTTPServer
	config        EurekaVerticleConfig
	registry      *Registry
	stopEviction  chan struct{}
}

// NewEurekaVerticle creates a new EurekaVerticle with default configuration
func NewEurekaVerticle() *EurekaVerticle {
	return NewEurekaVerticleWithConfig(DefaultEurekaVerticleConfig())
}

// NewEurekaVerticleWithConfig creates a new EurekaVerticle with custom configuration
func NewEurekaVerticleWithConfig(config EurekaVerticleConfig) *EurekaVerticle {
	failfast.NotNil(&config, "config")
	if config.RegistryConfig == nil {
		config.RegistryConfig = DefaultRegistryConfig()
	}

	return &EurekaVerticle{
		BaseVerticle: core.NewBaseVerticle("eureka"),
		config:       config,
		registry:     NewRegistry(config.RegistryConfig),
		stopEviction: make(chan struct{}),
	}
}

// Start initializes the HTTP server and registers Eureka routes
func (v *EurekaVerticle) Start(ctx core.FluxorContext) error {
	// Start base verticle first
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	logger := core.NewDefaultLogger()
	logger.Info("EurekaVerticle starting...")

	// Get HTTP address from config or context
	addr := v.config.Address
	if addr == "" {
		if val, ok := ctx.Config()["eureka_addr"].(string); ok && val != "" {
			addr = val
		} else {
			addr = ":8761" // Default Eureka port
		}
	}

	// Create FastHTTPServer
	serverConfig := web.DefaultFastHTTPServerConfig(addr)
	v.server = web.NewFastHTTPServer(ctx.GoCMD(), serverConfig)

	// Get router
	router := v.server.FastRouter()

	// Register Eureka routes
	Register(router, v.config.Prefix, v.registry)

	// Start eviction task if enabled
	if v.config.EnableEviction {
		v.startEvictionTask(ctx.GoCMD().Context())
	}

	// Start server on I/O bound execution
	v.ExecuteOn(func() {
		logger.Info("Eureka HTTP server starting on " + addr)
		if err := v.server.Start(); err != nil {
			logger.Error("Eureka server error: " + err.Error())
		}
	})

	logger.Info("EurekaVerticle started successfully")
	return nil
}

// Stop stops the HTTP server
func (v *EurekaVerticle) Stop(ctx core.FluxorContext) error {
	logger := core.NewDefaultLogger()
	logger.Info("EurekaVerticle stopping...")

	// Stop eviction task
	if v.config.EnableEviction {
		close(v.stopEviction)
	}

	// Stop HTTP server
	if v.server != nil {
		if err := v.server.Stop(); err != nil {
			logger.Error("Error stopping Eureka server: " + err.Error())
			return err
		}
	}

	return v.BaseVerticle.Stop(ctx)
}

// Registry returns the underlying registry (for testing/debugging)
func (v *EurekaVerticle) Registry() *Registry {
	return v.registry
}

// startEvictionTask starts a background task to evict expired instances
func (v *EurekaVerticle) startEvictionTask(ctx context.Context) {
	ticker := time.NewTicker(v.config.RegistryConfig.EvictionInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				evicted := v.registry.EvictExpired()
				if evicted > 0 {
					logger := core.NewDefaultLogger()
					logger.Info("Evicted expired instances", "count", evicted)
				}
			case <-v.stopEviction:
				ticker.Stop()
				return
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
