package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// snsClient implements SNSClient
type snsClient struct {
	client *sns.Client
}

// newSNSClient creates a new SNS client
func newSNSClient(cfg aws.Config) SNSClient {
	return &snsClient{
		client: sns.NewFromConfig(cfg),
	}
}

// Publish publishes a message to a topic
func (c *snsClient) Publish(ctx context.Context, topicARN string, message string) error {
	// Fail-fast: Validate inputs
	if topicARN == "" {
		return fmt.Errorf("topic ARN cannot be empty")
	}
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	input := &sns.PublishInput{
		TopicArn: aws.String(topicARN),
		Message:  aws.String(message),
	}

	_, err := c.client.Publish(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to publish message to SNS: %w", err)
	}

	return nil
}

// Subscribe subscribes to a topic
func (c *snsClient) Subscribe(ctx context.Context, topicARN, protocol, endpoint string) (string, error) {
	// Fail-fast: Validate inputs
	if topicARN == "" {
		return "", fmt.Errorf("topic ARN cannot be empty")
	}
	if protocol == "" {
		return "", fmt.Errorf("protocol cannot be empty")
	}
	if endpoint == "" {
		return "", fmt.Errorf("endpoint cannot be empty")
	}

	input := &sns.SubscribeInput{
		TopicArn: aws.String(topicARN),
		Protocol:  aws.String(protocol),
		Endpoint: aws.String(endpoint),
	}

	output, err := c.client.Subscribe(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to subscribe to SNS topic: %w", err)
	}

	return aws.ToString(output.SubscriptionArn), nil
}

// Unsubscribe unsubscribes from a topic
func (c *snsClient) Unsubscribe(ctx context.Context, subscriptionARN string) error {
	// Fail-fast: Validate inputs
	if subscriptionARN == "" {
		return fmt.Errorf("subscription ARN cannot be empty")
	}

	input := &sns.UnsubscribeInput{
		SubscriptionArn: aws.String(subscriptionARN),
	}

	_, err := c.client.Unsubscribe(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from SNS topic: %w", err)
	}

	return nil
}

// CreateTopic creates a new topic
func (c *snsClient) CreateTopic(ctx context.Context, topicName string) (string, error) {
	// Fail-fast: Validate inputs
	if topicName == "" {
		return "", fmt.Errorf("topic name cannot be empty")
	}

	input := &sns.CreateTopicInput{
		Name: aws.String(topicName),
	}

	output, err := c.client.CreateTopic(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create SNS topic: %w", err)
	}

	return aws.ToString(output.TopicArn), nil
}

// DeleteTopic deletes a topic
func (c *snsClient) DeleteTopic(ctx context.Context, topicARN string) error {
	// Fail-fast: Validate inputs
	if topicARN == "" {
		return fmt.Errorf("topic ARN cannot be empty")
	}

	input := &sns.DeleteTopicInput{
		TopicArn: aws.String(topicARN),
	}

	_, err := c.client.DeleteTopic(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete SNS topic: %w", err)
	}

	return nil
}
