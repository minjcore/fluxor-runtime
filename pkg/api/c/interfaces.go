package c

// LogicDoer defines business/use-case logic for module c.
type LogicDoer interface {
	Do(e *Entity) (*Entity, error)
}

// ServiceProcessor orchestrates logic and returns result for module c.
type ServiceProcessor interface {
	Process(e *Entity) (*Entity, error)
}

// ControllerHandler is the input/HTTP layer for module c (implementations call ServiceProcessor).
type ControllerHandler interface {
	Service() ServiceProcessor
}
