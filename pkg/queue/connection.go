package queue

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/fluxorio/fluxor/pkg/core"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection represents a RabbitMQ connection
type Connection struct {
	conn   *amqp.Connection
	config Config
	logger core.Logger
}

// NewConnection creates a new RabbitMQ connection
// Fail-fast: Validates configuration before connecting
func NewConnection(config Config) (*Connection, error) {
	// Fail-fast: Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Build connection URL
	url := config.URL
	if url == "" {
		username := config.Username
		if username == "" {
			username = "guest"
		}
		password := config.Password
		if password == "" {
			password = "guest"
		}
		host := config.Host
		if host == "" {
			host = "localhost"
		}
		port := config.Port
		if port == 0 {
			port = 5672
		}
		vhost := config.VHost
		if vhost == "" {
			vhost = "/"
		}

		url = fmt.Sprintf("amqp://%s:%s@%s:%d%s", username, password, host, port, vhost)
	}

	// Configure connection
	amqpConfig := amqp.Config{
		Heartbeat: config.Heartbeat,
		Locale:    "en_US",
	}

	if config.ChannelMax > 0 {
		// Safe conversion: ensure value fits in uint16 (max 65535)
		const maxUint16 = 65535
		if config.ChannelMax > maxUint16 {
			return nil, fmt.Errorf("fail-fast: ChannelMax (%d) exceeds maximum value (%d)", config.ChannelMax, maxUint16)
		}
		amqpConfig.ChannelMax = uint16(config.ChannelMax)
	}
	if config.FrameSize > 0 {
		amqpConfig.FrameSize = config.FrameSize
	}

	// TLS configuration
	logger := core.NewDefaultLogger()
	var tlsConfig *tls.Config
	if config.TLS != nil && config.TLS.Enabled {
		// Warn if certificate verification is disabled (security risk)
		if config.TLS.InsecureSkipVerify {
			logger.Error("TLS InsecureSkipVerify is enabled - certificate verification is disabled. This is insecure and should only be used for development/testing.")
		}
		// #nosec G402 -- InsecureSkipVerify is a documented configuration option
		// for development/testing with self-signed certificates. A warning is logged when enabled.
		tlsConfig = &tls.Config{
			InsecureSkipVerify: config.TLS.InsecureSkipVerify,
		}

		// Load CA certificate if provided
		if config.TLS.CAFile != "" {
			caCert, err := os.ReadFile(config.TLS.CAFile)
			if err != nil {
				return nil, fmt.Errorf("fail-fast: failed to read CA certificate: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("fail-fast: failed to parse CA certificate")
			}
			tlsConfig.RootCAs = caCertPool
		}

		// Load client certificate if provided
		if config.TLS.CertFile != "" && config.TLS.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(config.TLS.CertFile, config.TLS.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("fail-fast: failed to load client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		// Set TLS config in amqp config
		amqpConfig.TLSClientConfig = tlsConfig

		// Use amqps:// scheme for TLS
		if config.URL == "" {
			url = fmt.Sprintf("amqps://%s:%s@%s:%d%s", config.Username, config.Password, config.Host, config.Port, config.VHost)
		} else if len(config.URL) >= 5 && config.URL[:5] != "amqps" {
			// If URL is provided but not amqps, replace scheme
			if len(config.URL) >= 5 && config.URL[:5] == "amqp:" {
				url = "amqps" + config.URL[4:]
			}
		}

		logger.Info("Connecting with TLS", "host", config.Host, "port", config.Port)
	}

	// Connect with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectionTimeout)
	defer cancel()

	logger.Info("Connecting to RabbitMQ", "host", config.Host, "port", config.Port, "vhost", config.VHost)

	// Use a channel to handle the blocking DialConfig with timeout
	type result struct {
		conn *amqp.Connection
		err  error
	}
	resultCh := make(chan result, 1)

	// Dial in a goroutine so we can timeout
	go func() {
		conn, err := amqp.DialConfig(url, amqpConfig)
		resultCh <- result{conn: conn, err: err}
	}()

	// Wait for either connection or timeout
	var conn *amqp.Connection
	var err error
	select {
	case <-ctx.Done():
		logger.Error("RabbitMQ connection timeout", "timeout", config.ConnectionTimeout)
		return nil, fmt.Errorf("fail-fast: rabbitmq connection timeout: %w", ctx.Err())
	case res := <-resultCh:
		conn = res.conn
		err = res.err
	}

	if err != nil {
		logger.Error("Failed to connect to RabbitMQ", "error", err)
		return nil, fmt.Errorf("fail-fast: rabbitmq connection failed: %w", err)
	}

	if conn == nil {
		logger.Error("Connection is nil after DialConfig")
		return nil, fmt.Errorf("fail-fast: rabbitmq connection is nil")
	}

	logger.Info("Successfully connected to RabbitMQ", "host", config.Host, "port", config.Port)

	return &Connection{
		conn:   conn,
		config: config,
		logger: logger,
	}, nil
}

// Channel creates a new channel
// Fail-fast: Validates connection state
func (c *Connection) Channel() (*amqp.Channel, error) {
	if c == nil {
		return nil, &Error{Code: "INVALID_STATE", Message: "connection cannot be nil"}
	}
	if c.conn == nil {
		return nil, &Error{Code: "INVALID_STATE", Message: "connection not initialized"}
	}
	if c.conn.IsClosed() {
		return nil, &Error{Code: "INVALID_STATE", Message: "connection is closed"}
	}

	ch, err := c.conn.Channel()
	if err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to open channel", "error", err)
		}
		return nil, fmt.Errorf("fail-fast: failed to open channel: %w", err)
	}
	return ch, nil
}

// Close closes the connection
// Fail-fast: Validates connection state
func (c *Connection) Close() error {
	if c == nil {
		return &Error{Code: "INVALID_STATE", Message: "connection cannot be nil"}
	}
	if c.conn == nil {
		return &Error{Code: "INVALID_STATE", Message: "connection already closed"}
	}
	if c.logger != nil {
		c.logger.Info("Closing RabbitMQ connection")
	}
	return c.conn.Close()
}

// IsClosed checks if connection is closed
func (c *Connection) IsClosed() bool {
	if c == nil || c.conn == nil {
		return true
	}
	return c.conn.IsClosed()
}

// NotifyClose returns a channel that receives notifications when connection closes
func (c *Connection) NotifyClose() <-chan *amqp.Error {
	if c == nil || c.conn == nil {
		ch := make(chan *amqp.Error, 1)
		close(ch)
		return ch
	}
	return c.conn.NotifyClose(make(chan *amqp.Error, 1))
}
