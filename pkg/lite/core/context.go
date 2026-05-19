package core

import (
	"context"
	"log/slog"
)

type FluxorContext struct {
	id     string
	bus    *Bus
	worker *WorkerPool
	stdCtx context.Context
	logger *slog.Logger
}

func NewFluxorContext(stdCtx context.Context, bus *Bus, wp *WorkerPool, id string) *FluxorContext {
	if stdCtx == nil {
		stdCtx = context.Background()
	}
	return &FluxorContext{
		id:     id,
		bus:    bus,
		worker: wp,
		stdCtx: stdCtx,
		logger: slog.Default().With("component_id", id),
	}
}

func (c *FluxorContext) ID() string           { return c.id }
func (c *FluxorContext) Bus() *Bus            { return c.bus }
func (c *FluxorContext) Worker() *WorkerPool  { return c.worker }
func (c *FluxorContext) Log() *slog.Logger    { return c.logger }
func (c *FluxorContext) Ctx() context.Context { return c.stdCtx }
