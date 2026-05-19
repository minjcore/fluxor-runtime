package core

import (
	"sync"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// BaseRouter provides a Java-style abstract base class for routers
// It provides common router functionality and thread safety
type BaseRouter struct {
	// Name of the router (can be set by subclasses)
	name string

	// State management
	mu sync.RWMutex
}

// NewBaseRouter creates a new BaseRouter
func NewBaseRouter(name string) *BaseRouter {
	return &BaseRouter{
		name: name,
	}
}

// Name returns the router name
func (br *BaseRouter) Name() string {
	br.mu.RLock()
	defer br.mu.RUnlock()
	return br.name
}

// SetName sets the router name
func (br *BaseRouter) SetName(name string) {
	failfast.NotEmpty(name, "router name")
	br.mu.Lock()
	defer br.mu.Unlock()
	br.name = name
}
