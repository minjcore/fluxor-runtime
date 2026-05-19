# fx - Sugar Dev Pattern

The `fx` package provides the **Sugar Dev** pattern - syntactic sugar that makes Fluxor more accessible to Node.js developers by offering function-based APIs similar to Express.js.

## What is Sugar Dev?

**Sugar Dev** is a pattern that provides sweeter, more familiar syntax for developers coming from Node.js/Express.js backgrounds. Instead of the verbose struct/interface-based Verticle pattern, `fx` offers:

- 🍬 **Sweeter syntax** - Shorter, cleaner code
- 🎯 **Familiar patterns** - Express.js-like function-based APIs
- 🚀 **Simplicity** - No struct/interface boilerplate
- 💎 **Better Dev UX** - Easier to get started

## Quick Start

```go
package main

import (
    "context"
    "log"
    "reflect"
    
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/fx"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create application (Express.js-like!)
    app, err := fx.New(ctx,
        fx.Invoke(fx.NewInvoker(setupApplication)),
    )
    if err != nil {
        log.Fatalf("Failed to create app: %v", err)
    }
    
    if err := app.Start(); err != nil {
        log.Fatalf("Failed to start app: %v", err)
    }
    
    app.KeepAndServe() // Keep server running
}

func setupApplication(deps map[reflect.Type]interface{}) error {
    vertx := deps[reflect.TypeOf((*core.Vertx)(nil)).Elem()].(core.Vertx)
    
    // Create HTTP server
    cfg := web.DefaultFastHTTPServerConfig(":8080")
    server := web.NewFastHTTPServer(vertx, cfg)
    router := server.FastRouter()
    
    // Setup routes
    router.GETFast("/", func(ctx *web.FastRequestContext) error {
        return ctx.JSON(200, map[string]interface{}{
            "message": "Hello from Sugar Dev!",
        })
    })
    
    go server.Start()
    return nil
}
```

## Verticle Pattern vs fx Pattern (Sugar Dev)

### Low-level: Verticle Pattern

The traditional Fluxor pattern requires implementing the `Verticle` interface:

```go
type ApiVerticle struct {
    server *web.FastHTTPServer
}

func (v *ApiVerticle) Start(ctx core.FluxorContext) error {
    // Setup code
    cfg := web.DefaultFastHTTPServerConfig(":8080")
    v.server = web.NewFastHTTPServer(ctx.GoCMD(), cfg)
    
    router := v.server.FastRouter()
    router.GETFast("/", func(ctx *web.FastRequestContext) error {
        return ctx.JSON(200, map[string]interface{}{"message": "Hello"})
    })
    
    go v.server.Start()
    return nil
}

func (v *ApiVerticle) Stop(ctx core.FluxorContext) error {
    if v.server != nil {
        return v.server.Stop()
    }
    return nil
}

// Usage
app.DeployVerticle(NewApiVerticle())
```

### Sugar Dev: fx Pattern

The `fx` pattern uses simple functions instead:

```go
// Just a function - no struct/interface needed!
func setupApplication(deps map[reflect.Type]interface{}) error {
    vertx := deps[reflect.TypeOf((*core.Vertx)(nil)).Elem()].(core.Vertx)
    
    cfg := web.DefaultFastHTTPServerConfig(":8080")
    server := web.NewFastHTTPServer(vertx, cfg)
    
    router := server.FastRouter()
    router.GETFast("/", func(ctx *web.FastRequestContext) error {
        return ctx.JSON(200, map[string]interface{}{"message": "Hello"})
    })
    
    go server.Start()
    return nil
}

// Usage - Express.js-like!
app, _ := fx.New(ctx, fx.Invoke(fx.NewInvoker(setupApplication)))
app.Start()
```

## When to Use Each Pattern

**Use fx (Sugar Dev) when:**
- You're coming from Node.js/Express.js background
- Building simple to medium complexity apps
- Want Express.js-like patterns
- Prefer function-based code

**Use Verticles when:**
- You need complex lifecycle management
- Want explicit Start/Stop control
- Building production systems with clear component boundaries
- Need fine-grained control over component initialization

## Features

### Dependency Injection

`fx` automatically injects dependencies based on function parameters:

```go
func setupApp(deps map[reflect.Type]interface{}) error {
    // Access dependencies from the map
    vertx := deps[reflect.TypeOf((*core.Vertx)(nil)).Elem()].(core.Vertx)
    eventBus := deps[reflect.TypeOf((*core.EventBus)(nil)).Elem()].(core.EventBus)
    
    // Use dependencies...
    return nil
}
```

### Providers

Register custom dependencies:

```go
fx.New(ctx,
    fx.Provide(fx.NewValueProvider("my-config-value")),
    fx.Invoke(fx.NewInvoker(setupApp)),
)
```

### Lifecycle Management

`fx` handles application lifecycle automatically:

```go
app, _ := fx.New(ctx, fx.Invoke(fx.NewInvoker(setupApp)))
app.Start()  // Initialize and start
app.KeepAndServe()   // Block until shutdown
app.Stop()   // Stop gracefully
```

## Under the Hood

The `fx` package uses the same Fluxor Stream as Verticles:
- Uses `core.GoCMD` for the runtime
- Uses `core.EventBus` for messaging
- Same performance characteristics
- Same event-driven architecture

The only difference is the API surface - `fx` provides a function-based interface instead of struct/interface-based Verticles.

## See Also

- [Node.js Developer's Guide](../../NODEJS_APPROACH.md) - Learn how to approach Fluxor from a Node.js perspective
- [Fluxor Documentation](../../DOCUMENTATION.md) - Complete API reference
- [Primary Pattern Guide](../../docs/PRIMARY_PATTERN.md) - Traditional Verticle pattern guide

