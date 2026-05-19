//go:build linux || darwin

package entrypoint

import (
	"fmt"
	"path/filepath"
	"plugin"

	"github.com/fluxorio/fluxor/pkg/core"
)

// PluginVerticleSymbol is the exported symbol name plugins must provide:
//
//	func NewVerticle() core.Verticle
//
// Build the plugin with:
//
//	go build -buildmode=plugin -o myverticle.so ./path/to/plugin/package
//
// The plugin module must use the same major path/version of github.com/fluxorio/fluxor/pkg/core
// as this host binary (replace directive in go.mod is typical for local dev).
const PluginVerticleSymbol = "NewVerticle"

// LoadVerticleFromPlugin opens a shared object built with -buildmode=plugin and
// returns the Verticle from the exported NewVerticle function.
func LoadVerticleFromPlugin(pluginPath string) (core.Verticle, error) {
	abs, err := filepath.Abs(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("plugin path: %w", err)
	}
	p, err := plugin.Open(abs)
	if err != nil {
		return nil, fmt.Errorf("plugin.Open(%q): %w", abs, err)
	}
	sym, err := p.Lookup(PluginVerticleSymbol)
	if err != nil {
		return nil, fmt.Errorf("plugin.Lookup(%q): %w; export func NewVerticle() core.Verticle in the plugin", PluginVerticleSymbol, err)
	}
	newFn, ok := sym.(func() core.Verticle)
	if !ok {
		return nil, fmt.Errorf("symbol %q has wrong type %T; want func() core.Verticle", PluginVerticleSymbol, sym)
	}
	v := newFn()
	if v == nil {
		return nil, fmt.Errorf("%s returned nil Verticle", PluginVerticleSymbol)
	}
	return v, nil
}
