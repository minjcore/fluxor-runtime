// subprocess-host-demo — parent process that manages subprocess verticles.
// Starts embedded NATS, opens admin socket, then waits for fluxorctl spawn commands.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/cluster/eventbus"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
)

func main() {
	fmt.Println("subprocess-host-demo: starting with embedded NATS + SubprocessManager")
	fmt.Println("Admin socket: /tmp/fluxor-subproc.sock")

	app, err := entrypoint.NewMainVerticleWithOptions("",
		entrypoint.WithOptions(entrypoint.MainVerticleOptions{
			BootstrapHook:           entrypoint.StartEmbeddedNATS,
			EnableSubprocessManager: true,
			AdminSocketPath:         "/tmp/fluxor-subproc.sock",
			EventBusFactory: func(ctx context.Context, gocmd core.GoCMD, cfg map[string]any) (core.EventBus, error) {
				natsURL, _ := cfg["nats_url"].(string)
				return eventbus.NewClusterEventBusNATS(ctx, gocmd, eventbus.ClusterNATSConfig{
					URL: natsURL,
				})
			},
		}),
	)
	if err != nil {
		log.Fatalf("init: %v", err)
	}

	if err := app.Start(); err != nil {
		log.Fatalf("start: %v", err)
	}
}
