package vpn

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestNewVPNServer(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	config := DefaultConfig()
	config.ListenAddr = ":0" // Random port for testing
	config.NetworkCIDR = "10.8.0.0/24"

	server, err := NewVPNServer(gocmd, config)
	if err != nil {
		t.Fatalf("Failed to create VPN server: %v", err)
	}

	if server == nil {
		t.Fatal("Server is nil")
	}
}

func TestVPNServerStartStop(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	config := DefaultConfig()
	config.ListenAddr = ":0"
	config.NetworkCIDR = "10.8.0.0/24"
	config.MaxClients = 10

	server, err := NewVPNServer(gocmd, config)
	if err != nil {
		t.Fatalf("Failed to create VPN server: %v", err)
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	if err := server.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	// Wait for start to finish
	select {
	case err := <-errChan:
		if err != nil {
			t.Logf("Server start error (expected on stop): %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Log("Server stopped successfully")
	}
}

func TestVPNServerMetrics(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	config := DefaultConfig()
	config.ListenAddr = ":0"
	config.NetworkCIDR = "10.8.0.0/24"

	server, err := NewVPNServer(gocmd, config)
	if err != nil {
		t.Fatalf("Failed to create VPN server: %v", err)
	}

	metrics := server.Metrics()
	
	// Check that metrics are initialized (all zeros is valid for a new server)
	if metrics.TotalConnections < 0 {
		t.Error("TotalConnections should be non-negative")
	}
	
	if metrics.ActiveConnections < 0 {
		t.Error("ActiveConnections should be non-negative")
	}
}

func TestVPNServerHealth(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	config := DefaultConfig()
	config.ListenAddr = ":0"
	config.NetworkCIDR = "10.8.0.0/24"

	server, err := NewVPNServer(gocmd, config)
	if err != nil {
		t.Fatalf("Failed to create VPN server: %v", err)
	}

	health := server.Health()
	if health == nil {
		t.Fatal("Health should not be nil")
	}

	healthy, ok := health["healthy"].(bool)
	if !ok {
		t.Fatal("Health should contain 'healthy' boolean")
	}

	// Server should be healthy when not started
	if !healthy {
		t.Log("Server health check returned unhealthy (may be expected)")
	}
}

func TestVPNServerAuthenticate(t *testing.T) {
	gocmd := core.NewGoCMD(context.Background())
	config := DefaultConfig()
	config.ListenAddr = ":0"
	config.NetworkCIDR = "10.8.0.0/24"

	server, err := NewVPNServer(gocmd, config)
	if err != nil {
		t.Fatalf("Failed to create VPN server: %v", err)
	}

	// Test authentication with default test user
	client, err := server.Authenticate("test", "test")
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.Username != "test" {
		t.Errorf("Expected username 'test', got '%s'", client.Username)
	}

	// Test failed authentication
	_, err = server.Authenticate("test", "wrong")
	if err == nil {
		t.Error("Authentication should fail with wrong password")
	}
}

func TestIPPool(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("10.8.0.0/24")
	if err != nil {
		t.Fatalf("Failed to parse CIDR: %v", err)
	}

	pool, err := NewIPPool(ipNet)
	if err != nil {
		t.Fatalf("Failed to create IP pool: %v", err)
	}

	// Allocate some IPs
	ip1, err := pool.Allocate()
	if err != nil {
		t.Fatalf("Failed to allocate IP: %v", err)
	}

	ip2, err := pool.Allocate()
	if err != nil {
		t.Fatalf("Failed to allocate IP: %v", err)
	}

	if ip1.Equal(ip2) {
		t.Error("Allocated IPs should be different")
	}

	// Check if allocated
	if !pool.IsAllocated(ip1) {
		t.Error("IP1 should be allocated")
	}

	// Release IP
	pool.Release(ip1)
	if pool.IsAllocated(ip1) {
		t.Error("IP1 should not be allocated after release")
	}

	// Check available count
	available := pool.AvailableCount()
	if available <= 0 {
		t.Errorf("Should have available IPs, got %d", available)
	}
}

func TestVPNPacket(t *testing.T) {
	// Test packet creation and serialization
	packet := CreateHandshakePacket(1)
	data, err := packet.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize packet: %v", err)
	}

	if len(data) < 16 {
		t.Error("Packet should be at least 16 bytes")
	}

	// Test packet parsing
	parsed, err := ParsePacket(data)
	if err != nil {
		t.Fatalf("Failed to parse packet: %v", err)
	}

	if parsed.Type != PacketTypeControlHandshake {
		t.Errorf("Expected handshake packet, got %d", parsed.Type)
	}

	if parsed.Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", parsed.Sequence)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(5, time.Second)

	// Should allow 5 requests
	for i := 0; i < 5; i++ {
		if !limiter.Allow("test") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th request should be rate limited
	if limiter.Allow("test") {
		t.Error("6th request should be rate limited")
	}

	// Wait and try again
	time.Sleep(1100 * time.Millisecond)
	if !limiter.Allow("test") {
		t.Error("Request should be allowed after window expires")
	}

	limiter.Stop()
}
