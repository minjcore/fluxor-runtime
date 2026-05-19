package webfast

import (
	"time"

	"github.com/fluxorio/fluxor/pkg/lite/core"
	"github.com/valyala/fasthttp"
)

// FastHTTPVerticle runs a fasthttp server (optimized for high RPS).
type FastHTTPVerticle struct {
	addr   string
	router *Router
	server *fasthttp.Server
}

func NewFastHTTPVerticle(addr string, router *Router) *FastHTTPVerticle {
	return &FastHTTPVerticle{addr: addr, router: router}
}

func (v *FastHTTPVerticle) OnStart(ctx *core.FluxorContext) error {
	ctx.Log().Info("FastHTTPVerticle listening", "addr", v.addr)

	// Bind runtime context into router so handlers can access Bus/Worker/Log.
	v.router.Bind(ctx)

	v.server = &fasthttp.Server{
		Handler: v.router.Handler(),

		// Throughput-oriented defaults.
		Name:                          "fluxor-litefast",
		NoDefaultServerHeader:         true,
		DisableHeaderNamesNormalizing: true,
		ReduceMemoryUsage:             true,
		ReadTimeout:                   5 * time.Second,
		WriteTimeout:                  5 * time.Second,
		IdleTimeout:                   30 * time.Second,
	}

	// ListenAndServe is blocking; run async like a Verticle.
	go func() {
		if err := v.server.ListenAndServe(v.addr); err != nil {
			// fasthttp returns non-nil on shutdown as well; log best-effort.
			ctx.Log().Error("fasthttp server stopped", "err", err)
		}
	}()

	return nil
}

func (v *FastHTTPVerticle) OnStop() error {
	if v.server == nil {
		return nil
	}
	return v.server.Shutdown()
}
