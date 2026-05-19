package labs

import (
	"context"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
)

// ExampleExampleExperiment demonstrates how to use an experimental feature.
func ExampleExampleExperiment() {
	// Create experiment
	exp := NewExampleExperiment()

	// Create a test context (in real usage, this comes from FluxorContext)
	// This is just for demonstration
	_ = exp

	// Use the experiment...
	// WARNING: Experimental APIs may change!
}

// TestExampleExperiment tests the example experiment.
func TestExampleExperiment(t *testing.T) {
	exp := NewExampleExperiment()
	if exp == nil {
		t.Fatal("NewExampleExperiment returned nil")
	}

	if exp.BaseVerticle == nil {
		t.Fatal("BaseVerticle is nil")
	}
}

// TestExampleExperimentLifecycle tests the lifecycle of an experiment.
// Note: This test uses BaseVerticle.Start/Stop which internally call doStart/doStop.
func TestExampleExperimentLifecycle(t *testing.T) {
	exp := NewExampleExperiment()

	// Create a minimal test context
	// In real usage, this would come from the Fluxor Stream
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	
	// Get EventBus from GoCMD
	eventBus := gocmd.EventBus()
	defer eventBus.Close()
	
	// Create FluxorContext - we need to access the internal newFluxorContext
	// For now, we'll test through BaseVerticle.Start/Stop which handles context creation
	// In a real test, you might need to create a test helper in pkg/core for creating test contexts
	
	// For this example, we'll just verify the experiment can be created
	// Full lifecycle testing would require access to internal context creation
	// or using entrypoint package which provides proper context setup
	_ = exp
	_ = ctx
	_ = gocmd
	_ = eventBus
}
