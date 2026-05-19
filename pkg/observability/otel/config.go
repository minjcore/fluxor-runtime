package otel

import (
	"fmt"
)

// Config configures OpenTelemetry
type Config struct {
	// ServiceName is the name of the service
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// Exporter is the exporter type: "jaeger", "zipkin", "stdout", "none"
	Exporter string

	// Endpoint is the exporter endpoint URL
	Endpoint string

	// Environment is the deployment environment (dev, staging, prod)
	Environment string

	// SampleRate is the sampling rate (0.0 to 1.0)
	SampleRate float64
}

// DefaultConfig returns a default OpenTelemetry configuration
func DefaultConfig() Config {
	return Config{
		ServiceName:    "fluxor-service",
		ServiceVersion: "1.0.0",
		Exporter:       "stdout",
		Endpoint:       "",
		Environment:    "development",
		SampleRate:     1.0, // 100% sampling by default
	}
}

// Validate validates the configuration
func (c Config) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if c.SampleRate < 0.0 || c.SampleRate > 1.0 {
		return fmt.Errorf("sample rate must be between 0.0 and 1.0")
	}
	return nil
}
