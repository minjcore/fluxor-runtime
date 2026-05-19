# Entrypoint Package

The `entrypoint` package provides high-level application bootstrapping and reactive programming utilities for Fluxor applications.

## MainVerticle Pattern (Recommended)

The **MainVerticle** is a convenience bootstrapper for "main-like" applications that provides:
- **Config-driven initialization** - Load JSON/YAML config files
- **Automatic config injection** - Config is automatically injected into all verticles
- **Signal handling** - Graceful shutdown on SIGINT/SIGTERM
- **EventBus customization** - Swap to NATS/clustered EventBus with one option

### Entrypoint as Control Plane

The entrypoint (MainVerticle) is the **control plane** of the Fluxor runtime: it decides what to load (config), when to bootstrap, what to deploy, and when to start/stop. The **runtime** (`pkg/core` GoCMD) is the execution plane: it runs the EventBus and verticles. MainVerticle creates and drives GoCMD; it does not execute business logic itself.

| Responsibility | Control plane (Entrypoint) | Runtime (GoCMD) |
|----------------|----------------------------|-----------------|
| **Owns** | Config map, BootstrapHook cleanup, root context + cancel | EventBus, deployment map, root context (from entrypoint) |
| **API** | `NewMainVerticle`, `NewMainVerticleWithOptions`, `DeployVerticle`, `Start`, `Stop` | `DeployVerticle`, `UndeployVerticle`, `EventBus`, `Close` |

See [PRIMARY_PATTERN.md](../../docs/PRIMARY_PATTERN.md) for the full pattern.

### Quick Start

```go
package main

import (
    "log"
    "github.com/fluxorio/fluxor/pkg/entrypoint"
)

func main() {
    // 1. Create app with config
    app, err := entrypoint.NewMainVerticle("config.json")
    if err != nil {
        log.Fatal(err)
    }

    // 2. Deploy verticles (order matters for dependencies)
    app.DeployVerticle(NewApiVerticle())
    app.DeployVerticle(NewWorkerVerticle())

    // 3. Start (blocks until shutdown signal)
    app.Start()
}
```

### Basic Usage

```go
// Load config from file (JSON or YAML)
app, err := entrypoint.NewMainVerticle("config.json")

// Deploy verticles (order matters for dependencies)
app.DeployVerticle(myVerticle)

// Start application (blocks, handles SIGINT/SIGTERM)
app.Start()
```

### Advanced Usage with Cluster EventBus

```go
package main

import (
    "context"
    "log"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/entrypoint"
)

func main() {
    // Customize EventBus (e.g., NATS for distributed services)
    app, err := entrypoint.NewMainVerticleWithOptions("config.json", entrypoint.MainVerticleOptions{
        EventBusFactory: func(ctx context.Context, gocmd core.GoCMD, cfg map[string]any) (core.EventBus, error) {
            natsCfg, _ := cfg["nats"].(map[string]any)
            url, _ := natsCfg["url"].(string)
            prefix, _ := natsCfg["prefix"].(string)
            return clusterbus.NewClusterEventBusJetStream(ctx, gocmd, clusterbus.ClusterJetStreamConfig{
                URL:     url,
                Prefix:  prefix,
                Service: "my-service",
            })
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    app.DeployVerticle(NewApiVerticle())
    app.Start()
}
```

### Configuration

**config.json:**
```json
{
  "nats": {
    "url": "nats://127.0.0.1:4222",
    "prefix": "myapp"
  },
  "http_addr": ":8080",
  "service_name": "api-gateway"
}
```

The config is automatically loaded and injected into all verticles via `FluxorContext.Config()`:

```go
func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Access injected config
    cfg := ctx.Config()
    httpAddr := cfg["http_addr"].(string)
    serviceName := cfg["service_name"].(string)
    // ...
}
```

### API Reference

#### MainVerticle

```go
// NewMainVerticle loads config and creates app runtime
func NewMainVerticle(configPath string) (*MainVerticle, error)

// NewMainVerticleWithOptions allows customizing EventBus and GoCMD
func NewMainVerticleWithOptions(configPath string, opts MainVerticleOptions) (*MainVerticle, error)

// DeployVerticle deploys a verticle with config injection
func (m *MainVerticle) DeployVerticle(v core.Verticle) (string, error)

// Start blocks until SIGINT/SIGTERM, then gracefully shuts down
func (m *MainVerticle) Start() error

// Stop gracefully shuts down the application
func (m *MainVerticle) Stop() error

// GoCMD returns the underlying GoCMD (advanced usage)
func (m *MainVerticle) GoCMD() core.GoCMD

// Config returns the loaded config map
func (m *MainVerticle) Config() map[string]any
```

#### MainVerticleOptions

```go
type MainVerticleOptions struct {
    // EventBusFactory allows switching to clustered EventBus (e.g., NATS)
    EventBusFactory func(ctx context.Context, gocmd core.GoCMD, cfg map[string]any) (core.EventBus, error)

    // GoCMDOptions are passed to core.NewGoCMDWithOptions
    GoCMDOptions core.GoCMDOptions
}
```

### Lifecycle

1. **Initialization**: Load config, create GoCMD, setup EventBus
2. **Deployment**: Deploy verticles (config automatically injected)
3. **Start**: Block until shutdown signal (SIGINT/SIGTERM)
4. **Shutdown**: Cancel context, close GoCMD, cleanup resources

### Config Injection

All verticles deployed via `DeployVerticle()` automatically receive the loaded config in their `FluxorContext`. The config is injected using a wrapper pattern (`configInjectedVerticle`) that ensures config is available in both `Start()` and `Stop()` methods.

**Example:**
```go
func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    cfg := ctx.Config()
    
    // Access config values
    if httpAddr, ok := cfg["http_addr"].(string); ok {
        // Use httpAddr
    }
    
    return nil
}
```

### Complete Example

```go
package main

import (
    "context"
    "log"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/entrypoint"
    "github.com/fluxorio/fluxor/pkg/web"
)

type ApiVerticle struct {
    *core.BaseVerticle
    server *web.FastHTTPServer
}

func NewApiVerticle() *ApiVerticle {
    return &ApiVerticle{
        BaseVerticle: core.NewBaseVerticle("api-gateway"),
    }
}

func (v *ApiVerticle) Start(ctx core.FluxorContext) error {
    if err := v.BaseVerticle.Start(ctx); err != nil {
        return err
    }

    // Get config from context
    cfg := ctx.Config()
    addr := ":8080"
    if httpAddr, ok := cfg["http_addr"].(string); ok {
        addr = httpAddr
    }

    // Setup HTTP server
    v.server = web.NewFastHTTPServer(ctx.GoCMD(), web.DefaultFastHTTPServerConfig(addr))
    router := v.server.FastRouter()
    router.GETFast("/health", func(c *web.FastRequestContext) error {
        return c.JSON(200, map[string]any{"status": "ok"})
    })

    return v.server.Start()
}

func (v *ApiVerticle) Stop(ctx core.FluxorContext) error {
    if v.server != nil {
        v.server.Stop()
    }
    return v.BaseVerticle.Stop(ctx)
}

func main() {
    app, err := entrypoint.NewMainVerticle("config.json")
    if err != nil {
        log.Fatal(err)
    }

    app.DeployVerticle(NewApiVerticle())
    app.Start()
}
```

## ReactorRuntime Pattern (Experimental)

The `ReactorRuntime` provides an alternative reactor-based runtime pattern. **Note: This is experimental and may change.**

### Usage

```go
runtime := entrypoint.New()

// Deploy a reactor
id := runtime.Deploy(myReactor, config)

// Shutdown
runtime.Shutdown()
```

### Reactor Interface

```go
type Reactor interface {
    OnStart(ctx core.FluxorContext) error
    OnStop() error
}
```

**Note:** The ReactorRuntime pattern is experimental. Use MainVerticle for production applications.

## Reactive Programming (Future/Promise)

The package also provides reactive programming utilities for handling asynchronous operations.

### Future

```go
future := entrypoint.NewFuture()

// Register handlers
future.OnSuccess(func(result interface{}) {
    // Handle success
})

future.OnFailure(func(err error) {
    // Handle error
})

// Complete the future
future.Complete("result")

// Or fail it
future.Fail(errors.New("error occurred"))
```

### Promise

```go
promise := entrypoint.NewPromise()

// Use as Future
promise.OnSuccess(func(result interface{}) {
    // Handle success
})

// Complete later
promise.Complete("result")
```

### Async/Await Style

```go
future := entrypoint.NewFuture()

// In goroutine
go func() {
    result := doAsyncWork()
    future.Complete(result)
}()

// Await result
result, err := future.Await(ctx)
if err != nil {
    // Handle error
}
```

### Promise-Style Chaining

```go
promise := entrypoint.NewPromise()

promise.
    Then(func(data interface{}) (interface{}, error) {
        // Transform data
        return transformed, nil
    }).
    Then(func(data interface{}) (interface{}, error) {
        // Next step
        return result, nil
    }).
    Catch(func(err error) (interface{}, error) {
        // Handle error
        return nil, err
    })

// Complete the promise
promise.Complete(initialData)
```

## Runtime Utilities

The package provides a `Runtime` interface for task execution and verticle deployment.

### Usage

```go
runtime := entrypoint.NewRuntime(ctx)
defer runtime.Stop()

// Start runtime
runtime.Start(ctx)

// Deploy verticle
runtime.Deploy(myVerticle)

// Execute task
task := &MyTask{}
runtime.Execute(task)
```

### Runtime Interface

```go
type Runtime interface {
    Start(ctx context.Context) error
    Stop() error
    Execute(task Task) error
    Deploy(verticle core.Verticle) (string, error)
    GoCMD() core.GoCMD
}
```

**Note:** For production applications, prefer `MainVerticle` over `Runtime`. Use `Runtime` only when you need fine-grained control over task execution.

## Workflow Utilities

The package provides `Workflow` and `Step` interfaces for composing reactive workflows.

### Usage

```go
step1 := entrypoint.NewStep("step1", func(ctx context.Context, data interface{}) (interface{}, error) {
    // Process data
    return result, nil
})

step2 := entrypoint.NewStep("step2", func(ctx context.Context, data interface{}) (interface{}, error) {
    // Process with previous result
    return result, nil
})

workflow := entrypoint.NewWorkflow("my-workflow", step1, step2)
err := workflow.Execute(ctx)
```

To run a Workflow during MainVerticle bootstrap, use `WorkflowBootstrapHook(workflow)` from `workflow_bootstrap.go`.

### Workflow Interface

```go
type Workflow interface {
    Name() string
    Execute(ctx context.Context) error
    Steps() []Step
}

type Step interface {
    Execute(ctx context.Context, data interface{}) (interface{}, error)
    Name() string
}
```

**Note:** For production workflows, consider using `pkg/workflow` which provides a more complete workflow engine with JSON definitions, EventBus integration, and n8n-like features.

## Comparison: MainVerticle vs ReactorRuntime

| Feature | MainVerticle | ReactorRuntime |
|---------|--------------|----------------|
| **Status** | ✅ Production | ⚠️ Experimental |
| **Pattern** | Verticle-based | Reactor-based |
| **Config** | File-based, auto-inject | Manual config map |
| **Shutdown** | Signal handling (SIGINT/SIGTERM) | Manual Shutdown() |
| **Use Case** | Main applications | Alternative runtime |
| **Recommended** | ✅ Yes | ❌ Experimental only |

## Best Practices

1. **Use MainVerticle for production** - It's the recommended pattern
2. **Config injection** - Access config via `ctx.Config()` in verticle Start/Stop
3. **Deployment order** - Deploy dependencies before dependents
4. **Signal handling** - MainVerticle handles SIGINT/SIGTERM automatically
5. **EventBus customization** - Use `EventBusFactory` for clustered deployments
6. **Graceful shutdown** - MainVerticle handles cleanup automatically

## See Also

- `pkg/core` - Core types (Verticle, EventBus, FluxorContext)
- `pkg/web` - HTTP server utilities
- `pkg/workflow` - Workflow engine
- `docs/PRIMARY_PATTERN.md` - Complete MainVerticle pattern guide

