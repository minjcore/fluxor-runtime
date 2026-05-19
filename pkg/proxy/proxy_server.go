package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"golang.org/x/time/rate"
)

// proxyServer implements ProxyServer interface
type proxyServer struct {
	*core.BaseServer

	config      Config
	httpServer  *http.Server
	tcpListener net.Listener

	mu          sync.RWMutex
	backends    []Backend
	backendStatus map[string]*BackendStatus
	loadBalancer LoadBalancer
	healthChecker *HealthChecker
	metricsCollector *MetricsCollector
	circuitBreakers map[string]*CircuitBreaker

	// Metrics (atomic for thread-safety)
	totalConnections    int64
	activeConnections   int64
	rejectedConnections int64
	failedConnections   int64
	totalRequests       int64
	successfulRequests  int64
	failedRequests      int64
	bytesTransferred    int64

	// Rate limiting
	rateLimiter *rate.Limiter

	// Health check
	healthCheckCancel context.CancelFunc
	healthCheckCtx    context.Context

	// Connection tracking
	activeConns map[net.Conn]bool
	connMu      sync.RWMutex
	stopping    int32 // atomic: server stopping flag
}

// NewProxyServer creates a new proxy server
// Fail-fast: Validates configuration
func NewProxyServer(gocmd core.GoCMD, config Config) (ProxyServer, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create base server
	baseServer := core.NewBaseServer("proxy-server", gocmd)

	server := &proxyServer{
		BaseServer:      baseServer,
		config:          config,
		backends:        config.Backends,
		backendStatus:   make(map[string]*BackendStatus),
		activeConns:     make(map[net.Conn]bool),
		circuitBreakers: make(map[string]*CircuitBreaker),
	}

	// Initialize load balancer
	server.loadBalancer = newLoadBalancer(config.LoadBalancingStrategy)

	// Initialize health checker
	server.healthChecker = NewHealthChecker(config.HealthCheckTimeout)

	// Initialize metrics collector
	server.metricsCollector = NewMetricsCollector(1000, 10*time.Second)

	// Initialize rate limiter
	if config.RateLimit > 0 {
		server.rateLimiter = rate.NewLimiter(rate.Limit(config.RateLimit), config.RateLimit)
	}

	// Initialize backend status and circuit breakers
	for _, backend := range config.Backends {
		server.backendStatus[backend.URL] = &BackendStatus{
			Backend:   backend,
			Healthy:   true, // Assume healthy initially
			LastCheck: time.Now(),
		}
		// Create circuit breaker for each backend (5 failures, 30s open)
		server.circuitBreakers[backend.URL] = NewCircuitBreaker(5, 30*time.Second)
	}

	// Set hooks
	server.BaseServer.SetHooks(server.doStart, server.doStop)

	return server, nil
}

// doStart starts the proxy server
func (s *proxyServer) doStart() error {
	// Start health checking
	s.startHealthChecking()

	// Start appropriate server based on protocol
	switch s.config.Protocol {
	case "http":
		return s.startHTTPServer()
	case "tcp":
		return s.startTCPServer()
	case "both":
		// Start both in goroutines
		go func() {
			if err := s.startHTTPServer(); err != nil {
				s.logger().WithFields(map[string]interface{}{"error": err.Error()}).Error("HTTP proxy server error")
			}
		}()
		return s.startTCPServer()
	default:
		return fmt.Errorf("unsupported protocol: %s", s.config.Protocol)
	}
}

// doStop stops the proxy server
func (s *proxyServer) doStop() error {
	// Mark as stopping
	atomic.StoreInt32(&s.stopping, 1)

	// Stop health checking
	if s.healthCheckCancel != nil {
		s.healthCheckCancel()
	}

	// Stop HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger().WithFields(map[string]interface{}{"error": err.Error()}).Error("HTTP server shutdown error")
		}
	}

	// Stop TCP listener
	if s.tcpListener != nil {
		if err := s.tcpListener.Close(); err != nil {
			s.logger().WithFields(map[string]interface{}{"error": err.Error()}).Error("TCP listener close error")
		}
	}

	// Close all active connections
	s.connMu.Lock()
	for conn := range s.activeConns {
		conn.Close()
		delete(s.activeConns, conn)
	}
	s.connMu.Unlock()

	return nil
}

// startHTTPServer starts the HTTP proxy server
func (s *proxyServer) startHTTPServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHTTPProxy)

	s.httpServer = &http.Server{
		Addr:              s.config.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: s.config.ReadTimeout,
		ReadTimeout:       s.config.ReadTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		IdleTimeout:       s.config.IdleTimeout,
	}

	s.logger().WithFields(map[string]interface{}{"addr": s.config.ListenAddr}).Info("HTTP proxy server starting")
	return s.httpServer.ListenAndServe()
}

// startTCPServer starts the TCP proxy server
func (s *proxyServer) startTCPServer() error {
	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.tcpListener = listener
	s.logger().WithFields(map[string]interface{}{"addr": s.config.ListenAddr}).Info("TCP proxy server starting")

	for {
		conn, err := listener.Accept()
		if err != nil {
			// Check if server is stopping
			if atomic.LoadInt32(&s.stopping) == 1 {
				return nil
			}
			return fmt.Errorf("accept error: %w", err)
		}

		// Check max connections
		if s.config.MaxConnections > 0 {
			active := atomic.LoadInt64(&s.activeConnections)
			if active >= int64(s.config.MaxConnections) {
				conn.Close()
				atomic.AddInt64(&s.rejectedConnections, 1)
				continue
			}
		}

		atomic.AddInt64(&s.totalConnections, 1)
		atomic.AddInt64(&s.activeConnections, 1)

		// Track connection
		s.connMu.Lock()
		s.activeConns[conn] = true
		s.connMu.Unlock()

		// Handle connection in goroutine
		go s.handleTCPProxy(conn)
	}
}

// handleHTTPProxy handles HTTP proxy requests
func (s *proxyServer) handleHTTPProxy(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	atomic.AddInt64(&s.totalRequests, 1)

	// Rate limiting
	if s.rateLimiter != nil {
		if err := s.rateLimiter.Wait(r.Context()); err != nil {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			atomic.AddInt64(&s.failedRequests, 1)
			return
		}
	}

	// Select backend
	backend, err := s.selectBackend()
	if err != nil {
		s.logger().WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("No healthy backend available")
		http.Error(w, fmt.Sprintf("No healthy backend available: %v", err), http.StatusServiceUnavailable)
		atomic.AddInt64(&s.failedRequests, 1)
		return
	}

	// Create proxy request
	proxyReq := &ProxyRequest{
		BaseRequestContext: core.NewBaseRequestContext(),
		Context:            r.Context(),
		Request:            r,
		Response:           w,
		GoCMD:              s.GoCMD(),
		EventBus:           s.EventBus(),
		Backend:            backend,
		StartTime:          startTime,
	}

	// Forward request
	if err := s.forwardHTTPRequest(proxyReq); err != nil {
		s.logger().Error("HTTP proxy error", "error", err, "backend", backend.URL)
		http.Error(w, "Proxy error", http.StatusBadGateway)
		atomic.AddInt64(&s.failedRequests, 1)
		return
	}

	atomic.AddInt64(&s.successfulRequests, 1)
}

// handleTCPProxy handles TCP proxy connections
func (s *proxyServer) handleTCPProxy(clientConn net.Conn) {
	defer func() {
		atomic.AddInt64(&s.activeConnections, -1)
		s.connMu.Lock()
		delete(s.activeConns, clientConn)
		s.connMu.Unlock()
		clientConn.Close()
	}()

	// Select backend
	backend, err := s.selectBackend()
	if err != nil {
		s.logger().WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("No healthy backend available")
		atomic.AddInt64(&s.failedConnections, 1)
		return
	}

	// Create proxy connection
	proxyConn := &ProxyConnection{
		BaseRequestContext: core.NewBaseRequestContext(),
		Context:           context.Background(),
		ClientConn:        clientConn,
		GoCMD:             s.GoCMD(),
		EventBus:          s.EventBus(),
		Backend:           backend,
		StartTime:         time.Now(),
	}

	// Forward connection
	if err := s.forwardTCPConnection(proxyConn); err != nil {
		s.logger().WithFields(map[string]interface{}{
			"error":   err.Error(),
			"backend": backend.URL,
		}).Error("TCP proxy error")
		atomic.AddInt64(&s.failedConnections, 1)
	}
	
	// Decrement connection count
	s.mu.Lock()
	if status, exists := s.backendStatus[backend.URL]; exists {
		if status.Connections > 0 {
			status.Connections--
		}
	}
	s.mu.Unlock()
}

// forwardHTTPRequest forwards an HTTP request to the backend
func (s *proxyServer) forwardHTTPRequest(proxyReq *ProxyRequest) error {
	// Create request to backend
	backendURL := proxyReq.Backend.URL + proxyReq.Request.URL.Path
	if proxyReq.Request.URL.RawQuery != "" {
		backendURL += "?" + proxyReq.Request.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(proxyReq.Context, proxyReq.Request.Method, backendURL, proxyReq.Request.Body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	for key, values := range proxyReq.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Set X-Forwarded-For
	if clientIP := proxyReq.Request.RemoteAddr; clientIP != "" {
		req.Header.Set("X-Forwarded-For", clientIP)
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: proxyReq.Backend.Timeout,
	}

	// Forward request with circuit breaker protection
	var resp *http.Response
	cb, exists := s.circuitBreakers[proxyReq.Backend.URL]
	
	if exists {
		var cbErr error
		err = cb.Call(func() error {
			resp, cbErr = client.Do(req)
			return cbErr
		})
		
		if err != nil {
			return NewBackendError(ErrCodeCircuitBreakerOpen, "circuit breaker is open", proxyReq.Backend, err)
		}
	} else {
		// Forward request without circuit breaker
		resp, err = client.Do(req)
		if err != nil {
			return NewBackendError(ErrCodeBackendError, "backend request failed", proxyReq.Backend, err)
		}
	}
	defer resp.Body.Close()

	// Calculate response time
	responseTime := time.Since(proxyReq.StartTime)
	s.metricsCollector.RecordResponseTime(responseTime)

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			proxyReq.Response.Header().Add(key, value)
		}
	}

	// Set status code
	proxyReq.Response.WriteHeader(resp.StatusCode)

	// Copy response body
	bytesWritten, err := io.Copy(proxyReq.Response, resp.Body)
	if err != nil {
		return NewBackendError(ErrCodeBackendError, "failed to copy response body", proxyReq.Backend, err)
	}

	atomic.AddInt64(&s.bytesTransferred, bytesWritten)

	// Record metrics
	success := resp.StatusCode >= 200 && resp.StatusCode < 500
	s.metricsCollector.RecordRequest(proxyReq.Backend.URL, success, responseTime)

	return nil
}

// forwardTCPConnection forwards a TCP connection to the backend
func (s *proxyServer) forwardTCPConnection(proxyConn *ProxyConnection) error {
	// Parse backend URL
	backendAddr, err := parseBackendAddr(proxyConn.Backend.URL)
	if err != nil {
		return fmt.Errorf("invalid backend URL: %w", err)
	}

	// Connect to backend
	backendConn, err := net.DialTimeout("tcp", backendAddr, proxyConn.Backend.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to backend: %w", err)
	}
	defer backendConn.Close()

	proxyConn.BackendConn = backendConn

	// Bidirectional forwarding
	errChan := make(chan error, 2)

	// Client -> Backend
	go func() {
		bytesWritten, err := io.Copy(backendConn, proxyConn.ClientConn)
		if err != nil {
			errChan <- fmt.Errorf("client->backend error: %w", err)
			return
		}
		atomic.AddInt64(&s.bytesTransferred, bytesWritten)
		errChan <- nil
	}()

	// Backend -> Client
	go func() {
		bytesWritten, err := io.Copy(proxyConn.ClientConn, backendConn)
		if err != nil {
			errChan <- fmt.Errorf("backend->client error: %w", err)
			return
		}
		atomic.AddInt64(&s.bytesTransferred, bytesWritten)
		errChan <- nil
	}()

	// Wait for one direction to finish
	err = <-errChan
	return err
}

// selectBackend selects a backend using the load balancer
func (s *proxyServer) selectBackend() (*Backend, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get healthy backends (also check circuit breaker)
	healthyBackends := make([]BackendStatus, 0)
	for url, status := range s.backendStatus {
		// Check circuit breaker state
		cb, exists := s.circuitBreakers[url]
		if exists && cb.State() == StateOpen {
			continue // Skip if circuit breaker is open
		}
		
		if status.Healthy {
			healthyBackends = append(healthyBackends, *status)
		}
	}

	if len(healthyBackends) == 0 {
		return nil, NewProxyError(ErrCodeNoHealthyBackends, "no healthy backends available")
	}

	// Update connection counts for least-connections strategy
	for i := range healthyBackends {
		url := healthyBackends[i].Backend.URL
		if status, exists := s.backendStatus[url]; exists {
			healthyBackends[i].Connections = status.Connections
		}
	}

	// Select backend using load balancer
	backend, err := s.loadBalancer.SelectBackend(healthyBackends)
	if err != nil {
		return nil, NewProxyError(ErrCodeNoHealthyBackends, "failed to select backend")
	}

	// Increment connection count
	if status, exists := s.backendStatus[backend.URL]; exists {
		status.Connections++
	}

	return backend, nil
}

// startHealthChecking starts periodic health checks
func (s *proxyServer) startHealthChecking() {
	s.healthCheckCtx, s.healthCheckCancel = context.WithCancel(context.Background())

	go func() {
		ticker := time.NewTicker(s.config.HealthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-s.healthCheckCtx.Done():
				return
			case <-ticker.C:
				s.checkBackendHealth()
			}
		}
	}()
}

// checkBackendHealth checks health of all backends
func (s *proxyServer) checkBackendHealth() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for url, status := range s.backendStatus {
		healthy, latency, err := s.healthChecker.CheckHealth(status.Backend)
		status.Healthy = healthy
		status.LastCheck = time.Now()
		if err != nil {
			status.LastError = err
		} else {
			status.LastError = nil
		}
		if latency > 0 {
			status.Latency = &latency
		}
		s.backendStatus[url] = status

		// Update circuit breaker based on health
		if cb, exists := s.circuitBreakers[url]; exists {
			if !healthy {
				// Simulate failure for circuit breaker
				cb.Call(func() error {
					if err != nil {
						return err
					}
					return fmt.Errorf("health check failed")
				})
			} else {
				// Simulate success for circuit breaker
				cb.Call(func() error { return nil })
			}
		}
	}
}

// Helper functions
func parseBackendAddr(url string) (string, error) {
	// Remove protocol prefix
	if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
	} else if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
	} else if strings.HasPrefix(url, "tcp://") {
		url = strings.TrimPrefix(url, "tcp://")
	}

	// Extract host:port
	parts := strings.Split(url, "/")
	return parts[0], nil
}

// Metrics returns current proxy metrics
func (s *proxyServer) Metrics() ServerMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	healthyCount := 0
	for url, status := range s.backendStatus {
		// Check both health status and circuit breaker
		cb, exists := s.circuitBreakers[url]
		if status.Healthy && (!exists || cb.State() != StateOpen) {
			healthyCount++
		}
	}

	return ServerMetrics{
		TotalConnections:    atomic.LoadInt64(&s.totalConnections),
		ActiveConnections:   atomic.LoadInt64(&s.activeConnections),
		RejectedConnections: atomic.LoadInt64(&s.rejectedConnections),
		FailedConnections:   atomic.LoadInt64(&s.failedConnections),
		TotalRequests:       atomic.LoadInt64(&s.totalRequests),
		SuccessfulRequests:  atomic.LoadInt64(&s.successfulRequests),
		FailedRequests:      atomic.LoadInt64(&s.failedRequests),
		AverageResponseTime: s.metricsCollector.AverageResponseTime(),
		BackendCount:        len(s.backendStatus),
		HealthyBackends:     healthyCount,
		UnhealthyBackends:   len(s.backendStatus) - healthyCount,
		RequestsPerSecond:   s.metricsCollector.RequestsPerSecond(),
		BytesTransferred:    atomic.LoadInt64(&s.bytesTransferred),
	}
}

// Health returns the health status of the proxy server
func (s *proxyServer) Health() map[string]interface{} {
	metrics := s.Metrics()
	
	healthy := true
	issues := []string{}

	// Check if server is running
	if atomic.LoadInt32(&s.stopping) == 1 {
		healthy = false
		issues = append(issues, "server is stopping")
	}

	// Check for healthy backends
	if metrics.HealthyBackends == 0 {
		healthy = false
		issues = append(issues, "no healthy backends available")
	}

	// Check for high failure rate
	if metrics.TotalRequests > 0 {
		failureRate := float64(metrics.FailedRequests) / float64(metrics.TotalRequests)
		if failureRate > 0.1 { // More than 10% failure rate
			healthy = false
			issues = append(issues, fmt.Sprintf("high failure rate: %.2f%%", failureRate*100))
		}
	}

	return map[string]interface{}{
		"healthy":           healthy,
		"issues":            issues,
		"metrics":           metrics,
		"protocol":          s.config.Protocol,
		"listenAddr":        s.config.ListenAddr,
		"loadBalancingStrategy": s.config.LoadBalancingStrategy,
	}
}

// AddBackend adds a backend to the proxy pool
func (s *proxyServer) AddBackend(backend Backend) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate backend
	if backend.URL == "" {
		return &ConfigError{Code: "INVALID_BACKEND", Message: "Backend URL is required"}
	}

	// Set defaults
	if backend.Weight <= 0 {
		backend.Weight = 1
	}
	if backend.Timeout == 0 {
		backend.Timeout = s.config.ConnectionTimeout
	}

	// Add to backends
	s.backends = append(s.backends, backend)
	s.backendStatus[backend.URL] = &BackendStatus{
		Backend:   backend,
		Healthy:   true,
		LastCheck: time.Now(),
	}

	// Create circuit breaker for new backend
	s.circuitBreakers[backend.URL] = NewCircuitBreaker(5, 30*time.Second)

	return nil
}

// RemoveBackend removes a backend from the proxy pool
func (s *proxyServer) RemoveBackend(url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from backends
	for i, backend := range s.backends {
		if backend.URL == url {
			s.backends = append(s.backends[:i], s.backends[i+1:]...)
			break
		}
	}

		// Remove from status
		delete(s.backendStatus, url)

	// Remove circuit breaker
	delete(s.circuitBreakers, url)

	return nil
}

// GetBackends returns all configured backends
func (s *proxyServer) GetBackends() []Backend {
	s.mu.RLock()
	defer s.mu.RUnlock()

	backends := make([]Backend, len(s.backends))
	copy(backends, s.backends)
	return backends
}

// logger returns the server logger
func (s *proxyServer) logger() core.Logger {
	return s.BaseServer.Logger()
}
