package queue

import (
	"time"
)

// Config configures RabbitMQ connection
type Config struct {
	// URL is the RabbitMQ connection URL (e.g., "amqp://guest:guest@localhost:5672/")
	URL string

	// Username is the RabbitMQ username (optional if provided in URL)
	Username string

	// Password is the RabbitMQ password (optional if provided in URL)
	Password string

	// Host is the RabbitMQ host (default: "localhost")
	Host string

	// Port is the RabbitMQ port (default: 5672)
	Port int

	// VHost is the virtual host (default: "/")
	VHost string

	// ConnectionTimeout is the connection timeout (default: 30s)
	ConnectionTimeout time.Duration

	// Heartbeat is the heartbeat interval (default: 10s)
	Heartbeat time.Duration

	// ChannelMax is the maximum number of channels (default: 0 = unlimited)
	ChannelMax int

	// FrameSize is the maximum frame size (default: 0 = unlimited)
	FrameSize int

	// TLS configures TLS connection (optional)
	TLS *TLSConfig

	// Retry configures retry behavior
	Retry *RetryConfig
}

// TLSConfig configures TLS for RabbitMQ connection
type TLSConfig struct {
	// Enabled enables TLS
	Enabled bool

	// InsecureSkipVerify skips certificate verification (not recommended for production)
	InsecureSkipVerify bool

	// CertFile is the path to client certificate file
	CertFile string

	// KeyFile is the path to client key file
	KeyFile string

	// CAFile is the path to CA certificate file
	CAFile string
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	// MaxRetries is the maximum number of retries (default: 3)
	MaxRetries int

	// InitialInterval is the initial retry interval (default: 1s)
	InitialInterval time.Duration

	// MaxInterval is the maximum retry interval (default: 30s)
	MaxInterval time.Duration

	// Multiplier is the backoff multiplier (default: 2.0)
	Multiplier float64
}

// DefaultConfig returns default RabbitMQ configuration
func DefaultConfig() Config {
	return Config{
		Host:              "localhost",
		Port:              5672,
		VHost:             "/",
		ConnectionTimeout: 30 * time.Second,
		Heartbeat:         10 * time.Second,
		ChannelMax:        0,
		FrameSize:         0,
		Retry: &RetryConfig{
			MaxRetries:      3,
			InitialInterval: 1 * time.Second,
			MaxInterval:     30 * time.Second,
			Multiplier:      2.0,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.URL == "" {
		if c.Host == "" {
			return &Error{Code: "INVALID_CONFIG", Message: "Host cannot be empty if URL is not provided"}
		}
		if c.Port <= 0 {
			return &Error{Code: "INVALID_CONFIG", Message: "Port must be positive"}
		}
	}
	if c.ConnectionTimeout < 0 {
		return &Error{Code: "INVALID_CONFIG", Message: "ConnectionTimeout cannot be negative"}
	}
	if c.Heartbeat < 0 {
		return &Error{Code: "INVALID_CONFIG", Message: "Heartbeat cannot be negative"}
	}
	if c.ChannelMax < 0 {
		return &Error{Code: "INVALID_CONFIG", Message: "ChannelMax cannot be negative"}
	}
	if c.FrameSize < 0 {
		return &Error{Code: "INVALID_CONFIG", Message: "FrameSize cannot be negative"}
	}
	if c.Retry != nil {
		if c.Retry.MaxRetries < 0 {
			return &Error{Code: "INVALID_CONFIG", Message: "MaxRetries cannot be negative"}
		}
		if c.Retry.InitialInterval < 0 {
			return &Error{Code: "INVALID_CONFIG", Message: "InitialInterval cannot be negative"}
		}
		if c.Retry.MaxInterval < 0 {
			return &Error{Code: "INVALID_CONFIG", Message: "MaxInterval cannot be negative"}
		}
		if c.Retry.Multiplier < 1.0 {
			return &Error{Code: "INVALID_CONFIG", Message: "Multiplier must be >= 1.0"}
		}
	}
	return nil
}
