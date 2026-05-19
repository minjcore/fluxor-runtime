package web

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/valyala/fasthttp"
)

const (
	bindRetryAttempts = 10
	bindRetryInterval = 2 * time.Second
)

// clampToZero clamps value to 0 if negative (prevents negative CCU display)
func clampToZero(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

// FastHTTPServer implements Server using fasthttp for high performance
// Uses Executor and Mailbox abstractions to hide Go concurrency primitives
// Extends BaseServer for common lifecycle management
type FastHTTPServer struct {
	*core.BaseServer // Embed base server for lifecycle management
	router           *FastRouter
	server           *fasthttp.Server
	addr             string
	requestMailbox   concurrency.Mailbox  // Abstracted: hides chan *fasthttp.RequestCtx
	executor         concurrency.Executor // Abstracted: hides goroutine pool
	maxQueue         int
	workers          int
	// Metrics for monitoring
	queuedRequests     int64 // Atomic counter for queued requests
	rejectedRequests   int64 // Atomic counter for rejected requests (503)
	totalRequests      int64 // Atomic counter for total requests
	successfulRequests int64 // Atomic counter for successful requests (200-299)
	errorRequests      int64 // Atomic counter for error requests (500-599)
	bytesSent          int64 // Atomic counter for bytes sent (network output)
	bytesReceived      int64 // Atomic counter for bytes received (network input)
	// Little's Law tracking
	totalResponseTimeNs int64 // Atomic counter for total response time in nanoseconds
	lastMetricsTime     int64 // Atomic timestamp (nanoseconds) for last metrics calculation
	lastTotalRequests   int64 // Atomic counter for total requests at last metrics calculation
	// Backpressure controller for CCU-based limiting
	backpressure *BackpressureController
	// Start workers once (constructor and Start() might both call it).
	startWorkersOnce sync.Once
	// TLS configuration for HTTPS
	tlsConfig *TLSConfig
}

// FastHTTPServerConfig configures the fasthttp server
type FastHTTPServerConfig struct {
	Addr            string
	MaxQueue        int // Bounded queue for backpressure
	Workers         int // Number of worker goroutines
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxConns        int
	ReadBufferSize  int
	WriteBufferSize int

	// High-RPS tuning (from tools/performant: 137k+ RPS). Zero = use defaults below.
	Concurrency        int           // Max concurrent connections (0 = 200000)
	IdleTimeout        time.Duration // Keepalive idle (0 = 90s)
	MaxRequestsPerConn int           // Max requests per connection (0 = unlimited)

	// TLS Configuration for HTTPS
	TLSConfig *TLSConfig // TLS configuration (nil = HTTP only)
}

// TLSConfig configures TLS/HTTPS for the server
type TLSConfig struct {
	// Enabled enables TLS/HTTPS
	Enabled bool

	// CertFile is the path to the TLS certificate file
	CertFile string

	// KeyFile is the path to the TLS private key file
	KeyFile string

	// CAFile is the path to the CA certificate file (for client cert verification)
	CAFile string

	// ClientAuth specifies the client authentication mode
	// 0 = NoClientCert (default), 1 = RequestClientCert, 2 = RequireAnyClientCert,
	// 3 = VerifyClientCertIfGiven, 4 = RequireAndVerifyClientCert
	ClientAuth tls.ClientAuthType

	// MinVersion is the minimum TLS version (default: TLS 1.2)
	MinVersion uint16

	// MaxVersion is the maximum TLS version (default: TLS 1.3)
	MaxVersion uint16

	// CipherSuites is a list of supported cipher suites (nil = use defaults)
	CipherSuites []uint16

	// PreferServerCipherSuites controls whether the server prefers its cipher suites
	PreferServerCipherSuites bool

	// InsecureSkipVerify skips certificate verification (NOT recommended for production)
	InsecureSkipVerify bool
}

// DefaultFastHTTPServerConfig returns default configuration for 137k+ RPS (tuned from tools/performant).
func DefaultFastHTTPServerConfig(addr string) *FastHTTPServerConfig {
	return &FastHTTPServerConfig{
		Addr:               addr,
		MaxQueue:           10000, // Bounded queue for backpressure
		Workers:             100,  // Worker goroutines
		ReadTimeout:         20 * time.Second,  // Tuned for keepalive (performant)
		WriteTimeout:       30 * time.Second,   // Tuned for keepalive (performant)
		MaxConns:            100000,
		ReadBufferSize:      8192,
		WriteBufferSize:     8192,
		Concurrency:         200000,            // fasthttp global concurrency (137k+ RPS)
		IdleTimeout:        90 * time.Second,   // Keepalive recycling
		MaxRequestsPerConn:  0,                  // Unlimited, maximize connection reuse
		TLSConfig:           nil,                // HTTP only by default
	}
}

// DefaultTLSConfig returns secure TLS configuration defaults
// Uses TLS 1.2+ with modern cipher suites
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		Enabled:                  true,
		MinVersion:               tls.VersionTLS12,
		MaxVersion:               tls.VersionTLS13,
		PreferServerCipherSuites: true,
		ClientAuth:               tls.NoClientCert,
		CipherSuites: []uint16{
			// TLS 1.3 cipher suites (automatically used when TLS 1.3 is negotiated)
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			// TLS 1.2 cipher suites
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		},
	}
}

// NewTLSConfigFromFiles creates TLS configuration from certificate files
// Fail-fast: Validates cert and key files exist
func NewTLSConfigFromFiles(certFile, keyFile string) (*TLSConfig, error) {
	// Fail-fast: certFile cannot be empty
	if certFile == "" {
		return nil, fmt.Errorf("fail-fast: certFile cannot be empty")
	}
	// Fail-fast: keyFile cannot be empty
	if keyFile == "" {
		return nil, fmt.Errorf("fail-fast: keyFile cannot be empty")
	}

	// Verify files exist
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("fail-fast: cert file not found: %s", certFile)
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("fail-fast: key file not found: %s", keyFile)
	}

	cfg := DefaultTLSConfig()
	cfg.CertFile = certFile
	cfg.KeyFile = keyFile
	return cfg, nil
}

// BuildTLSConfig builds a crypto/tls.Config from TLSConfig
// Fail-fast: Validates required fields and fails on any configuration error
func (c *TLSConfig) BuildTLSConfig() (*tls.Config, error) {
	if c == nil || !c.Enabled {
		return nil, nil
	}

	// Fail-fast: CertFile and KeyFile are required when TLS is enabled
	if c.CertFile == "" {
		return nil, fmt.Errorf("fail-fast: CertFile is required when TLS is enabled")
	}
	if c.KeyFile == "" {
		return nil, fmt.Errorf("fail-fast: KeyFile is required when TLS is enabled")
	}

	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("fail-fast: failed to load TLS certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               c.MinVersion,
		MaxVersion:               c.MaxVersion,
		CipherSuites:             c.CipherSuites,
		PreferServerCipherSuites: c.PreferServerCipherSuites,
		ClientAuth:               c.ClientAuth,
		InsecureSkipVerify:       c.InsecureSkipVerify,

		// Performance optimizations for high-throughput TLS
		// Session tickets: Enabled by default in Go (SessionTicketsDisabled = false)
		// This allows TLS session resumption, reducing handshake overhead

		// Client session cache: Enable for session resumption (TLS 1.2)
		// Default: nil (no cache) - we'll use default for now
		// For high-throughput, consider: ClientSessionCache: tls.NewLRUClientSessionCache(1000)
	}

	// Set defaults if not specified
	if tlsConfig.MinVersion == 0 {
		tlsConfig.MinVersion = tls.VersionTLS12
	}
	if tlsConfig.MaxVersion == 0 {
		tlsConfig.MaxVersion = tls.VersionTLS13
	}

	// Performance: Prefer TLS 1.3 (faster handshake, better performance)
	// TLS 1.3 uses 1-RTT handshake vs 2-RTT in TLS 1.2
	// Go's crypto/tls automatically uses TLS 1.3 when both client and server support it

	// Load CA certificate for client verification if provided
	if c.CAFile != "" {
		caCert, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("fail-fast: failed to read CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("fail-fast: failed to parse CA certificate")
		}
		tlsConfig.ClientCAs = caCertPool
	}

	return tlsConfig, nil
}

// CCUBasedConfig returns configuration optimized for CCU (Concurrent Users)
// maxCCU: Maximum concurrent users to serve normally
// overflowCCU: Additional CCU that will receive 503 (fail-fast backpressure)
// Formula: QueueSize = maxCCU - Workers (to handle overflow with 503)
func CCUBasedConfig(addr string, maxCCU int, overflowCCU int) *FastHTTPServerConfig {
	// Calculate workers: typically 10-20% of max CCU for optimal throughput
	workers := maxCCU / 10
	if workers < 50 {
		workers = 50 // Minimum workers
	}
	if workers > 500 {
		workers = 500 // Maximum workers to prevent goroutine explosion
	}

	// Queue size = maxCCU - workers
	// This ensures we can queue up to maxCCU requests
	// When queue is full, additional requests get 503 immediately (fail-fast)
	queueSize := maxCCU - workers
	if queueSize < 100 {
		queueSize = 100 // Minimum queue size
	}

	// MaxConns should allow maxCCU + some buffer, but reject overflow
	maxConns := maxCCU + overflowCCU

	return &FastHTTPServerConfig{
		Addr:               addr,
		MaxQueue:           queueSize,
		Workers:            workers,
		ReadTimeout:        20 * time.Second,
		WriteTimeout:       30 * time.Second,
		MaxConns:           maxConns,
		ReadBufferSize:     8192,
		WriteBufferSize:    8192,
		Concurrency:        200000,
		IdleTimeout:        90 * time.Second,
		MaxRequestsPerConn: 0,
	}
}

// CCUBasedConfigWithUtilization returns configuration with target utilization percentage
// maxCCU: Maximum concurrent users capacity
// utilizationPercent: Target utilization under normal load (e.g., 67 for 67%)
// This leaves headroom for traffic spikes while maintaining stability
// Formula: NormalCapacity = maxCCU * (utilizationPercent / 100)
func CCUBasedConfigWithUtilization(addr string, maxCCU int, utilizationPercent int) *FastHTTPServerConfig {
	if utilizationPercent < 1 || utilizationPercent > 100 {
		utilizationPercent = 67 // Default to 67% if invalid
	}

	// Calculate normal capacity (target utilization)
	normalCapacity := int(float64(maxCCU) * float64(utilizationPercent) / 100.0)

	// Calculate workers: 10-20% of normal capacity
	workers := normalCapacity / 10
	if workers < 50 {
		workers = 50 // Minimum workers
	}
	if workers > 500 {
		workers = 500 // Maximum workers
	}

	// Queue size for normal capacity
	queueSize := normalCapacity - workers
	if queueSize < 100 {
		queueSize = 100 // Minimum queue size
	}

	// MaxConns allows up to maxCCU (100% capacity)
	// But backpressure will kick in at normalCapacity (utilizationPercent)
	maxConns := maxCCU

	return &FastHTTPServerConfig{
		Addr:               addr,
		MaxQueue:           queueSize,
		Workers:            workers,
		ReadTimeout:        20 * time.Second,
		WriteTimeout:       30 * time.Second,
		MaxConns:           maxConns,
		ReadBufferSize:     8192,
		WriteBufferSize:    8192,
		Concurrency:        200000,
		IdleTimeout:        90 * time.Second,
		MaxRequestsPerConn: 0,
	}
}

// HighThroughputIOBoundConfig returns configuration optimized for IO-bound workloads
// Designed for high RPS scenarios where CPU utilization is low (10-30%)
// Key optimizations:
//   - High worker count (IO-bound: many goroutines OK, not limited by CPU cores)
//   - Large queue size to handle bursts
//   - High connection limits
//   - Target: 100k+ RPS for IO-bound operations (HTTP, DB, file I/O)
//
// Usage:
//
//	config := HighThroughputIOBoundConfig(":8080")
//	server := web.NewFastHTTPServer(gocmd, config)
func HighThroughputIOBoundConfig(addr string) *FastHTTPServerConfig {
	// For IO-bound: Workers can be much higher than CPU cores
	// CPU idle during I/O → many goroutines OK
	workers := 2000 // High worker count for IO-bound

	// Large queue to handle traffic bursts
	// Queue size should accommodate peak load
	queueSize := 50000

	// High connection limit for concurrent clients
	maxConns := 100000

	return &FastHTTPServerConfig{
		Addr:               addr,
		MaxQueue:           queueSize,
		Workers:            workers,
		ReadTimeout:        20 * time.Second,
		WriteTimeout:       30 * time.Second,
		MaxConns:           maxConns,
		ReadBufferSize:     8192,
		WriteBufferSize:    8192,
		Concurrency:        200000,
		IdleTimeout:        90 * time.Second,
		MaxRequestsPerConn: 0,
	}
}

// HighThroughputIOBoundConfigWithTargetRPS returns configuration for specific RPS target
// targetRPS: Target requests per second (e.g., 100000 for 100k RPS)
// avgLatencyMs: Average request latency in milliseconds
//   - HTTP: 10ms (fast IO)
//   - HTTPS: 20ms (10ms IO + 10ms TLS overhead)
//   - TLS handshake: +5-10ms, encryption/decryption: +1-2ms per request
//
// Formula: Workers = (targetRPS * avgLatencyMs) / 1000
//
//	QueueSize = targetRPS * 0.5 (buffer for 500ms of traffic)
//
// Note: For TLS/HTTPS, use avgLatencyMs=20 to account for TLS overhead
func HighThroughputIOBoundConfigWithTargetRPS(addr string, targetRPS int, avgLatencyMs int) *FastHTTPServerConfig {
	// Calculate workers based on target RPS and latency
	// Workers = (RPS * latency_ms) / 1000
	// Example: 100k RPS * 10ms = 1000 workers
	workers := (targetRPS * avgLatencyMs) / 1000
	if workers < 100 {
		workers = 100 // Minimum workers
	}
	if workers > 5000 {
		workers = 5000 // Maximum workers to prevent excessive goroutines
	}

	// Queue size: Buffer for 500ms of traffic at target RPS
	// This handles traffic bursts and network latency spikes
	queueSize := targetRPS / 2
	if queueSize < 1000 {
		queueSize = 1000 // Minimum queue size
	}
	if queueSize > 100000 {
		queueSize = 100000 // Maximum queue size
	}

	// MaxConns: Allow concurrent connections up to target RPS
	// Each connection can handle multiple requests (HTTP/1.1 keep-alive)
	maxConns := targetRPS
	if maxConns < 10000 {
		maxConns = 10000 // Minimum connections
	}
	if maxConns > 200000 {
		maxConns = 200000 // Maximum connections
	}

	return &FastHTTPServerConfig{
		Addr:               addr,
		MaxQueue:           queueSize,
		Workers:            workers,
		ReadTimeout:        20 * time.Second,
		WriteTimeout:       30 * time.Second,
		MaxConns:           maxConns,
		ReadBufferSize:     8192,
		WriteBufferSize:    8192,
		Concurrency:        200000,
		IdleTimeout:        90 * time.Second,
		MaxRequestsPerConn: 0,
	}
}

// NewFastHTTPServer creates a new fasthttp server with reactor-based handling
func NewFastHTTPServer(gocmd core.GoCMD, config *FastHTTPServerConfig) *FastHTTPServer {
	if config == nil {
		config = DefaultFastHTTPServerConfig(":8080")
	}

	router := NewFastRouter()

	// Calculate normal CCU capacity (queue + workers)
	// This is the target utilization capacity (e.g., 67% of max)
	normalCapacity := config.MaxQueue + config.Workers

	// Create Mailbox abstraction (hides channel creation)
	requestMailbox := concurrency.NewBoundedMailbox(config.MaxQueue)

	// Create Executor for worker pool (hides goroutine creation)
	// Use gocmd context for executor
	gocmdCtx := gocmd.Context()
	executorConfig := concurrency.ExecutorConfig{
		Workers:   config.Workers,
		QueueSize: config.MaxQueue,
	}
	executor := concurrency.NewExecutor(gocmdCtx, executorConfig)

	s := &FastHTTPServer{
		BaseServer:     core.NewBaseServer("fasthttp-server", gocmd),
		router:         router,
		addr:           config.Addr,
		requestMailbox: requestMailbox, // Abstracted: hides chan
		executor:       executor,       // Abstracted: hides goroutines
		maxQueue:       config.MaxQueue,
		workers:        config.Workers,
		// Initialize backpressure controller with normal capacity
		// This ensures 67% utilization under normal load
		// Reset interval: 60 seconds (for metrics)
		backpressure: NewBackpressureController(normalCapacity, 60),
		tlsConfig:    config.TLSConfig, // Store TLS config
		server: buildFasthttpServerFromConfig(config),
	}

	// Wire BaseServer hooks (template method pattern).
	s.BaseServer.SetHooks(s.doStart, s.doStop)

	// Set handler after server is created
	s.server.Handler = s.handleRequest

	// Start request processing workers using Executor (hides goroutine creation)
	s.startRequestWorkers()

	// Auto-register with dashboard metrics collector (if available)
	// Registration is done via dashboard package to avoid circular dependency
	// Users can call dashboard.RegisterHTTPServerForMetrics() manually if needed
	registerHTTPServerForMetrics(s)

	return s
}

// buildFasthttpServerFromConfig builds fasthttp.Server with performant tuning (tools/performant: 137k+ RPS).
func buildFasthttpServerFromConfig(config *FastHTTPServerConfig) *fasthttp.Server {
	concurrency := config.Concurrency
	if concurrency <= 0 {
		concurrency = 200000
	}
	idleTimeout := config.IdleTimeout
	if idleTimeout <= 0 {
		idleTimeout = 90 * time.Second
	}
	readTimeout := config.ReadTimeout
	if readTimeout <= 0 {
		readTimeout = 20 * time.Second
	}
	writeTimeout := config.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = 30 * time.Second
	}
	return &fasthttp.Server{
		ReadTimeout:                   readTimeout,
		WriteTimeout:                  writeTimeout,
		IdleTimeout:                   idleTimeout,
		MaxConnsPerIP:                 config.MaxConns,
		MaxRequestsPerConn:            config.MaxRequestsPerConn,
		Concurrency:                   concurrency,
		ReadBufferSize:               config.ReadBufferSize,
		WriteBufferSize:               config.WriteBufferSize,
		DisableKeepalive:              false, // Keepalive critical for high throughput
		DisableHeaderNamesNormalizing: true,  // Skip header normalization for speed (performant)
		NoDefaultServerHeader:         true,
		NoDefaultDate:                 true,  // Reduce overhead (performant)
		NoDefaultContentType:          true,  // Don't set Content-Type automatically (performant)
		ReduceMemoryUsage:            false, // Must be false when RequestCtx is passed through channels
		TCPKeepalive:                 true,
		TCPKeepalivePeriod:            60 * time.Second,
	}
}

// registerHTTPServerForMetrics registers the server with dashboard metrics collector
// This function uses a registration callback to avoid circular dependency
func registerHTTPServerForMetrics(server *FastHTTPServer) {
	// Use registration callback if available
	// This is set by dashboard package via init() to avoid circular dependency
	if httpServerRegistrar != nil {
		httpServerRegistrar(server)
	}
}

// httpServerRegistrar is set by dashboard package to register servers
var httpServerRegistrar func(*FastHTTPServer)

// SetHTTPServerRegistrar sets the function to register HTTP servers with dashboard
// This is called by dashboard package to enable auto-registration
func SetHTTPServerRegistrar(registrar func(*FastHTTPServer)) {
	httpServerRegistrar = registrar
}

// GetHTTPServerMetrics returns metrics data for dashboard integration
// This allows dashboard package to get metrics without importing web package
func (s *FastHTTPServer) GetHTTPServerMetrics() HTTPServerMetricsData {
	metrics := s.Metrics()
	return HTTPServerMetricsData{
		QueuedRequests:        metrics.QueuedRequests,
		RejectedRequests:      metrics.RejectedRequests,
		TotalRequests:         metrics.TotalRequests,
		SuccessfulRequests:    metrics.SuccessfulRequests,
		ErrorRequests:         metrics.ErrorRequests,
		QueueCapacity:         metrics.QueueCapacity,
		QueueUtilization:      metrics.QueueUtilization,
		Workers:               metrics.Workers,
		CurrentCCU:            metrics.CurrentCCU,
		CCUUtilization:        metrics.CCUUtilization,
		BytesSent:             metrics.BytesSent,
		BytesReceived:         metrics.BytesReceived,
		AverageLatencyMs:      metrics.AverageLatencyMs,
		ArrivalRate:           metrics.ArrivalRate,
		ExpectedQueueLength:   metrics.ExpectedQueueLength,
		LittlesLawValidation:  metrics.LittlesLawValidation,
	}
}

// HTTPServerMetricsData contains metrics data for dashboard
type HTTPServerMetricsData struct {
	QueuedRequests        int64
	RejectedRequests      int64
	TotalRequests         int64
	SuccessfulRequests    int64
	ErrorRequests         int64
	QueueCapacity         int
	QueueUtilization      float64
	Workers               int
	CurrentCCU            int
	CCUUtilization        float64
	BytesSent             int64
	BytesReceived         int64
	// Little's Law metrics
	AverageLatencyMs      float64
	ArrivalRate           float64
	ExpectedQueueLength   float64
	LittlesLawValidation  float64
}

// NewFastHTTPServerTLS creates a new fasthttp server with TLS/HTTPS support
// This is a convenience constructor for HTTPS servers
// Fail-fast: Validates TLS configuration
func NewFastHTTPServerTLS(gocmd core.GoCMD, config *FastHTTPServerConfig, certFile, keyFile string) (*FastHTTPServer, error) {
	if config == nil {
		config = DefaultFastHTTPServerConfig(":443")
	}

	// Create TLS config
	tlsCfg, err := NewTLSConfigFromFiles(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("fail-fast: failed to create TLS config: %w", err)
	}

	config.TLSConfig = tlsCfg
	return NewFastHTTPServer(gocmd, config), nil
}

// isAddrInUse reports whether err is "address already in use" (bind failure).
func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Err != nil {
		if errno, ok := opErr.Err.(*os.SyscallError); ok && errno.Err == syscall.EADDRINUSE {
			return true
		}
	}
	return strings.Contains(err.Error(), "address already in use")
}

// doStart is called by BaseServer.Start() - implements hook method
func (s *FastHTTPServer) doStart() error {
	// Set handler if not already set
	if s.server.Handler == nil {
		s.server.Handler = s.handleRequest
	}
	// Start request processing workers using Executor (hides goroutine creation)
	s.startRequestWorkers()

	// Check if TLS is enabled
	if s.tlsConfig != nil && s.tlsConfig.Enabled {
		s.Logger().Info(fmt.Sprintf("Starting FastHTTP HTTPS server on %s (TLS enabled)", s.addr))

		// Build TLS config and set on server
		tlsCfg, err := s.tlsConfig.BuildTLSConfig()
		if err != nil {
			s.Logger().Error(fmt.Sprintf("Failed to build TLS config: %v", err))
			return fmt.Errorf("fail-fast: failed to build TLS config: %w", err)
		}

		// Set TLS config on fasthttp server
		s.server.TLSConfig = tlsCfg

		// Start listening with TLS (retry on address already in use)
		var lastErr error
		for attempt := 0; attempt < bindRetryAttempts; attempt++ {
			lastErr = s.server.ListenAndServeTLS(s.addr, s.tlsConfig.CertFile, s.tlsConfig.KeyFile)
			if lastErr == nil || !isAddrInUse(lastErr) {
				break
			}
			s.Logger().Error(fmt.Sprintf("Bind failed (attempt %d/%d): %v; retrying in %v", attempt+1, bindRetryAttempts, lastErr, bindRetryInterval))
			time.Sleep(bindRetryInterval)
		}
		if lastErr != nil {
			s.Logger().Error(fmt.Sprintf("FastHTTP HTTPS server error: %v", lastErr))
		}
		return lastErr
	}

	// Start HTTP server (no TLS), retry on address already in use
	s.Logger().Info(fmt.Sprintf("Starting FastHTTP server on %s", s.addr))
	var lastErr error
	for attempt := 0; attempt < bindRetryAttempts; attempt++ {
		lastErr = s.server.ListenAndServe(s.addr)
		if lastErr == nil || !isAddrInUse(lastErr) {
			break
		}
		s.Logger().Error(fmt.Sprintf("Bind failed (attempt %d/%d): %v; retrying in %v", attempt+1, bindRetryAttempts, lastErr, bindRetryInterval))
		time.Sleep(bindRetryInterval)
	}
	if lastErr != nil {
		s.Logger().Error(fmt.Sprintf("FastHTTP server error: %v", lastErr))
	}
	return lastErr
}

// IsTLS returns whether the server is configured for TLS/HTTPS
func (s *FastHTTPServer) IsTLS() bool {
	return s.tlsConfig != nil && s.tlsConfig.Enabled
}

// TLSInfo returns TLS configuration information
func (s *FastHTTPServer) TLSInfo() map[string]interface{} {
	if s.tlsConfig == nil || !s.tlsConfig.Enabled {
		return map[string]interface{}{
			"enabled": false,
		}
	}
	return map[string]interface{}{
		"enabled":    true,
		"certFile":   s.tlsConfig.CertFile,
		"keyFile":    s.tlsConfig.KeyFile,
		"minVersion": tlsVersionString(s.tlsConfig.MinVersion),
		"maxVersion": tlsVersionString(s.tlsConfig.MaxVersion),
	}
}

// tlsVersionString converts TLS version to human-readable string
func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

// doStop is called by BaseServer.Stop() - implements hook method
func (s *FastHTTPServer) doStop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close request mailbox (hides channel close)
	s.requestMailbox.Close()

	// Shutdown executor (hides goroutine cleanup)
	if err := s.executor.Shutdown(ctx); err != nil {
		return err
	}

	// Shutdown server
	return s.server.ShutdownWithContext(ctx)
}

// Router returns the router
func (s *FastHTTPServer) Router() Router {
	return s.router
}

// FastRouter returns the fast router for direct access
func (s *FastHTTPServer) FastRouter() *FastRouter {
	return s.router
}

// RegisterMetricsEndpoint registers a /metrics endpoint for server monitoring
// Returns JSON with current server metrics including:
//   - Queue length and utilization
//   - Rejected requests (backpressure)
//   - Worker count
//   - CCU (Concurrent Users) metrics
//   - Request statistics
//
// Usage:
//
//	server := web.NewFastHTTPServer(gocmd, config)
//	server.RegisterMetricsEndpoint()  // Adds GET /metrics
func (s *FastHTTPServer) RegisterMetricsEndpoint() {
	s.router.GETFast("/metrics", func(ctx *FastRequestContext) error {
		metrics := s.Metrics()

		// Calculate RPS (requests per second) if available
		// Note: This is a snapshot, not a rate calculation
		// For actual RPS, use time-based metrics collection

		response := map[string]interface{}{
			"queue": map[string]interface{}{
				"queued":      metrics.QueuedRequests,
				"capacity":    metrics.QueueCapacity,
				"utilization": fmt.Sprintf("%.2f%%", metrics.QueueUtilization),
			},
			"workers": metrics.Workers,
			"ccu": map[string]interface{}{
				"normal":      metrics.NormalCCU,
				"current":     metrics.CurrentCCU,
				"utilization": fmt.Sprintf("%.2f%%", metrics.CCUUtilization),
			},
			"requests": map[string]interface{}{
				"total":      metrics.TotalRequests,
				"successful": metrics.SuccessfulRequests,
				"errors":     metrics.ErrorRequests,
				"rejected":   metrics.RejectedRequests,
			},
			"bottleneck_indicators": map[string]interface{}{
				"queue_full":         metrics.QueueUtilization > 90.0,
				"rejecting_requests": metrics.RejectedRequests > 0,
				"high_ccu":           metrics.CCUUtilization > 90.0,
			},
			"littles_law": map[string]interface{}{
				"average_latency_ms":    fmt.Sprintf("%.2f", metrics.AverageLatencyMs),
				"arrival_rate":          fmt.Sprintf("%.2f", metrics.ArrivalRate),
				"expected_queue_length": fmt.Sprintf("%.2f", metrics.ExpectedQueueLength),
				"actual_queue_length":   metrics.QueuedRequests,
				"validation_ratio":      fmt.Sprintf("%.2f", metrics.LittlesLawValidation),
			},
		}

		return ctx.JSON(fasthttp.StatusOK, response)
	})
}

// Metrics returns current server metrics
func (s *FastHTTPServer) Metrics() ServerMetrics {
	bpMetrics := s.backpressure.GetMetrics()
	normalCapacity := int(bpMetrics.NormalCapacity)
	queued := atomic.LoadInt64(&s.queuedRequests)
	queueUtil := float64(queued) / float64(s.maxQueue) * 100
	if queueUtil > 100.0 {
		queueUtil = 100.0
	}

	totalRequests := atomic.LoadInt64(&s.totalRequests)
	totalResponseTimeNs := atomic.LoadInt64(&s.totalResponseTimeNs)

	// Calculate average latency (W) in milliseconds
	var averageLatencyMs float64
	if totalRequests > 0 {
		averageLatencyMs = float64(totalResponseTimeNs) / float64(totalRequests) / 1e6 // Convert nanoseconds to milliseconds
	}

	// Calculate arrival rate (λ) using sliding window
	currentTime := time.Now().UnixNano()
	lastMetricsTime := atomic.LoadInt64(&s.lastMetricsTime)
	lastTotalRequests := atomic.LoadInt64(&s.lastTotalRequests)

	var arrivalRate float64
	if lastMetricsTime > 0 {
		deltaTime := float64(currentTime-lastMetricsTime) / 1e9 // Convert nanoseconds to seconds
		deltaRequests := float64(totalRequests - lastTotalRequests)
		if deltaTime > 0 {
			arrivalRate = deltaRequests / deltaTime
		}
	}

	// Update sliding window for next calculation
	atomic.StoreInt64(&s.lastMetricsTime, currentTime)
	atomic.StoreInt64(&s.lastTotalRequests, totalRequests)

	// Calculate expected queue length using Little's Law: L = λ × W
	// Convert latency from milliseconds to seconds for calculation
	expectedQueueLength := arrivalRate * (averageLatencyMs / 1000.0)

	// Calculate validation ratio: ActualQueueLength / ExpectedQueueLength
	var littlesLawValidation float64
	if expectedQueueLength > 0 {
		littlesLawValidation = float64(queued) / expectedQueueLength
	}

	return ServerMetrics{
		QueuedRequests:       queued,
		RejectedRequests:     atomic.LoadInt64(&s.rejectedRequests),
		QueueCapacity:        s.maxQueue,
		Workers:              s.workers,
		QueueUtilization:     queueUtil,
		NormalCCU:            normalCapacity, // Normal capacity (target utilization, e.g., 67%)
		CurrentCCU:           clampToZero(int(bpMetrics.CurrentLoad)), // Clamp to 0 to prevent negative values
		CCUUtilization:       bpMetrics.Utilization, // Utilization relative to normal capacity
		TotalRequests:        totalRequests,
		SuccessfulRequests:   atomic.LoadInt64(&s.successfulRequests),
		ErrorRequests:        atomic.LoadInt64(&s.errorRequests),
		BytesSent:            atomic.LoadInt64(&s.bytesSent),
		BytesReceived:        atomic.LoadInt64(&s.bytesReceived),
		AverageLatencyMs:     averageLatencyMs,
		ArrivalRate:          arrivalRate,
		ExpectedQueueLength:  expectedQueueLength,
		LittlesLawValidation: littlesLawValidation,
	}
}

// ServerMetrics provides server performance metrics
type ServerMetrics struct {
	QueuedRequests        int64   // Current queued requests
	RejectedRequests      int64   // Total rejected requests (503)
	QueueCapacity         int     // Maximum queue capacity
	Workers               int     // Number of worker goroutines
	QueueUtilization      float64 // Queue utilization percentage
	NormalCCU             int     // Normal CCU capacity (target utilization, e.g., 67%)
	CurrentCCU            int     // Current CCU load
	CCUUtilization        float64 // CCU utilization percentage (relative to normal capacity)
	TotalRequests         int64   // Total requests processed (successful + rejected)
	SuccessfulRequests    int64   // Total successful requests (200-299)
	ErrorRequests         int64   // Total error requests (500-599)
	BytesSent             int64   // Total bytes sent (network output)
	BytesReceived         int64   // Total bytes received (network input)
	// Little's Law metrics
	AverageLatencyMs      float64 // W: Average response latency in milliseconds
	ArrivalRate           float64 // λ: Requests per second (arrival rate)
	ExpectedQueueLength   float64 // L = λ × W (calculated expected queue length)
	LittlesLawValidation  float64 // Ratio: ActualQueueLength / ExpectedQueueLength
}

// handleRequest is the main request handler - non-blocking, queues to workers
// Fail-fast: Returns 503 immediately when normal capacity exceeded (backpressure)
// Normal capacity is set to target utilization (e.g., 67%), leaving headroom for spikes
// This prevents system crash by rejecting overflow requests gracefully
func (s *FastHTTPServer) handleRequest(ctx *fasthttp.RequestCtx) {
	method := string(ctx.Method())
	path := string(ctx.Path())

	// Check if GoCMD context is cancelled
	gocmdCtx := s.GoCMD().Context()
	select {
	case <-gocmdCtx.Done():
		s.Logger().Info(fmt.Sprintf("GoCMD context cancelled: %v", gocmdCtx.Err()))
		ctx.Error("Service Unavailable", fasthttp.StatusServiceUnavailable)
		return
	default:
		// Context is still active
	}

	// Step 1: Check backpressure controller (normal capacity limiting)
	// Normal capacity = target utilization (e.g., 67% of max)
	// This ensures system operates at target utilization under normal load
	if !s.backpressure.TryAcquire() {
		// Fail-fast: Normal capacity exceeded, reject immediately
		// This maintains target utilization (e.g., 67%) under normal conditions
		s.Logger().Info(fmt.Sprintf("backpressure: capacity exceeded for %s %s", method, path))
		atomic.AddInt64(&s.rejectedRequests, 1)
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		ctx.SetContentType("application/json")
		response := map[string]interface{}{
			"error":   "capacity_exceeded",
			"message": "Server at normal capacity - backpressure applied",
			"code":    "BACKPRESSURE",
		}
		jsonData, err := core.JSONEncode(response)
		if err != nil {
			s.Logger().Error(fmt.Sprintf("failed to encode backpressure response: %v", err))
			return
		}
		if _, err := ctx.Write(jsonData); err != nil {
			s.Logger().Error(fmt.Sprintf("failed to write backpressure response: %v", err))
		}
		return
	}

	// Step 2: Process request synchronously to ensure response is sent correctly
	// fasthttp requires the handler to complete before sending response
	// We still use backpressure for rate limiting, but process in same goroutine
	// Use defer to ensure backpressure is always released, even on panic
	defer s.backpressure.Release()

	// Process with panic recovery to ensure backpressure is released
	defer func() {
		if r := recover(); r != nil {
			// Handler panic: return 500 error instead of crashing
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetContentType("application/json")
			requestID := string(ctx.Request.Header.Peek("X-Request-ID"))
			if requestID == "" {
				requestID = "unknown"
			}
			s.Logger().Error(fmt.Sprintf("handler panic (request_id=%s): %v", requestID, r))
			response := map[string]interface{}{
				"error":      "handler_panic",
				"message":    "Request handler failed",
				"request_id": requestID,
			}
			jsonData, err := core.JSONEncode(response)
			if err != nil {
				s.Logger().Error(fmt.Sprintf("failed to encode panic response: %v", err))
				return
			}
			if _, err := ctx.Write(jsonData); err != nil {
				s.Logger().Error(fmt.Sprintf("failed to write panic response: %v", err))
			}
		}
	}()

	s.processRequest(ctx)
}

// SetHandler sets the request handler
func (s *FastHTTPServer) SetHandler(handler func(*fasthttp.RequestCtx)) {
	s.server.Handler = handler
}

// startRequestWorkers starts request processing using Executor (hides goroutine creation)
func (s *FastHTTPServer) startRequestWorkers() {
	s.startWorkersOnce.Do(func() {
		// Submit worker tasks to executor (hides go func() calls)
		for i := 0; i < s.workers; i++ {
			task := concurrency.NewNamedTask(
				fmt.Sprintf("http-worker-%d", i),
				func(ctx context.Context) error {
					return s.processRequestFromMailbox(ctx)
				},
			)
			if err := s.executor.Submit(task); err != nil {
				// Log error but continue
				s.Logger().Error(fmt.Sprintf("failed to start worker %d: %v", i, err))
			}
		}
	})
}

// processRequestFromMailbox processes requests from mailbox (hides channel operations)
func (s *FastHTTPServer) processRequestFromMailbox(ctx context.Context) error {
	// Fail-fast: recover from panics to prevent system crash
	// Panic isolation: one worker panic doesn't crash entire system
	defer func() {
		if r := recover(); r != nil {
			// Log panic but don't re-panic to prevent system crash
			s.Logger().Error(fmt.Sprintf("panic in worker (isolated): %v", r))
		}
	}()

	// Use Mailbox abstraction (hides channel receive and select statement)
	for {
		msg, err := s.requestMailbox.Receive(ctx)
		if err != nil {
			// Mailbox closed or context cancelled
			return err
		}

		// Type assert to RequestCtx
		reqCtx, ok := msg.(*fasthttp.RequestCtx)
		if !ok {
			// Invalid message type - skip
			s.Logger().Error(fmt.Sprintf("invalid message type in mailbox: %T", msg))
			continue
		}

		method := string(reqCtx.Method())
		path := string(reqCtx.Path())
		s.Logger().Info(fmt.Sprintf("worker received request: %s %s", method, path))

		// Decrement queued counter when processing starts
		atomic.AddInt64(&s.queuedRequests, -1)

		// Process request with panic isolation
		func() {
			// Release backpressure capacity when request completes
			defer s.backpressure.Release()

			defer func() {
				if r := recover(); r != nil {
					// Handler panic: return 500 error instead of crashing
					// Panic isolation: one request panic doesn't crash system
					// Note: reqCtx here is *fasthttp.RequestCtx, not *FastRequestContext
					// We need to get request ID from the FastRequestContext created in processRequest
					reqCtx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
					reqCtx.SetContentType("application/json")
					// Extract request ID from header if available
					requestID := string(reqCtx.Request.Header.Peek("X-Request-ID"))
					if requestID == "" {
						requestID = "unknown"
					}
					s.Logger().Error(fmt.Sprintf("handler panic (request_id=%s): %v", requestID, r))
					if _, err := reqCtx.WriteString(fmt.Sprintf(`{"error":"handler_panic","message":"Request handler failed","request_id":"%s"}`, requestID)); err != nil {
						s.Logger().Error(fmt.Sprintf("failed to write panic response: %v", err))
					}
				}
			}()

			s.processRequest(reqCtx)
		}()
	}
}

// realIPFromRequest returns the client IP, trusting one proxy (e.g. GCLB).
// Uses X-Forwarded-For (leftmost = client when 1 hop) or X-Real-IP, else RemoteAddr.
func realIPFromRequest(ctx *fasthttp.RequestCtx) string {
	if xff := string(ctx.Request.Header.Peek("X-Forwarded-For")); xff != "" {
		// Leftmost IP = client when trusting the single proxy in front (GKE/GCLB)
		first := strings.TrimSpace(strings.Split(xff, ",")[0])
		if first != "" {
			// Optional: strip port if present (e.g. "1.2.3.4:12345" -> "1.2.3.4")
			if host, _, err := net.SplitHostPort(first); err == nil {
				return strings.TrimSpace(host)
			}
			return first
		}
	}
	if xri := string(ctx.Request.Header.Peek("X-Real-IP")); xri != "" {
		return strings.TrimSpace(xri)
	}
	addr := ctx.RemoteAddr()
	if addr == nil {
		return ""
	}
	return addr.String()
}

// processRequest processes a single request
func (s *FastHTTPServer) processRequest(ctx *fasthttp.RequestCtx) {
	// Fail-fast: validate inputs
	if ctx == nil {
		panic("request context cannot be nil")
	}
	if s.GoCMD() == nil {
		panic("gocmd cannot be nil")
	}
	if s.router == nil {
		panic("router cannot be nil")
	}

	// Generate or extract request ID from headers
	requestID := string(ctx.Request.Header.Peek("X-Request-ID"))
	if requestID == "" {
		requestID = core.GenerateRequestID()
	}

	method := string(ctx.Method())
	path := string(ctx.Path())
	s.Logger().Info(fmt.Sprintf("processing request: %s %s (request_id=%s)", method, path, requestID))

	// Real IP from X-Forwarded-For (1 hop: leftmost) or X-Real-IP (GCLB/Autopilot-friendly)
	reqCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         ctx,
		GoCMD:              s.GoCMD(),
		EventBus:           s.EventBus(),
		Params:             make(map[string]string),
		requestID:          requestID,
	}
	reqCtx.Set("real_ip", realIPFromRequest(ctx))

	// Set request ID in response header for tracing
	ctx.Response.Header.Set("X-Request-ID", requestID)

	// Track request start time for latency calculation
	startTime := time.Now()

	// Track request metrics
	atomic.AddInt64(&s.totalRequests, 1)
	
	// Track bytes received (request body + headers including request line)
	// Header.Header() returns the complete HTTP header including request line (GET /path HTTP/1.1)
	requestHeaderBytes := len(ctx.Request.Header.Header())
	requestBodyBytes := len(ctx.Request.Body())
	requestBytes := requestHeaderBytes + requestBodyBytes
	atomic.AddInt64(&s.bytesReceived, int64(requestBytes))

	// Route request - errors are propagated immediately (fail-fast)
	s.router.ServeFastHTTP(reqCtx)

	// Track latency after request processing
	duration := time.Since(startTime)
	atomic.AddInt64(&s.totalResponseTimeNs, duration.Nanoseconds())

	// Track response status
	statusCode := ctx.Response.StatusCode()
	bodyLen := len(ctx.Response.Body())
	
	// Track bytes sent (response body + headers including status line)
	// Header.Header() returns the complete HTTP header including status line (HTTP/1.1 200 OK)
	responseHeaderBytes := len(ctx.Response.Header.Header())
	responseBytes := bodyLen + responseHeaderBytes
	atomic.AddInt64(&s.bytesSent, int64(responseBytes))
	s.Logger().Info(fmt.Sprintf("request completed: %s %s -> status=%d body_len=%d (request_id=%s)", method, path, statusCode, bodyLen, requestID))

	if bodyLen == 0 && statusCode == 200 {
		s.Logger().Info(fmt.Sprintf("response body is empty but status is 200 for %s %s (request_id=%s)", method, path, requestID))
	}

	if statusCode >= 200 && statusCode < 300 {
		atomic.AddInt64(&s.successfulRequests, 1)
	} else if statusCode >= 500 {
		atomic.AddInt64(&s.errorRequests, 1)
	}
}

// FastRequestContext wraps fasthttp RequestCtx with Fluxor context
// Extends BaseRequestContext for common data storage functionality
type FastRequestContext struct {
	*core.BaseRequestContext // Embed base context for data storage
	RequestCtx               *fasthttp.RequestCtx
	GoCMD                    core.GoCMD
	EventBus                 core.EventBus
	Params                   map[string]string
	requestID                string // Request ID for tracing
}

// JSON writes JSON response (default format) - fail-fast
func (c *FastRequestContext) JSON(statusCode int, data interface{}) error {
	// Fail-fast: validate status code
	if statusCode < 100 || statusCode > 599 {
		return fmt.Errorf("invalid status code: %d", statusCode)
	}

	if c.RequestCtx == nil {
		return fmt.Errorf("RequestCtx is nil")
	}

	c.RequestCtx.SetStatusCode(statusCode)
	c.RequestCtx.SetContentType("application/json")

	// Fail-fast: JSON encoding errors are propagated immediately
	jsonData, err := core.JSONEncode(data)
	if err != nil {
		return fmt.Errorf("json encode error: %w", err)
	}

	n, err := c.RequestCtx.Write(jsonData)
	if err != nil {
		return fmt.Errorf("write response error: %w", err)
	}

	if n != len(jsonData) {
		return fmt.Errorf("incomplete write: wrote %d of %d bytes", n, len(jsonData))
	}
	return nil
}

// BindJSON binds JSON request body to a struct - fail-fast
func (c *FastRequestContext) BindJSON(v interface{}) error {
	// Fail-fast: validate target
	if v == nil {
		return fmt.Errorf("cannot bind to nil value")
	}

	body := c.RequestCtx.PostBody()
	if len(body) == 0 {
		return fmt.Errorf("empty request body")
	}

	// Fail-fast: JSON decoding errors are propagated immediately
	return core.JSONDecode(body, v)
}

// Text writes text response
func (c *FastRequestContext) Text(statusCode int, text string) error {
	if c.RequestCtx == nil {
		return fmt.Errorf("RequestCtx is nil")
	}

	c.RequestCtx.SetStatusCode(statusCode)
	c.RequestCtx.SetContentType("text/plain; charset=utf-8")

	textBytes := []byte(text)
	n, err := c.RequestCtx.Write(textBytes)
	if err != nil {
		return fmt.Errorf("write text response error: %w", err)
	}

	if n != len(textBytes) {
		return fmt.Errorf("incomplete write: wrote %d of %d bytes", n, len(textBytes))
	}
	return nil
}

// Query returns query parameter value
func (c *FastRequestContext) Query(key string) string {
	return string(c.RequestCtx.QueryArgs().Peek(key))
}

// Param returns path parameter value
func (c *FastRequestContext) Param(key string) string {
	return c.Params[key]
}

// Method returns HTTP method
func (c *FastRequestContext) Method() []byte {
	return c.RequestCtx.Method()
}

// Path returns request path
func (c *FastRequestContext) Path() []byte {
	return c.RequestCtx.Path()
}

// Error writes error response
func (c *FastRequestContext) Error(msg string, statusCode int) {
	c.RequestCtx.Error(msg, statusCode)
}

// RequestID returns the request ID for this request
func (c *FastRequestContext) RequestID() string {
	return c.requestID
}

// RealIP returns the client IP, from X-Forwarded-For (1 hop) or X-Real-IP when behind a proxy (e.g. GCLB).
func (c *FastRequestContext) RealIP() string {
	if v := c.Get("real_ip"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if c.RequestCtx != nil && c.RequestCtx.RemoteAddr() != nil {
		return c.RequestCtx.RemoteAddr().String()
	}
	return ""
}

// UserID extracts UserID from request
// Priority: 1. JWT Claims (if JWT middleware was used) > 2. Request Context (if set by middleware) > 3. HTTP Header (X-User-ID)
// UserID is used for EventLoop routing (same user → same EventLoop)
func (c *FastRequestContext) UserID() string {
	// Priority 1: JWT Claims (if JWT middleware was used)
	// Try common JWT claim keys: "user", "jwt", "claims"
	// Use dynamic import to avoid circular dependency - try to extract from claims directly
	jwtKeys := []string{"user", "jwt", "claims"}
	for _, key := range jwtKeys {
		if userID := c.getUserIDFromJWTClaims(key); userID != "" {
			return userID
		}
	}

	// Priority 2: Request Context (if set by middleware)
	if userID, ok := c.Get("user_id").(string); ok && userID != "" {
		return userID
	}

	// Priority 3: HTTP Header
	if userID := string(c.RequestCtx.Request.Header.Peek("X-User-ID")); userID != "" {
		return userID
	}

	return ""
}

// getUserIDFromJWTClaims attempts to extract UserID from JWT claims stored in context
// This method handles various JWT claim formats without importing auth package (to avoid circular dependency)
func (c *FastRequestContext) getUserIDFromJWTClaims(key string) string {
	// Check if claims exist in context
	claimsInterface := c.Get(key)
	if claimsInterface == nil {
		return ""
	}

	// Try to extract user_id from various claim formats
	// Handle map[string]interface{} (common JWT claims format)
	if claimsMap, ok := claimsInterface.(map[string]interface{}); ok {
		// Try common claim names: user_id, sub (JWT standard "subject"), id
		if userID, ok := claimsMap["user_id"].(string); ok && userID != "" {
			return userID
		}
		if userID, ok := claimsMap["sub"].(string); ok && userID != "" {
			return userID
		}
		if userID, ok := claimsMap["id"].(string); ok && userID != "" {
			return userID
		}
	}

	// Handle jwt.MapClaims (type assertion may fail, but try)
	// Note: We can't import jwt package here to avoid circular dependency
	// But jwt.MapClaims is essentially map[string]interface{}, so the above should work

	return ""
}

// FloxID extracts FloxID (Stream ID) from request
// Priority: 1. Header (X-Flox-ID) > 2. Path params (streamId, orderId, aggregateId, etc.) > 3. Context (if already set)
// FloxID is used for EventLoop routing (same stream → same EventLoop)
func (c *FastRequestContext) FloxID() string {
	// Priority 1: Check header X-Flox-ID
	if floxid := string(c.RequestCtx.Request.Header.Peek("X-Flox-ID")); floxid != "" {
		return floxid
	}

	// Priority 2: Check path params (common stream ID param names)
	// Check in order: streamId, orderId, aggregateId, entityId, id
	streamIDParams := []string{"streamId", "orderId", "aggregateId", "entityId", "id"}
	for _, param := range streamIDParams {
		if val := c.Params[param]; val != "" {
			return val
		}
	}

	// Priority 3: Check if FloxID already set in context (from middleware or previous extraction)
	if c.GoCMD != nil {
		if floxid := core.GetFloxID(c.GoCMD.Context()); floxid != "" {
			return floxid
		}
	}

	return ""
}

// Context returns a context with request ID, FloxID, and UserID (if available)
// FloxID and UserID are automatically extracted and set in context for EventBus routing
func (c *FastRequestContext) Context() context.Context {
	ctx := context.Background()
	if c.requestID != "" {
		ctx = core.WithRequestID(ctx, c.requestID)
	}
	
	// Auto-extract and set FloxID for EventBus routing (highest priority)
	if floxid := c.FloxID(); floxid != "" {
		ctx = core.WithFloxID(ctx, floxid)
	}
	
	// Note: UserID is passed via headers, not context, to maintain compatibility
	// EventBus will extract UserID from headers in GetRoutingHeaders()
	
	return ctx
}

// GetRoutingHeaders extracts all routing headers for EventBus routing
// Returns map with: X-Flox-ID, X-User-ID, X-Session-ID, X-Request-ID, X-Route-Key
// UserID is extracted using UserID() method (JWT > Context > Header priority)
func (c *FastRequestContext) GetRoutingHeaders() map[string]string {
	headers := make(map[string]string)
	
	// Extract FloxID (highest priority for routing)
	if floxid := c.FloxID(); floxid != "" {
		headers["X-Flox-ID"] = floxid
	}
	
	// Extract UserID using UserID() method (JWT > Context > Header priority)
	if userID := c.UserID(); userID != "" {
		headers["X-User-ID"] = userID
	}
	
	// Extract other routing headers
	if sessionID, ok := c.Get("session_id").(string); ok && sessionID != "" {
		headers["X-Session-ID"] = sessionID
	} else if sessionID := string(c.RequestCtx.Request.Header.Peek("X-Session-ID")); sessionID != "" {
		headers["X-Session-ID"] = sessionID
	}
	
	if requestID := c.RequestID(); requestID != "" {
		headers["X-Request-ID"] = requestID
	}
	
	if routeKey := string(c.RequestCtx.Request.Header.Peek("X-Route-Key")); routeKey != "" {
		headers["X-Route-Key"] = routeKey
	}
	
	return headers
}

// PublishWithRouting publishes a message to EventBus with automatic routing
// FloxID is automatically extracted and set in context for EventLoop routing
// Same stream ID → same EventLoop (sequential processing, event ordering preserved)
func (c *FastRequestContext) PublishWithRouting(address string, body interface{}) error {
	if c.EventBus == nil {
		return fmt.Errorf("EventBus is not available")
	}
	
	// Get context with FloxID automatically set
	ctx := c.Context()
	
	// Publish - FloxID from context will be used for routing (highest priority)
	return c.EventBus.PublishWithContext(ctx, address, body)
}

// SendWithRouting sends a request-reply message to EventBus with automatic routing
// FloxID is automatically extracted and set in context for EventLoop routing
// Returns reply message or error
func (c *FastRequestContext) SendWithRouting(address string, body interface{}, timeout time.Duration) (core.Message, error) {
	if c.EventBus == nil {
		return nil, fmt.Errorf("EventBus is not available")
	}
	
	// Get context with FloxID automatically set
	ctx := c.Context()
	
	// Send - FloxID from context will be used for routing (highest priority)
	return c.EventBus.SendWithContext(ctx, address, body, timeout)
}
