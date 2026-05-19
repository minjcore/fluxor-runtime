// Package connectors provides common interfaces and utilities for all connector implementations.
//
// All connectors (Airtable, Notion, GitHub, etc.) should implement the Connector interface
// to ensure consistent behavior and integration with the Fluxor framework.
package connectors

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core"
)

// Connector is the common interface that all connectors must implement.
// It provides lifecycle management, health checking, and metadata.
//
// Note: Connectors should embed core.BaseComponent for lifecycle management,
// but don't need to implement core.Component directly in this interface.
type Connector interface {
	// Name returns the connector name (e.g., "airtable", "notion", "github")
	Name() string

	// Type returns the connector type category
	// Examples: "database", "api", "storage", "messaging", "analytics"
	Type() ConnectorType

	// Version returns the connector version
	Version() string

	// IsHealthy checks if the connector is healthy and can serve requests
	IsHealthy(ctx context.Context) (bool, error)

	// GetMetadata returns connector metadata (capabilities, limits, etc.)
	GetMetadata() ConnectorMetadata

	// Start starts the connector (lifecycle management)
	Start(ctx core.FluxorContext) error

	// Stop stops the connector (lifecycle management)
	Stop(ctx core.FluxorContext) error

	// IsStarted returns whether the connector is started
	IsStarted() bool
}

// ConnectorType represents the category of connector
type ConnectorType string

const (
	// Database connectors (PostgreSQL, MongoDB, Redis, etc.)
	TypeDatabase ConnectorType = "database"

	// API connectors (REST APIs, GraphQL, etc.)
	TypeAPI ConnectorType = "api"

	// Storage connectors (S3, GCS, Azure Blob, etc.)
	TypeStorage ConnectorType = "storage"

	// Messaging connectors (Kafka, RabbitMQ, NATS, etc.)
	TypeMessaging ConnectorType = "messaging"

	// Analytics connectors (Google Analytics, Mixpanel, etc.)
	TypeAnalytics ConnectorType = "analytics"

	// CRM connectors (Salesforce, HubSpot, etc.)
	TypeCRM ConnectorType = "crm"

	// Productivity connectors (Airtable, Notion, Slack, etc.)
	TypeProductivity ConnectorType = "productivity"

	// Authentication connectors (Auth0, OAuth, etc.)
	TypeAuth ConnectorType = "auth"

	// Cloud connectors (AWS, Azure, GCP, etc.)
	TypeCloud ConnectorType = "cloud"

	// Other/Generic connectors
	TypeOther ConnectorType = "other"
)

// ConnectorMetadata provides information about the connector's capabilities
type ConnectorMetadata struct {
	// Name of the connector
	Name string `json:"name"`

	// Display name for UI
	DisplayName string `json:"displayName"`

	// Description of what the connector does
	Description string `json:"description"`

	// Version of the connector
	Version string `json:"version"`

	// Type category
	Type ConnectorType `json:"type"`

	// Author/Maintainer
	Author string `json:"author"`

	// Documentation URL
	DocsURL string `json:"docsUrl"`

	// Capabilities supported by this connector
	Capabilities []Capability `json:"capabilities"`

	// Rate limits (if applicable)
	RateLimits *RateLimitInfo `json:"rateLimits,omitempty"`

	// Authentication methods supported
	AuthMethods []string `json:"authMethods"`

	// Tags for categorization
	Tags []string `json:"tags"`
}

// Capability represents a feature/capability of the connector
type Capability struct {
	// Name of the capability (e.g., "read", "write", "delete", "stream")
	Name string `json:"name"`

	// Description of the capability
	Description string `json:"description"`

	// Whether this capability is enabled
	Enabled bool `json:"enabled"`
}

// RateLimitInfo provides rate limit information
type RateLimitInfo struct {
	// Requests per second
	RequestsPerSecond int `json:"requestsPerSecond,omitempty"`

	// Requests per minute
	RequestsPerMinute int `json:"requestsPerMinute,omitempty"`

	// Requests per hour
	RequestsPerHour int `json:"requestsPerHour,omitempty"`

	// Requests per day
	RequestsPerDay int `json:"requestsPerDay,omitempty"`

	// Burst size (for token bucket algorithm)
	BurstSize int `json:"burstSize,omitempty"`
}

// ConfigurableConnector is an optional interface for connectors that support
// dynamic configuration updates
type ConfigurableConnector interface {
	Connector

	// UpdateConfig updates the connector configuration
	UpdateConfig(config interface{}) error

	// GetConfig returns the current configuration
	GetConfig() interface{}

	// ValidateConfig validates a configuration without applying it
	ValidateConfig(config interface{}) error
}

// StreamingConnector is an optional interface for connectors that support
// streaming/real-time data
type StreamingConnector interface {
	Connector

	// Subscribe subscribes to a stream/topic
	Subscribe(ctx context.Context, topic string, handler StreamHandler) error

	// Unsubscribe unsubscribes from a stream/topic
	Unsubscribe(topic string) error
}

// StreamHandler handles streaming data
type StreamHandler func(ctx context.Context, data interface{}) error

// BatchOperationConnector is an optional interface for connectors that support
// batch operations for efficiency
type BatchOperationConnector interface {
	Connector

	// BatchCreate creates multiple items in a single operation
	BatchCreate(ctx context.Context, items []interface{}) ([]interface{}, error)

	// BatchUpdate updates multiple items in a single operation
	BatchUpdate(ctx context.Context, items []interface{}) ([]interface{}, error)

	// BatchDelete deletes multiple items in a single operation
	BatchDelete(ctx context.Context, ids []string) error
}

// CacheableConnector is an optional interface for connectors that support caching
type CacheableConnector interface {
	Connector

	// EnableCache enables caching with the given TTL
	EnableCache(ttl int) error

	// DisableCache disables caching
	DisableCache() error

	// ClearCache clears the cache
	ClearCache() error

	// IsCacheEnabled returns whether caching is enabled
	IsCacheEnabled() bool
}

// TransactionalConnector is an optional interface for connectors that support transactions
type TransactionalConnector interface {
	Connector

	// BeginTransaction starts a new transaction
	BeginTransaction(ctx context.Context) (Transaction, error)
}

// Transaction represents a database transaction
type Transaction interface {
	// Commit commits the transaction
	Commit() error

	// Rollback rolls back the transaction
	Rollback() error

	// Context returns the transaction context
	Context() context.Context
}

// MonitorableConnector is an optional interface for connectors that provide monitoring metrics
type MonitorableConnector interface {
	Connector

	// GetMetrics returns current metrics
	GetMetrics() ConnectorMetrics

	// ResetMetrics resets all metrics
	ResetMetrics()
}

// ConnectorMetrics provides monitoring information
type ConnectorMetrics struct {
	// Total requests made
	TotalRequests int64 `json:"totalRequests"`

	// Successful requests
	SuccessfulRequests int64 `json:"successfulRequests"`

	// Failed requests
	FailedRequests int64 `json:"failedRequests"`

	// Average response time in milliseconds
	AvgResponseTime float64 `json:"avgResponseTime"`

	// Peak response time in milliseconds
	PeakResponseTime float64 `json:"peakResponseTime"`

	// Current connections/sessions
	ActiveConnections int `json:"activeConnections"`

	// Errors by type
	ErrorsByType map[string]int64 `json:"errorsByType"`

	// Custom metrics (connector-specific)
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// ConnectorRegistry manages available connectors
type ConnectorRegistry struct {
	connectors map[string]Connector
}

// NewConnectorRegistry creates a new connector registry
func NewConnectorRegistry() *ConnectorRegistry {
	return &ConnectorRegistry{
		connectors: make(map[string]Connector),
	}
}

// Register registers a connector
func (r *ConnectorRegistry) Register(connector Connector) error {
	name := connector.Name()
	if _, exists := r.connectors[name]; exists {
		return &core.EventBusError{
			Code:    "CONNECTOR_ALREADY_REGISTERED",
			Message: "connector already registered: " + name,
		}
	}
	r.connectors[name] = connector
	return nil
}

// Unregister unregisters a connector
func (r *ConnectorRegistry) Unregister(name string) {
	delete(r.connectors, name)
}

// Get returns a connector by name
func (r *ConnectorRegistry) Get(name string) (Connector, bool) {
	connector, exists := r.connectors[name]
	return connector, exists
}

// List returns all registered connectors
func (r *ConnectorRegistry) List() []Connector {
	connectors := make([]Connector, 0, len(r.connectors))
	for _, connector := range r.connectors {
		connectors = append(connectors, connector)
	}
	return connectors
}

// ListByType returns connectors of a specific type
func (r *ConnectorRegistry) ListByType(connectorType ConnectorType) []Connector {
	connectors := make([]Connector, 0)
	for _, connector := range r.connectors {
		if connector.Type() == connectorType {
			connectors = append(connectors, connector)
		}
	}
	return connectors
}

// Global connector registry
var globalRegistry = NewConnectorRegistry()

// Register registers a connector globally
func Register(connector Connector) error {
	return globalRegistry.Register(connector)
}

// Get returns a connector from the global registry
func Get(name string) (Connector, bool) {
	return globalRegistry.Get(name)
}

// List returns all globally registered connectors
func List() []Connector {
	return globalRegistry.List()
}

// ListByType returns globally registered connectors of a specific type
func ListByType(connectorType ConnectorType) []Connector {
	return globalRegistry.ListByType(connectorType)
}
