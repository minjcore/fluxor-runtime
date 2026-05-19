package core

// Component is the basic runtime unit (similar to a Verticle).
//
// This minimal API is intentionally dependency-free (no imports) to keep the
// dependency graph acyclic.
type Component interface {
	OnStart(ctx *FluxorContext) error
	OnStop() error
}
