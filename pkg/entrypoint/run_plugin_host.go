package entrypoint

import (
	"fmt"
	"os"
)

// RunFromPluginSO loads a Verticle from a Go plugin (.so on Linux, .dylib often used on macOS
// though many builds still use .so naming), deploys it on a fresh MainVerticle with empty config,
// and blocks until SIGINT/SIGTERM (same lifecycle as MainVerticle.Start).
//
// An admin socket is opened at socketPath (default: "/tmp/fluxor-<pid>.sock") so additional
// verticles can be hot-deployed with `fluxorctl deploy --target <socket> plugin.so`.
func RunFromPluginSO(pluginPath string, socketPath ...string) error {
	v, err := LoadVerticleFromPlugin(pluginPath)
	if err != nil {
		return err
	}

	sock := fmt.Sprintf("/tmp/fluxor-%d.sock", os.Getpid())
	if len(socketPath) > 0 && socketPath[0] != "" {
		sock = socketPath[0]
	}

	m, err := NewMainVerticleWithOptions("", WithOptions(MainVerticleOptions{
		AdminSocketPath: sock,
	}))
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	if _, err := m.DeployVerticle(v); err != nil {
		_ = m.Stop()
		return fmt.Errorf("deploy: %w", err)
	}
	fmt.Printf("[fluxor] admin socket: %s\n", sock)
	return m.Start()
}
