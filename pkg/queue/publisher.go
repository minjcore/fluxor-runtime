package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	amqp "github.com/rabbitmq/amqp091-go"
)

// PublisherImpl implements Publisher interface
type PublisherImpl struct {
	conn    *Connection
	channel *amqp.Channel
	config  Config
	logger  core.Logger
}

// NewPublisher creates a new publisher
// Fail-fast: Validates connection state
func NewPublisher(conn *Connection) (*PublisherImpl, error) {
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

	return &PublisherImpl{
		conn:    conn,
		channel: ch,
		config:  conn.config,
		logger:  core.NewDefaultLogger(),
	}, nil
}

// Publish publishes a message to an exchange
// Fail-fast: Validates inputs before publishing
func (p *PublisherImpl) Publish(ctx context.Context, exchange, routingKey string, msg Message) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &Error{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}

	// Fail-fast: Validate exchange
	if err := ValidateExchangeName(exchange); err != nil {
		return err
	}

	// Fail-fast: Validate routing key
	if routingKey == "" {
		return &Error{Code: "INVALID_INPUT", Message: "routing key cannot be empty"}
	}

	// Fail-fast: Validate publisher state
	if p == nil {
		return &Error{Code: "INVALID_STATE", Message: "publisher cannot be nil"}
	}
	if p.channel == nil {
		return &Error{Code: "INVALID_STATE", Message: "publisher channel not initialized"}
	}
	if p.channel.IsClosed() {
		return &Error{Code: "INVALID_STATE", Message: "publisher channel is closed"}
	}

	// Encode body
	body, err := JSONEncode(msg.Body)
	if err != nil {
		return fmt.Errorf("fail-fast: failed to encode message body: %w", err)
	}

	// Prepare headers
	headers := make(amqp.Table)
	if msg.Headers != nil {
		for k, v := range msg.Headers {
			headers[k] = v
		}
	}

	// Set content type
	contentType := msg.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	// Set delivery mode
	deliveryMode := msg.DeliveryMode
	if deliveryMode == 0 {
		deliveryMode = 2 // Persistent by default
	}

	// Prepare publishing options
	publishing := amqp.Publishing{
		Headers:      headers,
		ContentType:  contentType,
		DeliveryMode: deliveryMode,
		Body:         body,
		Timestamp:    time.Now(),
	}

	if msg.Priority > 0 {
		publishing.Priority = msg.Priority
	}
	if msg.Expiration != "" {
		publishing.Expiration = msg.Expiration
	}
	if msg.MessageID != "" {
		publishing.MessageId = msg.MessageID
	}
	if msg.ReplyTo != "" {
		publishing.ReplyTo = msg.ReplyTo
	}
	if msg.CorrelationID != "" {
		publishing.CorrelationId = msg.CorrelationID
	}
	if !msg.Timestamp.IsZero() {
		publishing.Timestamp = msg.Timestamp
	}

	// Publish with context
	err = p.channel.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		publishing,
	)

	if err != nil {
		p.logger.Error("Failed to publish message", "exchange", exchange, "routing_key", routingKey, "error", err)
		return fmt.Errorf("fail-fast: failed to publish message: %w", err)
	}

	p.logger.Debug("Message published", "exchange", exchange, "routing_key", routingKey, "message_id", msg.MessageID)
	return nil
}

// Close closes the publisher
// Fail-fast: Validates state
func (p *PublisherImpl) Close() error {
	if p == nil {
		return &Error{Code: "INVALID_STATE", Message: "publisher cannot be nil"}
	}
	if p.channel == nil {
		return nil // Already closed
	}
	return p.channel.Close()
}
