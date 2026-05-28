package entrypoint

import (
	"context"
	"fmt"
	"os"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/cluster/eventbus"
)

// RunSubprocess starts a verticle in subprocess mode.
// Call this from the subprocess binary's main() instead of the usual MainVerticle bootstrap.
//
// Subprocess mode: FLUXOR_NATS_URL is set by parent → connects to parent's NATS EventBus.
// Standalone mode: FLUXOR_NATS_URL is empty → runs with in-process EventBus (local dev).
//
// Example:
//
//	func main() {
//	    if err := entrypoint.RunSubprocess(NewGatewayVerticle); err != nil {
//	        log.Fatal(err)
//	    }
//	}
func RunSubprocess(newVerticle func() core.Verticle) error {
	natsURL := os.Getenv("FLUXOR_NATS_URL")

	var opts []Option
	if natsURL != "" {
		opts = append(opts, WithOptions(MainVerticleOptions{
			EventBusFactory: func(ctx context.Context, gocmd core.GoCMD, cfg map[string]any) (core.EventBus, error) {
				return eventbus.NewClusterEventBusNATS(ctx, gocmd, eventbus.ClusterNATSConfig{
					URL:  natsURL,
					Name: os.Getenv("FLUXOR_DEPLOYMENT_ID"),
				})
			},
		}))
	}

	m, err := NewMainVerticleWithOptions("", opts...)
	if err != nil {
		return fmt.Errorf("subprocess init: %w", err)
	}
	if _, err := m.DeployVerticle(newVerticle()); err != nil {
		return fmt.Errorf("subprocess deploy: %w", err)
	}
	return m.Start()
}
