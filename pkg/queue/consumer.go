package queue

import (
	"context"
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core"
	amqp "github.com/rabbitmq/amqp091-go"
)

// ConsumerConfig configures consumer behavior
type ConsumerConfig struct {
	// Queue is the queue name
	Queue string

	// ConsumerTag is the consumer tag (optional)
	ConsumerTag string

	// AutoAck enables automatic acknowledgment (default: false)
	AutoAck bool

	// Exclusive makes this consumer exclusive (default: false)
	Exclusive bool

	// NoLocal prevents receiving messages published by this connection (default: false)
	NoLocal bool

	// NoWait enables no-wait mode (default: false)
	NoWait bool

	// PrefetchCount is the prefetch count (default: 0 = unlimited)
	PrefetchCount int

	// PrefetchSize is the prefetch size in bytes (default: 0 = unlimited)
	PrefetchSize int

	// Global applies prefetch globally (default: false)
	Global bool
}

// ConsumerImpl implements Consumer interface
type ConsumerImpl struct {
	conn    *Connection
	channel *amqp.Channel
	config  Config
	logger  core.Logger
}

// NewConsumer creates a new consumer
// Fail-fast: Validates connection state
func NewConsumer(conn *Connection) (*ConsumerImpl, error) {
	if conn == nil {
		return nil, &Error{Code: "INVALID_INPUT", Message: "connection cannot be nil"}
	}
	if conn.IsClosed() {
		return nil, &Error{Code: "INVALID_STATE", Message: "connection is closed"}
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &ConsumerImpl{
		conn:    conn,
		channel: ch,
		config:  conn.config,
		logger:  core.NewDefaultLogger(),
	}, nil
}

// Consume starts consuming messages from a queue
// Fail-fast: Validates inputs before consuming
func (c *ConsumerImpl) Consume(ctx context.Context, queue string, handler MessageHandler) error {
	// Use default consumer config
	config := ConsumerConfig{
		Queue:         queue,
		ConsumerTag:   "",
		AutoAck:       false,
		Exclusive:     false,
		NoLocal:       false,
		NoWait:        false,
		PrefetchCount: 0,
		PrefetchSize:  0,
		Global:        false,
	}
	return c.ConsumeWithConfig(ctx, config, handler)
}

// ConsumeWithConfig starts consuming with custom configuration
func (c *ConsumerImpl) ConsumeWithConfig(ctx context.Context, config ConsumerConfig, handler MessageHandler) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}

	// Fail-fast: Validate queue name
	if err := ValidateQueueName(config.Queue); err != nil {
		return err
	}

	// Fail-fast: Validate handler
	if handler == nil {
		return &Error{Code: "INVALID_INPUT", Message: "handler cannot be nil"}
	}

	// Fail-fast: Validate consumer state
	if c == nil {
		return &Error{Code: "INVALID_STATE", Message: "consumer cannot be nil"}
	}
	if c.channel == nil {
		return &Error{Code: "INVALID_STATE", Message: "consumer channel not initialized"}
	}
	if c.channel.IsClosed() {
		return &Error{Code: "INVALID_STATE", Message: "consumer channel is closed"}
	}

	c.logger.Info("Starting consumer", "queue", config.Queue, "consumer_tag", config.ConsumerTag)

	// Use internal consume method
	return c.consumeInternal(ctx, config, handler)
}

// consumeInternal is the shared implementation for consuming messages
func (c *ConsumerImpl) consumeInternal(ctx context.Context, config ConsumerConfig, handler MessageHandler) error {
	// Set prefetch (QoS)
	if err := c.channel.Qos(config.PrefetchCount, config.PrefetchSize, config.Global); err != nil {
		c.logger.Error("Failed to set QoS", "error", err)
		return fmt.Errorf("fail-fast: failed to set QoS: %w", err)
	}

	// Start consuming
	deliveries, err := c.channel.Consume(
		config.Queue,
		config.ConsumerTag,
		config.AutoAck,
		config.Exclusive,
		config.NoLocal,
		config.NoWait,
		nil, // args
	)
	if err != nil {
		c.logger.Error("Failed to start consuming", "queue", config.Queue, "error", err)
		return fmt.Errorf("fail-fast: failed to start consuming: %w", err)
	}

	c.logger.Info("Consumer started successfully", "queue", config.Queue)

	// Process deliveries
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Error("Consumer panic recovered", "panic", r, "queue", config.Queue)
			}
			c.logger.Info("Consumer goroutine stopped", "queue", config.Queue)
		}()

		for {
			select {
			case <-ctx.Done():
				c.logger.Debug("Consumer context cancelled", "queue", config.Queue)
				return
			case delivery, ok := <-deliveries:
				if !ok {
					c.logger.Info("Consumer delivery channel closed", "queue", config.Queue)
					return // Channel closed
				}

				// Convert to Delivery
				msg := &Delivery{
					Body:          delivery.Body,
					Headers:       convertHeaders(delivery.Headers),
					ContentType:   delivery.ContentType,
					DeliveryTag:   delivery.DeliveryTag,
					Exchange:      delivery.Exchange,
					RoutingKey:    delivery.RoutingKey,
					MessageID:     delivery.MessageId,
					Timestamp:     delivery.Timestamp,
					ReplyTo:       delivery.ReplyTo,
					CorrelationID: delivery.CorrelationId,
					Redelivered:   delivery.Redelivered,
				}

				// Handle message
				if err := handler(ctx, msg); err != nil {
					c.logger.Error("Message handler error", "queue", config.Queue, "routing_key", msg.RoutingKey, "error", err)
					// Nack message on error (requeue)
					if !config.AutoAck {
						if nackErr := delivery.Nack(false, true); nackErr != nil {
							c.logger.Error("Failed to nack message", "error", nackErr)
						}
					}
					continue
				}

				// Ack message on success
				if !config.AutoAck {
					if err := delivery.Ack(false); err != nil {
						c.logger.Error("Failed to ack message", "delivery_tag", delivery.DeliveryTag, "error", err)
					}
				}
			}
		}
	}()

	return nil
}

// Connection returns the underlying connection
func (c *ConsumerImpl) Connection() *Connection {
	return c.conn
}

// Close closes the consumer
// Fail-fast: Validates state
func (c *ConsumerImpl) Close() error {
	if c == nil {
		return &Error{Code: "INVALID_STATE", Message: "consumer cannot be nil"}
	}
	if c.channel == nil {
		return nil // Already closed
	}
	return c.channel.Close()
}

// convertHeaders converts amqp.Table to map[string]string
func convertHeaders(headers amqp.Table) map[string]string {
	if headers == nil {
		return make(map[string]string)
	}
	result := make(map[string]string)
	for k, v := range headers {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}
