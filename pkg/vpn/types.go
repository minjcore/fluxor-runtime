package vpn

import (
	"context"
	"net"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// VPNServer represents a VPN server that can handle VPN connections
type VPNServer interface {
	// Start starts the VPN server
	Start() error

	// Stop stops the VPN server gracefully
	Stop() error

	// Metrics returns current VPN server metrics
	Metrics() ServerMetrics

	// Health returns the health status of the VPN server
	Health() map[string]interface{}

	// AddClient adds a VPN client
	AddClient(client Client) error

	// RemoveClient removes a VPN client
	RemoveClient(clientID string) error

	// GetClient returns a client by ID
	GetClient(clientID string) (*Client, error)

	// ListClients returns all connected clients
	ListClients() []Client

	// Authenticate authenticates a client connection
	Authenticate(username, password string) (*Client, error)
}

// Client represents a VPN client
type Client struct {
	// ID is the unique client identifier
	ID string `json:"id"`

	// Username for authentication
	Username string `json:"username"`

	// AssignedIP is the IP address assigned to this client
	AssignedIP net.IP `json:"assignedIP"`

	// RemoteAddr is the client's remote address
	RemoteAddr net.Addr `json:"remoteAddr"`

	// ConnectedAt is when the client connected
	ConnectedAt time.Time `json:"connectedAt"`

	// LastSeen is the last time the client was seen
	LastSeen time.Time `json:"lastSeen"`

	// BytesReceived is the total bytes received from client
	BytesReceived int64 `json:"bytesReceived"`

	// BytesSent is the total bytes sent to client
	BytesSent int64 `json:"bytesSent"`

	// Status is the client connection status
	Status ClientStatus `json:"status"`

	// Metadata for custom client information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ClientStatus represents the status of a VPN client
type ClientStatus string

const (
	ClientStatusConnecting ClientStatus = "connecting"
	ClientStatusConnected  ClientStatus = "connected"
	ClientStatusDisconnected ClientStatus = "disconnected"
	ClientStatusAuthenticated ClientStatus = "authenticated"
)

// VPNConnection represents an active VPN connection
type VPNConnection struct {
	*core.BaseRequestContext

	Context    context.Context
	Client     *Client
	Conn       net.Conn
	GoCMD      core.GoCMD
	EventBus   core.EventBus
	StartTime  time.Time
	TunnelIP   net.IP
}

// ServerMetrics provides VPN server performance metrics
type ServerMetrics struct {
	// Connection metrics
	TotalConnections    int64   `json:"totalConnections"`
	ActiveConnections   int64   `json:"activeConnections"`
	RejectedConnections int64   `json:"rejectedConnections"`
	FailedConnections   int64   `json:"failedConnections"`

	// Client metrics
	TotalClients       int64   `json:"totalClients"`
	ActiveClients      int64   `json:"activeClients"`
	AuthenticatedClients int64 `json:"authenticatedClients"`

	// Traffic metrics
	BytesReceived      int64   `json:"bytesReceived"`
	BytesSent          int64   `json:"bytesSent"`
	PacketsReceived    int64   `json:"packetsReceived"`
	PacketsSent        int64   `json:"packetsSent"`

	// Performance metrics
	AverageLatency     float64 `json:"averageLatency"` // milliseconds
	PeakLatency        float64 `json:"peakLatency"`    // milliseconds
}

// Authenticator handles client authentication
type Authenticator interface {
	// Authenticate authenticates a client with username and password
	Authenticate(username, password string) (bool, error)

	// GetClientInfo retrieves client information after authentication
	GetClientInfo(username string) (*ClientInfo, error)
}

// ClientInfo contains information about a client
type ClientInfo struct {
	ID       string
	Username string
	IP       net.IP
	Metadata map[string]interface{}
}

// VPNError, ConfigError, and ProtocolError are defined in errors.go
