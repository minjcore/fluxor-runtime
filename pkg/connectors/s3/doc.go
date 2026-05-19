// Package s3 provides Amazon S3 integration for Fluxor.
//
// This package implements the Connector interface and provides a high-level
// API for S3 operations including uploading, downloading, deleting, and listing objects.
//
// Example usage:
//
//	// Create S3 component with configuration
//	config := s3.DefaultConfig()
//	config.Bucket = "my-bucket"
//	config.Region = "us-east-1"
//
//	component := s3.NewS3Component(config)
//
//	// Start the component
//	ctx := core.NewFluxorContext(...)
//	if err := component.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer component.Stop(ctx)
//
//	// Create client for operations
//	client, err := s3.NewClient(component)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Upload an object
//	data := []byte("Hello, S3!")
//	if err := client.PutObject(context.Background(), "my-bucket", "hello.txt", data, "text/plain"); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Download an object
//	downloaded, err := client.GetObject(context.Background(), "my-bucket", "hello.txt")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List objects
//	objects, err := client.ListObjects(context.Background(), "my-bucket", "prefix/")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Check if object exists
//	exists, err := client.ObjectExists(context.Background(), "my-bucket", "hello.txt")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Delete an object
//	if err := client.DeleteObject(context.Background(), "my-bucket", "hello.txt"); err != nil {
//	    log.Fatal(err)
//	}
//
// Features:
//   - Upload objects to S3
//   - Download objects from S3
//   - Delete objects from S3
//   - List objects in buckets
//   - Check if objects exist
//   - Support for default bucket configuration
//   - Support for AWS credentials (access key, IAM role, etc.)
//   - Support for LocalStack for local testing
//
// Configuration:
//   - AWS_REGION: AWS region (required)
//   - AWS_ACCESS_KEY_ID: AWS access key ID (optional if using IAM role)
//   - AWS_SECRET_ACCESS_KEY: AWS secret access key (optional if using IAM role)
//   - AWS_SESSION_TOKEN: AWS session token (optional, for temporary credentials)
//   - AWS_ENDPOINT: AWS endpoint URL (optional, for LocalStack)
//   - S3_BUCKET: Default S3 bucket name (optional)
//   - S3_DEFAULT_CONTENT_TYPE: Default content type for uploads (default: application/octet-stream)
//   - S3_TIMEOUT: Timeout for S3 API calls (default: 30s)
//   - S3_MAX_RETRIES: Maximum retries for S3 API calls (default: 3)
//   - S3_DEBUG: Enable debug logging (default: false)
package s3
