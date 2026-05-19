# Fluxor Service Mesh

This package provides internal service mesh capabilities for Fluxor applications, built on top of the Fluxor EventBus.

## Features

- **Circuit Breaker**: Protects your services from cascading failures by stopping calls to failing services.
- **Retries**: Configurable retry policies with exponential backoff for transient failures.
- **Service Abstraction**: Simple `Call` API that handles the complexity of reliability patterns.

## Usage

```go
import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/mesh"
)

func main() {
    // ... setup vertx ...
    eventBus := vertx.EventBus()
    
    // Create Mesh
    m := mesh.NewServiceMesh(eventBus)
    
    // Register Service (optional, initializes breaker)
    m.Register("my-service")
    
    // Call Service
    opts := mesh.CallOptions{
        Timeout: 5 * time.Second,
        RetryPolicy: mesh.DefaultRetryPolicy(),
    }
    
    resp, err := m.Call(ctx, "my-service", "my-action", payload, opts)
    if err != nil {
        log.Println("Call failed:", err)
    }
}
```

## Architecture

The Mesh layer sits between your application logic and the EventBus. It intercepts calls to apply resilience patterns before delegating to the underlying transport (NATS, in-memory, etc.).
