# EventBus Service Pattern

## Overview

Fluxor is an **event-driven runtime engine** (inspired by Vert.x and OTP). EventBus is the **core communication mechanism** and the **recommended, most stable, and principled pattern** for service communication.

This guide explains why EventBus is preferred over direct injection and provides practical guidance on when to use each pattern.

---

## EventBus Pattern (Recommended)

EventBus-based communication is the **recommended pattern** for service communication in Fluxor. It aligns with the event-driven architecture principles and provides the best foundation for scalable, maintainable systems.

### Why EventBus is Preferred

#### 1. **Decoupling**
- Services communicate via **addresses**, not direct references
- Verticles don't need to know service implementation details
- Services can be replaced or refactored without affecting callers

#### 2. **Location Transparency**
- Services can be **local** (in-process) or **distributed** (NATS/clustered)
- Same code works for both - just swap EventBus implementation
- Enables microservices architecture without code changes

#### 3. **Reactor Isolation**
- Handlers **always execute on reactors** (event loops)
- Maintains event loop semantics and sequential processing
- No blocking operations on event loops

#### 4. **No Shared Mutable State**
- Communication via **immutable messages**
- Eliminates race conditions and synchronization issues
- Follows functional programming principles

#### 5. **Backpressure Handling**
- Built-in **queue management** and **timeout handling**
- Automatic backpressure detection and error handling
- Prevents unbounded memory growth

#### 6. **Testability**
- Easy to **mock/test** by replacing EventBus implementation
- Can test verticles in isolation
- No need for real service instances

#### 7. **Scalability**
- Can **distribute services** across nodes without code changes
- Supports load balancing and service discovery
- Enables horizontal scaling

### EventBus Pattern Example

```go
// Step 1: Define service using BaseService
type UserService struct {
    *core.BaseService
    users map[string]map[string]interface{}
}

func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    request := msg.Body().(map[string]interface{})
    userID := request["userID"].(string)
    user, exists := s.users[userID]
    if !exists {
        return s.Fail(msg, 404, "User not found")
    }
    return s.Reply(msg, user)
}

func (s *UserService) doStart(ctx core.FluxorContext) error {
    s.users = make(map[string]map[string]interface{})
    return nil
}

// Step 2: Deploy service
service := &UserService{
    BaseService: core.NewBaseService("user-service", "user.service"),
}
gocmd.DeployVerticle(service)

// Step 3: Verticle calls service via EventBus
type APIVerticle struct {
    *core.BaseVerticle
}

func (v *APIVerticle) doStart(ctx core.FluxorContext) error {
    consumer := v.Consumer("api.request")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        request := msg.Body().(map[string]interface{})
        userID := request["userID"].(string)
        
        // Call service via EventBus (request-reply pattern)
        reply, err := v.Request("user.service", map[string]interface{}{
            "userID": userID,
        }, 5*time.Second)
        if err != nil {
            return msg.Fail(502, "Service unavailable: "+err.Error())
        }
        
        var user map[string]interface{}
        reply.DecodeBody(&user)
        return msg.Reply(user)
    })
    return nil
}
```

### CPU-Bound Services

**Important**: Services often perform CPU-bound work (computations, image processing, ML inference, etc.). Even though services use EventBus for communication, **CPU-bound work must use WorkerPool** to avoid blocking the event loop.

**Pattern**: EventBus for communication + WorkerPool for CPU-bound work

```go
type ImageProcessingService struct {
    *core.BaseService
}

func (s *ImageProcessingService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    request := msg.Body().(map[string]interface{})
    imageData := request["image"].([]byte)
    
    // CPU-bound work: Use WorkerPool (ExecuteBlocking)
    // This executes on a worker thread, not the event loop
    result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
        // CPU-intensive image processing
        processedImage := processImageCPUIntensive(imageData)
        return processedImage, nil
    }, 30*time.Second)
    
    if err != nil {
        return s.Fail(msg, 500, "Processing failed: "+err.Error())
    }
    
    // Reply with result (back on event loop)
    return s.Reply(msg, map[string]interface{}{
        "processedImage": result,
    })
}

// BAD: CPU-bound work directly on event loop
func (s *ImageProcessingService) doHandleRequest_BAD(ctx core.FluxorContext, msg core.Message) error {
    // NEVER DO THIS: Blocks event loop
    processedImage := processImageCPUIntensive(imageData)  // ❌ Blocks reactor
    return s.Reply(msg, processedImage)
}
```

**Key Points**:
- ✅ **EventBus**: Use for service communication (address-based, decoupled)
- ✅ **WorkerPool**: Use for CPU-bound work within service handlers
- ❌ **Never block**: Don't do CPU-intensive work directly on event loop
- ✅ **Maintain reactor isolation**: Handlers receive messages on event loops, execute CPU work on workers

### Startx Pattern: I/O-Bound Verticle + CPU-Bound Service

**Startx** is a common pattern where:
- **Verticle is I/O-bound**: Handles HTTP requests, database calls, network I/O
- **Service is CPU-bound**: Performs heavy computations using WorkerPool

This pattern demonstrates the separation of concerns:
- Verticle handles I/O operations (fast, non-blocking)
- Service handles CPU-intensive work (on WorkerPool, doesn't block event loop)

```go
// CPU-Bound Service (uses WorkerPool for heavy computations)
type ImageProcessingService struct {
    *core.BaseService
}

func (s *ImageProcessingService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    request := msg.Body().(map[string]interface{})
    imageData := request["image"].([]byte)
    operation := request["operation"].(string)
    
    // CPU-bound work: Use WorkerPool (ExecuteBlocking)
    // This executes on a worker thread, not the event loop
    result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
        // CPU-intensive image processing
        var processedImage []byte
        switch operation {
        case "resize":
            processedImage = resizeImage(imageData, 800, 600)
        case "compress":
            processedImage = compressImage(imageData)
        case "filter":
            processedImage = applyFilter(imageData)
        default:
            return nil, fmt.Errorf("unknown operation: %s", operation)
        }
        return processedImage, nil
    }, 30*time.Second)
    
    if err != nil {
        return s.Fail(msg, 500, "Processing failed: "+err.Error())
    }
    
    // Reply with result (back on event loop)
    return s.Reply(msg, map[string]interface{}{
        "processedImage": result,
        "size":           len(result.([]byte)),
    })
}

// I/O-Bound Verticle (handles HTTP, database, network)
type APIVerticle struct {
    *core.BaseVerticle
    imageServiceAddress string
}

func (v *APIVerticle) doStart(ctx core.FluxorContext) error {
    v.imageServiceAddress = "image.service"
    
    // I/O-bound: Handle HTTP requests (fast, non-blocking)
    consumer := v.Consumer("api.image.process")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        request := msg.Body().(map[string]interface{})
        imageData := request["image"].([]byte)
        operation := request["operation"].(string)
        
        // Call CPU-bound service via EventBus
        // Verticle doesn't block - service handles CPU work on WorkerPool
        reply, err := v.Request(v.imageServiceAddress, map[string]interface{}{
            "image":     imageData,
            "operation": operation,
        }, 30*time.Second)
        
        if err != nil {
            return msg.Fail(502, "Service unavailable: "+err.Error())
        }
        
        var result map[string]interface{}
        reply.DecodeBody(&result)
        return msg.Reply(result)
    })
    
    return nil
}

// Startx: Deploy both verticle and service
func Startx(gocmd core.GoCMD) error {
    // Deploy CPU-bound service first
    imageService := &ImageProcessingService{
        BaseService: core.NewBaseService("image-service", "image.service"),
    }
    if _, err := gocmd.DeployVerticle(imageService); err != nil {
        return fmt.Errorf("failed to deploy image service: %w", err)
    }
    
    // Deploy I/O-bound verticle
    apiVerticle := &APIVerticle{
        BaseVerticle: core.NewBaseVerticle("api-verticle"),
    }
    if _, err := gocmd.DeployVerticle(apiVerticle); err != nil {
        return fmt.Errorf("failed to deploy API verticle: %w", err)
    }
    
    return nil
}
```

**Key Benefits of Startx Pattern**:
- ✅ **Separation of concerns**: I/O and CPU work are separated
- ✅ **Non-blocking**: Verticle never blocks on CPU work
- ✅ **Scalability**: Service can be distributed, verticle stays I/O-focused
- ✅ **Resource efficiency**: CPU work uses WorkerPool, I/O uses event loops

---

## Direct Injection Pattern (Acceptable for Simple Cases)

Direct injection (constructor or field injection) is **acceptable** for simple cases but is **not the recommended pattern** for production, scalable systems.

### When Direct Injection is Acceptable

- **Simple, single-process applications** that will never need distribution
- **Tightly coupled components** that are always deployed together
- **Legacy code migration** where refactoring to EventBus is not feasible
- **Performance-critical paths** where EventBus overhead is unacceptable (rare)

### Direct Injection Limitations

#### 1. **Tight Coupling**
- Verticles depend on concrete service types
- Cannot replace services without modifying callers
- Harder to refactor and maintain

#### 2. **Testing Challenges**
- Requires **real service instances** for testing
- Harder to mock and test in isolation
- Integration tests become necessary

#### 3. **No Distribution**
- Cannot distribute services across nodes
- Locked into single-process architecture
- Cannot scale horizontally

#### 4. **Violates Event-Driven Principles**
- Direct method calls instead of message passing
- Shared mutable state risks
- Breaks reactor isolation

#### 5. **Shared Mutable State Risks**
- Services may share state with verticles
- Race conditions and synchronization issues
- Harder to reason about concurrency

### Direct Injection Example

```go
// Service (not using BaseService)
type URLShortenerService struct {
    db *sql.DB
}

func NewURLShortenerService(db *sql.DB) *URLShortenerService {
    return &URLShortenerService{db: db}
}

// Verticle with direct injection
type APIVerticle struct {
    service *URLShortenerService  // Direct reference
    port    string
}

func NewAPIVerticle(service *URLShortenerService, port string) *APIVerticle {
    return &APIVerticle{
        service: service,  // Direct injection
        port:    port,
    }
}

func (v *APIVerticle) Start(ctx core.FluxorContext) error {
    // Use service directly
    shortURL, err := v.service.ShortenURL(ctx.Context(), url, customCode)
    // ...
}
```

---

## Comparison Table

| Aspect | EventBus Pattern | Direct Injection |
|--------|-----------------|------------------|
| **Decoupling** | ✅ High (address-based) | ❌ Low (type-based) |
| **Location Transparency** | ✅ Yes (local/distributed) | ❌ No (single process) |
| **Reactor Isolation** | ✅ Yes (event loops) | ⚠️ Depends on implementation |
| **Shared State** | ✅ No (immutable messages) | ❌ Yes (shared references) |
| **Backpressure** | ✅ Built-in | ❌ Manual handling |
| **Testability** | ✅ Easy (mock EventBus) | ❌ Hard (real instances) |
| **Scalability** | ✅ Horizontal scaling | ❌ Vertical only |
| **Distribution** | ✅ Yes (NATS/clustered) | ❌ No |
| **Code Changes for Distribution** | ✅ None | ❌ Major refactoring |
| **Performance Overhead** | ⚠️ Small (message passing) | ✅ None (direct calls) |
| **Complexity** | ⚠️ Medium (addresses) | ✅ Low (direct calls) |
| **Recommended For** | ✅ Production, scalable systems | ⚠️ Simple, single-process apps |

---

## Migration Guide: Direct Injection → EventBus

If you have existing code using direct injection, here's how to migrate to EventBus:

### Step 1: Convert Service to BaseService

**Before (Direct Injection):**
```go
type UserService struct {
    db *sql.DB
}

func (s *UserService) GetUser(userID string) (*User, error) {
    // Direct database access
    // ...
}
```

**After (EventBus):**
```go
type UserService struct {
    *core.BaseService
    db *sql.DB
}

func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    request := msg.Body().(map[string]interface{})
    userID := request["userID"].(string)
    
    // Database access
    user, err := s.getUserFromDB(userID)
    if err != nil {
        return s.Fail(msg, 404, "User not found")
    }
    return s.Reply(msg, user)
}
```

### Step 2: Update Verticle to Use EventBus

**Before (Direct Injection):**
```go
type APIVerticle struct {
    userService *UserService  // Direct reference
}

func (v *APIVerticle) doStart(ctx core.FluxorContext) error {
    // Direct call
    user, err := v.userService.GetUser(userID)
    // ...
}
```

**After (EventBus):**
```go
type APIVerticle struct {
    *core.BaseVerticle
    userServiceAddress string
}

func (v *APIVerticle) doStart(ctx core.FluxorContext) error {
    v.userServiceAddress = "user.service"
    
    consumer := v.Consumer("api.request")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        // EventBus call
        reply, err := v.Request(v.userServiceAddress, map[string]interface{}{
            "userID": userID,
        }, 5*time.Second)
        // ...
    })
    return nil
}
```

### Step 3: Deploy Services Separately

**Before:**
```go
service := NewUserService(db)
verticle := NewAPIVerticle(service, port)  // Service injected
gocmd.DeployVerticle(verticle)
```

**After:**
```go
// Deploy service first
service := &UserService{
    BaseService: core.NewBaseService("user-service", "user.service"),
    db: db,
}
gocmd.DeployVerticle(service)

// Deploy verticle (no service reference needed)
verticle := &APIVerticle{
    BaseVerticle: core.NewBaseVerticle("api-verticle"),
}
gocmd.DeployVerticle(verticle)
```

---

## Best Practices

### ✅ DO: Use EventBus for Service Communication

1. **Always use BaseService** for services that need to be called by other verticles
2. **Use addresses, not types** - communicate via EventBus addresses
3. **Handle timeouts** - always specify timeout for Request calls
4. **Handle errors** - check for errors and return appropriate failure codes
5. **Use request-reply pattern** - use `Request()` for synchronous calls
6. **Deploy services independently** - services and verticles are separate deployments
7. **Use WorkerPool for CPU-bound work** - services are often CPU-bound; use `ExecuteBlocking()` for CPU-intensive operations

### ❌ DON'T: Anti-Patterns to Avoid

1. **Don't share service instances** - services should be deployed, not injected
2. **Don't use direct method calls** - use EventBus Request/Publish instead
3. **Don't ignore timeouts** - always specify timeout for Request calls
4. **Don't share mutable state** - communicate via immutable messages
5. **Don't block event loops** - use WorkerPool for blocking operations

---

## Examples

### Example 1: Simple Service (EventBus)

**I/O-bound service** (database, network calls):

```go
// Service
type CalculatorService struct {
    *core.BaseService
}

func (s *CalculatorService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    request := msg.Body().(map[string]interface{})
    a := request["a"].(float64)
    b := request["b"].(float64)
    op := request["op"].(string)
    
    var result float64
    switch op {
    case "+":
        result = a + b
    case "-":
        result = a - b
    case "*":
        result = a * b
    case "/":
        if b == 0 {
            return s.Fail(msg, 400, "Division by zero")
        }
        result = a / b
    default:
        return s.Fail(msg, 400, "Unknown operator")
    }
    
    return s.Reply(msg, map[string]interface{}{"result": result})
}

// Deploy
service := &CalculatorService{
    BaseService: core.NewBaseService("calculator-service", "calculator.service"),
}
gocmd.DeployVerticle(service)
```

### Example 1b: CPU-Bound Service (EventBus + WorkerPool)

**CPU-bound service** (image processing, ML inference, heavy computations):

```go
// Service
type ImageProcessingService struct {
    *core.BaseService
}

func (s *ImageProcessingService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    request := msg.Body().(map[string]interface{})
    imageData := request["image"].([]byte)
    
    // CPU-bound work: Use WorkerPool (ExecuteBlocking)
    result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
        // CPU-intensive work (runs on worker thread, not event loop)
        processedImage := processImage(imageData)
        return processedImage, nil
    }, 30*time.Second)
    
    if err != nil {
        return s.Fail(msg, 500, "Processing failed: "+err.Error())
    }
    
    return s.Reply(msg, map[string]interface{}{
        "processedImage": result,
    })
}

// Deploy
service := &ImageProcessingService{
    BaseService: core.NewBaseService("image-service", "image.service"),
}
gocmd.DeployVerticle(service)
```

### Example 2: Orchestrating Multiple Services (EventBus)

```go
type OrderVerticle struct {
    *core.BaseVerticle
    userServiceAddr    string
    paymentServiceAddr string
    inventoryServiceAddr string
}

func (v *OrderVerticle) doStart(ctx core.FluxorContext) error {
    v.userServiceAddr = "user.service"
    v.paymentServiceAddr = "payment.service"
    v.inventoryServiceAddr = "inventory.service"
    
    consumer := v.Consumer("order.create")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        request := msg.Body().(map[string]interface{})
        userID := request["userID"].(string)
        productID := request["productID"].(string)
        amount := request["amount"].(float64)
        
        // Step 1: Validate user
        _, err := v.Request(v.userServiceAddr, map[string]interface{}{
            "userID": userID,
        }, 5*time.Second)
        if err != nil {
            return msg.Fail(400, "Invalid user: "+err.Error())
        }
        
        // Step 2: Check inventory
        _, err = v.Request(v.inventoryServiceAddr, map[string]interface{}{
            "productID": productID,
        }, 5*time.Second)
        if err != nil {
            return msg.Fail(400, "Inventory check failed: "+err.Error())
        }
        
        // Step 3: Process payment
        _, err = v.Request(v.paymentServiceAddr, map[string]interface{}{
            "userID": userID,
            "amount": amount,
        }, 10*time.Second)
        if err != nil {
            return msg.Fail(402, "Payment failed: "+err.Error())
        }
        
        return msg.Reply(map[string]interface{}{
            "orderID": "order-123",
            "status":  "completed",
        })
    })
    return nil
}
```

---

## Summary

**EventBus is the recommended pattern** for service communication in Fluxor because it:

- ✅ Aligns with event-driven architecture principles
- ✅ Provides decoupling and location transparency
- ✅ Enables scalability and distribution
- ✅ Maintains reactor isolation
- ✅ Eliminates shared mutable state
- ✅ Improves testability

**Direct injection is acceptable** for:
- Simple, single-process applications
- Tightly coupled components
- Legacy code migration

However, **EventBus is preferred** for production, scalable systems.

---

## Related Documentation

- [BASE_CLASSES.md](./BASE_CLASSES.md) - Base classes and patterns
- [EVENTBUS_FLOW.md](./EVENTBUS_FLOW.md) - EventBus flow architecture
- [eventbus.go](./eventbus.go) - EventBus interface and implementation
