package entrypoint

import (
	"context"
	"fmt"
	"os"
	"time"

	natssrv "github.com/nats-io/nats-server/v2/server"
)

// StartEmbeddedNATS starts an embedded NATS server with JetStream enabled.
// This is a reusable BootstrapHook function that can be used in MainVerticleOptions.
//
// It returns:
//   - cleanup: function to shutdown the NATS server (called during app.Stop())
//   - result: map with "nats_url" key containing the server URL
//   - error: if server fails to start
//
// The server port can be configured via NATS_PORT environment variable.
// If not set, a random port is used.
func StartEmbeddedNATS(ctx context.Context, cfg map[string]any) (cleanup func() error, result map[string]any, err error) {
	// Use random port (-1) or check for NATS_PORT env var
	port := -1
	if portStr := os.Getenv("NATS_PORT"); portStr != "" {
		var p int
		if _, err := fmt.Sscanf(portStr, "%d", &p); err == nil && p > 0 {
			port = p
		}
	}

	opts := &natssrv.Options{
		Port:      port,
		Host:      "127.0.0.1",
		JetStream: true, // Enable JetStream for ClusterEventBusJetStream
	}

	server, err := natssrv.NewServer(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create NATS server: %w", err)
	}

	// Start server in goroutine
	go server.Start()

	// Wait for server to be ready
	if !server.ReadyForConnections(10 * time.Second) {
		// Wrap shutdown in recover to handle potential panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Ignore panic during failed startup shutdown
				}
			}()
			server.Shutdown()
		}()
		return nil, nil, fmt.Errorf("NATS server failed to start within timeout")
	}

	// Get the actual URL (handles random port assignment)
	url := server.ClientURL()

	// Return cleanup function and result map
	cleanupFn := func() error {
		// Wrap shutdown in recover to handle potential panic from NATS server
		// (known issue: shutdownEventing can panic if eventing channel is nil)
		defer func() {
			if r := recover(); r != nil {
				// Log but don't propagate panic - shutdown should be best-effort
				// The panic is from NATS server internal bug, not our code
			}
		}()
		server.Shutdown()
		return nil
	}

	resultMap := map[string]any{
		"nats_url": url,
	}

	return cleanupFn, resultMap, nil
}
