# VPN Server Package

High-performance VPN server with client management, IP allocation, and authentication support.

## 🎯 Overview

The VPN server package provides a production-ready VPN implementation with:

- ✅ **UDP and TCP protocol support** - Handle both UDP and TCP VPN connections
- ✅ **Client management** - Track and manage connected VPN clients
- ✅ **IP address pooling** - Automatic IP allocation from configurable network range
- ✅ **Authentication** - Password and certificate-based authentication
- ✅ **Fail-fast design** - Immediate validation and clear error messages
- ✅ **Keep-alive** - Automatic connection health monitoring
- ✅ **Graceful shutdown** - Clean connection termination
- ✅ **Rich metrics** - Performance monitoring and observability
- ✅ **EventBus integration** - Reactive patterns with Fluxor EventBus

---

## 🚀 Quick Start

### Basic VPN Server

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/vpn"
)

func main() {
    // Create runtime
    gocmd := core.NewGoCMD(context.Background())
    
    // Create VPN configuration
    config := vpn.DefaultConfig()
    config.ListenAddr = ":1194"
    config.Protocol = "udp"
    config.NetworkCIDR = "10.8.0.0/24"
    config.MaxClients = 100
    
    // Create VPN server
    vpnServer := vpn.NewVPNServer(gocmd, config)
    
    // Start VPN server (blocking)
    if err := vpnServer.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Using VPN Component

```go
type MyVerticle struct {
    *core.BaseVerticle
    vpn *vpn.VPNComponent
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Create VPN component
    config := vpn.DefaultConfig()
    config.ListenAddr = ":1194"
    config.Protocol = "udp"
    config.NetworkCIDR = "10.8.0.0/24"
    v.vpn = vpn.NewVPNComponent(config)
    
    // Start component
    return v.vpn.Start(ctx)
}

func (v *MyVerticle) Stop(ctx core.FluxorContext) error {
    if v.vpn != nil {
        return v.vpn.Stop(ctx)
    }
    return nil
}
```

---

## 📋 Configuration

### Config Structure

```go
type Config struct {
    ListenAddr           string        // Listen address (e.g., ":1194")
    Protocol             string        // "udp" or "tcp"
    NetworkCIDR          string        // VPN network CIDR (e.g., "10.8.0.0/24")
    MaxClients           int           // Max concurrent clients (0 = unlimited)
    ConnectionTimeout    time.Duration // Connection timeout
    KeepAliveInterval    time.Duration // Keep-alive ping interval
    IdleTimeout          time.Duration // Idle connection timeout
    EnableMetrics        bool          // Enable metrics collection
    AuthenticationMethod string        // "password", "certificate", "token"
    CertificatePath      string        // Path to server certificate
    PrivateKeyPath       string        // Path to server private key
    CACertificatePath    string        // Path to CA certificate
    DNSServers           []string      // DNS servers to push to clients
    Routes               []string      // Routes to push to clients
}
```

### Default Configuration

```go
config := vpn.DefaultConfig()
// ListenAddr: ":1194"
// Protocol: "udp"
// NetworkCIDR: "10.8.0.0/24"
// MaxClients: 100
// ConnectionTimeout: 30s
// KeepAliveInterval: 10s
// IdleTimeout: 300s
// EnableMetrics: true
// AuthenticationMethod: "password"
```

### Custom Configuration

```go
config := vpn.Config{
    ListenAddr:           ":1194",
    Protocol:             "udp",
    NetworkCIDR:          "10.8.0.0/24",
    MaxClients:           200,
    ConnectionTimeout:    30 * time.Second,
    KeepAliveInterval:    10 * time.Second,
    IdleTimeout:          300 * time.Second,
    AuthenticationMethod: "password",
    DNSServers:           []string{"8.8.8.8", "8.8.4.4"},
    Routes:               []string{"0.0.0.0/0"},
}
```

### Environment Variables

```bash
export VPN_LISTEN_ADDR=":1194"
export VPN_PROTOCOL="udp"
export VPN_NETWORK_CIDR="10.8.0.0/24"
export VPN_MAX_CLIENTS="100"
export VPN_AUTH_METHOD="password"
export VPN_KEEPALIVE_INTERVAL="10s"
export VPN_IDLE_TIMEOUT="300s"
```

---

## 🏗️ Architecture

### Components

```
VPN Server Architecture
│
├─ Listener (UDP/TCP)
│   └─ Accept connections/packets
│
├─ Authenticator
│   ├─ Password-based
│   └─ Certificate-based
│
├─ IP Pool
│   ├─ Allocate IPs to clients
│   └─ Release IPs on disconnect
│
├─ Client Manager
│   ├─ Track connected clients
│   ├─ Monitor client status
│   └─ Keep-alive checks
│
└─ Metrics Collector
    ├─ Connection metrics
    ├─ Client metrics
    └─ Traffic metrics
```

### Connection Flow

```
1. Client Connects
   ↓
2. Authentication
   → Reject if invalid
   ↓
3. IP Allocation
   → Fail if pool exhausted
   ↓
4. Client Registration
   ↓
5. Keep-Alive Monitoring
   ↓
6. Data Tunneling
   ↓
7. Client Disconnect
   ↓
8. IP Release
```

---

## 🎨 Usage Patterns

### Pattern 1: UDP VPN Server

```go
config := vpn.DefaultConfig()
config.ListenAddr = ":1194"
config.Protocol = "udp"
config.NetworkCIDR = "10.8.0.0/24"
config.MaxClients = 100

vpnServer := vpn.NewVPNServer(gocmd, config)
vpnServer.Start()
```

### Pattern 2: TCP VPN Server

```go
config := vpn.DefaultConfig()
config.ListenAddr = ":1194"
config.Protocol = "tcp"
config.NetworkCIDR = "10.8.0.0/24"

vpnServer := vpn.NewVPNServer(gocmd, config)
vpnServer.Start()
```

### Pattern 3: Custom Authentication

```go
// Create custom authenticator
authenticator := vpn.NewPasswordAuthenticator()
authenticator.AddUser("user1", "password1", &vpn.ClientInfo{
    ID:       "user1-id",
    Username: "user1",
    Metadata: map[string]interface{}{
        "role": "admin",
    },
})

// Use in server (you'll need to modify server to accept custom authenticator)
config := vpn.DefaultConfig()
// ... configure ...
```

### Pattern 4: Client Management

```go
vpnServer := vpn.NewVPNServer(gocmd, config)
vpnServer.Start()

// List all clients
clients := vpnServer.ListClients()
for _, client := range clients {
    fmt.Printf("Client: %s, IP: %s, Status: %s\n",
        client.Username, client.AssignedIP, client.Status)
}

// Get specific client
client, err := vpnServer.GetClient("client-id")
if err != nil {
    log.Fatal(err)
}

// Remove client
err = vpnServer.RemoveClient("client-id")
```

### Pattern 5: EventBus Integration

```go
// Listen for VPN events
eventBus := gocmd.EventBus()

eventBus.Consumer("vpn.ready").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var data map[string]interface{}
    if err := msg.Decode(&data); err != nil {
        return err
    }
    
    listenAddr := data["listenAddr"].(string)
    log.Printf("VPN ready on %s", listenAddr)
    return nil
})

eventBus.Consumer("vpn.client.connected").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var data map[string]interface{}
    if err := msg.Decode(&data); err != nil {
        return err
    }
    
    clientID := data["clientID"].(string)
    log.Printf("Client connected: %s", clientID)
    return nil
})
```

---

## 📊 Metrics & Monitoring

### Server Metrics

```go
metrics := vpnServer.Metrics()

fmt.Printf("Total Connections: %d\n", metrics.TotalConnections)
fmt.Printf("Active Connections: %d\n", metrics.ActiveConnections)
fmt.Printf("Total Clients: %d\n", metrics.TotalClients)
fmt.Printf("Active Clients: %d\n", metrics.ActiveClients)
fmt.Printf("Authenticated Clients: %d\n", metrics.AuthenticatedClients)
fmt.Printf("Bytes Received: %d\n", metrics.BytesReceived)
fmt.Printf("Bytes Sent: %d\n", metrics.BytesSent)
fmt.Printf("Packets Received: %d\n", metrics.PacketsReceived)
fmt.Printf("Packets Sent: %d\n", metrics.PacketsSent)
```

### Metrics Structure

```go
type ServerMetrics struct {
    TotalConnections     int64   // Total connections accepted
    ActiveConnections    int64   // Current active connections
    RejectedConnections  int64   // Connections rejected (max clients, etc.)
    FailedConnections    int64   // Failed connections
    TotalClients         int64   // Total clients connected
    ActiveClients        int64   // Current active clients
    AuthenticatedClients int64   // Authenticated clients
    BytesReceived        int64   // Total bytes received
    BytesSent            int64   // Total bytes sent
    PacketsReceived      int64   // Total packets received
    PacketsSent          int64   // Total packets sent
    AverageLatency       float64 // Average latency (ms)
    PeakLatency          float64 // Peak latency (ms)
}
```

### Health Check Endpoint

```go
// HTTP health endpoint
router.GET("/vpn/metrics", func(ctx *web.RequestContext) error {
    metrics := vpnServer.Metrics()
    return ctx.JSON(200, metrics)
})

router.GET("/vpn/health", func(ctx *web.RequestContext) error {
    metrics := vpnServer.Metrics()
    
    healthy := metrics.ActiveConnections > 0 || metrics.ActiveClients > 0
    statusCode := 200
    if !healthy {
        statusCode = 503
    }
    
    return ctx.JSON(statusCode, map[string]interface{}{
        "healthy":           healthy,
        "active_clients":    metrics.ActiveClients,
        "total_clients":     metrics.TotalClients,
    })
})
```

---

## 🔐 Authentication

### Password Authentication

Default authentication method using username/password.

```go
config := vpn.DefaultConfig()
config.AuthenticationMethod = "password"

// Default authenticator includes test user: test/test
// For production, add your own users:
authenticator := vpn.NewPasswordAuthenticator()
authenticator.AddUser("admin", "secure-password", &vpn.ClientInfo{
    ID:       "admin-001",
    Username: "admin",
    Metadata: map[string]interface{}{
        "role": "administrator",
    },
})
```

### Certificate Authentication

Certificate-based authentication (requires TLS configuration).

```go
config := vpn.DefaultConfig()
config.AuthenticationMethod = "certificate"
config.CertificatePath = "/path/to/server.crt"
config.PrivateKeyPath = "/path/to/server.key"
config.CACertificatePath = "/path/to/ca.crt"
```

---

## 🌐 IP Address Management

### IP Pool Configuration

The VPN server automatically manages IP addresses from the configured network range.

```go
config := vpn.DefaultConfig()
config.NetworkCIDR = "10.8.0.0/24" // Provides 254 IPs (10.8.0.1 - 10.8.0.254)

// Larger network for more clients
config.NetworkCIDR = "10.8.0.0/22" // Provides 1022 IPs
```

### IP Allocation

IPs are automatically allocated when clients connect and released when they disconnect.

```go
// When client connects:
// 1. Authenticate client
// 2. Allocate IP from pool
// 3. Assign IP to client
// 4. Release IP when client disconnects
```

---

## 💓 Keep-Alive

### Keep-Alive Configuration

```go
config := vpn.DefaultConfig()
config.KeepAliveInterval = 10 * time.Second // Send keep-alive every 10s
config.IdleTimeout = 300 * time.Second      // Disconnect after 5min idle
```

### Keep-Alive Behavior

- **Automatic**: Server sends keep-alive packets at configured interval
- **Client tracking**: Updates client's `LastSeen` timestamp
- **Idle timeout**: Clients inactive beyond `IdleTimeout` are disconnected
- **Connection health**: Detects and handles disconnected clients

---

## 🎯 Best Practices

### 1. Configure Appropriate Network Size

```go
// Small VPN (few clients)
config.NetworkCIDR = "10.8.0.0/24" // 254 IPs

// Medium VPN (moderate clients)
config.NetworkCIDR = "10.8.0.0/22" // 1022 IPs

// Large VPN (many clients)
config.NetworkCIDR = "10.8.0.0/16" // 65,534 IPs
```

### 2. Set Max Clients for Resource Protection

```go
config.MaxClients = 100 // Prevent resource exhaustion
```

### 3. Use Keep-Alive for Connection Health

```go
config.KeepAliveInterval = 10 * time.Second  // Frequent checks
config.IdleTimeout = 300 * time.Second       // 5 minute timeout
```

### 4. Monitor Metrics

```go
go func() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        metrics := vpnServer.Metrics()
        
        if metrics.ActiveClients >= int64(config.MaxClients) {
            logger.Warn("Maximum clients reached")
        }
        
        if metrics.FailedConnections > 0 {
            logger.Warn("Failed connections detected", "count", metrics.FailedConnections)
        }
    }
}()
```

### 5. Use EventBus for Client Events

```go
// Listen for client connection events
eventBus.Consumer("vpn.client.connected").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Log, update database, send notification, etc.
    return nil
})
```

---

## 🧪 Testing

### Unit Tests

```bash
go test ./pkg/vpn/...
```

### Integration Test

```go
func TestVPNServer(t *testing.T) {
    gocmd := core.NewGoCMD(context.Background())
    
    config := vpn.DefaultConfig()
    config.ListenAddr = ":0" // Random port
    config.NetworkCIDR = "10.8.0.0/24"
    
    vpnServer := vpn.NewVPNServer(gocmd, config)
    
    go vpnServer.Start()
    defer vpnServer.Stop()
    
    // Connect client and test
    conn, err := net.Dial("udp", vpnServer.Addr())
    require.NoError(t, err)
    defer conn.Close()
    
    // Send authentication packet
    // ... test authentication ...
}
```

---

## 🆚 Comparison

### vs OpenVPN

| Feature | pkg/vpn | OpenVPN |
|---------|---------|---------|
| **Protocol** | UDP/TCP | UDP/TCP |
| **Configuration** | Go code/YAML | Config files |
| **Dynamic Updates** | ✅ Runtime API | ⚠️ Restart required |
| **EventBus Integration** | ✅ Yes | ❌ No |
| **Metrics** | ✅ Built-in | ⚠️ External tools |
| **IP Management** | ✅ Automatic | ✅ Automatic |
| **Authentication** | ✅ Password/Cert | ✅ Password/Cert |

### vs pkg/tcp (TCP Server)

| Feature | pkg/vpn | pkg/tcp |
|---------|---------|---------|
| **Purpose** | VPN tunneling | Generic TCP |
| **Protocols** | UDP + TCP | TCP only |
| **IP Management** | ✅ Yes | ❌ No |
| **Client Tracking** | ✅ Yes | ❌ No |
| **Authentication** | ✅ Yes | ❌ No |
| **Use Case** | VPN connections | Generic TCP server |

---

## 📚 Related Documentation

- [pkg/tcp/README.md](../tcp/README.md) - TCP server implementation
- [pkg/proxy/README.md](../proxy/README.md) - Proxy server implementation
- [pkg/core/README.md](../core/README.md) - Core components

---

## 🔗 Integration Examples

### With pkg/web (Management API)

```go
// VPN server
vpnConfig := vpn.DefaultConfig()
vpnConfig.ListenAddr = ":1194"
vpnServer := vpn.NewVPNServer(gocmd, vpnConfig)
go vpnServer.Start()

// HTTP management API
httpServer := web.NewFastHTTPServer(gocmd, web.DefaultConfig(":8080"))
router := httpServer.Router()

router.GET("/vpn/clients", func(ctx *web.RequestContext) error {
    clients := vpnServer.ListClients()
    return ctx.JSON(200, clients)
})

router.DELETE("/vpn/clients/:id", func(ctx *web.RequestContext) error {
    clientID := ctx.Params["id"]
    err := vpnServer.RemoveClient(clientID)
    if err != nil {
        return ctx.JSON(404, map[string]string{"error": err.Error()})
    }
    return ctx.JSON(200, map[string]string{"status": "disconnected"})
})
```

### With EventBus

```go
// VPN publishes events
eventBus.Consumer("vpn.client.connected").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Update user session, log activity, etc.
    return nil
})

eventBus.Consumer("vpn.client.disconnected").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Cleanup, log activity, etc.
    return nil
})
```

---

## 🔒 Security Considerations

### 1. Use Strong Authentication

```go
// Use certificate-based auth for production
config.AuthenticationMethod = "certificate"
config.CertificatePath = "/path/to/server.crt"
config.PrivateKeyPath = "/path/to/server.key"
config.CACertificatePath = "/path/to/ca.crt"
```

### 2. Configure Firewall Rules

```bash
# Allow VPN port
ufw allow 1194/udp
ufw allow 1194/tcp
```

### 3. Limit Client Access

```go
config.MaxClients = 50 // Limit concurrent connections
```

### 4. Use TLS for Encryption

```go
// Configure TLS for encrypted connections
config.TLSConfig = &tls.Config{
    Certificates: []tls.Certificate{cert},
    MinVersion: tls.VersionTLS12,
}
```

---

## TUN client (thử nghiệm)

Có thể dùng TUN (card mạng ảo) với VPN server qua **vpn-tun-client**.

**Runbook deploy** (VPS Linux + STUN + firewall + systemd): [`docs/runbook_vpn_stun_deploy.md`](../docs/runbook_vpn_stun_deploy.md).

### Chỉ TUN (không VPN) — `tun-standalone`

Tạo interface TUN riêng, không handshake server: thử `ifconfig`/`ip`, route, hoặc debug stack mạng.

```bash
go build -o bin/tun-standalone ./cmd/tun-standalone
sudo ./bin/tun-standalone -up
# Tuỳ chọn: -tun-ip 10.9.0.1 -tun-gw 10.9.0.0
```

Tiến trình đọc và bỏ gói từ TUN để giữ FD mở; **Ctrl+C** thoát (interface biến mất). Linux: dùng `ip addr` / `ip link` (cần sudo).

### TCP (test local)

1. Chạy VPN server: `go run ./cmd/vpn-server` (TCP :1194).
2. Chạy client (**bắt buộc sudo**): `sudo ./bin/vpn-tun-client -server 127.0.0.1:1194 -up`

### UDP + STUN (194.233.73.36): Bước 1 STUN, bước 2 VPN over UDP

Luồng: client mở kết nối UDP tới STUN server (194.233.73.36:3478), sau đó chạy VPN over cùng UDP tới 194.233.73.36:1194.

1. **Trên 194.233.73.36:** Chạy STUN server (UDP 3478) và VPN server (UDP 1194):
   - STUN: `./bin/stun-server -addr :3478` (từ p2p-udp)
   - VPN: `go run ./cmd/vpn-server -udp` (UDP :1194, mở firewall `ufw allow 1194/udp`)

2. **Client (macOS, sudo):**
   ```bash
   sudo ./bin/vpn-tun-client -proto udp -server 194.233.73.36:1194 -stun 194.233.73.36:3478 -up
   ```
   - Bước 1: gửi STUN Binding Request tới 194.233.73.36:3478 (mở NAT / biết public IP:port).
   - Bước 2: handshake/auth/config và bridge TUN ↔ VPN qua UDP tới 194.233.73.36:1194 (cùng socket UDP).

3. Nếu chưa dùng `-up`: terminal khác `sudo ifconfig utunN 10.8.0.1 10.8.0.0 up`

Client tạo TUN, handshake/auth/config với server, rồi bridge: đọc gói từ TUN → gửi VPN Data; nhận VPN Data → ghi vào TUN. Khi server bật EchoData, gói gửi đi sẽ được trả lại (round-trip test).

---

## ⚠️ Limitations

- **TUN/TAP + forward**: Client TUN (cmd/vpn-tun-client); server forward gói theo dst IP (ipToClient). Client A (10.8.0.1) ping client B (10.8.0.2) qua VPN được. Ping 10.8.0.0 (gateway) nhận ICMP reply từ server. EchoData tắt mặc định.
- **Reconnect**: Client tự reconnect khi mất kết nối (sau 3s).
- **DNS/route push**: Server gửi DNS và routes trong config response. Client với `-apply-routes`: chạy **route thật** (macOS: `route add -net ...`, Linux: `ip route add ... dev <tun>`).
- **Forward ra Internet**: Server với `-forward` (chỉ Linux) tạo TUN gateway (10.8.0.254); gói dst ngoài pool được ghi vào TUN → kernel forward ra WAN; reply đọc từ TUN và gửi về client. Cần `sysctl -w net.ipv4.ip_forward=1` trên server.
- **Mã hóa**: Data tunnel dùng **ChaCha20-Poly1305** (AEAD). Key derive từ password + salt 16 byte (server gửi salt trong auth response). Chỉ payload Data được encrypt; control (handshake, auth, config) đi plain.
- **Protocol support**: Basic UDP/TCP protocol support. Full OpenVPN/WireGuard compatibility requires additional protocol implementation
- **Encryption**: TLS support is available but requires proper certificate configuration
- **Routing**: DNS and route pushing is configured but routing setup on server/client requires OS-level configuration

---

**Package**: `github.com/fluxorio/fluxor/pkg/vpn`  
**Status**: ✅ Stable  
**Test Coverage**: TBD  
**Last Updated**: 2026-01-16
