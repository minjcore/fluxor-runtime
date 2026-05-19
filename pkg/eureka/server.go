package eureka

import (
	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// ServerConfig configures the Eureka server (DEPRECATED: Use EurekaVerticleConfig instead)
// Kept for backward compatibility
type ServerConfig struct {
	// Address to bind the HTTP server (e.g., ":8761")
	Address string

	// Registry configuration
	RegistryConfig *RegistryConfig

	// Enable eviction of expired instances
	EnableEviction bool
}

// DefaultServerConfig returns default server configuration (DEPRECATED)
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Address:         ":8761",
		RegistryConfig:  DefaultRegistryConfig(),
		EnableEviction:  true,
	}
}

// ServerVerticle is DEPRECATED: Use EurekaVerticle instead
// ServerVerticle is kept for backward compatibility
// New code should use NewEurekaVerticle() or NewEurekaVerticleWithConfig()
type ServerVerticle = EurekaVerticle

// NewServerVerticle is DEPRECATED: Use NewEurekaVerticle() instead
// Creates a new Eureka server verticle (backward compatibility wrapper)
func NewServerVerticle(config *ServerConfig) *ServerVerticle {
	failfast.NotNil(config, "config")
	
	verticleConfig := EurekaVerticleConfig{
		Address:        config.Address,
		Prefix:         "",
		RegistryConfig: config.RegistryConfig,
		EnableEviction: config.EnableEviction,
	}
	
	return NewEurekaVerticleWithConfig(verticleConfig)
}
