// Package entrypoint re-exports supervisor and verticle types from core so apps
// can use the entrypoint API without importing pkg/core (e.g. for fx or other DI).
package entrypoint

import (
	"github.com/fluxorio/fluxor/pkg/core"
)

// Verticle is the unit of deployment. Re-exported from core so apps use entrypoint.Verticle.
type Verticle = core.Verticle

// SupervisorSpec defines supervisor configuration. Re-exported from core.
type SupervisorSpec = core.SupervisorSpec

// ChildSpec defines a child verticle specification. Re-exported from core.
type ChildSpec = core.ChildSpec

// RestartStrategy defines how a supervisor restarts children. Re-exported from core.
type RestartStrategy = core.RestartStrategy

// RestartConfig defines restart frequency limits. Re-exported from core.
type RestartConfig = core.RestartConfig

// RestartType defines when a child should be restarted. Re-exported from core.
type RestartType = core.RestartType

// Supervisor restart strategies and restart types (re-exported from core).
const (
	RestartStrategyOneForOne   = core.RestartStrategyOneForOne
	RestartStrategyOneForAll   = core.RestartStrategyOneForAll
	RestartStrategyRestForOne  = core.RestartStrategyRestForOne
	RestartPermanent  RestartType = core.RestartPermanent
	RestartTransient   RestartType = core.RestartTransient
	RestartTemporary   RestartType = core.RestartTemporary
)

// NewSupervisor creates a supervisor verticle from the given spec. Apps can deploy it
// via MainVerticle.DeployVerticle without importing pkg/core.
func NewSupervisor(spec SupervisorSpec) Verticle {
	return core.NewSupervisor(spec)
}
