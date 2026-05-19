# Webhook Package

The `webhook` package provides a secure webhook receiver for Fluxor applications. It supports HMAC signature validation, EventBus integration, and multiple webhook endpoints.

## Features

- **HMAC Signature Validation**: Supports HMAC-SHA1, HMAC-SHA256, and HMAC-SHA512
- **Multiple Endpoints**: Register multiple webhook endpoints with different configurations
- **EventBus Integration**: Automatically publishes webhook events to EventBus
- **FastHTTP Integration**: Middleware and router helpers for FastHTTP
- **Custom Validators**: Support for custom signature validation logic
- **GitHub/Stripe Support**: Built-in validators for GitHub and Stripe webhooks

## Usage

### Basic Example

```go
package main

import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/web"
    "github.com/fluxorio/fluxor/pkg/webhook"
)

func main() {
    ctx := context.Background()
    gocmd := core.NewGoCMD(ctx)
    defer gocmd.Close()
    fluxorCtx := core.NewFluxorContext(ctx, gocmd)

    // Create webhook receiver configuration
    config := &webhook.ReceiverConfig{
        Prefix: "/webhooks",
        Endpoints: []webhook.EndpointConfig{
            {
                Path:            "/github",
                Secret:          "your-github-webhook-secret",
                EventBusAddress: "webhook.github",
            },
        },
    }

    // Create receiver
    receiver := webhook.NewReceiver(config)
    if err := receiver.Start(fluxorCtx); err != nil {
        panic(err)
    }
    defer receiver.Stop(fluxorCtx)

    // Create HTTP server
    serverConfig := web.DefaultFastHTTPServerConfig(":8080")
    server := web.NewFastHTTPServer(gocmd, serverConfig)
    router := server.FastRouter()

    // Register webhook routes
    if err := webhook.RegisterRoutes(router, receiver); err != nil {
        panic(err)
    }

    // Start server
    if err := server.Start(); err != nil {
        panic(err)
    }
}
```

### Using Middleware

```go
// Create receiver
receiver := webhook.NewReceiver(config)
receiver.Start(fluxorCtx)

// Add webhook middleware
router.UseFast(webhook.FastHTTPMiddleware(receiver))

// Register other routes
router.GETFast("/health", healthHandler)
```

### GitHub Webhook Example

```go
config := &webhook.ReceiverConfig{
    Prefix: "/webhooks",
    Endpoints: []webhook.EndpointConfig{
        {
            Path:            "/github",
            Secret:          "your-github-webhook-secret",
            Algorithm:       webhook.SignatureAlgorithmHMACSHA256,
            SignatureHeader: "X-Hub-Signature-256",
            SignaturePrefix: "sha256=",
            EventBusAddress: "webhook.github",
        },
    },
}

receiver := webhook.NewReceiver(config)

// Or use the GitHub-specific validator
endpointConfig := webhook.EndpointConfig{
    Path:            "/github",
    EventBusAddress: "webhook.github",
    CustomValidator: webhook.NewGitHubSignatureValidator("your-github-webhook-secret"),
}
```

### Stripe Webhook Example

```go
endpointConfig := webhook.EndpointConfig{
    Path:            "/stripe",
    EventBusAddress: "webhook.stripe",
    CustomValidator: webhook.NewStripeSignatureValidator("your-stripe-webhook-secret"),
}

// Note: Stripe signature format is more complex (includes timestamp)
// For production, consider using Stripe's official SDK
```

### Custom Signature Validator

```go
type CustomValidator struct {
    secret string
}

func (v *CustomValidator) Validate(payload []byte, signature string, headers map[string]string) error {
    // Custom validation logic
    expectedSignature := computeSignature(payload, v.secret)
    if signature != expectedSignature {
        return fmt.Errorf("invalid signature")
    }
    return nil
}

endpointConfig := webhook.EndpointConfig{
    Path:            "/custom",
    EventBusAddress: "webhook.custom",
    CustomValidator: &CustomValidator{secret: "secret"},
}
```

### Skipping Validation (Not Recommended for Production)

```go
endpointConfig := webhook.EndpointConfig{
    Path:            "/insecure",
    EventBusAddress: "webhook.insecure",
    SkipValidation: true, // ⚠️ Only for development/testing
}
```

## EventBus Integration

Webhook events are automatically published to EventBus with the following structure:

```json
{
    "path": "/webhooks/github",
    "payload": <raw-payload-bytes>,
    "headers": {
        "X-Hub-Signature-256": "sha256=...",
        "Content-Type": "application/json",
        ...
    },
    "queryParams": {
        "param": "value"
    },
    "method": "POST"
}
```

To consume webhook events:

```go
consumer := eventBus.Consumer("webhook.github")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    event := msg.Body().(map[string]interface{})
    
    path := event["path"].(string)
    payload := event["payload"].([]byte)
    headers := event["headers"].(map[string]interface{})
    
    // Process webhook event
    return nil
})
```

## Configuration

### ReceiverConfig

- `Prefix` (string, required): URL prefix for all webhooks (e.g., "/webhooks")
- `Endpoints` ([]EndpointConfig, required): List of webhook endpoint configurations
- `OnError` (func(string, error) error, optional): Custom error handler

### EndpointConfig

- `Path` (string, required): Webhook path (e.g., "/github")
- `Secret` (string, optional): Secret for HMAC signature validation
- `Algorithm` (SignatureAlgorithm, optional): HMAC algorithm (default: HMAC-SHA256)
- `SignatureHeader` (string, optional): Header name containing signature (default: "X-Webhook-Signature")
- `SignaturePrefix` (string, optional): Signature prefix (default: "sha256=")
- `EventBusAddress` (string, required): EventBus address to publish events
- `CustomValidator` (SignatureValidator, optional): Custom signature validator
- `SkipValidation` (bool, optional): Skip signature validation (not recommended for production)

## Signature Algorithms

- `SignatureAlgorithmHMACSHA1`: HMAC-SHA1
- `SignatureAlgorithmHMACSHA256`: HMAC-SHA256 (default)
- `SignatureAlgorithmHMACSHA512`: HMAC-SHA512

## Security Considerations

1. **Always validate signatures in production**: Never use `SkipValidation: true` in production
2. **Use strong secrets**: Generate cryptographically random secrets
3. **Keep secrets secure**: Store secrets in secure storage (environment variables, secret managers)
4. **HTTPS only**: Always use HTTPS for webhook endpoints in production
5. **Rate limiting**: Consider adding rate limiting for webhook endpoints

## Examples

See the `examples/` directory for complete examples:
- GitHub webhook receiver
- Stripe webhook receiver
- Custom validator implementation

## Testing

```bash
go test ./pkg/webhook
```

## License

Part of the Fluxor project.