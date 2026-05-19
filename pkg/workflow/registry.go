package workflow

import (
	"sync"
)

// nodeRegistry implements NodeRegistry.
type nodeRegistry struct {
	handlers map[NodeType]NodeHandler
	mu       sync.RWMutex
}

// NewNodeRegistry creates a new node registry with built-in handlers.
func NewNodeRegistry() NodeRegistry {
	r := &nodeRegistry{
		handlers: make(map[NodeType]NodeHandler),
	}
	r.registerBuiltins()
	return r
}

func (r *nodeRegistry) Register(nodeType NodeType, handler NodeHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[nodeType] = handler
}

func (r *nodeRegistry) Get(nodeType NodeType) (NodeHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[nodeType]
	return h, ok
}

func (r *nodeRegistry) registerBuiltins() {
	// Register all built-in node handlers
	r.handlers[NodeTypeNoOp] = noOpHandler
	r.handlers[NodeTypeSet] = setHandler
	r.handlers[NodeTypeVariable] = variableHandler
	r.handlers[NodeTypeCondition] = conditionHandler
	r.handlers[NodeTypeWait] = waitHandler
	r.handlers[NodeTypeError] = errorHandler
	r.handlers[NodeTypeLoop] = loopHandler
	r.handlers[NodeTypeSplit] = splitHandler
	r.handlers[NodeTypeMerge] = mergeHandler
	r.handlers[NodeTypeSwitch] = switchHandler
}
