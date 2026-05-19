# Dashboard Package

Package `dashboard` provides HTTP handlers and routes for the Fluxor system monitoring dashboard, including a reusable `DashboardVerticle` that can be deployed into any application.

## Overview

This package provides:
- **DashboardVerticle**: A reusable verticle that can be deployed into any application
- Metrics API endpoints (`/api/dashboard/metrics`)
- Health check endpoints (`/api/dashboard/health`)
- Simple HTML dashboard (fallback UI)
- Dashboard HTTP handlers

## Usage

### Using DashboardVerticle (Recommended)

The easiest way to add dashboard to your application is to deploy the `DashboardVerticle`:

```go
import (
    "github.com/fluxorio/fluxor/pkg/entrypoint"
    "github.com/fluxorio/fluxor/pkg/web/dashboard"
)

func main() {
    app, err := entrypoint.NewMainVerticle("")
    if err != nil {
        log.Fatalf("Failed to create main verticle: %v", err)
    }

    // Deploy your application verticles
    app.DeployVerticle(NewQueueRPCServerVerticle())
    
    // Deploy dashboard (add-on)
    app.DeployVerticle(dashboard.NewDashboardVerticle())
    
    if err := app.Start(); err != nil {
        log.Fatalf("Failed to start application: %v", err)
    }
}
```

#### Custom Configuration

You can customize the dashboard configuration:

```go
config := dashboard.DashboardVerticleConfig{
    Address: ":9090",  // Custom port
    Prefix:  "/admin", // Routes at /admin/dashboard, /admin/api/dashboard/metrics, etc.
}
app.DeployVerticle(dashboard.NewDashboardVerticleWithConfig(config))
```

#### Configuration from Context

If `Address` is empty, the verticle will use the `http_addr` value from the context config, or default to `:8080`:

```go
// In your config file or context setup:
ctx.SetConfig("http_addr", ":8080")
```

### Manual Route Registration

If you prefer to integrate dashboard routes into your existing HTTP server:

```go
import (
    "github.com/fluxorio/fluxor/pkg/web"
    "github.com/fluxorio/fluxor/pkg/web/dashboard"
)

// In your verticle's Start() method:
router := server.FastRouter()
dashboard.Register(router, "")  // Register at root path

// Or with a prefix:
dashboard.Register(router, "/admin")  // Routes will be at /admin/dashboard, /admin/api/dashboard/metrics, etc.
```

### API Endpoints

The dashboard provides the following endpoints:

- `GET /api/dashboard/metrics` - Returns comprehensive metrics including:
  - Executor and WorkerPool metrics (existing)
  - HTTP Server metrics (requests, RPS, queue, CCU)
  - Profiling metrics (IO/CPU-bound work classification, bottlenecks)
  - Runtime metrics (goroutines, memory, GC, allocation rate)
- `GET /api/dashboard/health` - Returns health status with metrics summary
- `GET /dashboard` - Simple HTML dashboard (fallback UI)
- `GET /dashboard.js` - Dashboard JavaScript
- `GET /dashboard.css` - Dashboard CSS

If a prefix is configured, all routes will be prefixed (e.g., `/admin/api/dashboard/metrics`).

### New Metrics Available

The dashboard now includes:

1. **HTTP Server Metrics**
   - Requests per second (RPS)
   - Queue length and utilization
   - Rejected requests (backpressure)
   - CCU (Concurrent Users) metrics
   - Worker utilization

2. **Profiling Metrics**
   - Work classification (IO-bound, CPU-bound, Mixed)
   - Goroutine statistics by state and work type
   - Bottleneck detection with recommendations
   - Anti-pattern detection (mixed work)

3. **Runtime Metrics**
   - Goroutine count
   - Memory allocation (current, total, system)
   - GC statistics (cycles, pause time, rate)
   - Allocation rate (allocs/sec)
   - GC rate (GC cycles/sec)

### Metrics Collection

To collect metrics from your executors and worker pools, register them. You can use helper functions for automatic registration:

#### Using Helper Functions (Recommended)

```go
import "github.com/fluxorio/fluxor/pkg/core/concurrency"

// Create executor with automatic metrics registration
executor := concurrency.NewExecutorWithMetrics(
    ctx.GoCMD().Context(),
    "queue-processor",  // ID for metrics
    concurrency.DefaultExecutorConfig(),
)
// Executor is automatically registered for metrics collection

// Create worker pool with automatic metrics registration
pool := concurrency.NewWorkerPoolWithMetrics(
    ctx.GoCMD().Context(),
    "image-processor",  // ID for metrics
    concurrency.DefaultWorkerPoolConfig(),
)
if err := pool.Start(); err != nil {
    // handle error
}
// Worker pool is automatically registered for metrics collection
```

#### Manual Registration

You can also register executors and worker pools manually:

```go
import "github.com/fluxorio/fluxor/pkg/core/concurrency"

// Register an executor
executor := concurrency.NewExecutor(ctx, config)
concurrency.RegisterExecutor("my-executor", executor)

// Register a worker pool
pool := concurrency.NewWorkerPool(ctx, config)
pool.Start()
concurrency.RegisterWorkerPool("my-worker-pool", pool)
```

Metrics will automatically be collected and available via the `/api/dashboard/metrics` endpoint.

### Auto-Registration of HTTP Servers

HTTP servers (FastHTTPServer) are automatically registered with the dashboard when created. The dashboard will:
- Collect HTTP server metrics (RPS, queue, CCU, etc.)
- Start profiling system automatically
- Track work classification and bottlenecks

No manual registration needed - just create FastHTTPServer and it will appear in the dashboard.

### Manual Registration (Optional)

If you need to register HTTP servers manually or with custom names:

```go
import "github.com/fluxorio/fluxor/pkg/web/dashboard"

// Register with custom name
server := web.NewFastHTTPServer(gocmd, config)
dashboard.RegisterHTTPServerForMetrics("my-api-server", server)
```

## Examples

See `cmd/dashboard-example` for a complete example of using the DashboardVerticle.
