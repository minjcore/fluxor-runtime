// Package main builds as a Go plugin for fluxor-cli / fluxorctl dynamic load.
//
// From repo root:
//
//	go build -buildmode=plugin -o /tmp/demo_verticle.so ./cmd/demo-plugin-verticle/
package main

import (
	"log"

	"github.com/fluxorio/fluxor/pkg/core"
)

type demoVerticle struct{}

func (demoVerticle) Start(ctx core.FluxorContext) error {
	log.Printf("demo-plugin-verticle started (deployment=%s)", ctx.DeploymentID())
	return nil
}

func (demoVerticle) Stop(ctx core.FluxorContext) error {
	log.Printf("demo-plugin-verticle stopped")
	return nil
}

// NewVerticle is exported for the host (see pkg/entrypoint.PluginVerticleSymbol).
func NewVerticle() core.Verticle {
	return demoVerticle{}
}
