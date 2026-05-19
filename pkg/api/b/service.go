package b

// Service orchestrates logic and external dependencies for module b.
type Service struct {
	Logic LogicDoer
}

// Ensure Service implements ServiceProcessor.
var _ ServiceProcessor = (*Service)(nil)

// NewService creates a service with default logic.
func NewService() *Service {
	return &Service{Logic: &Logic{}}
}

// NewServiceWithLogic creates a service with injected logic (for testing or DI).
func NewServiceWithLogic(logic LogicDoer) *Service {
	return &Service{Logic: logic}
}

// Process delegates to logic and returns result.
func (s *Service) Process(e *Entity) (*Entity, error) {
	return s.Logic.Do(e)
}
