# Dashboard Example

Ví dụ đầy đủ về cách sử dụng Dashboard trong Fluxor application.

## Overview

Ví dụ này demonstrate:
1. Deploy `DashboardVerticle` (reusable component)
2. Tạo HTTP server với metrics tự động
3. Register executors cho metrics collection
4. Xem real-time metrics trong dashboard
5. Tích hợp profiling metrics (IO/CPU-bound detection)
6. Runtime metrics (GC, memory, allocations)

## Building

```bash
go build -o dashboard ./cmd/dashboard
```

Or from the project root:

```bash
go build -o bin/dashboard ./cmd/dashboard
```

## Running

```bash
./dashboard
```

The application will start an HTTP server on port 8080.

## Endpoints

- **Dashboard UI**: http://localhost:8080/dashboard
- **Metrics API**: http://localhost:8080/api/dashboard/metrics
- **Health API**: http://localhost:8080/api/dashboard/health
- **Test API**: http://localhost:9090/api/test (HTTP server for testing)

## Metrics Available

Dashboard hiển thị:

1. **Executor & WorkerPool Metrics**
   - Queue length và utilization
   - Throughput (tasks/sec)
   - Worker utilization
   - Completed/rejected tasks

2. **HTTP Server Metrics** (tự động từ FastHTTPServer)
   - Requests per second (RPS)
   - Queue length và utilization
   - Rejected requests (backpressure)
   - CCU (Concurrent Users)
   - Worker count

3. **Profiling Metrics** (tự động từ profiling system)
   - Work classification (IO-bound, CPU-bound, Mixed)
   - Goroutine statistics
   - Bottleneck detection với recommendations
   - Anti-pattern detection

4. **Runtime Metrics**
   - Goroutine count
   - Memory allocation (current, total, system)
   - GC statistics (cycles, pause time, rate)
   - Allocation rate (allocs/sec)
   - GC rate (GC cycles/sec)

## Features

- Reusable `DashboardVerticle` - deploy into any application
- Automatic metrics registration using helper functions
- Real-time metrics collection from executors and worker pools
- HTTP API for metrics data
- Simple HTML dashboard UI
- Health check endpoint

## Usage in Your Application

### Pattern 1: Deploy DashboardVerticle (Recommended)

The easiest way to add dashboard to your application:

```go
import (
    "github.com/fluxorio/fluxor/pkg/entrypoint"
    "github.com/fluxorio/fluxor/pkg/web/dashboard"
)

func main() {
    app, err := entrypoint.NewMainVerticle("")
    
    // Deploy your application verticles
    app.DeployVerticle(NewMyAppVerticle())
    
    // Deploy dashboard (add-on)
    app.DeployVerticle(dashboard.NewDashboardVerticle())
    
    app.Start()
}
```

### Pattern 2: Register Metrics with Helper Functions

In your application verticles, use helper functions to automatically register executors/worker pools:

```go
import "github.com/fluxorio/fluxor/pkg/core/concurrency"

// In your verticle's Start() method:
executor := concurrency.NewExecutorWithMetrics(
    ctx.GoCMD().Context(),
    "my-executor-id",  // ID for metrics
    concurrency.DefaultExecutorConfig(),
)
// Executor is automatically registered for metrics collection
```

### Pattern 3: Manual Route Registration

If you prefer to integrate dashboard routes into your existing HTTP server:

```go
import "github.com/fluxorio/fluxor/pkg/web/dashboard"

// In your verticle's Start() method:
router := server.FastRouter()
dashboard.Register(router, "")
```

### Custom Configuration

You can customize the dashboard configuration:

```go
config := dashboard.DashboardVerticleConfig{
    Address: ":9090",  // Custom port
    Prefix:  "/admin", // Routes at /admin/dashboard, etc.
}
app.DeployVerticle(dashboard.NewDashboardVerticleWithConfig(config))
```

## Example Code Structure

This example demonstrates:
- `TestMetricsVerticle`: A sample verticle that creates an executor and registers it for metrics
- `DashboardVerticle`: The reusable dashboard component deployed separately

The key pattern is that dashboard is a separate, reusable component that can be deployed into any application without modifying the application's core logic.
