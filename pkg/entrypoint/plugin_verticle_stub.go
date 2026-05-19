//go:build !(linux || darwin)

package entrypoint

import (
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core"
)

// PluginVerticleSymbol is documented on unix; kept for API symmetry.
const PluginVerticleSymbol = "NewVerticle"

// LoadVerticleFromPlugin is not available on this GOOS (Go plugins require linux or darwin).
func LoadVerticleFromPlugin(pluginPath string) (core.Verticle, error) {
	return nil, fmt.Errorf("Go plugin verticles (.so/.dylib) are only supported on linux and darwin (got path %q)", pluginPath)
}
