package aws

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// awsClient implements AWSClient
type awsClient struct {
	config aws.Config
	region string
	s3     S3Client
	sqs    SQSClient
	sns    SNSClient
	lambda LambdaClient
	ec2    EC2Client
}

// NewClient creates a new AWS client
// Fail-fast: Validates configuration
func NewClient(config Config) (AWSClient, error) {
	// Fail-fast: Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Load AWS SDK config
	ctx := context.Background()
	cfg, err := loadAWSConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := &awsClient{
		config: cfg,
		region: config.Region,
	}

	// Initialize service clients
	client.s3 = newS3Client(cfg)
	client.sqs = newSQSClient(cfg)
	client.sns = newSNSClient(cfg)
	client.lambda = newLambdaClient(cfg)
	client.ec2 = newEC2Client(cfg)

	return client, nil
}

// loadAWSConfig loads AWS SDK configuration
func loadAWSConfig(ctx context.Context, config Config) (aws.Config, error) {
	var opts []func(*awsconfig.LoadOptions) error

	// Set region
	opts = append(opts, awsconfig.WithRegion(config.Region))

	// Set credentials if provided
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		creds := credentials.NewStaticCredentialsProvider(
			config.AccessKeyID,
			config.SecretAccessKey,
			config.SessionToken,
		)
		opts = append(opts, awsconfig.WithCredentialsProvider(creds))
	}

	// Set custom endpoint (for LocalStack)
	if config.Endpoint != "" {
		opts = append(opts, awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           config.Endpoint,
					SigningRegion: region,
				}, nil
			}),
		))
	}

	// Load config
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, err
	}

	return cfg, nil
}

// S3 returns the S3 client
func (c *awsClient) S3() S3Client {
	return c.s3
}

// SQS returns the SQS client
func (c *awsClient) SQS() SQSClient {
	return c.sqs
}

// SNS returns the SNS client
func (c *awsClient) SNS() SNSClient {
	return c.sns
}

// Lambda returns the Lambda client
func (c *awsClient) Lambda() LambdaClient {
	return c.lambda
}

// EC2 returns the EC2 client
func (c *awsClient) EC2() EC2Client {
	return c.ec2
}

// Region returns the AWS region
func (c *awsClient) Region() string {
	return c.region
}

// getEnvOrDefault returns environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsGoogleCloud checks if the code is running on Google Cloud Platform.
// This function checks multiple indicators to determine GCP presence:
//   - Environment variables: GOOGLE_CLOUD_PROJECT, GCP_PROJECT, GCE_METADATA_HOST
//   - GCE metadata server availability (169.254.169.254)
//
// Returns true if running on GCP, false otherwise.
// This is useful for multi-cloud deployments where you need to detect
// the cloud provider at runtime.
func IsGoogleCloud() bool {
	// Check environment variables first (fastest method)
	if os.Getenv("GOOGLE_CLOUD_PROJECT") != "" {
		return true
	}
	if os.Getenv("GCP_PROJECT") != "" {
		return true
	}
	if os.Getenv("GCE_METADATA_HOST") != "" {
		return true
	}

	// Check for GCE_METADATA_ROOT environment variable (another GCP indicator)
	if os.Getenv("GCE_METADATA_ROOT") != "" {
		return true
	}

	// Try to reach GCE metadata server
	// GCP metadata server is at 169.254.169.254 (link-local address)
	return checkGCPMetadataServer()
}

// checkGCPMetadataServer attempts to connect to the GCE metadata server
// to verify if we're running on Google Cloud Platform.
func checkGCPMetadataServer() bool {
	client := &http.Client{
		Timeout: 500 * time.Millisecond, // Short timeout for fast detection
	}

	// Try the standard GCE metadata server endpoint
	req, err := http.NewRequestWithContext(
		context.Background(),
		"GET",
		"http://169.254.169.254/computeMetadata/v1/",
		nil,
	)
	if err != nil {
		return false
	}

	// GCE metadata server requires this header
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check for GCP metadata server response
	// GCP metadata server should return 200 and include "Google" in Metadata-Flavor header
	if resp.StatusCode == http.StatusOK {
		metadataFlavor := resp.Header.Get("Metadata-Flavor")
		if strings.Contains(strings.ToLower(metadataFlavor), "google") {
			return true
		}
	}

	return false
}

// GetGoogleCloudProject returns the Google Cloud Project ID if running on GCP,
// or an empty string otherwise.
// Checks environment variables: GOOGLE_CLOUD_PROJECT, GCP_PROJECT
func GetGoogleCloudProject() string {
	if project := os.Getenv("GOOGLE_CLOUD_PROJECT"); project != "" {
		return project
	}
	if project := os.Getenv("GCP_PROJECT"); project != "" {
		return project
	}
	return ""
}
