// Package aws provides AWS cloud integration for Fluxor.
//
// This package integrates AWS services (S3, SQS, SNS, Lambda, EC2) with Fluxor's
// EventBus and component system, following Fluxor's fail-fast and reactive patterns.
//
// Features:
//   - Full AWS SDK v2 integration
//   - EventBus integration for AWS events
//   - Component lifecycle management
//   - Support for S3, SQS, SNS, Lambda, EC2
//   - LocalStack support for local development
//   - Fail-fast validation
//
// Example usage:
//
//	import (
//	    "github.com/fluxorio/fluxor/pkg/cloud/aws"
//	    "github.com/fluxorio/fluxor/pkg/core"
//	)
//
//	// Create AWS component
//	awsConfig := aws.DefaultConfig()
//	awsConfig.Region = "us-east-1"
//	awsComponent := aws.NewAWSComponent(awsConfig)
//
//	// Start component (in a verticle)
//	verticle := &MyVerticle{
//	    aws: awsComponent,
//	}
//	verticle.Start(ctx)
//
//	// Use AWS services
//	s3, _ := awsComponent.S3()
//	s3.PutObject(ctx, "my-bucket", "key", []byte("data"), "text/plain")
//
//	// Listen to AWS events via EventBus
//	eventBus.Consumer("aws.ready").Handler(func(ctx core.FluxorContext, msg core.Message) error {
//	    // Handle AWS ready event
//	    return nil
//	})
package aws
