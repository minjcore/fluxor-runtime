package core

import (
	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// BaseService provides a Java-style abstract base class for service verticles.
// Services typically handle request-reply patterns and provide business logic.
// Combines BaseVerticle with service-specific patterns.
//
// RECOMMENDED PATTERN: EventBus-Based Communication
//
// EventBus is the most stable and principled pattern for service communication:
// - Decoupling: Services communicate via addresses, not direct references
// - Location transparency: Can be local or distributed (NATS/clustered)
// - Reactor isolation: Handlers execute on event loops
// - No shared mutable state: Communication via immutable messages
// - Backpressure handling: Built-in queue management
// - Testability: Easy to mock/test
// - Scalability: Can distribute without code changes
//
// Direct injection (constructor/field injection) is acceptable for:
// - Simple, single-process applications
// - Tightly coupled components that never need distribution
// - Legacy code migration
// However, EventBus is preferred for production, scalable systems.
//
// See EVENTBUS_SERVICE_PATTERN.md for detailed guidance.
//
// Usage Example (Override doHandleRequest):
//
//	type UserService struct {
//		*core.BaseService
//		users map[string]map[string]interface{}
//	}
//
//	func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
//		request := msg.Body().(map[string]interface{})
//		userID := request["userID"].(string)
//		user, exists := s.users[userID]
//		if !exists {
//			return s.Fail(msg, 404, "User not found")
//		}
//		return s.Reply(msg, user)
//	}
//
//	func (s *UserService) doStart(ctx core.FluxorContext) error {
//		s.users = make(map[string]map[string]interface{})
//		return nil
//	}
//
//	service := &UserService{
//		BaseService: core.NewBaseService("user-service", "user.service"),
//	}
//	gocmd.DeployVerticle(service)
//
// Usage Example (SetRequestHandler):
//
//	service := core.NewBaseService("calc-service", "calculator.service")
//	service.SetRequestHandler(func(ctx core.FluxorContext, msg core.Message) error {
//		request := msg.Body().(map[string]interface{})
//		result := request["a"].(float64) + request["b"].(float64)
//		return service.Reply(msg, map[string]interface{}{"result": result})
//	})
//	gocmd.DeployVerticle(service)
//
// Calling the service from a verticle:
//
//	type APIVerticle struct {
//		*core.BaseVerticle
//	}
//
//	func (v *APIVerticle) doStart(ctx core.FluxorContext) error {
//		consumer := v.Consumer("api.request")
//		consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
//			// Call the service using Request (request-reply pattern)
//			reply, err := v.Request("user.service", map[string]interface{}{
//				"userID": "123",
//			}, 5 * time.Second)
//			if err != nil {
//				return msg.Fail(502, "Service unavailable: "+err.Error())
//			}
//			var user map[string]interface{}
//			reply.DecodeBody(&user)
//			return msg.Reply(user)
//		})
//		return nil
//	}
//
// Startx Pattern: I/O-Bound Verticle + CPU-Bound Service
//
// Startx is a common pattern where the verticle is I/O-bound (HTTP, database)
// and the service is CPU-bound (heavy computations using WorkerPool):
//
//	// CPU-Bound Service (uses WorkerPool)
//	type ImageService struct {
//		*core.BaseService
//	}
//
//	func (s *ImageService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
//		request := msg.Body().(map[string]interface{})
//		imageData := request["image"].([]byte)
//
//		// CPU-bound work: Use WorkerPool (ExecuteBlocking)
//		result, err := ctx.GoCMD().ExecuteBlocking(func() (interface{}, error) {
//			return processImageCPUIntensive(imageData), nil
//		}, 30*time.Second)
//		if err != nil {
//			return s.Fail(msg, 500, "Processing failed: "+err.Error())
//		}
//		return s.Reply(msg, result)
//	}
//
//	// I/O-Bound Verticle (handles HTTP, database)
//	type APIVerticle struct {
//		*core.BaseVerticle
//		imageServiceAddress string
//	}
//
//	func (v *APIVerticle) doStart(ctx core.FluxorContext) error {
//		v.imageServiceAddress = "image.service"
//		consumer := v.Consumer("api.image.process")
//		consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
//			// Call CPU-bound service via EventBus (non-blocking)
//			reply, err := v.Request(v.imageServiceAddress, msg.Body(), 30*time.Second)
//			if err != nil {
//				return msg.Fail(502, "Service unavailable: "+err.Error())
//			}
//			return msg.Reply(reply.Body())
//		})
//		return nil
//	}
//
//	// Startx: Deploy both
//	func Startx(gocmd core.GoCMD) error {
//		imageService := &ImageService{
//			BaseService: core.NewBaseService("image-service", "image.service"),
//		}
//		gocmd.DeployVerticle(imageService)
//
//		apiVerticle := &APIVerticle{
//			BaseVerticle: core.NewBaseVerticle("api-verticle"),
//		}
//		gocmd.DeployVerticle(apiVerticle)
//		return nil
//	}
type BaseService struct {
	*BaseVerticle

	// Service address (where this service listens)
	address string

	// Request handler
	requestHandler MessageHandler
}

// NewBaseService creates a new BaseService
func NewBaseService(name, address string) *BaseService {
	// Fail-fast: validate address
	if err := ValidateAddress(address); err != nil {
		failfast.Err(err)
	}
	return &BaseService{
		BaseVerticle: NewBaseVerticle(name),
		address:      address,
	}
}

// doStart overrides BaseVerticle.doStart to register service handler
func (bs *BaseService) doStart(ctx FluxorContext) error {
	// Register service handler
	consumer := bs.Consumer(bs.address)
	consumer.Handler(bs.handleRequest)

	return nil
}

// handleRequest handles incoming service requests
func (bs *BaseService) handleRequest(ctx FluxorContext, msg Message) error {
	// Fail-fast: message cannot be nil
	failfast.NotNil(msg, "message")
	// If custom handler is set, use it
	if bs.requestHandler != nil {
		return bs.requestHandler(ctx, msg)
	}

	// Otherwise, call hook method
	return bs.doHandleRequest(ctx, msg)
}

// doHandleRequest is a hook method for subclasses to implement
// Default implementation returns error
func (bs *BaseService) doHandleRequest(ctx FluxorContext, msg Message) error {
	return &EventBusError{Code: "NOT_IMPLEMENTED", Message: "doHandleRequest must be implemented by subclass"}
}

// SetRequestHandler sets a custom request handler
func (bs *BaseService) SetRequestHandler(handler MessageHandler) {
	// Fail-fast: handler cannot be nil
	failfast.NotNil(handler, "handler")
	bs.requestHandler = handler
}

// Address returns the service address
func (bs *BaseService) Address() string {
	return bs.address
}

// Reply is a convenience method to reply to service requests
func (bs *BaseService) Reply(msg Message, body interface{}) error {
	return msg.Reply(body)
}

// Fail is a convenience method to fail service requests
func (bs *BaseService) Fail(msg Message, code int, message string) error {
	return msg.Fail(code, message)
}
