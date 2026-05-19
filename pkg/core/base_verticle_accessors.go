package core

// Name returns the verticle name
func (bv *BaseVerticle) Name() string {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.name
}

// Context returns the FluxorContext (set during Start)
func (bv *BaseVerticle) Context() FluxorContext {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.ctx
}

// EventBus returns the EventBus reference
func (bv *BaseVerticle) EventBus() EventBus {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.eventBus
}

// GoCMD returns the GoCMD reference (kept as GoCMD for backward compatibility)
func (bv *BaseVerticle) GoCMD() GoCMD {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.gocmd
}

// IsStarted returns whether the verticle has been started
func (bv *BaseVerticle) IsStarted() bool {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.started
}

// IsStopped returns whether the verticle has been stopped
func (bv *BaseVerticle) IsStopped() bool {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.stopped
}
