package core

import (
	"context"
	"time"
)

// FluxorContext represents the execution context for a verticle or handler.
//
// This is distinct from context.Context (Go's standard context):
//   - context.Context: Go's cancellation/deadline/value propagation
//   - FluxorContext: Fluxor's runtime context with access to GoCMD, EventBus, Config
//
// FluxorContext wraps a context.Context and provides additional Fluxor-specific
// functionality. Use Context() to get the underlying context.Context when needed
// for cancellation or passing to Go standard library functions.
type FluxorContext interface {
	// Context returns the underlying context.Context (Go's standard context)
	Context() context.Context

	// EventBus returns the event bus instance
	EventBus() EventBus

	// GoCMD returns the GoCMD instance (kept as GoCMD for backward compatibility)
	GoCMD() GoCMD

	// Config returns the configuration map
	Config() map[string]interface{}

	// SetConfig sets a configuration value
	SetConfig(key string, value interface{})

	// Deploy deploys a verticle
	Deploy(verticle Verticle) (string, error)

	// Undeploy undeploys a verticle by deployment ID
	Undeploy(deploymentID string) error

	// DeploymentID returns the current deployment's ID when called from within a verticle
	// (e.g. during Start or in a handler). Returns empty string if not in a deployment context.
	DeploymentID() string
}

// gocmdContext implements FluxorContext
type gocmdContext struct {
	goCtx        context.Context // renamed from 'ctx' for clarity: this is Go's context.Context
	gocmd        GoCMD
	config       map[string]interface{}
	deploymentID string // set when this context is for a specific deployment (e.g. during Start)
}

// NewFluxorContext creates a FluxorContext from Go context.Context and GoCMD.
// This is exported for use by cluster eventbus implementations.
func NewFluxorContext(goCtx context.Context, gocmd GoCMD) FluxorContext {
	if goCtx == nil {
		// Fail-fast: context cannot be nil
		panic("context cannot be nil")
	}
	return &gocmdContext{
		goCtx:  goCtx,
		gocmd:  gocmd,
		config: make(map[string]interface{}),
	}
}

// newFluxorContext is an alias for NewFluxorContext for backward compatibility
// within the core package.
func newFluxorContext(goCtx context.Context, gocmd GoCMD) FluxorContext {
	return NewFluxorContext(goCtx, gocmd)
}

// NewFluxorContextWithDeploymentID creates a FluxorContext with the given deployment ID.
// Used when starting a verticle so the verticle can call DeploymentID() and use Link/Monitor.
func NewFluxorContextWithDeploymentID(goCtx context.Context, gocmd GoCMD, deploymentID string) FluxorContext {
	if goCtx == nil {
		panic("context cannot be nil")
	}
	return &gocmdContext{
		goCtx:        goCtx,
		gocmd:        gocmd,
		config:       make(map[string]interface{}),
		deploymentID: deploymentID,
	}
}

// newFluxorContextWithDeploymentID is used by the core package when deploying a verticle.
func newFluxorContextWithDeploymentID(goCtx context.Context, gocmd GoCMD, deploymentID string) FluxorContext {
	return NewFluxorContextWithDeploymentID(goCtx, gocmd, deploymentID)
}

func (c *gocmdContext) DeploymentID() string {
	return c.deploymentID
}

// Context returns the underlying context.Context (Go's standard context)
func (c *gocmdContext) Context() context.Context {
	return c.goCtx
}

func (c *gocmdContext) EventBus() EventBus {
	if c.gocmd == nil {
		// Fail-fast: gocmd is nil
		panic("gocmd is nil, cannot get EventBus")
	}
	return c.gocmd.EventBus()
}

func (c *gocmdContext) GoCMD() GoCMD {
	return c.gocmd
}

func (c *gocmdContext) Config() map[string]interface{} {
	return c.config
}

func (c *gocmdContext) SetConfig(key string, value interface{}) {
	c.config[key] = value
}

func (c *gocmdContext) Deploy(verticle Verticle) (string, error) {
	return c.gocmd.DeployVerticle(verticle)
}

// DeployService deploys a service (convenience method)
func (c *gocmdContext) DeployService(service *BaseService) (string, error) {
	return c.gocmd.DeployService(service)
}

func (c *gocmdContext) Undeploy(deploymentID string) error {
	return c.gocmd.UndeployVerticle(deploymentID)
}

// GetTimezone returns the timezone location from the context config
// Returns UTC if timezone is not set in context
func GetTimezone(ctx FluxorContext) *time.Location {
	if ctx == nil {
		return time.UTC
	}
	config := ctx.Config()
	if config == nil {
		return time.UTC
	}

	// Try to get timezone location directly
	if location, ok := config["timezone.location"].(*time.Location); ok && location != nil {
		return location
	}

	// Try to get timezone string and parse it
	if tzStr, ok := config["timezone"].(string); ok && tzStr != "" {
		return ParseTimezone(tzStr)
	}

	// Try logger.timezone as fallback
	if tzStr, ok := config["logger.timezone"].(string); ok && tzStr != "" {
		return ParseTimezone(tzStr)
	}

	return time.UTC
}

// SetTimezone sets the timezone in the context config
// Stores both the timezone string and parsed location for efficient access
// Falls back to UTC if timezone is empty or invalid
func SetTimezone(ctx FluxorContext, timezone string) {
	if ctx == nil {
		return
	}
	// Fallback to UTC if timezone is empty
	if timezone == "" {
		timezone = "UTC"
	}
	location := ParseTimezone(timezone)
	ctx.SetConfig("timezone", timezone)
	ctx.SetConfig("timezone.location", location)
	// Also set logger.timezone for backward compatibility
	ctx.SetConfig("logger.timezone", timezone)
}
