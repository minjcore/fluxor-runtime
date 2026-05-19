# DevOps Package

The `devops` package provides DevOps utilities and tools for Fluxor applications. This package is designed to support common DevOps operations and can be extended with specific features as needed.

## Multi-runtime vision

This package is the **deploy** side of Fluxor’s multi-runtime story. The sibling package **pkg/shell** handles **local process run** (terminal/command interface: e.g. `fluxor run app.py`, `fluxor ps`, `fluxor logs` via `ProcessRuntime`). Together:

- **Local run**: `fluxor run app.py` / `main.go` → `pkg/shell.ProcessRuntime`
- **Deploy**: `fluxor deploy app.yaml` → `pkg/devops.DockerRuntime` (Docker Compose on VPS)

Other runtimes (e.g. Kubernetes, Docker Swarm) can be added later as further implementations of the same concepts.

## Current Status

This package is currently a foundational structure that can be extended with specific DevOps features. The basic structure includes:

- Health check types and interfaces
- Basic health status definitions

## Potential Use Cases

This package can be extended to support:

### Health Checks
- Component health monitoring
- Service health aggregation
- Health check endpoints
- Dependency health tracking

### Metrics Collection
- Application metrics
- Performance metrics
- Business metrics
- Custom metric collection

### Deployment Automation
- Deployment status tracking
- Rollback capabilities
- Version management
- Blue-green deployment support

### Monitoring Integration
- Prometheus metrics export
- OpenTelemetry integration
- Custom monitoring backends
- Alerting integration

### CI/CD Integration
- Build status tracking
- Pipeline integration
- Test result aggregation
- Release management

### Logging & Observability
- Structured logging
- Log aggregation
- Distributed tracing
- Error tracking

## Basic Usage

```go
package main

import (
    "context"
    "github.com/fluxorio/fluxor/pkg/devops"
)

func main() {
    ctx := context.Background()
    
    // Create a health check
    check := devops.NewHealthCheck(
        devops.HealthStatusHealthy,
        "Service is operational",
    )
    
    // Add details
    check.WithDetails("version", "1.0.0")
    check.WithDetails("uptime", "2h30m")
}
```

## Future Extensions

As the package evolves, it will include:

- Health check registry and aggregation
- Metrics collection and export
- Deployment automation tools
- Monitoring integrations
- CI/CD utilities

## Contributing

When adding new DevOps features, consider:

1. **Consistency**: Follow existing patterns in the Fluxor codebase
2. **Extensibility**: Design for future enhancements
3. **Integration**: Work well with other Fluxor packages
4. **Documentation**: Keep this README updated

