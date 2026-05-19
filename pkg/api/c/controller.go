package c

// Controller handles HTTP/input for module c and delegates to service.
type Controller struct {
	svc ServiceProcessor
}

// Ensure Controller implements ControllerHandler.
var _ ControllerHandler = (*Controller)(nil)

// NewController creates a controller with default service.
func NewController() *Controller {
	return &Controller{svc: NewService()}
}

// NewControllerWithService creates a controller with injected service (for testing or DI).
func NewControllerWithService(svc ServiceProcessor) *Controller {
	return &Controller{svc: svc}
}

// Service returns the service processor (for handlers that need it).
func (c *Controller) Service() ServiceProcessor {
	return c.svc
}
