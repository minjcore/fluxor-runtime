//go:build ignore
// This file contains Example* functions that reference types (DeployService, ServiceInjection, etc.)
// not defined in this repo; exclude from default build so go test ./pkg/core can run.
// To include: go test -tags=examples ./pkg/core (after defining the missing types).

package core_test

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core"
)

// ExampleBaseVerticle demonstrates using BaseVerticle (Java-style abstract class)
func ExampleBaseVerticle() {
	// Create a custom verticle by embedding BaseVerticle
	type MyVerticle struct {
		*core.BaseVerticle
	}

	// Create and use the verticle
	verticle := &MyVerticle{
		BaseVerticle: core.NewBaseVerticle("my-verticle"),
	}

	// Note: In real usage, you would override doStart method:
	// func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
	//     consumer := v.Consumer("my.address")
	//     consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
	//         return msg.Reply("processed")
	//     })
	//     return nil
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)

	// Deploy verticle
	gocmd.DeployVerticle(verticle)
}

// ExampleBaseService demonstrates using BaseService (Java-style service pattern)
// This example shows how to create a service that handles request-reply patterns
func ExampleBaseService() {
	// Step 1: Create a custom service by embedding BaseService
	// In your actual code, define this outside the example:
	//
	// type UserService struct {
	//     *core.BaseService
	//     users map[string]map[string]interface{} // In-memory store
	// }
	//
	// Step 2: Override doHandleRequest to implement service logic:
	//
	// func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     // Parse request body
	//     request, ok := msg.Body().(map[string]interface{})
	//     if !ok {
	//         return s.Fail(msg, 400, "Invalid request format")
	//     }
	//
	//     // Extract action and user ID
	//     action, _ := request["action"].(string)
	//     userID, _ := request["userID"].(string)
	//
	//     switch action {
	//     case "get":
	//         if userID == "" {
	//             return s.Fail(msg, 400, "userID is required")
	//         }
	//         user, exists := s.users[userID]
	//         if !exists {
	//             return s.Fail(msg, 404, "User not found")
	//         }
	//         return s.Reply(msg, user)
	//
	//     case "create":
	//         name, _ := request["name"].(string)
	//         if name == "" {
	//             return s.Fail(msg, 400, "name is required")
	//         }
	//         user := map[string]interface{}{
	//             "id":   userID,
	//             "name": name,
	//         }
	//         s.users[userID] = user
	//         return s.Reply(msg, user)
	//
	//     default:
	//         return s.Fail(msg, 400, "Unknown action: "+action)
	//     }
	// }
	//
	// Step 3: Optionally override doStart to initialize service state:
	//
	// func (s *UserService) doStart(ctx core.FluxorContext) error {
	//     s.users = make(map[string]map[string]interface{})
	//     // BaseService.doStart is automatically called to register the handler
	//     return nil
	// }

	// Create service instance
	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("user-service", "user.service"),
	}

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)

	// Deploy service - it will automatically register on "user.service" address
	// You can use DeployService (convenience method) or DeployVerticle (generic method)
	deploymentID, err := gocmd.DeployService(service.BaseService)
	if err != nil {
		// Handle deployment error
		return
	}
	_ = deploymentID // Use deploymentID for undeployment if needed

	// Example: Send request to service (request-reply pattern)
	// In a real application, this would be done from another verticle:
	//
	// reply, err := gocmd.Send("user.service", map[string]interface{}{
	//     "action": "create",
	//     "name":   "John Doe",
	// }, 5*time.Second)
	// if err != nil {
	//     log.Printf("Error: %v", err)
	// } else {
	//     user := reply.Body().(map[string]interface{})
	//     log.Printf("Created user: %v", user)
	// }
}

// ExampleBaseService_withHandler demonstrates using SetRequestHandler as an alternative
// to overriding doHandleRequest. This is useful when you want to set handlers dynamically
// or use function-based handlers instead of struct methods.
func ExampleBaseService_withHandler() {
	// Create service
	service := core.NewBaseService("calculator-service", "calculator.service")

	// Set a custom request handler instead of overriding doHandleRequest
	// This handler will be called for all requests to "calculator.service"
	service.SetRequestHandler(func(ctx core.FluxorContext, msg core.Message) error {
		// Parse request
		request, ok := msg.Body().(map[string]interface{})
		if !ok {
			return msg.Fail(400, "Invalid request format")
		}

		// Extract operands
		a, _ := request["a"].(float64)
		b, _ := request["b"].(float64)
		op, _ := request["op"].(string)

		// Perform calculation
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
				return msg.Fail(400, "Division by zero")
			}
			result = a / b
		default:
			return msg.Fail(400, "Unknown operator: "+op)
		}

		// Reply with result using convenience method
		return service.Reply(msg, map[string]interface{}{
			"result": result,
		})
	})

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)

	// Deploy service - it will automatically register on "calculator.service" address
	// Use DeployService for services (more semantic than DeployVerticle)
	deploymentID, err := gocmd.DeployService(service)
	if err != nil {
		// Handle deployment error
		return
	}
	_ = deploymentID // Use deploymentID for undeployment if needed

	// Example usage from another verticle:
	//
	// reply, err := gocmd.Send("calculator.service", map[string]interface{}{
	//     "a":  10.0,
	//     "b":  5.0,
	//     "op": "+",
	// }, 5*time.Second)
	// if err != nil {
	//     log.Printf("Error: %v", err)
	// } else {
	//     result := reply.Body().(map[string]interface{})
	//     log.Printf("Result: %v", result["result"]) // Output: Result: 15
	// }
}

// ExampleBaseHandler demonstrates using BaseHandler (Java-style handler pattern)
func ExampleBaseHandler() {
	// Create a custom handler by embedding BaseHandler
	type UserHandler struct {
		*core.BaseHandler
	}

	// Create handler
	handler := &UserHandler{
		BaseHandler: core.NewBaseHandler("user-handler"),
	}

	// Note: In real usage, you would override doHandle method:
	// func (h *UserHandler) doHandle(ctx core.FluxorContext, msg core.Message) error {
	//     var request map[string]interface{}
	//     if err := h.DecodeBody(msg, &request); err != nil {
	//         return h.Fail(msg, 400, "Invalid request")
	//     }
	//     userID := request["id"].(string)
	//     userData := map[string]interface{}{
	//         "id":   userID,
	//         "name": "John Doe",
	//     }
	//     return h.Reply(msg, userData)
	// }

	_ = handler // Use handler
}

// ExampleBaseComponent demonstrates using BaseComponent (Java-style component pattern)
func ExampleBaseComponent() {
	// Create a custom component by embedding BaseComponent
	type DatabaseComponent struct {
		*core.BaseComponent
		connection string
	}

	// Create component
	component := &DatabaseComponent{
		BaseComponent: core.NewBaseComponent("database"),
	}

	// Note: In real usage, you would override doStart and doStop methods:
	// func (c *DatabaseComponent) doStart(ctx core.FluxorContext) error {
	//     c.connection = "connected"
	//     return nil
	// }
	// func (c *DatabaseComponent) doStop(ctx core.FluxorContext) error {
	//     c.connection = "disconnected"
	//     return nil
	// }

	_ = component // Use component
}

// ExampleDeployService demonstrates using DeployService convenience method
// DeployService is a convenience method specifically for deploying BaseService instances
func ExampleDeployService() {
	// Create service
	service := core.NewBaseService("user-service", "user.service")

	// Deploy service using DeployService (convenience method)
	// This is equivalent to DeployVerticle(service) but provides a more semantic API
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	deploymentID, err := gocmd.DeployService(service)
	if err != nil {
		// Handle deployment error
		return
	}

	// Use deploymentID for undeployment if needed
	_ = deploymentID

	// You can also use DeployService from FluxorContext:
	// ctx := gocmd.Context()
	// fluxorCtx := core.NewFluxorContext(ctx, gocmd)
	// deploymentID, err := fluxorCtx.DeployService(service)
}

// ExampleServiceInjection_EventBus demonstrates EventBus-based service injection (RECOMMENDED PATTERN)
// EventBus is the most stable and principled pattern for service communication:
// - Decoupling: Services communicate via addresses, not direct references
// - Location transparency: Can be local or distributed (NATS/clustered)
// - Reactor isolation: Handlers execute on event loops
// - No shared mutable state: Communication via immutable messages
// - Backpressure handling: Built-in queue management
// - Testability: Easy to mock/test
// - Scalability: Can distribute without code changes
func ExampleServiceInjection_EventBus() {
	// Step 1: Define service using BaseService (EventBus-based)
	// In your actual code, define this outside the example:
	//
	// type UserService struct {
	//     *core.BaseService
	//     users map[string]map[string]interface{}
	// }
	//
	// func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     userID := request["userID"].(string)
	//     user, exists := s.users[userID]
	//     if !exists {
	//         return s.Fail(msg, 404, "User not found")
	//     }
	//     return s.Reply(msg, user)
	// }
	//
	// func (s *UserService) doStart(ctx core.FluxorContext) error {
	//     s.users = make(map[string]map[string]interface{})
	//     s.users["123"] = map[string]interface{}{"id": "123", "name": "John Doe"}
	//     return nil
	// }

	// Step 2: Define verticle that uses service via EventBus
	// In your actual code, define this outside the example:
	//
	// type APIVerticle struct {
	//     *core.BaseVerticle
	//     userServiceAddress string  // Store address, not service reference
	// }
	//
	// func (v *APIVerticle) doStart(ctx core.FluxorContext) error {
	//     v.userServiceAddress = "user.service"  // EventBus address
	//
	//     consumer := v.Consumer("api.request")
	//     consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
	//         request := msg.Body().(map[string]interface{})
	//         userID := request["userID"].(string)
	//
	//         // Call service via EventBus (request-reply pattern)
	//         reply, err := v.Request(v.userServiceAddress, map[string]interface{}{
	//             "userID": userID,
	//         }, 5*time.Second)
	//         if err != nil {
	//             return msg.Fail(502, "Service unavailable: "+err.Error())
	//         }
	//
	//         var user map[string]interface{}
	//         reply.DecodeBody(&user)
	//         return msg.Reply(user)
	//     })
	//     return nil
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Step 3: Deploy service first (independent deployment)
	userService := core.NewBaseService("user-service", "user.service")
	deploymentID, err := gocmd.DeployService(userService)
	if err != nil {
		// Handle deployment error
		return
	}
	_ = deploymentID // Use deploymentID for undeployment if needed

	// Step 4: Deploy verticle (no service reference needed - uses EventBus address)
	apiVerticle := &struct {
		*core.BaseVerticle
	}{
		BaseVerticle: core.NewBaseVerticle("api-verticle"),
	}
	gocmd.DeployVerticle(apiVerticle)

	// Benefits of EventBus pattern:
	// - Services and verticles are decoupled
	// - Can swap to clustered EventBus without code changes
	// - Easy to test by mocking EventBus
	// - Services can be distributed across nodes
}

// ExampleServiceInjection_Direct demonstrates direct injection (ACCEPTABLE for simple cases)
// Direct injection is acceptable for:
// - Simple, single-process applications
// - Tightly coupled components that never need distribution
// - Legacy code migration
// However, EventBus is preferred for production, scalable systems.
func ExampleServiceInjection_Direct() {
	// Direct injection pattern (acceptable for simple cases)
	// In your actual code, define this outside the example:
	//
	// type URLShortenerService struct {
	//     db *sql.DB
	// }
	//
	// func NewURLShortenerService(db *sql.DB) *URLShortenerService {
	//     return &URLShortenerService{db: db}
	// }
	//
	// type APIVerticle struct {
	//     service *URLShortenerService  // Direct reference (tight coupling)
	//     port    string
	// }
	//
	// func NewAPIVerticle(service *URLShortenerService, port string) *APIVerticle {
	//     return &APIVerticle{
	//         service: service,  // Direct injection
	//         port:    port,
	//     }
	// }
	//
	// func (v *APIVerticle) Start(ctx core.FluxorContext) error {
	//     // Use service directly (no EventBus)
	//     shortURL, err := v.service.ShortenURL(ctx.Context(), url, customCode)
	//     // ...
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Limitations of direct injection:
	// - Tight coupling: Verticle depends on concrete service type
	// - Cannot distribute services across nodes
	// - Harder to test (requires real service instances)
	// - Violates event-driven principles
	// - Shared mutable state risks

	// For production systems, prefer EventBus pattern (see ExampleServiceInjection_EventBus)
	_ = gocmd
}

// ExampleVerticle_usingService demonstrates how a verticle uses a service
// This is the most common pattern: a verticle calls services via request-reply
// RECOMMENDED: Use EventBus pattern (see ExampleServiceInjection_EventBus)
func ExampleVerticle_usingService() {
	// Step 1: Define a service (e.g., UserService)
	// In your actual code, this would be in a separate file:
	//
	// type UserService struct {
	//     *core.BaseService
	//     users map[string]map[string]interface{}
	// }
	//
	// func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     userID := request["userID"].(string)
	//     user, exists := s.users[userID]
	//     if !exists {
	//         return s.Fail(msg, 404, "User not found")
	//     }
	//     return s.Reply(msg, user)
	// }
	//
	// func (s *UserService) doStart(ctx core.FluxorContext) error {
	//     s.users = make(map[string]map[string]interface{})
	//     s.users["123"] = map[string]interface{}{"id": "123", "name": "John Doe"}
	//     return nil
	// }

	// Step 2: Define a verticle that uses the service
	// In your actual code, this would be in a separate file:
	//
	// type APIVerticle struct {
	//     *core.BaseVerticle
	//     userServiceAddress string
	// }
	//
	// func (v *APIVerticle) doStart(ctx core.FluxorContext) error {
	//     v.userServiceAddress = "user.service"
	//
	//     // Register HTTP handler that calls the service
	//     consumer := v.Consumer("api.request")
	//     consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
	//         // Extract user ID from request
	//         request := msg.Body().(map[string]interface{})
	//         userID := request["userID"].(string)
	//
	//         // Call the service using Request (request-reply pattern)
	//         reply, err := v.Request(v.userServiceAddress, map[string]interface{}{
	//             "userID": userID,
	//         }, 5*time.Second)
	//         if err != nil {
	//             // Service call failed
	//             return msg.Fail(502, "Service unavailable: "+err.Error())
	//         }
	//
	//         // Decode service response
	//         var user map[string]interface{}
	//         if err := reply.DecodeBody(&user); err != nil {
	//             return msg.Fail(500, "Failed to decode response")
	//         }
	//
	//         // Forward service response to original request
	//         return msg.Reply(user)
	//     })
	//
	//     return nil
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Step 3: Deploy the service first
	userService := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("user-service", "user.service"),
	}
	gocmd.DeployVerticle(userService)

	// Step 4: Deploy the verticle that uses the service
	apiVerticle := &struct {
		*core.BaseVerticle
	}{
		BaseVerticle: core.NewBaseVerticle("api-verticle"),
	}
	gocmd.DeployVerticle(apiVerticle)

	// Step 5: Example of calling the service from the verticle
	// In a real application, this would happen inside the verticle's handler:
	//
	// reply, err := apiVerticle.Request("user.service", map[string]interface{}{
	//     "userID": "123",
	// }, 5*time.Second)
	// if err != nil {
	//     log.Printf("Service call failed: %v", err)
	// } else {
	//     var user map[string]interface{}
	//     reply.DecodeBody(&user)
	//     log.Printf("User: %v", user)
	// }
}

// ExampleVerticle_usingService_multipleCalls demonstrates calling services multiple times
// and handling errors properly
func ExampleVerticle_usingService_multipleCalls() {
	// This example shows a verticle that orchestrates multiple service calls
	//
	// type OrderVerticle struct {
	//     *core.BaseVerticle
	//     userServiceAddr    string
	//     paymentServiceAddr string
	//     inventoryServiceAddr string
	// }
	//
	// func (v *OrderVerticle) doStart(ctx core.FluxorContext) error {
	//     v.userServiceAddr = "user.service"
	//     v.paymentServiceAddr = "payment.service"
	//     v.inventoryServiceAddr = "inventory.service"
	//
	//     consumer := v.Consumer("order.create")
	//     consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
	//         request := msg.Body().(map[string]interface{})
	//         userID := request["userID"].(string)
	//         productID := request["productID"].(string)
	//         amount := request["amount"].(float64)
	//
	//         // Step 1: Validate user
	//         userReply, err := v.Request(v.userServiceAddr, map[string]interface{}{
	//             "userID": userID,
	//         }, 5*time.Second)
	//         if err != nil {
	//             return msg.Fail(400, "Invalid user: "+err.Error())
	//         }
	//
	//         // Step 2: Check inventory
	//         invReply, err := v.Request(v.inventoryServiceAddr, map[string]interface{}{
	//             "productID": productID,
	//         }, 5*time.Second)
	//         if err != nil {
	//             return msg.Fail(400, "Inventory check failed: "+err.Error())
	//         }
	//
	//         // Step 3: Process payment
	//         paymentReply, err := v.Request(v.paymentServiceAddr, map[string]interface{}{
	//             "userID": userID,
	//             "amount": amount,
	//         }, 10*time.Second)
	//         if err != nil {
	//             return msg.Fail(402, "Payment failed: "+err.Error())
	//         }
	//
	//         // All services succeeded, return success
	//         return msg.Reply(map[string]interface{}{
	//             "orderID": "order-123",
	//             "status":  "completed",
	//         })
	//     })
	//
	//     return nil
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Deploy services using DeployService (convenience method)
	userService := core.NewBaseService("user-service", "user.service")
	paymentService := core.NewBaseService("payment-service", "payment.service")
	inventoryService := core.NewBaseService("inventory-service", "inventory.service")

	_, _ = gocmd.DeployService(userService)
	_, _ = gocmd.DeployService(paymentService)
	_, _ = gocmd.DeployService(inventoryService)

	// Deploy orchestrator verticle
	orderVerticle := &struct {
		*core.BaseVerticle
	}{
		BaseVerticle: core.NewBaseVerticle("order-verticle"),
	}
	gocmd.DeployVerticle(orderVerticle)

	// Example usage:
	// reply, err := orderVerticle.Request("order.create", map[string]interface{}{
	//     "userID":    "123",
	//     "productID": "prod-456",
	//     "amount":    99.99,
	// }, 15*time.Second)
	// if err != nil {
	//     log.Printf("Order creation failed: %v", err)
	// } else {
	//     var order map[string]interface{}
	//     reply.DecodeBody(&order)
	//     log.Printf("Order created: %v", order)
	// }
}

// ExampleServiceInjection_EventBus_Orchestration demonstrates orchestrating multiple services via EventBus
// This shows how to coordinate multiple service calls while maintaining decoupling
func ExampleServiceInjection_EventBus_Orchestration() {
	// This example shows orchestrating multiple services via EventBus
	// All services communicate via EventBus addresses, maintaining decoupling
	//
	// type OrderVerticle struct {
	//     *core.BaseVerticle
	//     userServiceAddr    string      // EventBus address, not reference
	//     paymentServiceAddr string     // EventBus address, not reference
	//     inventoryServiceAddr string    // EventBus address, not reference
	// }
	//
	// func (v *OrderVerticle) doStart(ctx core.FluxorContext) error {
	//     v.userServiceAddr = "user.service"
	//     v.paymentServiceAddr = "payment.service"
	//     v.inventoryServiceAddr = "inventory.service"
	//
	//     consumer := v.Consumer("order.create")
	//     consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
	//         request := msg.Body().(map[string]interface{})
	//         userID := request["userID"].(string)
	//         productID := request["productID"].(string)
	//         amount := request["amount"].(float64)
	//
	//         // Step 1: Validate user via EventBus
	//         _, err := v.Request(v.userServiceAddr, map[string]interface{}{
	//             "userID": userID,
	//         }, 5*time.Second)
	//         if err != nil {
	//             return msg.Fail(400, "Invalid user: "+err.Error())
	//         }
	//
	//         // Step 2: Check inventory via EventBus
	//         _, err = v.Request(v.inventoryServiceAddr, map[string]interface{}{
	//             "productID": productID,
	//         }, 5*time.Second)
	//         if err != nil {
	//             return msg.Fail(400, "Inventory check failed: "+err.Error())
	//         }
	//
	//         // Step 3: Process payment via EventBus
	//         _, err = v.Request(v.paymentServiceAddr, map[string]interface{}{
	//             "userID": userID,
	//             "amount": amount,
	//         }, 10*time.Second)
	//         if err != nil {
	//             return msg.Fail(402, "Payment failed: "+err.Error())
	//         }
	//
	//         return msg.Reply(map[string]interface{}{
	//             "orderID": "order-123",
	//             "status":  "completed",
	//         })
	//     })
	//     return nil
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Deploy services independently (EventBus-based) using DeployService
	userService := core.NewBaseService("user-service", "user.service")
	paymentService := core.NewBaseService("payment-service", "payment.service")
	inventoryService := core.NewBaseService("inventory-service", "inventory.service")

	_, _ = gocmd.DeployService(userService)
	_, _ = gocmd.DeployService(paymentService)
	_, _ = gocmd.DeployService(inventoryService)

	// Deploy orchestrator verticle (no service references - uses EventBus addresses)
	orderVerticle := &struct {
		*core.BaseVerticle
	}{
		BaseVerticle: core.NewBaseVerticle("order-verticle"),
	}
	gocmd.DeployVerticle(orderVerticle)

	// Benefits of EventBus orchestration:
	// - All services are decoupled
	// - Services can be distributed across nodes
	// - Easy to add/remove services
	// - Can swap to clustered EventBus without code changes
}

// ExampleStartx demonstrates Startx pattern: I/O-Bound Verticle + CPU-Bound Service
// Startx is a common pattern where:
// - Verticle is I/O-bound (handles HTTP, database, network I/O)
// - Service is CPU-bound (performs heavy computations using WorkerPool)
func ExampleStartx() {
	// Step 1: Define CPU-Bound Service (uses WorkerPool for heavy computations)
	// In your actual code, define this outside the example:
	//
	// type ImageProcessingService struct {
	//     *core.BaseService
	// }
	//
	// func (s *ImageProcessingService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     imageData := request["image"].([]byte)
	//     operation := request["operation"].(string)
	//
	//     // CPU-bound work: Use WorkerPool (ExecuteBlocking)
	//     // This executes on a worker thread, not the event loop
	//     result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
	//         // CPU-intensive image processing
	//         var processedImage []byte
	//         switch operation {
	//         case "resize":
	//             processedImage = resizeImage(imageData, 800, 600)
	//         case "compress":
	//             processedImage = compressImage(imageData)
	//         default:
	//             return nil, fmt.Errorf("unknown operation: %s", operation)
	//         }
	//         return processedImage, nil
	//     }, 30*time.Second)
	//
	//     if err != nil {
	//         return s.Fail(msg, 500, "Processing failed: "+err.Error())
	//     }
	//
	//     return s.Reply(msg, map[string]interface{}{
	//         "processedImage": result,
	//     })
	// }

	// Step 2: Define I/O-Bound Verticle (handles HTTP, database, network)
	// In your actual code, define this outside the example:
	//
	// type APIVerticle struct {
	//     *core.BaseVerticle
	//     imageServiceAddress string
	// }
	//
	// func (v *APIVerticle) doStart(ctx core.FluxorContext) error {
	//     v.imageServiceAddress = "image.service"
	//
	//     // I/O-bound: Handle HTTP requests (fast, non-blocking)
	//     consumer := v.Consumer("api.image.process")
	//     consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
	//         request := msg.Body().(map[string]interface{})
	//         imageData := request["image"].([]byte)
	//         operation := request["operation"].(string)
	//
	//         // Call CPU-bound service via EventBus
	//         // Verticle doesn't block - service handles CPU work on WorkerPool
	//         reply, err := v.Request(v.imageServiceAddress, map[string]interface{}{
	//             "image":     imageData,
	//             "operation": operation,
	//         }, 30*time.Second)
	//
	//         if err != nil {
	//             return msg.Fail(502, "Service unavailable: "+err.Error())
	//         }
	//
	//         var result map[string]interface{}
	//         reply.DecodeBody(&result)
	//         return msg.Reply(result)
	//     })
	//
	//     return nil
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Step 3: Startx - Deploy CPU-bound service first using DeployService
	imageService := core.NewBaseService("image-service", "image.service")
	_, _ = gocmd.DeployService(imageService)

	// Step 4: Deploy I/O-bound verticle
	apiVerticle := &struct {
		*core.BaseVerticle
	}{
		BaseVerticle: core.NewBaseVerticle("api-verticle"),
	}
	gocmd.DeployVerticle(apiVerticle)

	// Key Benefits of Startx Pattern:
	// - Separation of concerns: I/O and CPU work are separated
	// - Non-blocking: Verticle never blocks on CPU work
	// - Scalability: Service can be distributed, verticle stays I/O-focused
	// - Resource efficiency: CPU work uses WorkerPool, I/O uses event loops
}

// ExampleService_Validation demonstrates a service with input validation
func ExampleService_Validation() {
	// Service with input validation
	// In your actual code, define this outside the example:
	//
	// type UserService struct {
	//     *core.BaseService
	//     users map[string]map[string]interface{}
	// }
	//
	// func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     userID, ok := request["userID"].(string)
	//     if !ok || userID == "" {
	//         return s.Fail(msg, 400, "userID is required")
	//     }
	//
	//     email, ok := request["email"].(string)
	//     if !ok || email == "" {
	//         return s.Fail(msg, 400, "email is required")
	//     }
	//
	//     // Validate email format
	//     if !isValidEmail(email) {
	//         return s.Fail(msg, 400, "invalid email format")
	//     }
	//
	//     // Process valid request
	//     user := map[string]interface{}{
	//         "userID": userID,
	//         "email":  email,
	//     }
	//     s.users[userID] = user
	//     return s.Reply(msg, user)
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("user-service", "user.service"),
	}
	gocmd.DeployVerticle(service)
}

// ExampleService_ErrorHandling demonstrates comprehensive error handling
func ExampleService_ErrorHandling() {
	// Service with comprehensive error handling
	// In your actual code, define this outside the example:
	//
	// type DataService struct {
	//     *core.BaseService
	//     db *sql.DB
	// }
	//
	// func (s *DataService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     id, _ := request["id"].(string)
	//
	//     // Use WorkerPool for database access (I/O-bound but blocking)
	//     result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
	//         var data map[string]interface{}
	//         err := s.db.QueryRow("SELECT * FROM data WHERE id = $1", id).Scan(&data)
	//         if err == sql.ErrNoRows {
	//             return nil, &NotFoundError{ID: id}
	//         }
	//         if err != nil {
	//             return nil, fmt.Errorf("database error: %w", err)
	//         }
	//         return data, nil
	//     }, 5*time.Second)
	//
	//     if err != nil {
	//         if _, ok := err.(*NotFoundError); ok {
	//             return s.Fail(msg, 404, "data not found: "+id)
	//         }
	//         return s.Fail(msg, 500, "internal error: "+err.Error())
	//     }
	//
	//     return s.Reply(msg, result)
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("data-service", "data.service"),
	}
	gocmd.DeployVerticle(service)
}

// ExampleService_Caching demonstrates a service with caching
func ExampleService_Caching() {
	// Service with caching
	// In your actual code, define this outside the example:
	//
	// type CacheService struct {
	//     *core.BaseService
	//     cache map[string]interface{}
	//     mu    sync.RWMutex
	// }
	//
	// func (s *CacheService) doStart(ctx core.FluxorContext) error {
	//     s.cache = make(map[string]interface{})
	//     return nil
	// }
	//
	// func (s *CacheService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     key, _ := request["key"].(string)
	//     action, _ := request["action"].(string)
	//
	//     switch action {
	//     case "get":
	//         s.mu.RLock()
	//         value, exists := s.cache[key]
	//         s.mu.RUnlock()
	//         if !exists {
	//             return s.Fail(msg, 404, "key not found: "+key)
	//         }
	//         return s.Reply(msg, map[string]interface{}{"value": value})
	//
	//     case "set":
	//         value := request["value"]
	//         s.mu.Lock()
	//         s.cache[key] = value
	//         s.mu.Unlock()
	//         return s.Reply(msg, map[string]interface{}{"status": "ok"})
	//
	//     default:
	//         return s.Fail(msg, 400, "unknown action: "+action)
	//     }
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("cache-service", "cache.service"),
	}
	gocmd.DeployVerticle(service)
}

// ExampleService_Chaining demonstrates service chaining (service calling another service)
func ExampleService_Chaining() {
	// Service chaining: service calls another service
	// In your actual code, define this outside the example:
	//
	// type OrderService struct {
	//     *core.BaseService
	//     userServiceAddress string
	//     productServiceAddress string
	// }
	//
	// func (s *OrderService) doStart(ctx core.FluxorContext) error {
	//     s.userServiceAddress = "user.service"
	//     s.productServiceAddress = "product.service"
	//     return nil
	// }
	//
	// func (s *OrderService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     userID := request["userID"].(string)
	//     productID := request["productID"].(string)
	//
	//     // Step 1: Validate user
	//     userReply, err := s.Request(s.userServiceAddress, map[string]interface{}{
	//         "userID": userID,
	//     }, 5*time.Second)
	//     if err != nil {
	//         return s.Fail(msg, 400, "invalid user: "+err.Error())
	//     }
	//
	//     // Step 2: Get product
	//     productReply, err := s.Request(s.productServiceAddress, map[string]interface{}{
	//         "productID": productID,
	//     }, 5*time.Second)
	//     if err != nil {
	//         return s.Fail(msg, 400, "product not found: "+err.Error())
	//     }
	//
	//     // Step 3: Create order
	//     order := map[string]interface{}{
	//         "orderID":   "order-123",
	//         "userID":    userID,
	//         "productID": productID,
	//         "status":    "created",
	//     }
	//     return s.Reply(msg, order)
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Deploy dependent services first using DeployService
	userService := core.NewBaseService("user-service", "user.service")
	productService := core.NewBaseService("product-service", "product.service")
	_, _ = gocmd.DeployService(userService)
	_, _ = gocmd.DeployService(productService)

	// Deploy service that chains to others
	orderService := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("order-service", "order.service"),
	}
	gocmd.DeployVerticle(orderService)
}

// ExampleService_TimeoutHandling demonstrates service with timeout handling
func ExampleService_TimeoutHandling() {
	// Service with timeout handling for external calls
	// In your actual code, define this outside the example:
	//
	// type ExternalAPIService struct {
	//     *core.BaseService
	//     apiURL string
	// }
	//
	// func (s *ExternalAPIService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     data := request["data"]
	//
	//     // External API call with timeout (I/O-bound)
	//     result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
	//         client := &http.Client{Timeout: 10 * time.Second}
	//         resp, err := client.Post(s.apiURL, "application/json", bytes.NewReader(data))
	//         if err != nil {
	//             return nil, err
	//         }
	//         defer resp.Body.Close()
	//         return io.ReadAll(resp.Body)
	//     }, 15*time.Second) // Timeout longer than HTTP client timeout
	//
	//     if err != nil {
	//         if err == context.DeadlineExceeded {
	//             return s.Fail(msg, 504, "external API timeout")
	//         }
	//         return s.Fail(msg, 502, "external API error: "+err.Error())
	//     }
	//
	//     return s.Reply(msg, result)
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("external-api-service", "external.api.service"),
	}
	gocmd.DeployVerticle(service)
}

// ExampleService_DatabaseAccess demonstrates service with database access using WorkerPool
func ExampleService_DatabaseAccess() {
	// Service with database access (I/O-bound, uses WorkerPool)
	// In your actual code, define this outside the example:
	//
	// type TodoService struct {
	//     *core.BaseService
	//     db *sql.DB
	// }
	//
	// func (s *TodoService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     action := request["action"].(string)
	//     userID := request["userID"].(int)
	//
	//     // Database access: Use WorkerPool (I/O-bound but blocking)
	//     result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
	//         switch action {
	//         case "create":
	//             title := request["title"].(string)
	//             var id int
	//             err := s.db.QueryRow("INSERT INTO todos (user_id, title) VALUES ($1, $2) RETURNING id",
	//                 userID, title).Scan(&id)
	//             if err != nil {
	//                 return nil, err
	//             }
	//             return map[string]interface{}{"id": id, "userID": userID, "title": title}, nil
	//
	//         case "list":
	//             rows, err := s.db.Query("SELECT id, user_id, title FROM todos WHERE user_id = $1", userID)
	//             if err != nil {
	//                 return nil, err
	//             }
	//             defer rows.Close()
	//             todos := make([]map[string]interface{}, 0)
	//             for rows.Next() {
	//                 var id, uid int
	//                 var title string
	//                 rows.Scan(&id, &uid, &title)
	//                 todos = append(todos, map[string]interface{}{"id": id, "userID": uid, "title": title})
	//             }
	//             return todos, nil
	//
	//         default:
	//             return nil, fmt.Errorf("unknown action: %s", action)
	//         }
	//     }, 5*time.Second)
	//
	//     if err != nil {
	//         return s.Fail(msg, 500, "database error: "+err.Error())
	//     }
	//
	//     return s.Reply(msg, result)
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("todo-service", "todo.service"),
	}
	gocmd.DeployVerticle(service)
}

// ExampleService_AsyncProcessing demonstrates service with async processing
func ExampleService_AsyncProcessing() {
	// Service with async processing (fire-and-forget pattern)
	// In your actual code, define this outside the example:
	//
	// type NotificationService struct {
	//     *core.BaseService
	//     emailServiceAddress string
	// }
	//
	// func (s *NotificationService) doStart(ctx core.FluxorContext) error {
	//     s.emailServiceAddress = "email.service"
	//     return nil
	// }
	//
	// func (s *NotificationService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     userID := request["userID"].(string)
	//     message := request["message"].(string)
	//
	//     // Process notification synchronously (quick)
	//     notificationID := "notif-" + userID
	//     notification := map[string]interface{}{
	//         "id":      notificationID,
	//         "userID":  userID,
	//         "message": message,
	//     }
	//
	//     // Send email asynchronously (fire-and-forget)
	//     s.Publish(s.emailServiceAddress, map[string]interface{}{
	//         "userID":  userID,
	//         "subject": "Notification",
	//         "body":    message,
	//     })
	//
	//     return s.Reply(msg, notification)
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("notification-service", "notification.service"),
	}
	gocmd.DeployVerticle(service)
}

// ExampleService_HTTPInterface demonstrates a service with both EventBus and HTTP interfaces
// Services can expose HTTP endpoints while also being callable via EventBus
func ExampleService_HTTPInterface() {
	// Service with HTTP interface (can be called via HTTP or EventBus)
	// In your actual code, define this outside the example:
	//
	// type UserService struct {
	//     *core.BaseService
	//     server *web.FastHTTPServer
	//     users  map[string]map[string]interface{}
	//     port   string
	// }
	//
	// func (s *UserService) doStart(ctx core.FluxorContext) error {
	//     s.users = make(map[string]map[string]interface{})
	//
	//     // Start HTTP server
	//     config := web.DefaultFastHTTPServerConfig(":" + s.port)
	//     s.server = web.NewFastHTTPServer(ctx.GoCMD(), config)
	//     router := s.server.FastRouter()
	//
	//     // Note: Import "github.com/fluxorio/fluxor/pkg/web" for web package
	//
	//     // HTTP endpoints
	//     router.GETFast("/users/:id", func(c *web.FastRequestContext) error {
	//         userID := c.Param("id")
	//         user, exists := s.users[userID]
	//         if !exists {
	//             return c.JSON(404, map[string]interface{}{"error": "user not found"})
	//         }
	//         return c.JSON(200, user)
	//     })
	//
	//     router.POSTFast("/users", func(c *web.FastRequestContext) error {
	//         var req map[string]interface{}
	//         if err := c.BindJSON(&req); err != nil {
	//             return c.JSON(400, map[string]interface{}{"error": "invalid JSON"})
	//         }
	//         userID := req["id"].(string)
	//         s.users[userID] = req
	//         return c.JSON(201, req)
	//     })
	//
	//     // Start HTTP server (non-blocking)
	//     go func() {
	//         if err := s.server.Start(); err != nil {
	//             // Handle error
	//         }
	//     }()
	//
	//     return nil
	// }
	//
	// func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     // EventBus interface (same service, different access method)
	//     request := msg.Body().(map[string]interface{})
	//     userID := request["userID"].(string)
	//     action := request["action"].(string)
	//
	//     switch action {
	//     case "get":
	//         user, exists := s.users[userID]
	//         if !exists {
	//             return s.Fail(msg, 404, "user not found")
	//         }
	//         return s.Reply(msg, user)
	//
	//     case "create":
	//         user := map[string]interface{}{
	//             "id":   userID,
	//             "name": request["name"].(string),
	//         }
	//         s.users[userID] = user
	//         return s.Reply(msg, user)
	//
	//     default:
	//         return s.Fail(msg, 400, "unknown action: "+action)
	//     }
	// }
	//
	// func (s *UserService) doStop(ctx core.FluxorContext) error {
	//     if s.server != nil {
	//         return s.server.Stop()
	//     }
	//     return nil
	// }
	//
	// // Usage: Service can be accessed via HTTP or EventBus
	// service := &UserService{
	//     BaseService: core.NewBaseService("user-service", "user.service"),
	//     port:        "8080",
	// }
	// gocmd.DeployVerticle(service)
	//
	// // Access via HTTP:
	// //   GET http://localhost:8080/users/123
	// //   POST http://localhost:8080/users
	//
	// // Access via EventBus:
	// //   reply, err := verticle.Request("user.service", map[string]interface{}{
	// //       "action": "get",
	// //       "userID": "123",
	// //   }, 5*time.Second)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("user-service", "user.service"),
	}
	gocmd.DeployVerticle(service)

	// Benefits of dual interface:
	// - HTTP: Easy integration with web clients, REST APIs
	// - EventBus: Internal service-to-service communication
	// - Same business logic: Both interfaces use the same service methods
	// - Flexibility: Choose the right interface for each use case
}

// ExampleService_Consistent demonstrates service consistency patterns
// Services must be consistent: same input always produces same output
func ExampleService_Consistent() {
	// Consistent service: same input always produces same output
	// In your actual code, define this outside the example:
	//
	// type CalculationService struct {
	//     *core.BaseService
	// }
	//
	// func (s *CalculationService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     a := request["a"].(float64)
	//     b := request["b"].(float64)
	//     op := request["op"].(string)
	//
	//     // Consistent: Same inputs always produce same output
	//     var result float64
	//     switch op {
	//     case "+":
	//         result = a + b  // Deterministic
	//     case "-":
	//         result = a - b  // Deterministic
	//     case "*":
	//         result = a * b  // Deterministic
	//     case "/":
	//         if b == 0 {
	//             return s.Fail(msg, 400, "division by zero")
	//         }
	//         result = a / b  // Deterministic
	//     default:
	//         return s.Fail(msg, 400, "unknown operator")
	//     }
	//
	//     // Consistent response format
	//     return s.Reply(msg, map[string]interface{}{
	//         "result": result,
	//         "op":     op,
	//     })
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("calculation-service", "calculation.service"),
	}
	gocmd.DeployVerticle(service)
}

// ExampleService_Transactional demonstrates transactional service patterns
// Services must be transactional: all-or-nothing operations
func ExampleService_Transactional() {
	// Transactional service: all-or-nothing operations
	// In your actual code, define this outside the example:
	//
	// type OrderService struct {
	//     *core.BaseService
	//     db *sql.DB
	// }
	//
	// func (s *OrderService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     userID := request["userID"].(int)
	//     productID := request["productID"].(int)
	//     quantity := request["quantity"].(int)
	//
	//     // Transactional: Use database transaction (all-or-nothing)
	//     result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
	//         tx, err := s.db.Begin()
	//         if err != nil {
	//             return nil, err
	//         }
	//         defer tx.Rollback()
	//
	//         // Step 1: Check inventory (with lock)
	//         var stock int
	//         err = tx.QueryRow("SELECT stock FROM products WHERE id = $1 FOR UPDATE", productID).Scan(&stock)
	//         if err != nil {
	//             return nil, err
	//         }
	//         if stock < quantity {
	//             return nil, fmt.Errorf("insufficient stock: %d available, %d requested", stock, quantity)
	//         }
	//
	//         // Step 2: Update inventory
	//         _, err = tx.Exec("UPDATE products SET stock = stock - $1 WHERE id = $2", quantity, productID)
	//         if err != nil {
	//             return nil, err
	//         }
	//
	//         // Step 3: Create order
	//         var orderID int
	//         err = tx.QueryRow("INSERT INTO orders (user_id, product_id, quantity) VALUES ($1, $2, $3) RETURNING id",
	//             userID, productID, quantity).Scan(&orderID)
	//         if err != nil {
	//             return nil, err
	//         }
	//
	//         // Commit transaction (all-or-nothing)
	//         if err := tx.Commit(); err != nil {
	//             return nil, err
	//         }
	//
	//         return map[string]interface{}{
	//             "orderID": orderID,
	//             "status":  "created",
	//         }, nil
	//     }, 10*time.Second)
	//
	//     if err != nil {
	//         return s.Fail(msg, 500, "transaction failed: "+err.Error())
	//     }
	//
	//     return s.Reply(msg, result)
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("order-service", "order.service"),
	}
	gocmd.DeployVerticle(service)
}

// ExampleService_Durable demonstrates durable service patterns
// Services must be durable: data persists across restarts
func ExampleService_Durable() {
	// Durable service: data persists across restarts
	// In your actual code, define this outside the example:
	//
	// type DataService struct {
	//     *core.BaseService
	//     db *sql.DB  // Persistent storage
	// }
	//
	// func (s *DataService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     action := request["action"].(string)
	//     key := request["key"].(string)
	//     value := request["value"]
	//
	//     // Durable: Write to persistent storage
	//     result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
	//         switch action {
	//         case "set":
	//             // Durable write: fsync to disk
	//             _, err := s.db.Exec("INSERT INTO data (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2",
	//                 key, value)
	//             if err != nil {
	//                 return nil, err
	//             }
	//             // Ensure durability (PostgreSQL does this by default, but can be explicit)
	//             _, err = s.db.Exec("SELECT pg_sync()")  // Force sync to disk
	//             return map[string]interface{}{"status": "saved"}, err
	//
	//         case "get":
	//             var val string
	//             err := s.db.QueryRow("SELECT value FROM data WHERE key = $1", key).Scan(&val)
	//             if err == sql.ErrNoRows {
	//                 return nil, fmt.Errorf("key not found: %s", key)
	//             }
	//             return map[string]interface{}{"key": key, "value": val}, err
	//
	//         default:
	//             return nil, fmt.Errorf("unknown action: %s", action)
	//         }
	//     }, 5*time.Second)
	//
	//     if err != nil {
	//         return s.Fail(msg, 500, "operation failed: "+err.Error())
	//     }
	//
	//     return s.Reply(msg, result)
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("data-service", "data.service"),
	}
	gocmd.DeployVerticle(service)
}

// ExampleService_Idempotent demonstrates idempotent service patterns
// Services must be idempotent: same request can be safely retried
func ExampleService_Idempotent() {
	// Idempotent service: same request can be safely retried
	// In your actual code, define this outside the example:
	//
	// type PaymentService struct {
	//     *core.BaseService
	//     db *sql.DB
	// }
	//
	// func (s *PaymentService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     paymentID := request["paymentID"].(string)  // Idempotency key
	//     userID := request["userID"].(string)
	//     amount := request["amount"].(float64)
	//
	//     // Idempotent: Check if payment already processed
	//     result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
	//         // Check if payment already exists (idempotency check)
	//         var existingPaymentID string
	//         var status string
	//         err := s.db.QueryRow("SELECT id, status FROM payments WHERE idempotency_key = $1", paymentID).
	//             Scan(&existingPaymentID, &status)
	//
	//         if err == nil {
	//             // Payment already processed - return existing result (idempotent)
	//             return map[string]interface{}{
	//                 "paymentID": existingPaymentID,
	//                 "status":     status,
	//                 "idempotent": true,
	//             }, nil
	//         }
	//
	//         if err != sql.ErrNoRows {
	//             return nil, err
	//         }
	//
	//         // New payment - process it
	//         tx, err := s.db.Begin()
	//         if err != nil {
	//             return nil, err
	//         }
	//         defer tx.Rollback()
	//
	//         // Insert payment with idempotency key (unique constraint)
	//         var newPaymentID string
	//         err = tx.QueryRow("INSERT INTO payments (idempotency_key, user_id, amount, status) VALUES ($1, $2, $3, 'processing') RETURNING id",
	//             paymentID, userID, amount).Scan(&newPaymentID)
	//         if err != nil {
	//             // Duplicate key = idempotent retry (race condition)
	//             if strings.Contains(err.Error(), "duplicate key") {
	//                 tx.Rollback()
	//                 // Retry read
	//                 s.db.QueryRow("SELECT id, status FROM payments WHERE idempotency_key = $1", paymentID).
	//                     Scan(&existingPaymentID, &status)
	//                 return map[string]interface{}{
	//                     "paymentID": existingPaymentID,
	//                     "status":     status,
	//                     "idempotent": true,
	//                 }, nil
	//             }
	//             return nil, err
	//         }
	//
	//         // Process payment (external API call, etc.)
	//         // ... payment processing logic ...
	//
	//         // Update status
	//         _, err = tx.Exec("UPDATE payments SET status = 'completed' WHERE id = $1", newPaymentID)
	//         if err != nil {
	//             return nil, err
	//         }
	//
	//         if err := tx.Commit(); err != nil {
	//             return nil, err
	//         }
	//
	//         return map[string]interface{}{
	//             "paymentID": newPaymentID,
	//             "status":     "completed",
	//             "idempotent": false,
	//         }, nil
	//     }, 30*time.Second)
	//
	//     if err != nil {
	//         return s.Fail(msg, 500, "payment failed: "+err.Error())
	//     }
	//
	//     return s.Reply(msg, result)
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("payment-service", "payment.service"),
	}
	gocmd.DeployVerticle(service)
}

// ExampleService_Stateless demonstrates stateless service patterns
// Services must be stateless: no shared mutable state between requests
func ExampleService_Stateless() {
	// Stateless service: no shared mutable state between requests
	// In your actual code, define this outside the example:
	//
	// type AuthService struct {
	//     *core.BaseService
	//     db *sql.DB  // External state (database), not in-memory
	//     jwtSecret string  // Configuration, not mutable state
	// }
	//
	// func (s *AuthService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     action := request["action"].(string)
	//
	//     // Stateless: Each request is independent, no shared mutable state
	//     result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
	//         switch action {
	//         case "login":
	//             username := request["username"].(string)
	//             password := request["password"].(string)
	//
	//             // Read from database (external state, not in-memory)
	//             var userID string
	//             var passwordHash string
	//             err := s.db.QueryRow("SELECT id, password_hash FROM users WHERE username = $1", username).
	//                 Scan(&userID, &passwordHash)
	//             if err == sql.ErrNoRows {
	//                 return nil, fmt.Errorf("invalid credentials")
	//             }
	//             if err != nil {
	//                 return nil, err
	//             }
	//
	//             // Verify password (stateless operation)
	//             if !verifyPassword(password, passwordHash) {
	//                 return nil, fmt.Errorf("invalid credentials")
	//             }
	//
	//             // Generate token (stateless, deterministic)
	//             token := generateJWT(userID, s.jwtSecret)
	//
	//             return map[string]interface{}{
	//                 "token":  token,
	//                 "userID": userID,
	//             }, nil
	//
	//         case "validate":
	//             token := request["token"].(string)
	//             // Validate token (stateless, no shared state)
	//             claims, err := validateJWT(token, s.jwtSecret)
	//             if err != nil {
	//                 return nil, fmt.Errorf("invalid token: %w", err)
	//             }
	//             return map[string]interface{}{
	//                 "valid":  true,
	//                 "userID": claims["userID"],
	//             }, nil
	//
	//         default:
	//             return nil, fmt.Errorf("unknown action: %s", action)
	//         }
	//     }, 5*time.Second)
	//
	//     if err != nil {
	//         return s.Fail(msg, 401, "auth failed: "+err.Error())
	//     }
	//
	//     return s.Reply(msg, result)
	// }
	//
	// // Stateless: No instance variables that change between requests
	// // All state is external (database) or configuration (jwtSecret)

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("auth-service", "auth.service"),
	}
	gocmd.DeployVerticle(service)
}

// ExampleService_AllPrinciples demonstrates service implementing all principles
// Services must be: Consistent, Transactional, Durable, Idempotent, Stateless
func ExampleService_AllPrinciples() {
	// Service implementing all principles
	// In your actual code, define this outside the example:
	//
	// type OrderService struct {
	//     *core.BaseService
	//     db *sql.DB  // Durable storage
	// }
	//
	// func (s *OrderService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
	//     request := msg.Body().(map[string]interface{})
	//     orderID := request["orderID"].(string)      // Idempotency key
	//     userID := request["userID"].(string)
	//     productID := request["productID"].(string)
	//     quantity := request["quantity"].(int)
	//
	//     // All principles in one service:
	//     result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
	//         // 1. IDEMPOTENT: Check if order already exists
	//         var existingOrderID string
	//         var status string
	//         err := s.db.QueryRow("SELECT id, status FROM orders WHERE idempotency_key = $1", orderID).
	//             Scan(&existingOrderID, &status)
	//         if err == nil {
	//             // Already processed - return existing result (idempotent)
	//             return map[string]interface{}{
	//                 "orderID": existingOrderID,
	//                 "status":  status,
	//             }, nil
	//         }
	//         if err != sql.ErrNoRows {
	//             return nil, err
	//         }
	//
	//         // 2. TRANSACTIONAL: All-or-nothing operation
	//         tx, err := s.db.Begin()
	//         if err != nil {
	//             return nil, err
	//         }
	//         defer tx.Rollback()
	//
	//         // 3. CONSISTENT: Same inputs produce same outputs
	//         // Check inventory with lock (consistent read)
	//         var stock int
	//         err = tx.QueryRow("SELECT stock FROM products WHERE id = $1 FOR UPDATE", productID).Scan(&stock)
	//         if err != nil {
	//             return nil, err
	//         }
	//         if stock < quantity {
	//             return nil, fmt.Errorf("insufficient stock: %d", stock)
	//         }
	//
	//         // 4. DURABLE: Write to persistent storage
	//         var newOrderID string
	//         err = tx.QueryRow("INSERT INTO orders (idempotency_key, user_id, product_id, quantity, status) VALUES ($1, $2, $3, $4, 'created') RETURNING id",
	//             orderID, userID, productID, quantity).Scan(&newOrderID)
	//         if err != nil {
	//             return nil, err
	//         }
	//
	//         // Update inventory (durable)
	//         _, err = tx.Exec("UPDATE products SET stock = stock - $1 WHERE id = $2", quantity, productID)
	//         if err != nil {
	//             return nil, err
	//         }
	//
	//         // 5. STATELESS: No shared mutable state, all state in database
	//         // Commit transaction (durable, transactional)
	//         if err := tx.Commit(); err != nil {
	//             return nil, err
	//         }
	//
	//         return map[string]interface{}{
	//             "orderID": newOrderID,
	//             "status":  "created",
	//         }, nil
	//     }, 10*time.Second)
	//
	//     if err != nil {
	//         return s.Fail(msg, 500, "order creation failed: "+err.Error())
	//     }
	//
	//     // Consistent response format
	//     return s.Reply(msg, result)
	// }

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	service := &struct {
		*core.BaseService
	}{
		BaseService: core.NewBaseService("order-service", "order.service"),
	}
	gocmd.DeployVerticle(service)
}
