package managers

import (
	"errors"
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core"
)

// GetManagers retrieves Managers from FluxorContext
// Car analogy: Car part accesses Managers through car wiring (FluxorContext)
func GetManagers(ctx core.FluxorContext) (*Managers, error) {
	if ctx == nil {
		return nil, errors.New("fluxor context cannot be nil")
	}

	config := ctx.Config()
	if config == nil {
		return nil, errors.New("context config is nil")
	}

	managersValue, exists := config[ManagersKey]
	if !exists {
		return nil, fmt.Errorf("Managers not found in context (key: %s)", ManagersKey)
	}

	managers, ok := managersValue.(*Managers)
	if !ok {
		return nil, fmt.Errorf("Managers value in context is not *Managers type, got %T", managersValue)
	}

	return managers, nil
}

// WithManagers stores Managers reference in FluxorContext config
// Car analogy: Stores Managers reference in car wiring
func WithManagers(ctx core.FluxorContext, managers *Managers) {
	if ctx == nil || managers == nil {
		return
	}
	ctx.SetConfig(ManagersKey, managers)
}

// AttachToContext attaches Managers to a FluxorContext
// This is a convenience method that stores Managers in context config
func (m *Managers) AttachToContext(ctx core.FluxorContext) {
	WithManagers(ctx, m)
}
