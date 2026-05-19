package web

import (
	"net/http"
	"time"

	"github.com/fluxorio/fluxor/pkg/lite/core"
	"github.com/fluxorio/fluxor/pkg/lite/fx"
)

type HttpVerticle struct {
	port   string
	router *Router
	server *http.Server
}

func NewHttpVerticle(port string, router *Router) *HttpVerticle {
	return &HttpVerticle{port: port, router: router}
}

func (v *HttpVerticle) OnStart(ctx *core.FluxorContext) error {
	ctx.Log().Info("HttpVerticle listening", "port", v.port)

	handler := func(w http.ResponseWriter, r *http.Request) {
		c := fx.NewContext(w, r, ctx)
		if err := v.router.Handle(c); err != nil {
			ctx.Log().Error("handler error", "err", err)
		}
	}

	v.server = &http.Server{
		Addr:              ":" + v.port,
		Handler:           http.HandlerFunc(handler),
		ReadHeaderTimeout: 5 * time.Second, // mitigate Slowloris
	}

	go func() {
		if err := v.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ctx.Log().Error("server crashed", "err", err)
		}
	}()

	return nil
}

func (v *HttpVerticle) OnStop() error {
	if v.server != nil {
		return v.server.Close()
	}
	return nil
}
