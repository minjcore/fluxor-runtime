package queue

import (
	"context"
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core"
)

// QueueComponent provides RabbitMQ connection using Premium Pattern
// Similar to DatabaseComponent but for message queues
type QueueComponent struct {
	*core.BaseComponent
	config     Config
	connection *Connection
}

// NewQueueComponent creates a new queue component
// Fail-fast: Validates configuration
func NewQueueComponent(config Config) *QueueComponent {
	// Fail-fast: Validate configuration
	if err := config.Validate(); err != nil {
		panic(err.Error())
	}

	return &QueueComponent{
		BaseComponent: core.NewBaseComponent("queue"),
		config:        config,
	}
}

// doStart initializes the RabbitMQ connection
// Fail-fast: Validates state and configuration before starting
func (c *QueueComponent) doStart(ctx core.FluxorContext) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	logger := core.NewDefaultLogger()
	logger.Info(fmt.Sprintf("QueueComponent.doStart: Creating connection to %s:%d", c.config.Host, c.config.Port))

	// Create connection (NewConnection also validates config)
	conn, err := NewConnection(c.config)
	if err != nil {
		logger.Error(fmt.Sprintf("QueueComponent.doStart: NewConnection failed: %v", err))
		return fmt.Errorf("failed to create RabbitMQ connection: %w", err)
	}
	if conn == nil {
		logger.Error("QueueComponent.doStart: NewConnection returned nil connection without error")
		return &core.EventBusError{Code: "CONNECTION_FAILED", Message: "NewConnection returned nil connection without error"}
	}

	logger.Info("QueueComponent.doStart: Connection created successfully, setting connection field")
	c.connection = conn
	logger.Info("QueueComponent.doStart: Connection field set, component ready")

	// Notify via EventBus (Premium Pattern integration)
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("queue.ready", map[string]interface{}{
			"component": c.Name(),
			"host":      c.config.Host,
			"port":      c.config.Port,
			"vhost":     c.config.VHost,
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// doStop closes the RabbitMQ connection
func (c *QueueComponent) doStop(ctx core.FluxorContext) error {
	if c.connection != nil {
		return c.connection.Close()
	}
	return nil
}

// Connection returns the RabbitMQ connection
// Fail-fast: Panics if component not started
func (c *QueueComponent) Connection() *Connection {
	if c == nil {
		panic("QueueComponent cannot be nil")
	}
	if c.connection == nil {
		panic("queue component not started - call Start() first")
	}
	return c.connection
}

// Publisher creates a new publisher
// Fail-fast: Panics if component not started
func (c *QueueComponent) Publisher() (Publisher, error) {
	if c == nil {
		return nil, &core.EventBusError{Code: "INVALID_STATE", Message: "QueueComponent cannot be nil"}
	}
	if c.connection == nil {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "queue component not started - call Start() first"}
	}
	return NewPublisher(c.connection)
}

// Consumer creates a new consumer
// Fail-fast: Panics if component not started
func (c *QueueComponent) Consumer() (Consumer, error) {
	if c == nil {
		return nil, &core.EventBusError{Code: "INVALID_STATE", Message: "QueueComponent cannot be nil"}
	}
	if c.connection == nil {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "queue component not started - call Start() first"}
	}
	return NewConsumer(c.connection)
}

// Ping tests the connection health
// Fail-fast: Validates state and inputs before pinging
// Performs actual health check by attempting to create a test channel
func (c *QueueComponent) Ping(ctx context.Context) error {
	if c == nil {
		return &core.EventBusError{Code: "INVALID_STATE", Message: "QueueComponent cannot be nil"}
	}
	if c.connection == nil {
		return &core.EventBusError{Code: "NOT_STARTED", Message: "queue component not started - call Start() first"}
	}
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	if c.connection.IsClosed() {
		return &core.EventBusError{Code: "INVALID_STATE", Message: "connection is closed"}
	}

	// Perform actual health check by creating and immediately closing a test channel
	testChannel, err := c.connection.Channel()
	if err != nil {
		return &core.EventBusError{Code: "HEALTH_CHECK_FAILED", Message: fmt.Sprintf("connection health check failed: %v", err)}
	}
	defer testChannel.Close()

	return nil
}
