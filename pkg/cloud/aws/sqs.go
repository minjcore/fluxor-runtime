package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// sqsClient implements SQSClient
type sqsClient struct {
	client *sqs.Client
}

// newSQSClient creates a new SQS client
func newSQSClient(cfg aws.Config) SQSClient {
	return &sqsClient{
		client: sqs.NewFromConfig(cfg),
	}
}

// SendMessage sends a message to a queue
func (c *sqsClient) SendMessage(ctx context.Context, queueURL string, body string) error {
	// Fail-fast: Validate inputs
	if queueURL == "" {
		return fmt.Errorf("queue URL cannot be empty")
	}
	if body == "" {
		return fmt.Errorf("message body cannot be empty")
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(body),
	}

	_, err := c.client.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send message to SQS: %w", err)
	}

	return nil
}

// ReceiveMessages receives messages from a queue
func (c *sqsClient) ReceiveMessages(ctx context.Context, queueURL string, maxMessages int) ([]SQSMessage, error) {
	// Fail-fast: Validate inputs
	if queueURL == "" {
		return nil, fmt.Errorf("queue URL cannot be empty")
	}
	if maxMessages <= 0 || maxMessages > 10 {
		maxMessages = 10 // SQS limit
	}

	input := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: int32(maxMessages),
		WaitTimeSeconds:     20, // Long polling
	}

	output, err := c.client.ReceiveMessage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to receive messages from SQS: %w", err)
	}

	messages := make([]SQSMessage, 0, len(output.Messages))
	for _, msg := range output.Messages {
		m := SQSMessage{
			MessageID:     aws.ToString(msg.MessageId),
			ReceiptHandle: aws.ToString(msg.ReceiptHandle),
			Body:          aws.ToString(msg.Body),
			Attributes:    make(map[string]string),
		}

		// Copy attributes
		for k, v := range msg.Attributes {
			m.Attributes[k] = v
		}

		messages = append(messages, m)
	}

	return messages, nil
}

// DeleteMessage deletes a message from a queue
func (c *sqsClient) DeleteMessage(ctx context.Context, queueURL, receiptHandle string) error {
	// Fail-fast: Validate inputs
	if queueURL == "" {
		return fmt.Errorf("queue URL cannot be empty")
	}
	if receiptHandle == "" {
		return fmt.Errorf("receipt handle cannot be empty")
	}

	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	}

	_, err := c.client.DeleteMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete message from SQS: %w", err)
	}

	return nil
}

// CreateQueue creates a new queue
func (c *sqsClient) CreateQueue(ctx context.Context, queueName string) (string, error) {
	// Fail-fast: Validate inputs
	if queueName == "" {
		return "", fmt.Errorf("queue name cannot be empty")
	}

	input := &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	}

	output, err := c.client.CreateQueue(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create SQS queue: %w", err)
	}

	return aws.ToString(output.QueueUrl), nil
}

// DeleteQueue deletes a queue
func (c *sqsClient) DeleteQueue(ctx context.Context, queueURL string) error {
	// Fail-fast: Validate inputs
	if queueURL == "" {
		return fmt.Errorf("queue URL cannot be empty")
	}

	input := &sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	}

	_, err := c.client.DeleteQueue(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete SQS queue: %w", err)
	}

	return nil
}
