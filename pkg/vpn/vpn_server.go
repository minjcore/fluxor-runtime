package vpn

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/songgao/water"
)

// clientCryptoState giữ AEAD + sendSeq cho mỗi client (ChaCha20-Poly1305).
type clientCryptoState struct {
	aead    cipherAEAD
	sendSeq uint64
}

// vpnServer implements VPNServer interface
type vpnServer struct {
	*core.BaseServer

	config      Config
	listener    net.Listener
	udpConn     *net.UDPConn
	gatewayTUN  *water.Interface // TUN gateway để forward ra Internet (chỉ khi EnableForwarding)
	gatewayTUNMu sync.Mutex      // bảo vệ Write

	mu            sync.RWMutex
	clients       map[string]*Client
	ipToClient    map[string]*Client // AssignedIP.String() -> client (UDP forward)
	clientCrypto  map[string]*clientCryptoState
	ipPool        *IPPool
	authenticator Authenticator
	rateLimiter *ConnectionRateLimiter
	tlsConfig   *tls.Config

	// Connection state tracking
	connectionStates map[string]ConnectionState
	connectionSeqs   map[string]uint32

	// Metrics (atomic for thread-safety)
	totalConnections    int64
	activeConnections   int64
	rejectedConnections int64
	failedConnections   int64
	totalClients        int64
	activeClients       int64
	authenticatedClients int64
	bytesReceived       int64
	bytesSent           int64
	packetsReceived     int64
	packetsSent         int64

	// Connection tracking
	activeConns map[net.Conn]bool
	connMu      sync.RWMutex
	stopping    int32 // atomic: server stopping flag

	// Keep-alive
	keepAliveCancel context.CancelFunc
	keepAliveCtx    context.Context
}

// NewVPNServer creates a new VPN server
// Fail-fast: Validates configuration
func NewVPNServer(gocmd core.GoCMD, config Config) (VPNServer, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create base server
	baseServer := core.NewBaseServer("vpn-server", gocmd)

	// Parse network CIDR
	_, ipNet, err := net.ParseCIDR(config.NetworkCIDR)
	if err != nil {
		return nil, &ConfigError{
			Code:    "INVALID_NETWORK_CIDR",
			Message: fmt.Sprintf("Invalid network CIDR: %v", err),
		}
	}

	// Create IP pool
	ipPool, err := NewIPPool(ipNet)
	if err != nil {
		return nil, &ConfigError{
			Code:    "IP_POOL_ERROR",
			Message: fmt.Sprintf("Failed to create IP pool: %v", err),
		}
	}

	server := &vpnServer{
		BaseServer:       baseServer,
		config:           config,
		clients:          make(map[string]*Client),
		ipToClient:       make(map[string]*Client),
		clientCrypto:     make(map[string]*clientCryptoState),
		ipPool:           ipPool,
		activeConns:      make(map[net.Conn]bool),
		authenticator:    NewPasswordAuthenticator(), // Default authenticator
		connectionStates: make(map[string]ConnectionState),
		connectionSeqs:   make(map[string]uint32),
	}

	// Initialize rate limiter (max 10 connections per minute per IP)
	server.rateLimiter = NewConnectionRateLimiter(10, time.Minute)

	// Load TLS configuration if certificate auth is enabled
	if config.AuthenticationMethod == "certificate" && config.CertificatePath != "" {
		tlsCfg, err := LoadTLSConfig(TLSConfig{
			CertFile:   config.CertificatePath,
			KeyFile:    config.PrivateKeyPath,
			CAFile:     config.CACertificatePath,
			MinVersion: tls.VersionTLS12,
		})
		if err != nil {
			return nil, &ConfigError{
				Code:    "TLS_CONFIG_ERROR",
				Message: fmt.Sprintf("Failed to load TLS config: %v", err),
			}
		}
		server.tlsConfig = tlsCfg
	}

	// Set hooks
	server.BaseServer.SetHooks(server.doStart, server.doStop)

	return server, nil
}

// doStart starts the VPN server
func (s *vpnServer) doStart() error {
	// Start keep-alive
	s.startKeepAlive()

	// Start appropriate server based on protocol
	switch s.config.Protocol {
	case "udp":
		return s.startUDPServer()
	case "tcp":
		return s.startTCPServer()
	default:
		return fmt.Errorf("unsupported protocol: %s", s.config.Protocol)
	}
}

// doStop stops the VPN server
func (s *vpnServer) doStop() error {
	// Mark as stopping
	atomic.StoreInt32(&s.stopping, 1)

	// Stop keep-alive
	if s.keepAliveCancel != nil {
		s.keepAliveCancel()
	}

	// Stop listener
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.logger().Error("Listener close error", "error", err)
		}
	}

	if s.gatewayTUN != nil {
		s.gatewayTUN.Close()
		s.gatewayTUN = nil
	}
	if s.udpConn != nil {
		if err := s.udpConn.Close(); err != nil {
			s.logger().Error("UDP connection close error", "error", err)
		}
	}

	// Close all active connections
	s.connMu.Lock()
	for conn := range s.activeConns {
		conn.Close()
		delete(s.activeConns, conn)
	}
	s.connMu.Unlock()

	// Disconnect all clients
	s.mu.Lock()
	for _, client := range s.clients {
		client.Status = ClientStatusDisconnected
		// Release IPs
		if client.AssignedIP != nil {
			s.ipPool.Release(client.AssignedIP)
		}
	}
	s.clients = make(map[string]*Client)
	s.connectionStates = make(map[string]ConnectionState)
	s.connectionSeqs = make(map[string]uint32)
	s.mu.Unlock()

	// Stop rate limiter
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}

	return nil
}

// startTCPServer starts the TCP VPN server
func (s *vpnServer) startTCPServer() error {
	var listener net.Listener
	var err error

	// Use TLS if configured
	if s.tlsConfig != nil {
		listener, err = tls.Listen("tcp", s.config.ListenAddr, s.tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to listen with TLS: %w", err)
		}
		s.logger().Info("TCP VPN server starting with TLS", "addr", s.config.ListenAddr)
	} else {
		listener, err = net.Listen("tcp", s.config.ListenAddr)
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
		s.logger().Info("TCP VPN server starting", "addr", s.config.ListenAddr)
	}

	s.listener = listener

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check if server is stopping
			if atomic.LoadInt32(&s.stopping) == 1 {
				return nil
			}
			return fmt.Errorf("accept error: %w", err)
		}

		// Check rate limiting
		clientIP := conn.RemoteAddr().(*net.TCPAddr).IP.String()
		if !s.rateLimiter.Allow(clientIP) {
			conn.Close()
			atomic.AddInt64(&s.rejectedConnections, 1)
			s.logger().WithFields(map[string]interface{}{"ip": clientIP}).Error("Connection rate limited")
			continue
		}

		// Check max clients
		active := atomic.LoadInt64(&s.activeClients)
		if s.config.MaxClients > 0 && active >= int64(s.config.MaxClients) {
			conn.Close()
			atomic.AddInt64(&s.rejectedConnections, 1)
			continue
		}

		atomic.AddInt64(&s.totalConnections, 1)
		atomic.AddInt64(&s.activeConnections, 1)

		// Track connection
		s.connMu.Lock()
		s.activeConns[conn] = true
		s.connMu.Unlock()

		// Handle connection in goroutine
		go s.handleConnection(conn)
	}
}

// startUDPServer starts the UDP VPN server
func (s *vpnServer) startUDPServer() error {
	if s.config.EnableForwarding {
		if err := s.startGatewayTUN(); err != nil {
			s.logger().Error("Gateway TUN failed (forwarding disabled)", "error", err)
		}
	}

	addr, err := net.ResolveUDPAddr("udp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen UDP: %w", err)
	}

	s.udpConn = conn
	s.logger().Info("UDP VPN server starting", "addr", s.config.ListenAddr)

	buffer := make([]byte, 4096)
	for {
		// Check if stopping
		if atomic.LoadInt32(&s.stopping) == 1 {
			return nil
		}

		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Timeout is expected, continue
			}
			if atomic.LoadInt32(&s.stopping) == 1 {
				return nil
			}
			continue
		}

		// Không rate-limit từng gói UDP: ConnectionRateLimiter (10/phút/IP) trước đây áp vào
		// mọi datagram → sau ~10 gói (handshake + ping) client bị drop toàn bộ. Chỉ TCP accept
		// dùng rate limit (new connection).

		// Check max clients
		active := atomic.LoadInt64(&s.activeClients)
		if s.config.MaxClients > 0 && active >= int64(s.config.MaxClients) {
			atomic.AddInt64(&s.rejectedConnections, 1)
			continue
		}

		atomic.AddInt64(&s.totalConnections, 1)
		atomic.AddInt64(&s.packetsReceived, 1)

		// Copy payload + địa chỉ trước khi vào goroutine: vòng lặp tái dùng buffer và
		// net có thể tái dùng *UDPAddr — nếu không copy, gói bị ghi đè → decrypt lỗi / mất reply (ping chập chờn).
		pkt := make([]byte, n)
		copy(pkt, buffer[:n])
		addrCopy := &net.UDPAddr{
			IP:   append(net.IP(nil), clientAddr.IP...),
			Port: clientAddr.Port,
			Zone: clientAddr.Zone,
		}
		go s.handleUDPPacket(conn, addrCopy, pkt)
	}
}

// startGatewayTUN tạo TUN trên server để forward gói ra Internet (chỉ Linux).
func (s *vpnServer) startGatewayTUN() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("gateway TUN chỉ hỗ trợ Linux")
	}
	tun, err := water.New(water.Config{DeviceType: water.TUN})
	if err != nil {
		return err
	}
	s.gatewayTUN = tun
	name := tun.Name()
	gwIP := s.config.GatewayTUNIP
	if gwIP == "" {
		gwIP = "10.8.0.254"
	}
	_, ipNet, err := net.ParseCIDR(s.config.NetworkCIDR)
	if err != nil || ipNet == nil {
		_, ipNet, _ = net.ParseCIDR("10.8.0.0/24")
	}
	ones, _ := ipNet.Mask.Size()
	gwCIDR := fmt.Sprintf("%s/%d", gwIP, ones)
	cmd := exec.Command("ip", "addr", "add", gwCIDR, "dev", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		s.logger().Error("ip addr add failed", "out", string(out), "error", err)
	}
	cmd = exec.Command("ip", "link", "set", name, "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		s.logger().Error("ip link set up failed", "out", string(out), "error", err)
	}
	s.logger().Info("Gateway TUN up", "iface", name, "ip", gwCIDR)
	go s.gatewayTUNReadLoop()
	return nil
}

// gatewayTUNReadLoop đọc gói từ TUN (reply từ Internet), gửi về client theo dst IP.
func (s *vpnServer) gatewayTUNReadLoop() {
	buf := make([]byte, 2000)
	for s.gatewayTUN != nil {
		n, err := s.gatewayTUN.Read(buf)
		if err != nil {
			if s.gatewayTUN != nil {
				s.logger().Error("Gateway TUN read", "error", err)
			}
			return
		}
		if n < 20 || buf[0]>>4 != 4 {
			continue
		}
		dstIP := net.IP(buf[16:20])
		s.mu.RLock()
		target := s.ipToClient[dstIP.String()]
		s.mu.RUnlock()
		if target == nil {
			continue
		}
		payload := s.encryptPayloadForClient(target, buf[:n])
		forwardData, _ := CreateDataPacket(payload, 0).Serialize()
		s.sendDataToClient(target, forwardData)
	}
}

// handleConnection handles a TCP VPN connection
func (s *vpnServer) handleConnection(conn net.Conn) {
	defer func() {
		atomic.AddInt64(&s.activeConnections, -1)
		s.connMu.Lock()
		delete(s.activeConns, conn)
		s.connMu.Unlock()
		conn.Close()
	}()

	// Set timeouts
	conn.SetReadDeadline(time.Now().Add(s.config.IdleTimeout))
	conn.SetWriteDeadline(time.Now().Add(s.config.IdleTimeout))

	// Create VPN connection context
	vpnConn := &VPNConnection{
		BaseRequestContext: core.NewBaseRequestContext(),
		Context:           context.Background(),
		Conn:              conn,
		GoCMD:             s.GoCMD(),
		EventBus:          s.EventBus(),
		StartTime:         time.Now(),
	}

	// Handle VPN protocol
	if err := s.handleVPNProtocol(vpnConn); err != nil {
		s.logger().WithFields(map[string]interface{}{
			"error":  err.Error(),
			"remote": conn.RemoteAddr().String(),
		}).Error("VPN connection error")
		atomic.AddInt64(&s.failedConnections, 1)
		
		// Publish error event
		eventBus := s.EventBus()
		if eventBus != nil {
			eventBus.Publish("vpn.connection.error", map[string]interface{}{
				"remote": conn.RemoteAddr().String(),
				"error":  err.Error(),
			})
		}
	}
}

// handleUDPPacket handles a UDP VPN packet
func (s *vpnServer) handleUDPPacket(conn *net.UDPConn, clientAddr *net.UDPAddr, data []byte) {
	atomic.AddInt64(&s.bytesReceived, int64(len(data)))

	// Find or create client
	clientID := clientAddr.String()
	client := s.getOrCreateClient(clientID, clientAddr)

	// Update client stats
	atomic.AddInt64(&client.BytesReceived, int64(len(data)))
	client.LastSeen = time.Now()

	// Handle VPN protocol packet
	response, err := s.handleVPNPacket(client, data)
	if err != nil {
		s.logger().Error("VPN packet error", "error", err, "client", clientID)
		return
	}

	// Send response if any
	if response != nil {
		_, err := conn.WriteToUDP(response, clientAddr)
		if err != nil {
			s.logger().Error("Failed to send UDP response", "error", err)
			return
		}
		atomic.AddInt64(&s.bytesSent, int64(len(response)))
		atomic.AddInt64(&s.packetsSent, 1)
		atomic.AddInt64(&client.BytesSent, int64(len(response)))
	}
}

// handleVPNProtocol handles VPN protocol for TCP connections
func (s *vpnServer) handleVPNProtocol(vpnConn *VPNConnection) error {
	// Read initial packet
	buffer := make([]byte, 4096)
	
	// Set read deadline for initial handshake
	vpnConn.Conn.SetReadDeadline(time.Now().Add(s.config.ConnectionTimeout))
	n, err := vpnConn.Conn.Read(buffer)
	if err != nil {
		return NewVPNErrorWithError(ErrCodeTimeout, "failed to read initial packet", err)
	}

	atomic.AddInt64(&s.bytesReceived, int64(n))
	atomic.AddInt64(&s.packetsReceived, 1)

	// Create or get client
	clientID := vpnConn.Conn.RemoteAddr().String()
	client := s.getOrCreateClient(clientID, vpnConn.Conn.RemoteAddr())
	if client == nil {
		return NewVPNError(ErrCodeIPPoolExhausted, "failed to allocate IP for client")
	}
	vpnConn.Client = client

	s.logger().WithFields(map[string]interface{}{
		"client": clientID,
		"size":   n,
	}).Debug("Processing VPN packet")

	// Handle packet
	response, err := s.handleVPNPacket(client, buffer[:n])
	if err != nil {
		s.logger().WithFields(map[string]interface{}{
		"error":  err.Error(),
		"client": clientID,
	}).Error("Failed to handle VPN packet")
		return NewVPNErrorWithError(ErrCodeInvalidPacket, "packet handling failed", err)
	}

	// Send response if any
	if response != nil {
		_, err := vpnConn.Conn.Write(response)
		if err != nil {
			return fmt.Errorf("write error: %w", err)
		}
		atomic.AddInt64(&s.bytesSent, int64(len(response)))
		atomic.AddInt64(&s.packetsSent, 1)
		atomic.AddInt64(&client.BytesSent, int64(len(response)))
	}

	// Keep connection alive and handle more packets
	for {
		// Set read deadline
		vpnConn.Conn.SetReadDeadline(time.Now().Add(s.config.IdleTimeout))

		n, err := vpnConn.Conn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout - send keep-alive
				continue
			}
			return fmt.Errorf("read error: %w", err)
		}

		atomic.AddInt64(&s.bytesReceived, int64(n))
		atomic.AddInt64(&s.packetsReceived, 1)
		atomic.AddInt64(&client.BytesReceived, int64(n))
		client.LastSeen = time.Now()

		// Handle packet
		response, err := s.handleVPNPacket(client, buffer[:n])
		if err != nil {
			return err
		}

		// Send response if any
		if response != nil {
			_, err := vpnConn.Conn.Write(response)
			if err != nil {
				return fmt.Errorf("write error: %w", err)
			}
			atomic.AddInt64(&s.bytesSent, int64(len(response)))
			atomic.AddInt64(&s.packetsSent, 1)
			atomic.AddInt64(&client.BytesSent, int64(len(response)))
		}
	}
}

// handleVPNPacket handles a VPN protocol packet
func (s *vpnServer) handleVPNPacket(client *Client, data []byte) ([]byte, error) {
	// Parse VPN packet
	packet, err := ParsePacket(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse packet: %w", err)
	}

	// Get connection state
	s.mu.RLock()
	state, exists := s.connectionStates[client.ID]
	seq := s.connectionSeqs[client.ID]
	s.mu.RUnlock()

	if !exists {
		state = StateInitial
		seq = 0
	}

	// Handle packet based on type and state
	switch packet.Type {
	case PacketTypeControlHandshake:
		return s.handleHandshake(client, packet, &state, &seq)
	case PacketTypeControlAuth:
		return s.handleAuth(client, packet, &state, &seq)
	case PacketTypeControlConfig:
		return s.handleConfig(client, packet, &state, &seq)
	case PacketTypeControlKeepAlive:
		return s.handleKeepAlive(client, packet, &state, &seq)
	case PacketTypeControlDisconnect:
		return s.handleDisconnect(client, packet, &state, &seq)
	case PacketTypeData:
		if state != StateConnected {
			return nil, fmt.Errorf("data packet received in non-connected state: %d", state)
		}
		return s.handleData(client, packet, &seq)
	default:
		return nil, fmt.Errorf("unknown packet type: %d", packet.Type)
	}
}

// handleHandshake handles handshake packets
func (s *vpnServer) handleHandshake(client *Client, packet *VPNPacket, state *ConnectionState, seq *uint32) ([]byte, error) {
	if *state != StateInitial {
		return nil, fmt.Errorf("handshake received in invalid state: %d", *state)
	}

	*state = StateHandshaking
	*seq = packet.Sequence + 1

	// Update connection state
	s.mu.Lock()
	s.connectionStates[client.ID] = *state
	s.connectionSeqs[client.ID] = *seq
	s.mu.Unlock()

	// Send handshake response
	response := CreateHandshakePacket(*seq)
	return response.Serialize()
}

// handleAuth handles authentication packets
func (s *vpnServer) handleAuth(client *Client, packet *VPNPacket, state *ConnectionState, seq *uint32) ([]byte, error) {
	if *state != StateHandshaking {
		return nil, NewVPNError(ErrCodeInvalidState, fmt.Sprintf("auth packet received in invalid state: %d", *state))
	}

	*state = StateAuthenticating
	*seq = packet.Sequence + 1

	// Parse auth data
	username, password, err := ParseAuthData(packet)
	if err != nil {
		return nil, NewVPNErrorWithError(ErrCodeInvalidPacket, "failed to parse auth data", err)
	}

	s.logger().WithFields(map[string]interface{}{
		"client":   client.ID,
		"username": username,
	}).Info("Authenticating client")

	// Authenticate
	authenticated, err := s.authenticator.Authenticate(username, password)
	if err != nil || !authenticated {
		*state = StateDisconnected
		s.mu.Lock()
		s.connectionStates[client.ID] = *state
		s.mu.Unlock()
		s.logger().WithFields(map[string]interface{}{
			"client":   client.ID,
			"username": username,
		}).Error("Authentication failed")
		return nil, NewVPNError(ErrCodeAuthenticationFailed, "authentication failed")
	}

	// Get client info
	clientInfo, err := s.authenticator.GetClientInfo(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get client info: %w", err)
	}

	// Update client
	s.mu.Lock()
	client.Username = username
	client.Status = ClientStatusAuthenticated
	if clientInfo.IP != nil {
		client.AssignedIP = clientInfo.IP
	}
	s.mu.Unlock()

	*state = StateConfiguring
	s.mu.Lock()
	s.connectionStates[client.ID] = *state
	s.mu.Unlock()

	// ChaCha20-Poly1305: salt + derive key, gửi salt trong auth response
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("random salt: %w", err)
	}
	key := DeriveKey(salt, password)
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.clientCrypto[client.ID] = &clientCryptoState{aead: aead, sendSeq: 0}
	s.mu.Unlock()

	atomic.AddInt64(&s.authenticatedClients, 1)

	response := CreateAuthResponseWithSalt(*seq, salt)
	return response.Serialize()
}

// handleConfig handles configuration packets
func (s *vpnServer) handleConfig(client *Client, packet *VPNPacket, state *ConnectionState, seq *uint32) ([]byte, error) {
	if *state != StateConfiguring {
		return nil, fmt.Errorf("config packet received in invalid state: %d", *state)
	}

	*state = StateConnected
	*seq = packet.Sequence + 1

	// Update connection state
	s.mu.Lock()
	s.connectionStates[client.ID] = *state
	s.connectionSeqs[client.ID] = *seq
	client.Status = ClientStatusConnected
	s.mu.Unlock()

	// Publish event
	eventBus := s.EventBus()
	if eventBus != nil {
		eventBus.Publish("vpn.client.connected", map[string]interface{}{
			"clientID":    client.ID,
			"username":    client.Username,
			"assignedIP":  client.AssignedIP.String(),
			"remoteAddr":  client.RemoteAddr.String(),
		})
	}

	// Send config response with DNS và routes (push)
	dns := s.config.DNSServers
	if len(dns) == 0 {
		dns = []string{"8.8.8.8", "8.8.4.4"}
	}
	routes := s.config.Routes
	if len(routes) == 0 {
		routes = []string{s.config.NetworkCIDR}
	}
	response := CreateConfigResponsePacket(*seq, dns, routes)
	return response.Serialize()
}

// handleKeepAlive handles keep-alive packets
func (s *vpnServer) handleKeepAlive(client *Client, packet *VPNPacket, state *ConnectionState, seq *uint32) ([]byte, error) {
	*seq = packet.Sequence + 1

	// Update client last seen
	s.mu.Lock()
	client.LastSeen = time.Now()
	s.connectionSeqs[client.ID] = *seq
	s.mu.Unlock()

	// Send keep-alive response
	response := CreateKeepAlivePacket(*seq)
	return response.Serialize()
}

// handleDisconnect handles disconnect packets
func (s *vpnServer) handleDisconnect(client *Client, packet *VPNPacket, state *ConnectionState, seq *uint32) ([]byte, error) {
	*state = StateDisconnected
	*seq = packet.Sequence + 1

	// Update connection state
	s.mu.Lock()
	s.connectionStates[client.ID] = *state
	client.Status = ClientStatusDisconnected
	s.mu.Unlock()

	// Release IP, forward map, và crypto
	s.mu.Lock()
	delete(s.clientCrypto, client.ID)
	if client.AssignedIP != nil {
		delete(s.ipToClient, client.AssignedIP.String())
		s.ipPool.Release(client.AssignedIP)
	}
	s.mu.Unlock()

	// Publish event
	eventBus := s.EventBus()
	if eventBus != nil {
		eventBus.Publish("vpn.client.disconnected", map[string]interface{}{
			"clientID": client.ID,
			"username": client.Username,
		})
	}

	return nil, nil // No response needed
}

// handleData handles data packets: forward to client by dst IP, or ICMP reply for .0, or EchoData fallback.
func (s *vpnServer) handleData(client *Client, packet *VPNPacket, seq *uint32) ([]byte, error) {
	*seq = packet.Sequence + 1

	s.mu.Lock()
	client.LastSeen = time.Now()
	s.connectionSeqs[client.ID] = *seq
	s.mu.Unlock()

	data := packet.Data
	s.mu.RLock()
	cc := s.clientCrypto[client.ID]
	s.mu.RUnlock()
	if cc != nil {
		plain, err := DecryptData(cc.aead, data)
		if err != nil {
			s.logger().Error("Decrypt Data failed", "client", client.ID, "error", err)
			resp, _ := CreateACKPacket(*seq).Serialize()
			return resp, nil
		}
		data = plain
	}

	if len(data) < 20 || data[0]>>4 != 4 {
		return s.ackOrEcho(client, packet, seq, data)
	}

	dstIP := net.IP(data[16:20])
	ipNet := s.ipPool.Network()
	if ipNet == nil || !ipNet.Contains(dstIP) {
		// Dst ngoài pool: forward ra Internet qua gateway TUN (nếu bật)
		if s.gatewayTUN != nil {
			s.gatewayTUNMu.Lock()
			_, _ = s.gatewayTUN.Write(data)
			s.gatewayTUNMu.Unlock()
		}
		return nil, nil
	}

	dstKey := dstIP.String()
	s.mu.RLock()
	target := s.ipToClient[dstKey]
	s.mu.RUnlock()

	if target != nil && target.ID != client.ID {
		payload := s.encryptPayloadForClient(target, data)
		forwardData, _ := CreateDataPacket(payload, 0).Serialize()
		if s.sendDataToClient(target, forwardData) {
			return nil, nil
		}
	}

	if len(data) >= 28 && data[9] == 1 && data[20] == 8 {
		reply := make([]byte, len(data))
		copy(reply, data)
		copy(reply[12:16], data[16:20])
		copy(reply[16:20], data[12:16])
		reply[20] = 0
		reply[22], reply[23] = 0, 0
		binary.BigEndian.PutUint16(reply[22:24], icmpChecksum(reply[20:]))
		reply[10], reply[11] = 0, 0
		binary.BigEndian.PutUint16(reply[10:12], ipChecksum(reply[:20]))
		payload := s.encryptPayloadForClient(client, reply)
		resp, _ := CreateDataPacket(payload, *seq).Serialize()
		return resp, nil
	}

	return s.ackOrEcho(client, packet, seq, data)
}

// ackOrEcho: nếu EchoData thì echo gói (plaintext đã decrypt), không thì ACK.
func (s *vpnServer) ackOrEcho(client *Client, packet *VPNPacket, seq *uint32, plaintext []byte) ([]byte, error) {
	if s.config.EchoData && len(plaintext) > 0 {
		payload := s.encryptPayloadForClient(client, plaintext)
		resp, _ := CreateDataPacket(payload, *seq).Serialize()
		return resp, nil
	}
	resp, _ := CreateACKPacket(*seq).Serialize()
	return resp, nil
}

// encryptPayloadForClient: nếu client có crypto thì encrypt plaintext (nonce+Seal), tăng sendSeq; không thì trả về plaintext.
func (s *vpnServer) encryptPayloadForClient(client *Client, plaintext []byte) []byte {
	s.mu.Lock()
	cc := s.clientCrypto[client.ID]
	if cc == nil {
		s.mu.Unlock()
		return plaintext
	}
	enc, _ := EncryptData(cc.aead, plaintext, cc.sendSeq)
	cc.sendSeq++
	s.mu.Unlock()
	return enc
}

// sendDataToClient gửi data (VPN Data packet) tới client qua UDP. Trả về true nếu gửi thành công.
func (s *vpnServer) sendDataToClient(target *Client, data []byte) bool {
	if s.udpConn == nil || target.RemoteAddr == nil {
		return false
	}
	udpAddr, ok := target.RemoteAddr.(*net.UDPAddr)
	if !ok {
		return false
	}
	_, err := s.udpConn.WriteToUDP(data, udpAddr)
	if err != nil {
		s.logger().Error("Forward to client failed", "client", target.ID, "error", err)
		return false
	}
	atomic.AddInt64(&s.bytesSent, int64(len(data)))
	atomic.AddInt64(&s.packetsSent, 1)
	atomic.AddInt64(&target.BytesSent, int64(len(data)))
	return true
}

// getOrCreateClient gets or creates a client
func (s *vpnServer) getOrCreateClient(clientID string, remoteAddr net.Addr) *Client {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, exists := s.clients[clientID]
	if !exists {
		// Assign IP from pool
		assignedIP, err := s.ipPool.Allocate()
		if err != nil {
			s.logger().Error("Failed to allocate IP", "error", err)
			return nil
		}

		client = &Client{
			ID:          clientID,
			RemoteAddr:  remoteAddr,
			AssignedIP:  assignedIP,
			ConnectedAt: time.Now(),
			LastSeen:    time.Now(),
			Status:      ClientStatusConnecting,
			Metadata:    make(map[string]interface{}),
		}

		s.clients[clientID] = client
		s.ipToClient[assignedIP.String()] = client
		atomic.AddInt64(&s.totalClients, 1)
		atomic.AddInt64(&s.activeClients, 1)
	}

	return client
}

// startKeepAlive starts keep-alive monitoring
func (s *vpnServer) startKeepAlive() {
	s.keepAliveCtx, s.keepAliveCancel = context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(s.config.KeepAliveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-s.keepAliveCtx.Done():
				return
			case <-ticker.C:
				s.checkKeepAlive()
			}
		}
	}()
}

// checkKeepAlive checks and removes idle clients
func (s *vpnServer) checkKeepAlive() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for clientID, client := range s.clients {
		if now.Sub(client.LastSeen) > s.config.IdleTimeout {
			s.logger().Info("Disconnecting idle client", "client", clientID)
			delete(s.clientCrypto, clientID)
			if client.AssignedIP != nil {
				delete(s.ipToClient, client.AssignedIP.String())
				s.ipPool.Release(client.AssignedIP)
			}
			delete(s.clients, clientID)
			atomic.AddInt64(&s.activeClients, -1)
			client.Status = ClientStatusDisconnected
		}
	}
}

// Metrics returns current VPN server metrics
func (s *vpnServer) Metrics() ServerMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	authenticatedCount := int64(0)
	totalLatency := float64(0)
	peakLatency := float64(0)
	latencyCount := int64(0)

	for _, client := range s.clients {
		if client.Status == ClientStatusAuthenticated || client.Status == ClientStatusConnected {
			authenticatedCount++
		}
		// Calculate average latency from connection duration
		if !client.ConnectedAt.IsZero() {
			latency := time.Since(client.ConnectedAt).Milliseconds()
			totalLatency += float64(latency)
			latencyCount++
			if float64(latency) > peakLatency {
				peakLatency = float64(latency)
			}
		}
	}

	avgLatency := float64(0)
	if latencyCount > 0 {
		avgLatency = totalLatency / float64(latencyCount)
	}

	return ServerMetrics{
		TotalConnections:     atomic.LoadInt64(&s.totalConnections),
		ActiveConnections:    atomic.LoadInt64(&s.activeConnections),
		RejectedConnections:  atomic.LoadInt64(&s.rejectedConnections),
		FailedConnections:    atomic.LoadInt64(&s.failedConnections),
		TotalClients:         atomic.LoadInt64(&s.totalClients),
		ActiveClients:        atomic.LoadInt64(&s.activeClients),
		AuthenticatedClients: authenticatedCount,
		BytesReceived:        atomic.LoadInt64(&s.bytesReceived),
		BytesSent:           atomic.LoadInt64(&s.bytesSent),
		PacketsReceived:     atomic.LoadInt64(&s.packetsReceived),
		PacketsSent:         atomic.LoadInt64(&s.packetsSent),
		AverageLatency:       avgLatency,
		PeakLatency:          peakLatency,
	}
}

// Health returns the health status of the VPN server
func (s *vpnServer) Health() map[string]interface{} {
	metrics := s.Metrics()
	
	healthy := true
	issues := []string{}

	// Check if server is running
	if atomic.LoadInt32(&s.stopping) == 1 {
		healthy = false
		issues = append(issues, "server is stopping")
	}

	// Check connection limits
	if s.config.MaxClients > 0 && metrics.ActiveClients >= int64(s.config.MaxClients) {
		healthy = false
		issues = append(issues, "maximum clients reached")
	}

	// Check IP pool
	availableIPs := s.ipPool.AvailableCount()
	if availableIPs == 0 {
		healthy = false
		issues = append(issues, "IP pool exhausted")
	}

	// Check for high failure rate
	if metrics.TotalConnections > 0 {
		failureRate := float64(metrics.FailedConnections) / float64(metrics.TotalConnections)
		if failureRate > 0.1 { // More than 10% failure rate
			healthy = false
			issues = append(issues, fmt.Sprintf("high failure rate: %.2f%%", failureRate*100))
		}
	}

	return map[string]interface{}{
		"healthy":    healthy,
		"issues":     issues,
		"metrics":    metrics,
		"availableIPs": availableIPs,
		"protocol":   s.config.Protocol,
		"listenAddr": s.config.ListenAddr,
	}
}

// AddClient adds a VPN client
func (s *vpnServer) AddClient(client Client) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Assign IP if not set
	if client.AssignedIP == nil {
		ip, err := s.ipPool.Allocate()
		if err != nil {
			return &VPNError{
				Code:    "IP_POOL_EXHAUSTED",
				Message: "No available IP addresses in pool",
			}
		}
		client.AssignedIP = ip
	}

	client.ConnectedAt = time.Now()
	client.LastSeen = time.Now()
	client.Status = ClientStatusConnecting

	s.clients[client.ID] = &client
	atomic.AddInt64(&s.totalClients, 1)
	atomic.AddInt64(&s.activeClients, 1)

	return nil
}

// RemoveClient removes a VPN client
func (s *vpnServer) RemoveClient(clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, exists := s.clients[clientID]
	if !exists {
		return &VPNError{
			Code:    "CLIENT_NOT_FOUND",
			Message: fmt.Sprintf("Client not found: %s", clientID),
		}
	}

	// Release IP
	s.ipPool.Release(client.AssignedIP)

	delete(s.clients, clientID)
	atomic.AddInt64(&s.activeClients, -1)
	client.Status = ClientStatusDisconnected

	return nil
}

// GetClient returns a client by ID
func (s *vpnServer) GetClient(clientID string) (*Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[clientID]
	if !exists {
		return nil, &VPNError{
			Code:    "CLIENT_NOT_FOUND",
			Message: fmt.Sprintf("Client not found: %s", clientID),
		}
	}

	return client, nil
}

// ListClients returns all connected clients
func (s *vpnServer) ListClients() []Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]Client, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, *client)
	}

	return clients
}

// Authenticate authenticates a client connection
func (s *vpnServer) Authenticate(username, password string) (*Client, error) {
	// Use authenticator
	authenticated, err := s.authenticator.Authenticate(username, password)
	if err != nil {
		return nil, &VPNError{
			Code:    "AUTH_ERROR",
			Message: fmt.Sprintf("Authentication error: %v", err),
		}
	}

	if !authenticated {
		return nil, &VPNError{
			Code:    "AUTH_FAILED",
			Message: "Invalid username or password",
		}
	}

	// Get client info
	clientInfo, err := s.authenticator.GetClientInfo(username)
	if err != nil {
		return nil, &VPNError{
			Code:    "CLIENT_INFO_ERROR",
			Message: fmt.Sprintf("Failed to get client info: %v", err),
		}
	}

	// Create or update client
	client := &Client{
		ID:          clientInfo.ID,
		Username:    clientInfo.Username,
		AssignedIP:  clientInfo.IP,
		ConnectedAt: time.Now(),
		LastSeen:    time.Now(),
		Status:      ClientStatusAuthenticated,
		Metadata:    clientInfo.Metadata,
	}

	atomic.AddInt64(&s.authenticatedClients, 1)

	return client, nil
}

// logger returns the server logger
func (s *vpnServer) logger() core.Logger {
	return s.BaseServer.Logger()
}

func ipChecksum(b []byte) uint16 {
	var sum uint32
	for i := 0; i < len(b); i += 2 {
		if i+1 < len(b) {
			sum += uint32(binary.BigEndian.Uint16(b[i:]))
		}
	}
	for sum > 0xffff {
		sum = sum>>16 + sum&0xffff
	}
	return ^uint16(sum)
}

func icmpChecksum(b []byte) uint16 {
	var sum uint32
	for i := 0; i < len(b); i += 2 {
		if i+1 < len(b) {
			sum += uint32(binary.BigEndian.Uint16(b[i:]))
		} else {
			sum += uint32(b[i]) << 8
		}
	}
	for sum > 0xffff {
		sum = sum>>16 + sum&0xffff
	}
	return ^uint16(sum)
}
