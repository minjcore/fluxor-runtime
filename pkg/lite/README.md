# Lite Package - Minimal Fluxor

Minimal Fluxor implementation in ~500 lines of code with zero circular dependencies. Perfect for embedded applications, learning Fluxor, and understanding the core concepts.

## 🎯 Overview

The `lite` package is a simplified, stripped-down version of Fluxor that provides:

- ✅ **Minimal footprint**: ~500 LOC total
- ✅ **Zero circular dependencies**: Clean acyclic architecture
- ✅ **Core concepts**: EventBus, Components, Context
- ✅ **HTTP server**: Basic routing and handlers
- ✅ **Fast compilation**: Quick build times
- ✅ **Easy to understand**: Learn Fluxor patterns quickly

### When to Use Lite

**Use Lite for:**
- 🎓 Learning Fluxor concepts
- 📦 Embedded applications
- 🚀 Minimal footprint requirements
- 🔬 Prototyping and experiments
- 📖 Understanding core architecture

**Use Full Fluxor for:**
- 🏢 Production applications
- 🔥 Feature-rich microservices
- 🔄 Complex workflows
- 📊 Advanced observability
- 🌐 Distributed systems

---

## 📦 Package Structure

```
pkg/lite/
├── core/          # Minimal core (~150 LOC)
│   ├── bus.go         # Simple EventBus
│   ├── component.go   # Component interface
│   ├── context.go     # FluxorContext
│   └── worker.go      # Worker pool
│
├── web/           # Minimal HTTP server (~100 LOC)
│   ├── router.go      # Basic router
│   └── http_verticle.go
│
├── fx/            # Request context (~80 LOC)
│   ├── context.go     # HTTP context helpers
│   └── fast_context.go
│
├── webfast/       # FastHTTP variant (~100 LOC)
│   ├── router.go      # FastHTTP router
│   ├── cache.go       # Simple cache
│   └── verticle.go
│
└── fluxor/        # Runtime (~70 LOC)
    └── runtime.go     # Application runtime
```

**Total**: ~500 LOC (vs Full Fluxor: ~8000 LOC)

---

## 🚀 Quick Start

### Hello World

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/lite/core"
    "github.com/fluxorio/fluxor/pkg/lite/web"
    "github.com/fluxorio/fluxor/pkg/lite/fluxor"
    "github.com/fluxorio/fluxor/pkg/lite/fx"
)

func main() {
    // Create app
    app := fluxor.NewApp()
    
    // Create HTTP verticle
    httpVerticle := web.NewHttpVerticle(":8080")
    
    // Add routes
    httpVerticle.Router().GET("/", func(ctx fx.Context) error {
        return ctx.Text(200, "Hello from Lite!")
    })
    
    httpVerticle.Router().GET("/ping", func(ctx fx.Context) error {
        return ctx.JSON(200, map[string]string{"status": "ok"})
    })
    
    // Deploy and run
    app.Deploy(httpVerticle)
    app.Run()
}
```

Run:
```bash
go run main.go
curl http://localhost:8080/
# Hello from Lite!
```

---

## 🏗️ Architecture

### Lite vs Full Fluxor

| Component | Lite | Full Fluxor |
|-----------|------|-------------|
| **EventBus** | Simple in-memory | Local + Cluster (NATS) |
| **HTTP Server** | Basic net/http | FastHTTP + Backpressure |
| **Router** | Simple map-based | Advanced with middleware |
| **Context** | Basic helpers | Rich with validation |
| **Concurrency** | Basic worker pool | Enterprise (Executors, Mailboxes) |
| **Workflow** | ❌ Not included | ✅ n8n-like engine |
| **Scheduler** | ❌ Not included | ✅ Cron + intervals |
| **Cache** | Simple map | Multi-tier (Memory/Redis) |
| **Metrics** | ❌ Not included | ✅ Prometheus + OTEL |

---

## 📚 Core Components

### 1. EventBus (lite/core/bus.go)

Simple in-memory event bus:

```go
bus := core.NewBus()

// Publish
bus.Publish("user.created", map[string]interface{}{
    "id": 123,
    "name": "John",
})

// Subscribe
bus.Subscribe("user.created", func(data interface{}) {
    user := data.(map[string]interface{})
    fmt.Printf("User created: %v\n", user)
})
```

### 2. Component (lite/core/component.go)

Basic component interface:

```go
type Component interface {
    Start(ctx FluxorContext) error
    Stop(ctx FluxorContext) error
}
```

### 3. FluxorContext (lite/core/context.go)

Minimal context:

```go
type FluxorContext interface {
    Context() context.Context
    Bus() Bus
}
```

### 4. HTTP Router (lite/web/router.go)

Simple routing:

```go
router := web.NewRouter()

router.GET("/users", func(ctx fx.Context) error {
    return ctx.JSON(200, []string{"Alice", "Bob"})
})

router.POST("/users", func(ctx fx.Context) error {
    // Create user
    return ctx.JSON(201, map[string]string{"status": "created"})
})
```

### 5. Request Context (lite/fx/context.go)

HTTP helpers:

```go
type Context interface {
    // Response helpers
    Text(code int, text string) error
    JSON(code int, data interface{}) error
    
    // Request helpers
    Query(key string) string
    PathParam(key string) string
}
```

---

## 🎨 Usage Patterns

### Pattern 1: REST API

```go
func main() {
    app := fluxor.NewApp()
    
    httpVerticle := web.NewHttpVerticle(":8080")
    router := httpVerticle.Router()
    
    // Routes
    router.GET("/users", listUsers)
    router.GET("/users/:id", getUser)
    router.POST("/users", createUser)
    router.PUT("/users/:id", updateUser)
    router.DELETE("/users/:id", deleteUser)
    
    app.Deploy(httpVerticle)
    app.Run()
}

func listUsers(ctx fx.Context) error {
    users := []User{
        {ID: 1, Name: "Alice"},
        {ID: 2, Name: "Bob"},
    }
    return ctx.JSON(200, users)
}

func getUser(ctx fx.Context) error {
    id := ctx.PathParam("id")
    user := findUserByID(id)
    if user == nil {
        return ctx.JSON(404, map[string]string{"error": "not found"})
    }
    return ctx.JSON(200, user)
}
```

### Pattern 2: EventBus Communication

```go
type UserService struct {
    bus core.Bus
}

func (s *UserService) Start(ctx core.FluxorContext) error {
    s.bus = ctx.Bus()
    
    // Subscribe to events
    s.bus.Subscribe("user.created", s.onUserCreated)
    
    return nil
}

func (s *UserService) onUserCreated(data interface{}) {
    user := data.(map[string]interface{})
    fmt.Printf("New user: %v\n", user)
    
    // Send welcome email
    s.bus.Publish("email.send", map[string]interface{}{
        "to": user["email"],
        "subject": "Welcome!",
    })
}

func (s *UserService) Stop(ctx core.FluxorContext) error {
    return nil
}
```

### Pattern 3: Multi-Component App

```go
func main() {
    app := fluxor.NewApp()
    
    // HTTP server
    httpVerticle := web.NewHttpVerticle(":8080")
    setupRoutes(httpVerticle.Router())
    
    // Background services
    userService := &UserService{}
    emailService := &EmailService{}
    
    // Deploy all
    app.Deploy(httpVerticle)
    app.Deploy(userService)
    app.Deploy(emailService)
    
    app.Run()
}
```

### Pattern 4: FastHTTP Variant (Higher Performance)

```go
import "github.com/fluxorio/fluxor/pkg/lite/webfast"

func main() {
    app := fluxor.NewApp()
    
    // Use FastHTTP variant for higher RPS
    fastVerticle := webfast.NewVerticle(":8080")
    router := fastVerticle.Router()
    
    router.GET("/ping", func(ctx fx.FastContext) error {
        return ctx.Text(200, "pong")
    })
    
    app.Deploy(fastVerticle)
    app.Run()
}
```

---

## 🔧 Configuration

### HTTP Server Config

```go
config := web.HttpVerticleConfig{
    Addr:         ":8080",
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 5 * time.Second,
}

httpVerticle := web.NewHttpVerticleWithConfig(config)
```

### EventBus Config

```go
// Lite EventBus is simple - no configuration needed
bus := core.NewBus()

// All operations are synchronous (no queues, no backpressure)
```

---

## ⚡ Performance

### Benchmarks

```bash
go test -bench=. ./pkg/lite/...
```

**Sample Results**:
```
BenchmarkRouter/GET-8           5000000    250 ns/op
BenchmarkBus/Publish-8          3000000    450 ns/op
BenchmarkFastRouter/GET-8      10000000    120 ns/op
```

### Performance Tips

1. **Use FastHTTP variant** for high-RPS scenarios
2. **Avoid blocking** in EventBus handlers
3. **Keep handlers simple** - no heavy computation
4. **Use connection pooling** for database access
5. **Profile before optimizing** - measure first

---

## 🆚 Comparison

### Lite vs Full Fluxor

| Aspect | Lite | Full Fluxor |
|--------|------|-------------|
| **LOC** | ~500 | ~8000 |
| **Compilation** | ⚡ Fast (1s) | 🚀 Good (5s) |
| **Memory** | 🟢 Low (~10 MB) | 🟡 Medium (~50 MB) |
| **Features** | 🔵 Basic | 🟣 Complete |
| **Circular Deps** | ✅ Zero | ⚠️ Some internal |
| **Learning Curve** | 🟢 Easy | 🟡 Medium |

### When to Migrate from Lite to Full

Migrate when you need:
- ✅ Advanced workflow engine
- ✅ Task scheduling (cron)
- ✅ Multi-tier caching
- ✅ Service mesh patterns
- ✅ Distributed EventBus (NATS)
- ✅ Production observability
- ✅ CCU-based backpressure

---

## 🎓 Learning Path

### 1. Start with Lite

```go
// Simple, easy to understand
app := fluxor.NewApp()
http := web.NewHttpVerticle(":8080")
app.Deploy(http)
app.Run()
```

### 2. Add EventBus

```go
// Component communication
service1.bus.Publish("event", data)
service2.bus.Subscribe("event", handler)
```

### 3. Graduate to Full Fluxor

```go
// When you need more features
import "github.com/fluxorio/fluxor/pkg/entrypoint"

app, _ := entrypoint.NewMainVerticle("config.json")
app.DeployVerticle(NewApiVerticle())
app.Start()
```

---

## 🧪 Testing

### Unit Tests

```go
func TestRouter(t *testing.T) {
    router := web.NewRouter()
    
    router.GET("/test", func(ctx fx.Context) error {
        return ctx.Text(200, "ok")
    })
    
    // Test routing
    handler := router.Match("GET", "/test")
    assert.NotNil(t, handler)
}
```

### Integration Tests

```go
func TestHTTPServer(t *testing.T) {
    app := fluxor.NewApp()
    http := web.NewHttpVerticle(":0")  // Random port
    
    http.Router().GET("/ping", func(ctx fx.Context) error {
        return ctx.Text(200, "pong")
    })
    
    app.Deploy(http)
    go app.Run()
    
    // Make request
    resp, _ := http.Get("http://localhost:" + port + "/ping")
    body, _ := io.ReadAll(resp.Body)
    assert.Equal(t, "pong", string(body))
}
```

---

## 📚 Examples

### Complete REST API

See [cmd/lite/main.go](../../cmd/lite/main.go) for a complete example.

### Key Files to Study

1. **lite/core/bus.go** - Understand EventBus pattern
2. **lite/web/router.go** - Understand routing
3. **lite/fluxor/runtime.go** - Understand runtime
4. **lite/fx/context.go** - Understand request context

---

## 🔗 Migration Guide

### From Lite to Full Fluxor

#### 1. Replace imports

```go
// Before (Lite)
import (
    "github.com/fluxorio/fluxor/pkg/lite/core"
    "github.com/fluxorio/fluxor/pkg/lite/web"
)

// After (Full)
import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/web"
    "github.com/fluxorio/fluxor/pkg/entrypoint"
)
```

#### 2. Update runtime

```go
// Before (Lite)
app := fluxor.NewApp()
app.Deploy(verticle)
app.Run()

// After (Full)
app, _ := entrypoint.NewMainVerticle("config.json")
app.DeployVerticle(verticle)
app.Start()
```

#### 3. Use Premium Patterns

```go
// After (Full) - Use BaseVerticle
type MyVerticle struct {
    *core.BaseVerticle
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Premium pattern with lifecycle management
    return v.BaseVerticle.Start(ctx)
}
```

---

## 🎯 Best Practices

### 1. Keep It Simple

```go
// ✅ Good: Simple, direct
router.GET("/users", func(ctx fx.Context) error {
    return ctx.JSON(200, users)
})

// ❌ Over-engineered for Lite
router.Use(middleware1, middleware2, middleware3)
```

### 2. Use EventBus for Decoupling

```go
// ✅ Good: Decoupled via EventBus
http.POST("/users", func(ctx fx.Context) error {
    bus.Publish("user.created", user)
    return ctx.JSON(201, user)
})

// ❌ Tight coupling
http.POST("/users", func(ctx fx.Context) error {
    emailService.SendWelcome(user)  // Direct dependency
    return ctx.JSON(201, user)
})
```

### 3. Migrate When Needed

Don't force Lite to do everything - migrate to Full Fluxor when your needs grow.

---

## 📈 Roadmap

### Current State (v1.0)
- ✅ Basic EventBus
- ✅ Simple HTTP server
- ✅ Minimal runtime
- ✅ Zero circular deps

### Future Improvements
- 🔵 Middleware support
- 🔵 Better error handling
- 🔵 Request validation
- 🔵 More examples

---

**Package**: `github.com/fluxorio/fluxor/pkg/lite`  
**Status**: ✅ Stable (B+ Grade)  
**LOC**: ~500  
**Test Coverage**: 85%  
**Last Updated**: 2026-01-04

---

## 🔗 Related Documentation

- [PKG_INDEX.md](../../PKG_INDEX.md) - All packages
- [PKG_DECISION_GUIDE.md](../../PKG_DECISION_GUIDE.md) - Choose right package
- [DOCUMENTATION.md](../../DOCUMENTATION.md) - Full Fluxor docs
- [cmd/lite](../../cmd/lite) - Complete example
