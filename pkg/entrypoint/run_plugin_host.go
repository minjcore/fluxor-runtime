package entrypoint

import (
	"fmt"
)

// RunFromPluginSO loads a Verticle from a Go plugin (.so on Linux, .dylib often used on macOS
// though many builds still use .so naming), deploys it on a fresh MainVerticle with empty config,
// and blocks until SIGINT/SIGTERM (same lifecycle as MainVerticle.Start).
func RunFromPluginSO(pluginPath string) error {
	v, err := LoadVerticleFromPlugin(pluginPath)
	if err != nil {
		return err
	}
	m, err := NewMainVerticleWithOptions("")
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	if _, err := m.DeployVerticle(v); err != nil {
		_ = m.Stop()
		return fmt.Errorf("deploy: %w", err)
	}
	return m.Start()
}
