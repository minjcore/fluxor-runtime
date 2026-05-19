package a

// Logic contains business/use-case logic for module a.
type Logic struct{}

// Ensure Logic implements LogicDoer.
var _ LogicDoer = (*Logic)(nil)

// Do applies business rules and returns result.
func (l *Logic) Do(e *Entity) (*Entity, error) {
	if e == nil {
		return nil, nil
	}
	return e, nil
}
