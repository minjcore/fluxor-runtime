package proxy

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// ProxyServer represents a proxy server that can forward connections
type ProxyServer interface {
	// Start starts the proxy server
	Start() error

	// Stop stops the proxy server gracefully
	Stop() error

	// Metrics returns current proxy metrics
	Metrics() ServerMetrics

	// AddBackend adds a backend server to the proxy pool
	AddBackend(backend Backend) error

	// RemoveBackend removes a backend server from the proxy pool
	RemoveBackend(url string) error

	// GetBackends returns all configured backends
	GetBackends() []Backend

	// Health returns the health status of the proxy server
	Health() map[string]interface{}
}

// Backend represents a backend server to proxy to
type Backend struct {
	// URL is the backend server URL (e.g., "http://localhost:3000" or "tcp://localhost:9000")
	URL string `json:"url"`

	// Weight is the load balancing weight (higher = more traffic)
	Weight int `json:"weight" default:"1"`

	// HealthCheckURL is optional health check endpoint (HTTP only)
	HealthCheckURL string `json:"healthCheckURL,omitempty"`

	// HealthCheckInterval is the interval between health checks
	HealthCheckInterval time.Duration `json:"healthCheckInterval,omitempty" default:"30s"`

	// MaxConnections is the maximum concurrent connections to this backend
	MaxConnections int `json:"maxConnections,omitempty" default:"0"`

	// Timeout is the connection timeout for this backend
	Timeout time.Duration `json:"timeout,omitempty" default:"30s"`

	// Metadata for custom backend information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BackendStatus represents the health status of a backend
type BackendStatus struct {
	Backend     Backend    `json:"backend"`
	Healthy     bool       `json:"healthy"`
	LastCheck   time.Time  `json:"lastCheck"`
	LastError   error      `json:"lastError,omitempty"`
	Connections int        `json:"connections"`
	Latency     *time.Duration `json:"latency,omitempty"`
}

// LoadBalancer defines the load balancing strategy
type LoadBalancer interface {
	// SelectBackend selects a backend from the available healthy backends
	SelectBackend(backends []BackendStatus) (*Backend, error)
}

// ServerMetrics provides proxy server performance metrics
type ServerMetrics struct {
	// Connection metrics
	TotalConnections    int64   `json:"totalConnections"`
	ActiveConnections   int64   `json:"activeConnections"`
	RejectedConnections int64   `json:"rejectedConnections"`
	FailedConnections   int64   `json:"failedConnections"`

	// Request metrics (HTTP)
	TotalRequests       int64   `json:"totalRequests"`
	SuccessfulRequests  int64   `json:"successfulRequests"`
	FailedRequests      int64   `json:"failedRequests"`
	AverageResponseTime float64 `json:"averageResponseTime"` // milliseconds

	// Backend metrics
	BackendCount      int `json:"backendCount"`
	HealthyBackends   int `json:"healthyBackends"`
	UnhealthyBackends int `json:"unhealthyBackends"`

	// Throughput
	RequestsPerSecond float64 `json:"requestsPerSecond"`
	BytesTransferred  int64    `json:"bytesTransferred"`
}

// ProxyRequest represents a proxy request context
type ProxyRequest struct {
	*core.BaseRequestContext

	Context    context.Context
	Request    *http.Request
	Response   http.ResponseWriter
	GoCMD      core.GoCMD
	EventBus   core.EventBus
	Backend    *Backend
	StartTime  time.Time
}

// ProxyConnection represents a TCP proxy connection context
type ProxyConnection struct {
	*core.BaseRequestContext

	Context     context.Context
	ClientConn  net.Conn
	BackendConn net.Conn
	GoCMD       core.GoCMD
	EventBus    core.EventBus
	Backend     *Backend
	StartTime   time.Time
}

// ProxyError, ConfigError, and BackendError are defined in errors.go
