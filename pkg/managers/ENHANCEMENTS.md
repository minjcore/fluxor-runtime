# Managers Package Enhancements

This document describes the new features added to the Managers (Exmanagerstion Control Unit) package.

## 🎯 Overview

The Managers package has been enhanced with three major features:

1. **Service Registry** - Track and manage all deployed services
2. **Health Checking** - Aggregate health checks from all components
3. **Shutdown Coordination** - Graceful multi-phase shutdown

---

## 1. Service Registry

### Purpose

Track all deployed services, their status, dependencies, and metadata in a centralized registry.

### Features

- ✅ Service registration with type and dependencies
- ✅ Service status tracking (starting, healthy, unhealthy, stopping, stopped)
- ✅ Service metadata storage
- ✅ Dependency tracking
- ✅ Health summary aggregation

### Usage

```go
// Register a service
serviceInfo := managersInstance.RegisterService(
    "user-service",           // Service name
    "verticle",              // Service type
    []string{"database"},    // Dependencies
)

// Update service status
managersInstance.UpdateServiceStatus("user-service", managers.ServiceStatusHealthy)

// Set service metadata
serviceInfo.Metadata["version"] = "1.0.0"
serviceInfo.Metadata["endpoint"] = "/api/users"

// List all services
services := managersInstance.ListServices()
for _, svc := range services {
    fmt.Printf("%s: %s\n", svc.Name, svc.Status)
}

// Check if all services are healthy
if managersInstance.IsAllServicesHealthy() {
    fmt.Println("All services healthy!")
}

// Get health summary
summary := managersInstance.ServiceHealthSummary()
fmt.Printf("Healthy: %d, Unhealthy: %d\n",
    summary[managers.ServiceStatusHealthy],
    summary[managers.ServiceStatusUnhealthy])
```

### Integration Pattern

```go
// Verticle auto-registers with Managers
type UserServiceVerticle struct {
    *core.BaseVerticle
}

func (v *UserServiceVerticle) Start(ctx core.FluxorContext) error {
    // Get Managers from context
    managersInstance, _ := managers.GetManagers(ctx)
    
    // Register service
    managersInstance.RegisterService(
        "user-service",
        "verticle",
        []string{"database", "cache"},
    )
    
    // Start service
    if err := v.BaseVerticle.Start(ctx); err != nil {
        managersInstance.UpdateServiceStatus("user-service", managers.ServiceStatusUnhealthy)
        return err
    }
    
    // Mark as healthy
    managersInstance.UpdateServiceStatus("user-service", managers.ServiceStatusHealthy)
    
    return nil
}

func (v *UserServiceVerticle) Stop(ctx core.FluxorContext) error {
    managersInstance, _ := managers.GetManagers(ctx)
    
    // Mark as stopping
    managersInstance.UpdateServiceStatus("user-service", managers.ServiceStatusStopping)
    
    // Stop service
    if err := v.BaseVerticle.Stop(ctx); err != nil {
        return err
    }
    
    // Mark as stopped
    managersInstance.UpdateServiceStatus("user-service", managers.ServiceStatusStopped)
    managersInstance.UnregisterService("user-service")
    
    return nil
}
```

---

## 2. Health Checking

### Purpose

Aggregate health checks from all components and provide unified health status.

### Features

- ✅ Register custom health checks
- ✅ Run all health checks with timeout
- ✅ Per-check results with timing
- ✅ Aggregate health status
- ✅ Default health checks for Managers components

### Usage

```go
// Register health check
managersInstance.RegisterHealthCheck("database", func(ctx context.Context) error {
    // Check database connection
    return db.Ping(ctx)
})

managersInstance.RegisterHealthCheck("redis", func(ctx context.Context) error {
    // Check Redis connection
    _, err := redisClient.Ping(ctx).Result()
    return err
})

managersInstance.RegisterHealthCheck("external-api", func(ctx context.Context) error {
    // Check external API
    resp, err := http.Get("https://api.example.com/health")
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return fmt.Errorf("unhealthy: status %d", resp.StatusCode)
    }
    return nil
})

// Run all health checks
ctx := context.Background()
results := managersInstance.CheckHealth(ctx)

for _, result := range results {
    fmt.Printf("%s: healthy=%v, duration=%v\n",
        result.Name,
        result.Healthy,
        result.Duration)
    if !result.Healthy {
        fmt.Printf("  Error: %s\n", result.Error)
    }
}

// Check overall health
if managersInstance.IsHealthy(ctx) {
    fmt.Println("System is healthy")
} else {
    fmt.Println("System is unhealthy")
}

// Register default health checks
managersInstance.DefaultHealthChecks()
```

### HTTP Health Endpoint

```go
// Health check endpoint
router.GETFast("/health", func(ctx *web.FastRequestContext) error {
    healthy := managersInstance.IsHealthy(ctx.Context())
    status := 200
    if !healthy {
        status = 503
    }
    return ctx.JSON(status, map[string]interface{}{
        "healthy": healthy,
    })
})

// Detailed health endpoint
router.GETFast("/health/detailed", func(ctx *web.FastRequestContext) error {
    results := managersInstance.CheckHealth(ctx.Context())
    
    healthy := true
    details := make([]map[string]interface{}, len(results))
    
    for i, result := range results {
        if !result.Healthy {
            healthy = false
        }
        
        details[i] = map[string]interface{}{
            "name":      result.Name,
            "healthy":   result.Healthy,
            "duration":  result.Duration.String(),
            "timestamp": result.Timestamp,
        }
        if result.Error != "" {
            details[i]["error"] = result.Error
        }
    }
    
    status := 200
    if !healthy {
        status = 503
    }
    
    return ctx.JSON(status, map[string]interface{}{
        "healthy": healthy,
        "checks":  details,
    })
})
```

---

## 3. Shutdown Coordination

### Purpose

Coordinate graceful shutdown of all services in phases.

### Features

- ✅ Three-phase shutdown (PreStop, Stop, PostStop)
- ✅ Custom shutdown hooks per phase
- ✅ Timeout support
- ✅ Service status tracking during shutdown
- ✅ Default shutdown hooks

### Shutdown Phases

1. **PreStop**: Preparation phase
   - Mark services as stopping
   - Stop accepting new requests
   - Drain connections

2. **Stop**: Main shutdown phase
   - Stop services
   - Close connections
   - Release resources

3. **PostStop**: Finalization phase
   - Update service status
   - Log completion
   - Final cleanup

### Usage

```go
// Register custom shutdown hooks
managersInstance.RegisterShutdownHook(managers.ShutdownPhasePreStop, func(ctx context.Context) error {
    // Drain HTTP connections
    fmt.Println("Draining HTTP connections...")
    time.Sleep(5 * time.Second) // Grace period
    return nil
})

managersInstance.RegisterShutdownHook(managers.ShutdownPhaseStop, func(ctx context.Context) error {
    // Stop background workers
    fmt.Println("Stopping background workers...")
    return stopWorkers()
})

managersInstance.RegisterShutdownHook(managers.ShutdownPhasePostStop, func(ctx context.Context) error {
    // Log shutdown completion
    fmt.Println("Shutdown complete")
    return nil
})

// Register default hooks (recommended)
managersInstance.DefaultShutdownHooks()

// Perform graceful shutdown
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := managersInstance.Shutdown(ctx); err != nil {
    log.Printf("Shutdown error: %v", err)
}
```

### Signal Handling

```go
func main() {
    app, _ := entrypoint.NewMainVerticle("config.json")
    
    // Create Managers
    managersInstance, _ := managers.NewManagersWithGoCMD(app.GoCMD(), managers.DefaultConfig())
    
    // Setup Managers
    managersInstance.DefaultHealthChecks()
    managersInstance.DefaultShutdownHooks()
    
    // Deploy services
    app.DeployVerticle(NewUserServiceVerticle())
    app.DeployVerticle(NewOrderServiceVerticle())
    
    // Handle SIGINT/SIGTERM
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        fmt.Println("Received shutdown signal")
        
        // Graceful shutdown via Managers
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
        if err := managersInstance.Shutdown(ctx); err != nil {
            log.Printf("Shutdown error: %v", err)
            os.Exit(1)
        }
        
        os.Exit(0)
    }()
    
    app.Start()
}
```

---

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/managers"
    "github.com/fluxorio/fluxor/pkg/entrypoint"
)

func main() {
    app, _ := entrypoint.NewMainVerticle("config.json")
    
    // Create Managers
    managersInstance, _ := managers.NewManagersWithGoCMD(app.GoCMD(), managers.DefaultConfig())
    
    // Create and register components
    logger, _ := managersInstance.CreateLogger()
    cache, _ := managersInstance.CreateCache()
    httpServer, _ := managersInstance.CreateHTTPServer(app.GoCMD())
    
    managersInstance.RegisterLogger(logger)
    managersInstance.RegisterCache(cache)
    managersInstance.RegisterHTTPServer(httpServer)
    
    // Wire components
    managersInstance.Wire()
    
    // Setup health checks
    managersInstance.DefaultHealthChecks()
    managersInstance.RegisterHealthCheck("custom", func(ctx context.Context) error {
        // Custom health check
        return nil
    })
    
    // Setup shutdown coordination
    managersInstance.DefaultShutdownHooks()
    managersInstance.RegisterShutdownHook(managers.ShutdownPhasePreStop, func(ctx context.Context) error {
        fmt.Println("Custom PreStop hook")
        return nil
    })
    
    // Start heartbeat
    managersInstance.StartHeartbeat()
    
    // Deploy services
    userService := NewUserServiceVerticle()
    app.DeployVerticle(userService)
    
    // Register service with Managers
    managersInstance.RegisterService("user-service", "verticle", []string{"cache"})
    managersInstance.UpdateServiceStatus("user-service", managers.ServiceStatusHealthy)
    
    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        fmt.Println("Shutting down gracefully...")
        
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
        if err := managersInstance.Shutdown(ctx); err != nil {
            fmt.Printf("Shutdown error: %v\n", err)
            os.Exit(1)
        }
        
        os.Exit(0)
    }()
    
    // HTTP endpoints for health and status
    router := httpServer.(*web.Server).Router()
    
    router.GET("/health", func(ctx *web.RequestContext) error {
        healthy := managersInstance.IsHealthy(ctx.Context())
        status := 200
        if !healthy {
            status = 503
        }
        return ctx.JSON(status, map[string]interface{}{"healthy": healthy})
    })
    
    router.GET("/services", func(ctx *web.RequestContext) error {
        services := managersInstance.ListServices()
        return ctx.JSON(200, services)
    })
    
    router.GET("/health/detailed", func(ctx *web.RequestContext) error {
        results := managersInstance.CheckHealth(ctx.Context())
        return ctx.JSON(200, results)
    })
    
    // Start application
    app.Start()
}
```

---

## Best Practices

### 1. Service Registration

✅ **Register services on Start()**
```go
func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    managers, _ := managers.GetManagers(ctx)
    managers.RegisterService("my-service", "verticle", []string{"database"})
    // ...
}
```

✅ **Update status throughout lifecycle**
```go
managers.UpdateServiceStatus("my-service", managers.ServiceStatusHealthy)
```

✅ **Unregister on Stop()**
```go
func (v *MyVerticle) Stop(ctx core.FluxorContext) error {
    managers, _ := managers.GetManagers(ctx)
    managers.UnregisterService("my-service")
    // ...
}
```

### 2. Health Checks

✅ **Keep checks fast** (< 1 second)
```go
managers.RegisterHealthCheck("db", func(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
    defer cancel()
    return db.Ping(ctx)
})
```

✅ **Use default checks**
```go
managers.DefaultHealthChecks()  // HTTP server, cache, EventBus
```

✅ **Expose health endpoint**
```go
router.GET("/health", healthHandler)
router.GET("/health/detailed", detailedHealthHandler)
```

### 3. Shutdown Coordination

✅ **Register default hooks**
```go
managers.DefaultShutdownHooks()
```

✅ **Add custom hooks for cleanup**
```go
managers.RegisterShutdownHook(managers.ShutdownPhasePostStop, func(ctx context.Context) error {
    return cleanupTempFiles()
})
```

✅ **Handle signals properly**
```go
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
go func() {
    <-sigChan
    managers.Shutdown(context.WithTimeout(context.Background(), 30*time.Second))
}()
```

---

## API Reference

### Service Registry

```go
// Register service
RegisterService(name, serviceType string, dependencies []string) *ServiceInfo

// Unregister service
UnregisterService(name string)

// Get service info
GetService(name string) (*ServiceInfo, bool)

// List all services
ListServices() []*ServiceInfo

// Update service status
UpdateServiceStatus(name string, status ServiceStatus)

// Health summary
ServiceHealthSummary() map[ServiceStatus]int

// Check if all healthy
IsAllServicesHealthy() bool
```

### Health Checking

```go
// Register health check
RegisterHealthCheck(name string, check HealthCheck)

// Unregister health check
UnregisterHealthCheck(name string)

// Run all checks
CheckHealth(ctx context.Context) []HealthCheckResult

// Check overall health
IsHealthy(ctx context.Context) bool

// Default checks
DefaultHealthChecks()
```

### Shutdown Coordination

```go
// Register shutdown hook
RegisterShutdownHook(phase ShutdownPhase, hook ShutdownHook)

// Perform shutdown
Shutdown(ctx context.Context) error

// Default hooks
DefaultShutdownHooks()
```

---

**Last Updated**: 2026-01-04  
**Framework Version**: v1.5.x
