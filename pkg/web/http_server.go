package web

import (
	"context"
	"net/http"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// httpServer implements Server
// Extends BaseServer for common lifecycle management
type httpServer struct {
	*core.BaseServer // Embed base server for lifecycle management
	router           *router
	httpServer       *http.Server
}

// NewServer creates a new HTTP server
func NewServer(gocmd core.GoCMD, addr string) Server {
	r := NewRouter().(*router)

	s := &httpServer{
		BaseServer: core.NewBaseServer("http-server", gocmd),
		router:     r,
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           r,
			ReadHeaderTimeout: 5 * time.Second, // mitigate Slowloris
		},
	}
	s.BaseServer.SetHooks(s.doStart, s.doStop)
	return s
}

// doStart is called by BaseServer.Start() - implements hook method
func (s *httpServer) doStart() error {
	// Inject GoCMD and EventBus into router handlers
	// This is done by wrapping the router's ServeHTTP

	return s.httpServer.ListenAndServe()
}

// doStop is called by BaseServer.Stop() - implements hook method
func (s *httpServer) doStop() error {
	ctx := context.Background()
	return s.httpServer.Shutdown(ctx)
}

func (s *httpServer) Router() Router {
	return s.router
}

// InjectGoCMD injects GoCMD and EventBus into request context
func (s *httpServer) InjectGoCMD(ctx *RequestContext) {
	ctx.GoCMD = s.GoCMD()
	ctx.EventBus = s.EventBus()
}
