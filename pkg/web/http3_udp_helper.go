package web

import (
	"fmt"
	"net"

	"github.com/fluxorio/fluxor/pkg/udp"
)

// getOrCreatePacketConn gets or creates a UDP PacketConn for HTTP/3
// Priority: 1) Config.PacketConn, 2) Config.UDPServer, 3) Create new
func getOrCreatePacketConn(config *HTTP3ServerConfig) (net.PacketConn, error) {
	// Use provided PacketConn if available
	if config.PacketConn != nil {
		return config.PacketConn, nil
	}

	// Try to get PacketConn from UDP server if provided
	if config.UDPServer != nil {
		packetConn, err := getPacketConnFromUDPServer(config.UDPServer)
		if err != nil {
			return nil, fmt.Errorf("failed to get PacketConn from UDP server: %w", err)
		}
		if packetConn != nil {
			return packetConn, nil
		}
	}

	// Return nil - http3.Server will create its own PacketConn
	// This is the default behavior when no PacketConn is provided
	return nil, nil
}

// getPacketConnFromUDPServer gets the PacketConn from a UDP server
// The UDP server must be started for this to work
func getPacketConnFromUDPServer(udpServer *udp.UDPServer) (net.PacketConn, error) {
	// Check if UDP server is started by checking ListeningAddr
	// If it returns empty, server is not started
	listeningAddr := udpServer.ListeningAddr()
	if listeningAddr == "" {
		return nil, fmt.Errorf("UDP server is not started - start it before HTTP/3 server")
	}

	// Get PacketConn from UDP server
	packetConn := udpServer.PacketConn()
	if packetConn == nil {
		return nil, fmt.Errorf("UDP server PacketConn is nil - server may not be fully started")
	}

	return packetConn, nil
}
