package entrypoint

import (
	"context"
)

// WorkflowBootstrapHook returns a BootstrapHook that runs the given Workflow during
// MainVerticle bootstrap. The workflow runs before GoCMD/EventBus creation.
//
// On success it returns a no-op cleanup and nil result. On failure it returns
// (nil, nil, err) so MainVerticle fails startup.
//
// Example:
//
//	workflow := entrypoint.NewWorkflow("bootstrap", step1, step2)
//	opts := entrypoint.MainVerticleOptions{
//	    BootstrapHook: entrypoint.WorkflowBootstrapHook(workflow),
//	}
//	app, err := entrypoint.NewMainVerticleWithOptions("config.json", opts)
func WorkflowBootstrapHook(workflow Workflow) func(context.Context, map[string]any) (func() error, map[string]any, error) {
	return func(ctx context.Context, cfg map[string]any) (func() error, map[string]any, error) {
		if err := workflow.Execute(ctx); err != nil {
			return nil, nil, err
		}
		return func() error { return nil }, nil, nil
	}
}
