package core

import (
	"sync"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// BaseRequestContext provides a Java-style abstract base class for request contexts
// It provides common data storage functionality with thread safety
type BaseRequestContext struct {
	// State management
	mu sync.RWMutex

	// Data storage for key-value pairs
	data map[string]interface{}
}

// NewBaseRequestContext creates a new BaseRequestContext
func NewBaseRequestContext() *BaseRequestContext {
	return &BaseRequestContext{
		data: make(map[string]interface{}),
	}
}

// Set stores a value in the context
func (brc *BaseRequestContext) Set(key string, value interface{}) {
	failfast.NotEmpty(key, "key")
	brc.mu.Lock()
	defer brc.mu.Unlock()
	if brc.data == nil {
		brc.data = make(map[string]interface{})
	}
	brc.data[key] = value
}

// Get retrieves a value from the context
func (brc *BaseRequestContext) Get(key string) interface{} {
	brc.mu.RLock()
	defer brc.mu.RUnlock()
	if brc.data == nil {
		return nil
	}
	return brc.data[key]
}

// GetAll returns all stored data (for debugging/logging)
func (brc *BaseRequestContext) GetAll() map[string]interface{} {
	brc.mu.RLock()
	defer brc.mu.RUnlock()
	if brc.data == nil {
		return make(map[string]interface{})
	}
	// Return a copy to prevent external modification
	result := make(map[string]interface{})
	for k, v := range brc.data {
		result[k] = v
	}
	return result
}

// Delete removes a value from the context
func (brc *BaseRequestContext) Delete(key string) {
	brc.mu.Lock()
	defer brc.mu.Unlock()
	if brc.data != nil {
		delete(brc.data, key)
	}
}

// Clear removes all data from the context
func (brc *BaseRequestContext) Clear() {
	brc.mu.Lock()
	defer brc.mu.Unlock()
	brc.data = make(map[string]interface{})
}
