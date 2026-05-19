package core

// Verticle represents a unit of deployment in Fluxor
// Similar to Vert.x verticles, these are isolated units of work
type Verticle interface {
	// Start is called when the verticle is deployed
	Start(ctx FluxorContext) error

	// Stop is called when the verticle is undeployed
	Stop(ctx FluxorContext) error
}

// AsyncVerticle represents a verticle that handles asynchronous operations
type AsyncVerticle interface {
	Verticle

	// AsyncStart is called asynchronously when the verticle is deployed
	AsyncStart(ctx FluxorContext, resultHandler func(error))

	// AsyncStop is called asynchronously when the verticle is undeployed
	AsyncStop(ctx FluxorContext, resultHandler func(error))
}
