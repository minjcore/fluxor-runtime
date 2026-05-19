package a

// LogicDoer defines business/use-case logic for module a.
type LogicDoer interface {
	Do(e *Entity) (*Entity, error)
}

// ServiceProcessor orchestrates logic and returns result for module a.
type ServiceProcessor interface {
	Process(e *Entity) (*Entity, error)
}

// ControllerHandler is the input/HTTP layer for module a (implementations call ServiceProcessor).
type ControllerHandler interface {
	Service() ServiceProcessor
}
