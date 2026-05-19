package b

// LogicDoer defines business/use-case logic for module b.
type LogicDoer interface {
	Do(e *Entity) (*Entity, error)
}

// ServiceProcessor orchestrates logic and returns result for module b.
type ServiceProcessor interface {
	Process(e *Entity) (*Entity, error)
}

// ControllerHandler is the input/HTTP layer for module b (implementations call ServiceProcessor).
type ControllerHandler interface {
	Service() ServiceProcessor
}
