package managers

import (
	"errors"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// httpServerWrapper wraps a web.Server to send signals to Managers on start/stop
// Car analogy: Wraps engine to send signals to Managers when engine starts/stops
type httpServerWrapper struct {
	web.Server
	managers *Managers
}

// Start wraps the server Start() method to send signal to Managers
func (w *httpServerWrapper) Start() error {
	// Signal Managers that HTTP server is starting (before blocking call)
	// Note: Start() will block on ListenAndServe(), so we signal immediately
	w.managers.signalHTTPServerStarted()

	// Call the underlying Start() which blocks until server stops
	err := w.Server.Start()
	return err
}

// Stop wraps the server Stop() method to send signal to Managers
func (w *httpServerWrapper) Stop() error {
	err := w.Server.Stop()
	if err == nil {
		// Signal Managers that HTTP server stopped
		w.managers.signalHTTPServerStopped()
	}
	return err
}

// CreateHTTPServer creates an HTTP server and wraps it to send signals to Managers
func (m *Managers) CreateHTTPServer(gocmd core.GoCMD) (web.Server, error) {
	if gocmd == nil {
		return nil, errors.New("gocmd cannot be nil")
	}

	m.mu.RLock()
	addr := m.config.HTTPAddr
	m.mu.RUnlock()

	server := web.NewServer(gocmd, addr)

	// Wrap server to send signals to Managers
	wrapped := &httpServerWrapper{
		Server:   server,
		managers: m,
	}

	return wrapped, nil
}

// CreateFastHTTPServer creates a FastHTTP server
// Note: FastHTTPServer is a concrete type that doesn't implement web.Server interface directly
// For signal support, applications should manually call managers signal methods when starting/stopping
// or register the server and handle signals externally
func (m *Managers) CreateFastHTTPServer(gocmd core.GoCMD, config *web.FastHTTPServerConfig) (*web.FastHTTPServer, error) {
	if gocmd == nil {
		return nil, errors.New("gocmd cannot be nil")
	}

	if config == nil {
		m.mu.RLock()
		addr := m.config.HTTPAddr
		m.mu.RUnlock()
		config = web.DefaultFastHTTPServerConfig(addr)
	}

	server := web.NewFastHTTPServer(gocmd, config)

	// Note: FastHTTPServer signals would need to be handled manually or via wrapper
	// For now, return the server directly
	// Applications should call managers.signalHTTPServerStarted()/Stopped() manually
	// or use a wrapper verticle that handles signals

	return server, nil
}
