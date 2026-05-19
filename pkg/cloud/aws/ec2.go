package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// ec2Client implements EC2Client
type ec2Client struct {
	client *ec2.Client
}

// newEC2Client creates a new EC2 client
func newEC2Client(cfg aws.Config) EC2Client {
	return &ec2Client{
		client: ec2.NewFromConfig(cfg),
	}
}

// DescribeInstances describes EC2 instances
func (c *ec2Client) DescribeInstances(ctx context.Context, instanceIDs []string) ([]EC2Instance, error) {
	input := &ec2.DescribeInstancesInput{}
	if len(instanceIDs) > 0 {
		input.InstanceIds = instanceIDs
	}

	output, err := c.client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe EC2 instances: %w", err)
	}

	var instances []EC2Instance
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			inst := EC2Instance{
				InstanceID:   aws.ToString(instance.InstanceId),
				InstanceType: string(instance.InstanceType),
				Tags:        make(map[string]string),
			}

			// Get state
			if instance.State != nil {
				inst.State = string(instance.State.Name)
			}

			// Get private IP
			if instance.PrivateIpAddress != nil {
				inst.PrivateIP = *instance.PrivateIpAddress
			}

			// Get public IP
			if instance.PublicIpAddress != nil {
				inst.PublicIP = *instance.PublicIpAddress
			}

			// Get tags
			for _, tag := range instance.Tags {
				if tag.Key != nil && tag.Value != nil {
					inst.Tags[*tag.Key] = *tag.Value
				}
			}

			instances = append(instances, inst)
		}
	}

	return instances, nil
}

// StartInstance starts an EC2 instance
func (c *ec2Client) StartInstance(ctx context.Context, instanceID string) error {
	// Fail-fast: Validate inputs
	if instanceID == "" {
		return fmt.Errorf("instance ID cannot be empty")
	}

	input := &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	}

	_, err := c.client.StartInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to start EC2 instance: %w", err)
	}

	return nil
}

// StopInstance stops an EC2 instance
func (c *ec2Client) StopInstance(ctx context.Context, instanceID string) error {
	// Fail-fast: Validate inputs
	if instanceID == "" {
		return fmt.Errorf("instance ID cannot be empty")
	}

	input := &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	}

	_, err := c.client.StopInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to stop EC2 instance: %w", err)
	}

	return nil
}

// TerminateInstance terminates an EC2 instance
func (c *ec2Client) TerminateInstance(ctx context.Context, instanceID string) error {
	// Fail-fast: Validate inputs
	if instanceID == "" {
		return fmt.Errorf("instance ID cannot be empty")
	}

	input := &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	}

	_, err := c.client.TerminateInstances(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to terminate EC2 instance: %w", err)
	}

	return nil
}
