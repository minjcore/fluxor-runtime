package labs

import (
	"github.com/fluxorio/fluxor/pkg/core"
)

// ExampleExperiment demonstrates how to create a core-like experiment
// in labs following Fluxor patterns.
//
// This is a template/example for experimental core features.
// When stable, this would move to pkg/core.
//
// Example: This could be an experimental BaseActor or BaseStream
// that might eventually become part of core.
type ExampleExperiment struct {
	*core.BaseVerticle
	// Add your experimental core fields here
}

// NewExampleExperiment creates a new experimental core-like instance.
//
// WARNING: This is an experimental API and may change without notice.
// This is a candidate for promotion to pkg/core when stable.
func NewExampleExperiment() *ExampleExperiment {
	return &ExampleExperiment{
		BaseVerticle: core.NewBaseVerticle("example-experiment"),
	}
}

// doStart initializes the experiment.
// This follows the standard Verticle lifecycle pattern.
// When this becomes stable core, it would follow the same pattern.
func (e *ExampleExperiment) doStart(ctx core.FluxorContext) error {
	// Initialize your experimental core feature here
	// Remember: Never block the reactor!
	// Follow all Fluxor core patterns (non-blocking, event-driven, etc.)
	return nil
}

// doStop cleans up the experiment.
func (e *ExampleExperiment) doStop(ctx core.FluxorContext) error {
	// Cleanup your experimental core feature here
	return nil
}
