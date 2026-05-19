package web

import (
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fx"
	"net/http"
	"time"
)

// HttpVerticle is a verticle that runs an HTTP server
// Extends BaseVerticle for common lifecycle management
type HttpVerticle struct {
	*core.BaseVerticle // Embed base verticle for lifecycle management
	port               string
	router             Router
	server             *http.Server
}

func NewHttpVerticle(port string, router Router) *HttpVerticle {
	return &HttpVerticle{
		BaseVerticle: core.NewBaseVerticle("http-verticle"),
		port:         port,
		router:       router,
	}
}

// doStart is called by BaseVerticle.Start() - implements hook method
func (v *HttpVerticle) doStart(ctx core.FluxorContext) error {
	logger := core.NewDefaultLogger()
	logger.Info("HttpVerticle Listening", "port", v.port)

	handler := func(w http.ResponseWriter, r *http.Request) {
		c := fx.NewContext(w, r, ctx)
		// Router interface should implement http.Handler
		if httpHandler, ok := v.router.(http.Handler); ok {
			httpHandler.ServeHTTP(c.W, c.R)
		}
	}

	v.server = &http.Server{
		Addr:              ":" + v.port,
		Handler:           http.HandlerFunc(handler),
		ReadHeaderTimeout: 5 * time.Second, // mitigate Slowloris
	}

	go func() {
		if err := v.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server crashed", "err", err)
		}
	}()
	return nil
}

// doStop is called by BaseVerticle.Stop() - implements hook method
func (v *HttpVerticle) doStop(ctx core.FluxorContext) error {
	if v.server != nil {
		return v.server.Close()
	}
	return nil
}
