# Eureka-like Service Registry

A Eureka-compatible service registry implementation for Fluxor, providing service discovery and registration capabilities.

## Features

- ✅ **Service Registration**: Services can register themselves with metadata
- ✅ **Service Discovery**: Discover healthy instances of services
- ✅ **Heartbeat/Renewal**: Automatic lease renewal to keep services registered
- ✅ **Lease Expiration**: Automatic eviction of expired instances
- ✅ **REST API**: Eureka-compatible REST endpoints
- ✅ **Client Library**: Easy-to-use client for service registration
- ✅ **Multiple Instances**: Support for multiple instances of the same service
- ✅ **Status Management**: Update service instance status (UP, DOWN, OUT_OF_SERVICE, etc.)

## Architecture

The service registry consists of:

1. **EurekaVerticle**: Reusable verticle that runs the registry HTTP server with REST API (like dashboard)
2. **Registry**: Core registry implementation managing service instances
3. **Client**: Client library for services to register and discover
4. **Handler**: HTTP handlers for REST API endpoints

## Quick Start

### 1. Start the Registry Server (Recommended: Using EurekaVerticle)

The easiest way to add Eureka service registry to your application is to deploy the `EurekaVerticle`:

```go
package main

import (
	"log"

	"github.com/fluxorio/fluxor/pkg/entrypoint"
	"github.com/fluxorio/fluxor/pkg/eureka"
)

func main() {
	// Create main application
	app, err := entrypoint.NewMainVerticle("config.json")
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	// Deploy your application verticles
	app.DeployVerticle(NewMyServiceVerticle())
	
	// Deploy Eureka service registry (add-on)
	app.DeployVerticle(eureka.NewEurekaVerticle())
	
	// Start application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}
}
```

#### Custom Configuration

You can customize the Eureka configuration:

```go
config := eureka.EurekaVerticleConfig{
	Address: ":8761",  // Custom port (default: :8761)
	Prefix:  "",      // Route prefix (default: "" for root)
	RegistryConfig: &eureka.RegistryConfig{
		DefaultLeaseDuration: 90 * time.Second,
		RenewalThreshold:     30 * time.Second,
		EvictionInterval:     60 * time.Second,
	},
	EnableEviction: true,  // Enable automatic eviction
}
app.DeployVerticle(eureka.NewEurekaVerticleWithConfig(config))
```

#### Configuration from Context

If `Address` is empty, the verticle will use the `eureka_addr` value from the context config, or default to `:8761`:

```go
// In your config file:
{
  "eureka_addr": ":8761"
}
```

### Alternative: Manual Route Registration

If you prefer to integrate Eureka routes into your existing HTTP server:

```go
import (
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/eureka"
)

// In your verticle's Start() method:
router := server.FastRouter()

// Create registry
registry := eureka.NewRegistry(eureka.DefaultRegistryConfig())

// Register Eureka routes
eureka.Register(router, "", registry)  // "" = no prefix, "/eureka" = /eureka prefix
```
```

### 2. Register a Service

```go
package main

import (
	"context"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/eureka"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
)

type MyServiceVerticle struct {
	*core.BaseVerticle
	client *eureka.Client
}

func NewMyServiceVerticle() *MyServiceVerticle {
	// Create service instance
	instance := &eureka.ServiceInstance{
		ServiceName: "my-service",
		Host:        "localhost",
		Port:        8080,
		Status:      eureka.InstanceStatusUp,
		Metadata: map[string]string{
			"version": "1.0.0",
			"region":  "us-east-1",
		},
	}

	// Create client
	clientConfig := eureka.DefaultClientConfig("http://localhost:8761", instance)
	client := eureka.NewClient(clientConfig)

	return &MyServiceVerticle{
		BaseVerticle: core.NewBaseVerticle("my-service"),
		client:       client,
	}
}

func (v *MyServiceVerticle) Start(ctx core.FluxorContext) error {
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	// Register with Eureka
	registerCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := v.client.Register(registerCtx); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	// Start heartbeat
	heartbeatCtx := ctx.GoCMD().Context()
	if err := v.client.StartHeartbeat(heartbeatCtx); err != nil {
		return fmt.Errorf("failed to start heartbeat: %w", err)
	}

	return nil
}

func (v *MyServiceVerticle) Stop(ctx core.FluxorContext) error {
	// Stop heartbeat
	v.client.StopHeartbeat()

	// Unregister
	unregisterCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = v.client.Unregister(unregisterCtx)

	return v.BaseVerticle.Stop(ctx)
}
```

### 3. Discover Services

```go
// Discover instances of a service
discoverCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

instances, err := client.Discover(discoverCtx, "my-service")
if err != nil {
	log.Fatal(err)
}

for _, instance := range instances {
	fmt.Printf("Found instance: %s:%d (status: %s)\n",
		instance.Host, instance.Port, instance.Status)
}
```

## REST API

The registry server provides Eureka-compatible REST endpoints:

### Register Instance
```
POST /eureka/apps/{appName}
Content-Type: application/json

{
  "instance": {
    "instanceId": "my-service:localhost:8080",
    "serviceName": "my-service",
    "host": "localhost",
    "port": 8080,
    "status": "UP",
    "metadata": {
      "version": "1.0.0"
    }
  }
}
```

### Renew Lease (Heartbeat)
```
PUT /eureka/apps/{appName}/{instanceId}
```

### Unregister Instance
```
DELETE /eureka/apps/{appName}/{instanceId}
```

### Get Service Instances
```
GET /eureka/apps/{appName}
GET /eureka/apps/{appName}?status=all  # Include unhealthy/expired
```

### Get All Services
```
GET /eureka/apps
```

### Get Instance
```
GET /eureka/apps/{appName}/{instanceId}
```

### Update Instance Status
```
PUT /eureka/apps/{appName}/{instanceId}/status?value=DOWN
```

### Delete Status Override
```
DELETE /eureka/apps/{appName}/{instanceId}/status
```

### Health Check
```
GET /health
```

### Statistics
```
GET /stats
```

## Configuration

### Server Configuration

```go
config := &eureka.ServerConfig{
	Address: ":8761",  // Server address
	RegistryConfig: &eureka.RegistryConfig{
		DefaultLeaseDuration: 90 * time.Second,  // Lease duration
		RenewalThreshold:     30 * time.Second,  // Renewal interval
		EvictionInterval:     60 * time.Second,   // Eviction check interval
	},
	EnableEviction: true,  // Enable automatic eviction
}
```

### Client Configuration

```go
config := &eureka.ClientConfig{
	RegistryURL:    "http://localhost:8761",
	Instance:       instance,
	RenewalInterval: 30 * time.Second,  // Heartbeat interval
	RequestTimeout:  5 * time.Second,   // HTTP request timeout
}
```

## Instance Status

- `UP`: Instance is healthy and available
- `DOWN`: Instance is down/unhealthy
- `STARTING`: Instance is starting up
- `OUT_OF_SERVICE`: Instance is intentionally taken out of service

## Lease Management

- **Lease Duration**: Default 90 seconds
- **Renewal Interval**: Services should renew every 30 seconds (1/3 of lease)
- **Expiration**: Instances that don't renew within lease duration are evicted
- **Automatic Eviction**: Server periodically evicts expired instances

## Best Practices

1. **Always Start Heartbeat**: After registering, always start the heartbeat to keep the lease alive
2. **Graceful Shutdown**: Unregister on shutdown to immediately remove from registry
3. **Health Checks**: Update status to DOWN if health checks fail
4. **Metadata**: Use metadata to store version, region, zone, etc.
5. **Instance ID**: Use unique instance IDs (e.g., `{serviceName}:{host}:{port}`)

## Example: Complete Service

```go
type UserServiceVerticle struct {
	*core.BaseVerticle
	client *eureka.Client
}

func (v *UserServiceVerticle) Start(ctx core.FluxorContext) error {
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	// Get host and port from config
	host := "localhost"
	port := 8080

	// Create instance
	instance := &eureka.ServiceInstance{
		ServiceName: "user-service",
		InstanceID:  fmt.Sprintf("user-service:%s:%d", host, port),
		Host:        host,
		Port:        port,
		Status:      eureka.InstanceStatusUp,
		Metadata: map[string]string{
			"version": "1.0.0",
			"region":  "us-east-1",
		},
		HealthCheckURL: fmt.Sprintf("http://%s:%d/health", host, port),
	}

	// Create and configure client
	clientConfig := eureka.DefaultClientConfig("http://localhost:8761", instance)
	client := eureka.NewClient(clientConfig)
	v.client = client

	// Register
	if err := client.Register(ctx.GoCMD().Context()); err != nil {
		return err
	}

	// Start heartbeat
	if err := client.StartHeartbeat(ctx.GoCMD().Context()); err != nil {
		return err
	}

	// Set up health check endpoint
	// ... your HTTP server setup ...

	return nil
}

func (v *UserServiceVerticle) Stop(ctx core.FluxorContext) error {
	// Stop heartbeat
	v.client.StopHeartbeat()

	// Unregister
	_ = v.client.Unregister(ctx.GoCMD().Context())

	return v.BaseVerticle.Stop(ctx)
}
```

## Integration with Fluxor Patterns

The Eureka registry follows Fluxor patterns:

- **Reusable Verticle**: Like `dashboard.DashboardVerticle`, `EurekaVerticle` can be deployed into any application
- **HTTP Server**: Runs as a FastHTTP server with REST API endpoints
- **Non-Blocking**: All operations are non-blocking
- **Fail-Fast**: Uses fail-fast validation for all inputs
- **Event-Driven**: Uses EventBus for internal communication (future enhancement)

## Comparison with Dashboard Pattern

The Eureka implementation follows the same reusable pattern as the dashboard:

| Feature | Dashboard | Eureka |
|---------|-----------|--------|
| Reusable Verticle | `DashboardVerticle` | `EurekaVerticle` |
| Manual Registration | `dashboard.Register()` | `eureka.Register()` |
| Config | `DashboardVerticleConfig` | `EurekaVerticleConfig` |
| Default Port | `:8080` | `:8761` |
| Route Prefix | Supported | Supported |

## Future Enhancements

- [ ] Clustered registry (multiple registry servers)
- [ ] EventBus integration for service discovery events
- [ ] Load balancing strategies (round-robin, weighted, etc.)
- [ ] Service mesh integration
- [ ] Metrics and observability
- [ ] WebSocket support for real-time updates
