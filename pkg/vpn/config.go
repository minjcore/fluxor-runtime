package vpn

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/config"
)

// Config represents VPN server configuration.
// It embeds BaseConfig from the config package, providing common configuration
// features like service name, server settings, profile, and environment.
//
// Example usage:
//
//	// Create config with BaseConfig defaults
//	cfg := vpn.DefaultConfig()
//	cfg.ListenAddr = ":1194"
//	cfg.Protocol = "udp"
//	cfg.NetworkCIDR = "10.8.0.0/24"
//	cfg.Service.Name = "vpn-service"
//
//	// Or load from file (BaseConfig supports YAML/JSON loading)
//	var cfg vpn.Config
//	if err := config.Load("vpn-config.yaml", &cfg); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate both VPN-specific and BaseConfig fields
//	if err := cfg.Validate(); err != nil {
//	    log.Fatal(err)
//	}
type Config struct {
	// Embed BaseConfig to inherit common configuration features
	// This provides: Service, Server, Profile, Environment, and lifecycle methods
	config.BaseConfig

	// ListenAddr is the address to listen on (e.g., ":1194")
	ListenAddr string `json:"listenAddr" env:"VPN_LISTEN_ADDR" default:":1194" description:"VPN server listen address"`

	// Protocol is the VPN protocol ("udp" or "tcp")
	Protocol string `json:"protocol" env:"VPN_PROTOCOL" default:"udp" description:"VPN protocol (udp or tcp)"`

	// NetworkCIDR is the VPN network CIDR (e.g., "10.8.0.0/24")
	NetworkCIDR string `json:"networkCIDR" env:"VPN_NETWORK_CIDR" description:"VPN network CIDR for client IP assignment"`

	// MaxClients is the maximum number of concurrent clients (0 = unlimited)
	MaxClients int `json:"maxClients" env:"VPN_MAX_CLIENTS" default:"100" description:"Maximum concurrent VPN clients"`

	// ConnectionTimeout is the timeout for establishing connections
	ConnectionTimeout time.Duration `json:"connectionTimeout" env:"VPN_CONNECTION_TIMEOUT" default:"30s" description:"Connection timeout"`

	// KeepAliveInterval is the keep-alive ping interval
	KeepAliveInterval time.Duration `json:"keepAliveInterval" env:"VPN_KEEPALIVE_INTERVAL" default:"10s" description:"Keep-alive ping interval"`

	// IdleTimeout is the idle connection timeout
	IdleTimeout time.Duration `json:"idleTimeout" env:"VPN_IDLE_TIMEOUT" default:"300s" description:"Idle connection timeout"`

	// EnableMetrics enables metrics collection
	EnableMetrics bool `json:"enableMetrics" env:"VPN_ENABLE_METRICS" default:"true" description:"Enable metrics collection"`

	// TLSConfig for encrypted VPN connections (optional)
	TLSConfig interface{} `json:"tlsConfig,omitempty" description:"TLS configuration for encrypted VPN"`

	// AuthenticationMethod is the authentication method ("password", "certificate", "token")
	AuthenticationMethod string `json:"authenticationMethod" env:"VPN_AUTH_METHOD" default:"password" description:"Authentication method"`

	// CertificatePath is the path to server certificate (for certificate-based auth)
	CertificatePath string `json:"certificatePath" env:"VPN_CERT_PATH" description:"Path to server certificate"`

	// PrivateKeyPath is the path to server private key
	PrivateKeyPath string `json:"privateKeyPath" env:"VPN_KEY_PATH" description:"Path to server private key"`

	// CA certificate path (for client certificate verification)
	CACertificatePath string `json:"caCertificatePath" env:"VPN_CA_CERT_PATH" description:"Path to CA certificate"`

	// DNS servers to push to clients
	DNSServers []string `json:"dnsServers" env:"VPN_DNS_SERVERS" description:"DNS servers to push to clients"`

	// Routes to push to clients
	Routes []string `json:"routes" env:"VPN_ROUTES" description:"Routes to push to clients"`

	// EchoData khi true: server trả lại gói Data (để test client TUN, round-trip)
	EchoData bool `json:"echoData" env:"VPN_ECHO_DATA" description:"Echo data packets back to sender (for TUN client testing)"`

	// EnableForwarding: khi true, server tạo TUN gateway và forward gói dst ngoài pool ra Internet.
	EnableForwarding bool `json:"enableForwarding" env:"VPN_ENABLE_FORWARDING" description:"Forward traffic to internet via server TUN"`

	// GatewayTUNIP: IP của TUN gateway trên server (vd 10.8.0.254), phải trong NetworkCIDR.
	GatewayTUNIP string `json:"gatewayTUNIP" env:"VPN_GATEWAY_TUN_IP" description:"Server TUN gateway IP for forwarding"`
}

// DefaultConfig returns a default VPN configuration with BaseConfig initialized.
// Uses environment variables: VPN_LISTEN_ADDR, VPN_PROTOCOL, VPN_NETWORK_CIDR
// The BaseConfig is initialized with defaults from config.NewBaseConfig().
func DefaultConfig() Config {
	cfg := Config{
		BaseConfig:            *config.NewBaseConfig(),
		ListenAddr:            getEnvOrDefault("VPN_LISTEN_ADDR", ":1194"),
		Protocol:              getEnvOrDefault("VPN_PROTOCOL", "udp"),
		NetworkCIDR:           getEnvOrDefault("VPN_NETWORK_CIDR", "10.8.0.0/24"),
		MaxClients:            100,
		ConnectionTimeout:     30 * time.Second,
		KeepAliveInterval:     10 * time.Second,
		IdleTimeout:           300 * time.Second,
		EnableMetrics:         true,
		AuthenticationMethod:  getEnvOrDefault("VPN_AUTH_METHOD", "password"),
		DNSServers:            []string{"8.8.8.8", "8.8.4.4"},
		Routes:                []string{},
	}

	return cfg
}

// Validate validates the configuration, including both VPN-specific fields
// and BaseConfig fields. This demonstrates how to extend BaseConfig validation.
// Fail-fast: Returns error if required fields are missing
func (c *Config) Validate() error {
	// Validate BaseConfig first (if it has validators)
	baseValidators := c.BaseConfig.GetValidators()
	for _, validator := range baseValidators {
		if err := validator.Validate(c); err != nil {
			return &ConfigError{Code: "BASE_CONFIG_VALIDATION", Message: err.Error()}
		}
	}

	// Validate listen address
	if c.ListenAddr == "" {
		c.ListenAddr = getEnvOrDefault("VPN_LISTEN_ADDR", ":1194")
	}

	// Validate protocol
	if c.Protocol == "" {
		c.Protocol = getEnvOrDefault("VPN_PROTOCOL", "udp")
	}
	if c.Protocol != "udp" && c.Protocol != "tcp" {
		return &ConfigError{
			Code:    "INVALID_PROTOCOL",
			Message: "Protocol must be 'udp' or 'tcp'",
		}
	}

	// Validate network CIDR
	if c.NetworkCIDR == "" {
		c.NetworkCIDR = getEnvOrDefault("VPN_NETWORK_CIDR", "10.8.0.0/24")
	}
	_, ipNet, err := net.ParseCIDR(c.NetworkCIDR)
	if err != nil {
		return &ConfigError{
			Code:    "INVALID_NETWORK_CIDR",
			Message: fmt.Sprintf("Invalid network CIDR: %s", err.Error()),
		}
	}
	// Store parsed network for later use
	_ = ipNet

	// Validate authentication method
	if c.AuthenticationMethod == "" {
		c.AuthenticationMethod = getEnvOrDefault("VPN_AUTH_METHOD", "password")
	}
	validAuthMethods := map[string]bool{
		"password":    true,
		"certificate": true,
		"token":       true,
	}
	if !validAuthMethods[c.AuthenticationMethod] {
		return &ConfigError{
			Code:    "INVALID_AUTH_METHOD",
			Message: "Authentication method must be one of: password, certificate, token",
		}
	}

	// Validate certificate paths if using certificate auth
	if c.AuthenticationMethod == "certificate" {
		if c.CertificatePath == "" {
			c.CertificatePath = getEnvOrDefault("VPN_CERT_PATH", "")
			if c.CertificatePath == "" {
				return &ConfigError{
					Code:    "MISSING_CERTIFICATE",
					Message: "Certificate path is required for certificate authentication",
				}
			}
		}
		if c.PrivateKeyPath == "" {
			c.PrivateKeyPath = getEnvOrDefault("VPN_KEY_PATH", "")
			if c.PrivateKeyPath == "" {
				return &ConfigError{
					Code:    "MISSING_PRIVATE_KEY",
					Message: "Private key path is required for certificate authentication",
				}
			}
		}

		// Check if files exist
		if _, err := os.Stat(c.CertificatePath); os.IsNotExist(err) {
			return &ConfigError{
				Code:    "CERTIFICATE_NOT_FOUND",
				Message: fmt.Sprintf("Certificate file not found: %s", c.CertificatePath),
			}
		}
		if _, err := os.Stat(c.PrivateKeyPath); os.IsNotExist(err) {
			return &ConfigError{
				Code:    "PRIVATE_KEY_NOT_FOUND",
				Message: fmt.Sprintf("Private key file not found: %s", c.PrivateKeyPath),
			}
		}
	}

	// Validate timeouts
	if c.ConnectionTimeout <= 0 {
		c.ConnectionTimeout = 30 * time.Second
	}
	if c.KeepAliveInterval <= 0 {
		c.KeepAliveInterval = 10 * time.Second
	}
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = 300 * time.Second
	}

	// Validate DNS servers
	if len(c.DNSServers) == 0 {
		c.DNSServers = []string{"8.8.8.8", "8.8.4.4"}
	}
	for _, dns := range c.DNSServers {
		if net.ParseIP(dns) == nil {
			return &ConfigError{
				Code:    "INVALID_DNS_SERVER",
				Message: fmt.Sprintf("Invalid DNS server IP: %s", dns),
			}
		}
	}

	return nil
}

// GetNetwork returns the parsed network CIDR
func (c *Config) GetNetwork() (*net.IPNet, error) {
	_, ipNet, err := net.ParseCIDR(c.NetworkCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid network CIDR: %w", err)
	}
	return ipNet, nil
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
