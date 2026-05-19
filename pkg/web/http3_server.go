package web

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/udp"
	"github.com/quic-go/quic-go/http3"
)

// HTTP3Server implements Server using HTTP/3 (QUIC) for high performance
// Extends BaseServer for common lifecycle management
// HTTP/3 provides improved performance over HTTP/2, especially for high-latency connections
type HTTP3Server struct {
	*core.BaseServer // Embed base server for lifecycle management
	router           *router
	http3Server      *http3.Server
	httpServer       *http.Server // Fallback HTTP/1.1 and HTTP/2 server
	addr             string
	tlsConfig        *tls.Config
	enableFallback   bool // Enable HTTP/1.1 and HTTP/2 fallback

	// Metrics for monitoring
	totalRequests      int64 // Atomic counter for total requests
	http3Requests      int64 // Atomic counter for HTTP/3 requests
	fallbackRequests   int64 // Atomic counter for fallback requests
	successfulRequests int64 // Atomic counter for successful requests (200-299)
	errorRequests      int64 // Atomic counter for error requests (500-599)
	bytesSent          int64 // Atomic counter for bytes sent
	bytesReceived      int64 // Atomic counter for bytes received

	// Server lifecycle
	mu     sync.RWMutex
	stopping int32 // atomic: server stopping flag

	// UDP PacketConn (reused or created)
	packetConn net.PacketConn

	// Backpressure controller (from UDP package)
	backpressure *udp.BackpressureController

	// Additional metrics for backpressure
	rejectedRequests int64 // Atomic counter for rejected requests (backpressure)
	queuedRequests   int64 // Atomic counter for queued requests
}

// HTTP3ServerConfig configures the HTTP/3 server
type HTTP3ServerConfig struct {
	// Addr is the address to listen on (e.g., ":443")
	Addr string

	// TLSConfig is the TLS configuration (required for HTTP/3)
	TLSConfig *tls.Config

	// EnableFallback enables HTTP/1.1 and HTTP/2 fallback server
	// Note: HTTP/3 uses UDP (QUIC) while HTTP/1.1/2 use TCP, so they typically
	// need different ports. For same-port support, configure different addresses
	// or use Alt-Svc header to indicate HTTP/3 availability.
	EnableFallback bool
	
	// FallbackAddr is the address for the fallback server (HTTP/1.1/2)
	// If empty, uses the same address as Addr (may cause port conflicts)
	FallbackAddr string

	// ReadTimeout is the read timeout for connections
	ReadTimeout time.Duration

	// WriteTimeout is the write timeout for connections
	WriteTimeout time.Duration

	// IdleTimeout is the idle connection timeout
	IdleTimeout time.Duration

	// MaxHeaderBytes is the maximum size of request headers
	MaxHeaderBytes int

	// ReadBufferSize is the read buffer size for QUIC connections
	ReadBufferSize int

	// WriteBufferSize is the write buffer size for QUIC connections
	WriteBufferSize int

	// PacketConn is an optional UDP PacketConn to reuse
	// If provided, HTTP/3 will use this PacketConn instead of creating a new one
	// This allows sharing UDP socket with other services (e.g., raw UDP server)
	PacketConn net.PacketConn

	// UDPServer is an optional UDP server to reuse PacketConn from
	// If provided, PacketConn will be obtained from this server
	// Note: The UDP server must be started before HTTP/3 server
	UDPServer *udp.UDPServer

	// Backpressure configuration
	// NormalCapacity is the target capacity for normal operations (e.g., 80% of max)
	// 0 means no backpressure (unlimited)
	NormalCapacity int

	// MaxRequestsPerSecond limits requests per second (0 = unlimited)
	MaxRequestsPerSecond int
}

// DefaultHTTP3ServerConfig returns default configuration for HTTP/3 server
func DefaultHTTP3ServerConfig(addr string, tlsConfig *tls.Config) *HTTP3ServerConfig {
	return &HTTP3ServerConfig{
		Addr:            addr,
		TLSConfig:       tlsConfig,
		EnableFallback:  true, // Enable fallback by default
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		MaxHeaderBytes:  1 << 20, // 1MB
		ReadBufferSize:  65536,  // 64KB
		WriteBufferSize: 65536,  // 64KB
	}
}

// NewHTTP3Server creates a new HTTP/3 server
// Fail-fast: Validates configuration
func NewHTTP3Server(gocmd core.GoCMD, config *HTTP3ServerConfig) (Server, error) {
	// Fail-fast: Validate TLS config (required for HTTP/3)
	if config.TLSConfig == nil {
		return nil, fmt.Errorf("fail-fast: TLS configuration is required for HTTP/3")
	}

	// Fail-fast: Validate address
	if config.Addr == "" {
		return nil, fmt.Errorf("fail-fast: address cannot be empty")
	}

	// Create base server
	baseServer := core.NewBaseServer("http3-server", gocmd)

	// Create router
	r := NewRouter().(*router)

	// Get or create PacketConn (may be nil if not provided)
	packetConn, err := getOrCreatePacketConn(config)
	if err != nil {
		return nil, fmt.Errorf("fail-fast: failed to get/create PacketConn: %w", err)
	}

	// Initialize backpressure if configured
	var backpressure *udp.BackpressureController
	if config.NormalCapacity > 0 {
		backpressure = udp.NewBackpressureController(config.NormalCapacity, 60)
	}

	// Create HTTP/3 server
	server := &HTTP3Server{
		BaseServer:     baseServer,
		router:         r,
		addr:           config.Addr,
		tlsConfig:      config.TLSConfig,
		enableFallback: config.EnableFallback,
		backpressure:   backpressure,
		packetConn:     packetConn, // May be nil - http3.Server will create its own
	}

	// Create HTTP/3 server
	server.http3Server = &http3.Server{
		Addr:      config.Addr,
		Handler:   server.createHandler(),
		TLSConfig: config.TLSConfig,
		// QUICConfig can be set via quic.Config if needed
		// For now, use http3.Server defaults
	}

	// Create fallback HTTP/1.1 and HTTP/2 server if enabled
	if config.EnableFallback {
		fallbackAddr := config.FallbackAddr
		if fallbackAddr == "" {
			// Use same address (may cause port conflicts - HTTP/3 uses UDP, HTTP/1.1/2 use TCP)
			// In practice, you should use different ports or configure properly
			fallbackAddr = config.Addr
		}
		server.httpServer = &http.Server{
			Addr:           fallbackAddr,
			Handler:        server.createHandler(),
			TLSConfig:      config.TLSConfig,
			ReadTimeout:    config.ReadTimeout,
			WriteTimeout:   config.WriteTimeout,
			IdleTimeout:    config.IdleTimeout,
			MaxHeaderBytes: config.MaxHeaderBytes,
		}
	}

	// Set hooks
	server.BaseServer.SetHooks(server.doStart, server.doStop)

	return server, nil
}

// createHandler creates an HTTP handler that wraps the router
func (s *HTTP3Server) createHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&s.totalRequests, 1)

		// Check backpressure before processing request
		if s.backpressure != nil {
			if !s.backpressure.TryAcquire() {
				// Capacity exceeded, reject request
				atomic.AddInt64(&s.rejectedRequests, 1)
				http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
				return
			}
			// Release backpressure when request completes
			defer s.backpressure.Release()
		}

		// Detect HTTP/3 vs fallback
		if r.ProtoMajor == 3 {
			atomic.AddInt64(&s.http3Requests, 1)
		} else {
			atomic.AddInt64(&s.fallbackRequests, 1)
		}

		// Track bytes received
		if r.ContentLength > 0 {
			atomic.AddInt64(&s.bytesReceived, r.ContentLength)
		}

		// Inject GoCMD and EventBus into request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "gocmd", s.GoCMD())
		ctx = context.WithValue(ctx, "eventbus", s.EventBus())
		r = r.WithContext(ctx)

		// Wrap response writer to track bytes sent
		wrappedWriter := &responseWriter{
			ResponseWriter: w,
			bytesSent:      &s.bytesSent,
		}

		// Create a custom router wrapper that injects GoCMD/EventBus
		s.routerWithInjection(wrappedWriter, r)

		// Track successful request (router handles errors internally)
		atomic.AddInt64(&s.successfulRequests, 1)
	})
}

// responseWriter wraps http.ResponseWriter to track bytes sent
type responseWriter struct {
	http.ResponseWriter
	bytesSent *int64
	statusCode int
	written    bool
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	n, err := rw.ResponseWriter.Write(b)
	atomic.AddInt64(rw.bytesSent, int64(n))
	return n, err
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.written = true
	rw.ResponseWriter.WriteHeader(statusCode)
}

// doStart starts the HTTP/3 server
func (s *HTTP3Server) doStart() error {
	s.logger().WithFields(map[string]interface{}{
		"addr":          s.addr,
		"fallback":      s.enableFallback,
		"http3_enabled": true,
	}).Info("HTTP/3 server starting")

	// Start fallback server in goroutine if enabled
	// Note: HTTP/3 uses UDP (QUIC) while HTTP/1.1/2 use TCP, so they need different ports
	// For same-port support, use Alt-Svc header or configure different addresses
	if s.enableFallback && s.httpServer != nil {
		go func() {
			// Use ListenAndServeTLS with empty strings since certificates are in TLSConfig
			// This works when TLSConfig.Certificates is already populated
			if err := s.httpServer.ListenAndServeTLS("", ""); err != nil {
				if atomic.LoadInt32(&s.stopping) == 0 {
					s.logger().WithFields(map[string]interface{}{
						"error": err.Error(),
					}).Error("HTTP fallback server error")
				}
			}
		}()
	}

	// Start HTTP/3 server (blocking)
	// Use PacketConn if provided, otherwise let http3.Server create its own
	if s.packetConn != nil {
		return s.http3Server.Serve(s.packetConn)
	}
	return s.http3Server.ListenAndServe()
}

// doStop stops the HTTP/3 server
func (s *HTTP3Server) doStop() error {
	// Mark as stopping
	atomic.StoreInt32(&s.stopping, 1)

	// Stop HTTP/3 server gracefully
	if s.http3Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.http3Server.Shutdown(ctx); err != nil {
			s.logger().WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("HTTP/3 server shutdown error")
		}
	}

	// Note: Don't close packetConn if it was provided by user/UDPServer
	// The owner (UDPServer or user) is responsible for closing it

	// Stop fallback server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger().WithFields(map[string]interface{}{
				"error": err.Error(),
			}).Error("HTTP fallback server shutdown error")
		}
	}

	return nil
}

// routerWithInjection wraps router.ServeHTTP and injects GoCMD/EventBus into RequestContext
func (s *HTTP3Server) routerWithInjection(w http.ResponseWriter, r *http.Request) {
	// Get GoCMD and EventBus from context
	gocmd, _ := r.Context().Value("gocmd").(core.GoCMD)
	eventbus, _ := r.Context().Value("eventbus").(core.EventBus)

	// Create a wrapper that intercepts the router's ServeHTTP
	// and injects GoCMD/EventBus into RequestContext
	wrapper := &routerWrapper{
		router:   s.router,
		gocmd:    gocmd,
		eventbus: eventbus,
	}
	wrapper.ServeHTTP(w, r)
}

// routerWrapper wraps the router to inject GoCMD and EventBus
// It duplicates the router's ServeHTTP logic to inject GoCMD/EventBus
type routerWrapper struct {
	router   *router
	gocmd    core.GoCMD
	eventbus core.EventBus
}

func (rw *routerWrapper) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rw.router.mu.RLock()
	defer rw.router.mu.RUnlock()

	for _, route := range rw.router.routes {
		if route.method == req.Method && rw.matchPath(route.path, req.URL.Path) {
			ctx := &RequestContext{
				BaseRequestContext: core.NewBaseRequestContext(),
				Context:            req.Context(),
				Request:            req,
				Response:           w,
				GoCMD:              rw.gocmd,
				EventBus:           rw.eventbus,
				Params:             rw.extractParams(route.path, req.URL.Path),
			}

			if err := route.handler(ctx); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	http.NotFound(w, req)
}

// matchPath matches a pattern against a path (duplicated from router)
func (rw *routerWrapper) matchPath(pattern, path string) bool {
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			continue // Parameter
		}
		if part != pathParts[i] {
			return false
		}
	}

	return true
}

// extractParams extracts parameters from path (duplicated from router)
func (rw *routerWrapper) extractParams(pattern, path string) map[string]string {
	params := make(map[string]string)
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			paramName := strings.TrimPrefix(part, ":")
			if i < len(pathParts) {
				params[paramName] = pathParts[i]
			}
		}
	}

	return params
}

// Router returns the router
func (s *HTTP3Server) Router() Router {
	return s.router
}

// Metrics returns server metrics
func (s *HTTP3Server) Metrics() HTTP3Metrics {
	var bpMetrics *udp.BackpressureMetrics
	var utilization float64
	var normalCapacity int64
	var currentLoad int64

	if s.backpressure != nil {
		bpMetricsVal := s.backpressure.GetMetrics()
		bpMetrics = &bpMetricsVal
		utilization = bpMetricsVal.Utilization
		normalCapacity = bpMetricsVal.NormalCapacity
		currentLoad = bpMetricsVal.CurrentLoad
	}

	return HTTP3Metrics{
		TotalRequests:       atomic.LoadInt64(&s.totalRequests),
		HTTP3Requests:       atomic.LoadInt64(&s.http3Requests),
		FallbackRequests:    atomic.LoadInt64(&s.fallbackRequests),
		SuccessfulRequests:   atomic.LoadInt64(&s.successfulRequests),
		ErrorRequests:       atomic.LoadInt64(&s.errorRequests),
		RejectedRequests:    atomic.LoadInt64(&s.rejectedRequests),
		BytesSent:           atomic.LoadInt64(&s.bytesSent),
		BytesReceived:       atomic.LoadInt64(&s.bytesReceived),
		BackpressureMetrics: bpMetrics,
		QueueUtilization:    0, // Not applicable for HTTP/3 (QUIC handles queuing)
		NormalCapacity:      normalCapacity,
		CurrentLoad:         currentLoad,
		Utilization:         utilization,
	}
}

// logger returns the server logger
func (s *HTTP3Server) logger() core.Logger {
	return s.BaseServer.Logger()
}

// HTTP3Metrics provides HTTP/3 server performance metrics
// Aligned with UDP package's ServerMetrics structure for consistency
type HTTP3Metrics struct {
	// Request metrics
	TotalRequests      int64   `json:"totalRequests"`
	HTTP3Requests      int64   `json:"http3Requests"`
	FallbackRequests   int64   `json:"fallbackRequests"`
	SuccessfulRequests int64   `json:"successfulRequests"`
	ErrorRequests      int64   `json:"errorRequests"`
	RejectedRequests   int64   `json:"rejectedRequests"` // Backpressure rejections

	// Traffic metrics
	BytesSent     int64 `json:"bytesSent"`
	BytesReceived int64 `json:"bytesReceived"`

	// Backpressure metrics (aligned with UDP package)
	BackpressureMetrics *udp.BackpressureMetrics `json:"backpressure,omitempty"`
	QueueUtilization    float64                   `json:"queueUtilization"` // 0-100%
	NormalCapacity      int64                     `json:"normalCapacity"`     // Target capacity
	CurrentLoad         int64                     `json:"currentLoad"`       // Current load
	Utilization         float64                   `json:"utilization"`       // Utilization percentage
}
