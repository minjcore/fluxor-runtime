package udp

import (
	"context"
	"net"

	"github.com/fluxorio/fluxor/pkg/core"
)

// Server represents a UDP server abstraction, aligned with pkg/tcp.Server.
type Server interface {
	// Start starts the server (blocking, like TCP servers).
	Start() error

	// Stop stops the server gracefully.
	Stop() error

	// SetHandler sets the packet handler (fail-fast on nil).
	SetHandler(handler PacketHandler)

	// Metrics returns current server metrics.
	Metrics() ServerMetrics
}

// PacketHandler handles a single UDP packet.
// Implementations should be fail-fast and must not block forever.
// The handler receives the packet data and remote address.
type PacketHandler func(ctx *PacketContext) error

// Middleware is a UDP packet middleware, aligned with pkg/tcp.Middleware.
// It wraps a PacketHandler with extra behavior (auth, metrics, tracing, etc.).
type Middleware func(next PacketHandler) PacketHandler

// PacketContext provides per-packet context and convenient access to GoCMD/EventBus.
// Mirrors tcp.ConnContext's shape but for UDP packets (connectionless).
type PacketContext struct {
	*core.BaseRequestContext

	Context    context.Context
	Conn       net.PacketConn // PacketConn for sending responses
	GoCMD      core.GoCMD
	EventBus   core.EventBus
	Data       []byte   // Received packet data
	RemoteAddr net.Addr // Remote address that sent the packet
	LocalAddr  net.Addr // Local address that received the packet
}

// WriteTo writes data to the remote address that sent the packet.
// This is a convenience method for responding to UDP packets.
func (ctx *PacketContext) WriteTo(data []byte) (int, error) {
	return ctx.Conn.WriteTo(data, ctx.RemoteAddr)
}

// ServerMetrics provides UDP server performance metrics.
type ServerMetrics struct {
	QueuedPackets      int64   // Current queued packets
	DroppedPackets     int64   // Total dropped packets (backpressure/queue full)
	QueueCapacity      int     // Maximum queue capacity
	Workers            int     // Number of worker goroutines
	QueueUtilization   float64 // Queue utilization percentage
	NormalPPS           int     // Normal packets per second (target utilization baseline)
	CurrentPPS          int     // Current packets per second
	PPSUtilization      float64 // Utilization percentage (relative to normal PPS)
	TotalReceived       int64   // Total packets received
	HandledPackets      int64   // Total packets handled
	ErrorPackets        int64   // Total packets that returned error
	ActivePackets       int64   // Current in-flight packets (queued + handling)
	MaxPacketsPerSecond int     // Configured max packets per second (0 = unlimited)
}
