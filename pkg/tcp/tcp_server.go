package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/fluxorio/fluxor/pkg/core/netpoll"
)

// TCPServer implements a fail-fast, backpressured TCP server.
// It mirrors pkg/web.FastHTTPServer's structure: BaseServer + Mailbox + Executor + Backpressure.
type TCPServer struct {
	*core.BaseServer

	addr   string
	config *TCPServerConfig

	mu       sync.RWMutex
	listener net.Listener
	stopping int32

	connMailbox concurrency.Mailbox
	executor    concurrency.Executor
	workers     int
	maxQueue    int

	startWorkersOnce sync.Once

	handler      ConnectionHandler
	middlewares  []Middleware
	effective    ConnectionHandler
	backpressure *concurrency.BackpressureController
	maxConns     int
	activeConns  int64 // atomic: in-flight (queued + processing)

	// Metrics (atomic for thread-safety)
	queuedConnections   int64
	rejectedConnections int64
	totalAccepted       int64
	handledConnections  int64
	errorConnections    int64

	// netpoll: set when UseNetPoll is true and platform supports epoll/kqueue
	netpollPoller  netpoll.Poller
	listenerFile   *os.File
	listenerFd     int
}

// TCPServerConfig configures the TCP server.
type TCPServerConfig struct {
	Addr string

	// Backpressure: bounded queue + worker pool.
	MaxQueue int
	Workers  int
	// MaxConns bounds concurrent in-flight connections (queued + handling).
	// 0 means unlimited.
	MaxConns int

	// AcceptGoroutines controls the number of goroutines calling Accept().
	// Default: 1 (single accept goroutine, recommended for most cases).
	// Multiple accept goroutines can improve accept rate under extreme load
	// but may cause "thundering herd" - use with caution.
	// Recommended: 1-4, typically 1 or runtime.GOMAXPROCS(0)
	AcceptGoroutines int

	// TLSConfig enables TLS when non-nil.
	TLSConfig *tls.Config

	// Connection settings.
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// StopTimeout is the timeout for graceful shutdown (waiting for workers to finish).
	// Default: 5 seconds.
	StopTimeout time.Duration

	// UseNetPoll, when true, uses epoll (Linux) or kqueue (BSD/macOS) for the accept loop
	// instead of blocking Accept() in multiple goroutines. Single accept goroutine waits
	// on the listener fd via Poller.Wait; when readable, Accept() is called. Falls back
	// to the normal accept loop if NewPoller() returns ErrNotSupported (e.g. Windows).
	// Ignored when TLSConfig is set (TLS listener cannot use raw fd polling).
	UseNetPoll bool
}

// DefaultTCPServerConfig returns a sensible default configuration.
func DefaultTCPServerConfig(addr string) *TCPServerConfig {
	if addr == "" {
		addr = ":9000"
	}
	return &TCPServerConfig{
		Addr:             addr,
		MaxQueue:         1000,
		Workers:          50,
		MaxConns:         0,
		AcceptGoroutines: 1, // Single accept goroutine (recommended)
		TLSConfig:        nil,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
		StopTimeout:      5 * time.Second,
	}
}

// HighThroughputIOBoundConfig returns configuration optimized for IO-bound workloads.
// Designed for high connection rate scenarios where CPU utilization is low (10-30%).
// Key optimizations:
//   - High worker count (IO-bound: many goroutines OK, not limited by CPU cores)
//   - Large queue size to handle connection bursts
//   - High connection limits
//   - Multiple accept goroutines for maximum accept rate (runtime.GOMAXPROCS aware)
//   - Target: 100k+ connections/second for IO-bound operations (network I/O, file I/O)
//
// Usage:
//
//	config := HighThroughputIOBoundConfig(":9000")
//	server := tcp.NewTCPServer(gocmd, config)
func HighThroughputIOBoundConfig(addr string) *TCPServerConfig {
	if addr == "" {
		addr = ":9000"
	}
	// For IO-bound: Workers can be much higher than CPU cores
	// CPU idle during I/O → many goroutines OK
	workers := 2000 // High worker count for IO-bound

	// Large queue to handle connection bursts
	// Queue size should accommodate peak load
	queueSize := 50000

	// High connection limit for concurrent clients
	maxConns := 100000

	// Use multiple accept goroutines for maximum accept rate
	// Limit to GOMAXPROCS to avoid excessive context switching
	acceptGoroutines := runtime.GOMAXPROCS(0)
	if acceptGoroutines > 4 {
		acceptGoroutines = 4 // Cap at 4 to prevent thundering herd
	}
	if acceptGoroutines < 1 {
		acceptGoroutines = 1
	}

	return &TCPServerConfig{
		Addr:             addr,
		MaxQueue:         queueSize,
		Workers:          workers,
		MaxConns:         maxConns,
		AcceptGoroutines: acceptGoroutines,
		TLSConfig:        nil,
		ReadTimeout:      10 * time.Second,
		WriteTimeout:     10 * time.Second,
	}
}

// HighThroughputIOBoundConfigWithTargetRPS returns configuration for specific connection rate target.
// targetRPS: Target connections per second (e.g., 100000 for 100k connections/sec)
// avgLatencyMs: Average connection handling latency in milliseconds
//   - Plain TCP: 5ms (fast IO)
//   - TLS TCP: 15ms (5ms IO + 10ms TLS overhead)
//   - TLS handshake: +5-10ms, encryption/decryption: +1-2ms per connection
//
// Formula: Workers = (targetRPS * avgLatencyMs) / 1000
//
//	QueueSize = targetRPS * 0.5 (buffer for 500ms of traffic)
//
// Note: For TLS/encrypted TCP, use avgLatencyMs=15 to account for TLS overhead
func HighThroughputIOBoundConfigWithTargetRPS(addr string, targetRPS int, avgLatencyMs int) *TCPServerConfig {
	if addr == "" {
		addr = ":9000"
	}
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
	// This handles connection bursts and network latency spikes
	queueSize := targetRPS / 2
	if queueSize < 1000 {
		queueSize = 1000 // Minimum queue size
	}
	if queueSize > 100000 {
		queueSize = 100000 // Maximum queue size
	}

	// MaxConns: Allow concurrent connections up to target RPS
	// Each connection can handle multiple requests (keep-alive)
	maxConns := targetRPS
	if maxConns < 10000 {
		maxConns = 10000 // Minimum connections
	}
	if maxConns > 200000 {
		maxConns = 200000 // Maximum connections
	}

	// Use single accept goroutine for CPU-bound (accept is fast, workers are the bottleneck)
	acceptGoroutines := 1

	return &TCPServerConfig{
		Addr:             addr,
		MaxQueue:         queueSize,
		Workers:          workers,
		MaxConns:         maxConns,
		AcceptGoroutines: acceptGoroutines,
		TLSConfig:        nil,
		ReadTimeout:      10 * time.Second,
		WriteTimeout:     10 * time.Second,
	}
}

// CPUBoundConfig returns configuration optimized for CPU-bound workloads.
// Designed for scenarios where connection handling involves CPU-intensive operations
// (crypto, data processing, computation).
// Key optimizations:
//   - Limited workers (CPU-bound: workers ≤ CPU cores)
//   - Moderate queue size
//   - Workers pinned to OS threads for CPU-bound native code
//
// Usage:
//
//	config := CPUBoundConfig(":9000")
//	server := tcp.NewTCPServer(gocmd, config)
//	// In handler, use runtime.LockOSThread() for CPU-bound work
func CPUBoundConfig(addr string) *TCPServerConfig {
	if addr == "" {
		addr = ":9000"
	}
	// For CPU-bound: Workers should be ≤ CPU cores
	// Too many workers = context switching overhead, cache thrashing
	numCPU := runtime.NumCPU()
	workers := numCPU // 1 worker per CPU core

	// Moderate queue size (CPU-bound work is slower, don't need huge queue)
	queueSize := 1000

	// MaxConns: Limited by CPU capacity
	// CPU-bound can't handle as many concurrent connections
	maxConns := workers * 10 // 10 connections per worker

	return &TCPServerConfig{
		Addr:             addr,
		MaxQueue:         queueSize,
		Workers:          workers,
		MaxConns:         maxConns,
		AcceptGoroutines: 1, // Single accept goroutine for CPU-bound
		TLSConfig:        nil,
		ReadTimeout:      30 * time.Second, // Longer timeout for CPU-bound work
		WriteTimeout:     30 * time.Second,
	}
}

// HighThroughputIOBoundTLSConfig returns IO-bound config optimized for TLS/encrypted TCP.
// Accounts for TLS overhead (CPU-bound TLS operations in IO-bound workers).
// Key optimizations:
//   - Increased workers to handle TLS overhead (handshake, encryption/decryption)
//   - Larger queue for TLS connection bursts
//   - TLS session resumption enabled (reduces handshake overhead)
//
// Usage:
//
//	config := HighThroughputIOBoundTLSConfig(":9443")
//	tlsCfg, _ := tls.LoadX509KeyPair("cert.pem", "key.pem")
//	config.TLSConfig = &tls.Config{
//	    Certificates: []tls.Certificate{tlsCfg},
//	    SessionTicketsDisabled: false, // Enable session resumption
//	}
//	server := tcp.NewTCPServer(gocmd, config)
func HighThroughputIOBoundTLSConfig(addr string) *TCPServerConfig {
	if addr == "" {
		addr = ":9443"
	}
	// For TLS: Increase workers to handle TLS overhead
	// TLS handshake: +5-10ms, encryption/decryption: +1-2ms per connection
	// Total latency: ~15ms (5ms IO + 10ms TLS) vs 5ms for plain TCP
	workers := 2000 // Higher than plain IO-bound to account for TLS overhead

	// Large queue for TLS connection bursts
	queueSize := 50000

	// High connection limit (TLS can handle many connections with session resumption)
	maxConns := 100000

	return &TCPServerConfig{
		Addr:         addr,
		MaxQueue:     queueSize,
		Workers:      workers,
		MaxConns:     maxConns,
		TLSConfig:    nil,              // Set by caller
		ReadTimeout:  30 * time.Second, // Longer timeout for TLS handshake
		WriteTimeout: 30 * time.Second,
	}
}

// HighThroughputIOBoundTLSConfigWithTargetRPS returns TLS config for specific connection rate target.
// Accounts for TLS overhead in latency calculation.
// targetRPS: Target connections per second (e.g., 100000 for 100k connections/sec)
// avgLatencyMs: Average connection handling latency in milliseconds
//   - Plain TCP: 5ms
//   - TLS TCP: 15ms (5ms IO + 10ms TLS overhead)
//
// Usage:
//
//	config := HighThroughputIOBoundTLSConfigWithTargetRPS(":9443", 100000, 15)
//	tlsCfg, _ := tls.LoadX509KeyPair("cert.pem", "key.pem")
//	config.TLSConfig = &tls.Config{
//	    Certificates: []tls.Certificate{tlsCfg},
//	    SessionTicketsDisabled: false, // Enable session resumption
//	    CipherSuites: []uint16{
//	        tls.TLS_AES_128_GCM_SHA256, // TLS 1.3 (fastest, hardware accelerated)
//	    },
//	}
//	server := tcp.NewTCPServer(gocmd, config)
func HighThroughputIOBoundTLSConfigWithTargetRPS(addr string, targetRPS int, avgLatencyMs int) *TCPServerConfig {
	if addr == "" {
		addr = ":9443"
	}
	// Calculate workers accounting for TLS overhead
	// Workers = (RPS * latency_ms) / 1000
	// Example: 100k RPS * 15ms (TLS) = 1500 workers
	workers := (targetRPS * avgLatencyMs) / 1000
	if workers < 200 {
		workers = 200 // Minimum workers for TLS (higher than plain TCP)
	}
	if workers > 5000 {
		workers = 5000 // Maximum workers
	}

	// Queue size: Buffer for 500ms of traffic
	queueSize := targetRPS / 2
	if queueSize < 1000 {
		queueSize = 1000
	}
	if queueSize > 100000 {
		queueSize = 100000
	}

	// MaxConns: Allow concurrent connections
	maxConns := targetRPS
	if maxConns < 10000 {
		maxConns = 10000
	}
	if maxConns > 200000 {
		maxConns = 200000
	}

	// Use multiple accept goroutines for TLS with target RPS
	acceptGoroutines := runtime.GOMAXPROCS(0)
	if acceptGoroutines > 4 {
		acceptGoroutines = 4
	}
	if acceptGoroutines < 1 {
		acceptGoroutines = 1
	}

	return &TCPServerConfig{
		Addr:             addr,
		MaxQueue:         queueSize,
		Workers:          workers,
		MaxConns:         maxConns,
		AcceptGoroutines: acceptGoroutines,
		TLSConfig:        nil,              // Set by caller
		ReadTimeout:      30 * time.Second, // Longer timeout for TLS handshake
		WriteTimeout:     30 * time.Second,
	}
}

// NewTCPServer creates a new TCP server.
func NewTCPServer(gocmd core.GoCMD, config *TCPServerConfig) *TCPServer {
	if config == nil {
		config = DefaultTCPServerConfig(":9000")
	}
	if config.Addr == "" {
		config.Addr = ":9000"
	}
	if config.MaxQueue < 1 {
		config.MaxQueue = 100
	}
	if config.Workers < 1 {
		config.Workers = 1
	}
	if config.MaxConns < 0 {
		config.MaxConns = 0
	}
	if config.AcceptGoroutines < 1 {
		config.AcceptGoroutines = 1
	}
	if config.AcceptGoroutines > runtime.GOMAXPROCS(0)*2 {
		// Cap accept goroutines to prevent excessive context switching
		config.AcceptGoroutines = runtime.GOMAXPROCS(0)
		if config.AcceptGoroutines < 1 {
			config.AcceptGoroutines = 1
		}
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = 5 * time.Second
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 5 * time.Second
	}

	normalCapacity := config.MaxQueue + config.Workers
	connMailbox := concurrency.NewBoundedMailbox(config.MaxQueue)
	executor := concurrency.NewExecutor(gocmd.Context(), concurrency.ExecutorConfig{
		Workers:   config.Workers,
		QueueSize: config.MaxQueue,
	})

	s := &TCPServer{
		BaseServer:   core.NewBaseServer("tcp-server", gocmd),
		addr:         config.Addr,
		config:       config,
		connMailbox:  connMailbox,
		executor:     executor,
		workers:      config.Workers,
		maxQueue:     config.MaxQueue,
		maxConns:     config.MaxConns,
		backpressure: concurrency.NewBackpressureController(normalCapacity, 60),
		handler:      defaultConnectionHandler,
	}
	s.effective = s.handler

	// Wire BaseServer hooks (template method pattern).
	s.BaseServer.SetHooks(s.doStart, s.doStop)

	// Start workers once (constructor and Start() both might be called by different patterns).
	s.startConnWorkers()

	return s
}

func defaultConnectionHandler(ctx *ConnContext) error {
	// Default: do nothing. Connection will be closed by server.
	return nil
}

// SetHandler sets the connection handler (fail-fast on nil).
func (s *TCPServer) SetHandler(handler ConnectionHandler) {
	if handler == nil {
		panic("tcp handler cannot be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = handler
	s.rebuildHandlerLocked()
}

// Use adds middleware to the TCP server. Best practice: call before Start().
// Fail-fast: panics if any middleware is nil.
func (s *TCPServer) Use(mw ...Middleware) {
	if len(mw) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range mw {
		if m == nil {
			panic("tcp middleware cannot be nil")
		}
		s.middlewares = append(s.middlewares, m)
	}
	s.rebuildHandlerLocked()
}

func (s *TCPServer) rebuildHandlerLocked() {
	h := s.handler
	// Wrap like web middleware: last added runs outermost.
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		h = s.middlewares[i](h)
	}
	s.effective = h
}

// ListeningAddr returns the actual listening address (useful when Addr is ":0").
// Returns empty string if not currently listening.
func (s *TCPServer) ListeningAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// doStart is called by BaseServer.Start() - implements hook method.
// Note: Like FastHTTPServer, Start() is a blocking call.
func (s *TCPServer) doStart() error {
	// Ensure workers are running.
	s.startConnWorkers()

	var (
		ln  net.Listener
		err error
	)
	if s.config.TLSConfig != nil {
		ln, err = tls.Listen("tcp", s.addr, s.config.TLSConfig)
	} else {
		ln, err = net.Listen("tcp", s.addr)
	}
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()

	// Use netpoll accept loop when enabled and we have a TCP listener (no TLS)
	if s.config.UseNetPoll && s.config.TLSConfig == nil {
		if tcpLn, ok := ln.(*net.TCPListener); ok {
			poller, pollErr := netpoll.NewPoller()
			if pollErr == nil {
				f, fileErr := tcpLn.File()
				if fileErr == nil {
					fd := int(f.Fd())
					if addErr := poller.Add(fd, netpoll.Read); addErr == nil {
						s.mu.Lock()
						s.netpollPoller = poller
						s.listenerFile = f
						s.listenerFd = fd
						s.mu.Unlock()
						return s.runAcceptNetPoll(ln, poller, fd)
					}
					_ = f.Close()
				}
				_ = poller.Close()
			}
			// Fall through to normal accept loop
		}
	}

	// Start accept goroutines (1 or more for maximum accept rate)
	acceptGoroutines := s.config.AcceptGoroutines
	if acceptGoroutines < 1 {
		acceptGoroutines = 1
	}

	var wg sync.WaitGroup
	errCh := make(chan error, acceptGoroutines)

	// Launch multiple accept goroutines for maximum accept rate
	for i := 0; i < acceptGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				conn, err := ln.Accept()
				if err != nil {
					// If we're stopping, treat "closed listener" as clean shutdown.
					if atomic.LoadInt32(&s.stopping) == 1 {
						return
					}
					// Some platforms wrap the "closed" error; handle that too.
					if errors.Is(err, net.ErrClosed) {
						return
					}
					// Only send error from first goroutine to avoid multiple errors
					if id == 0 {
						select {
						case errCh <- err:
						default:
						}
					}
					return
				}

				atomic.AddInt64(&s.totalAccepted, 1)
				if !s.tryAcquireConnSlot() {
					atomic.AddInt64(&s.rejectedConnections, 1)
					_ = conn.Close()
					continue
				}
				s.enqueueConn(conn)
			}
		}(i)
	}

	// Wait for all accept goroutines to finish
	wg.Wait()

	// Check for errors
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

// runAcceptNetPoll runs a single accept goroutine using epoll/kqueue on the listener fd.
func (s *TCPServer) runAcceptNetPoll(ln net.Listener, p netpoll.Poller, listenerFd int) error {
	ctx := s.GoCMD().Context()
	for atomic.LoadInt32(&s.stopping) == 0 {
		events, err := p.Wait(ctx, 10*time.Millisecond)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		for _, e := range events {
			if e.FD != listenerFd || !e.ReadReady || e.Err != nil {
				continue
			}
			conn, err := ln.Accept()
			if err != nil {
				if atomic.LoadInt32(&s.stopping) == 1 || errors.Is(err, net.ErrClosed) {
					return nil
				}
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
			atomic.AddInt64(&s.totalAccepted, 1)
			if !s.tryAcquireConnSlot() {
				atomic.AddInt64(&s.rejectedConnections, 1)
				_ = conn.Close()
				continue
			}
			s.enqueueConn(conn)
		}
	}
	return nil
}

// doStop is called by BaseServer.Stop() - implements hook method.
func (s *TCPServer) doStop() error {
	atomic.StoreInt32(&s.stopping, 1)

	s.mu.Lock()
	ln := s.listener
	s.listener = nil
	p := s.netpollPoller
	s.netpollPoller = nil
	lf := s.listenerFile
	s.listenerFile = nil
	s.mu.Unlock()

	if p != nil {
		_ = p.Wake()
		_ = p.Close()
	}
	if lf != nil {
		_ = lf.Close()
	}

	// Close listener to break Accept().
	if ln != nil {
		_ = ln.Close()
	}

	// Close mailbox and shutdown executor.
	s.connMailbox.Close()

	timeout := s.config.StopTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.executor.Shutdown(ctx)
}

// Metrics returns current server metrics.
func (s *TCPServer) Metrics() ServerMetrics {
	bp := s.backpressure.GetMetrics()

	queued := atomic.LoadInt64(&s.queuedConnections)
	queueUtil := float64(queued) / float64(s.maxQueue) * 100
	if s.maxQueue <= 0 {
		queueUtil = 0
	} else if queueUtil > 100.0 {
		queueUtil = 100.0
	}

	return ServerMetrics{
		QueuedConnections:   queued,
		RejectedConnections: atomic.LoadInt64(&s.rejectedConnections),
		QueueCapacity:       s.maxQueue,
		Workers:             s.workers,
		QueueUtilization:    queueUtil,
		NormalCCU:           int(bp.NormalCapacity),
		CurrentCCU:          int(bp.CurrentLoad),
		CCUUtilization:      bp.Utilization,
		TotalAccepted:       atomic.LoadInt64(&s.totalAccepted),
		HandledConnections:  atomic.LoadInt64(&s.handledConnections),
		ErrorConnections:    atomic.LoadInt64(&s.errorConnections),
		ActiveConnections:   atomic.LoadInt64(&s.activeConns),
		MaxConns:            s.maxConns,
	}
}

func (s *TCPServer) tryAcquireConnSlot() bool {
	// Unlimited: track active for metrics only.
	if s.maxConns <= 0 {
		atomic.AddInt64(&s.activeConns, 1)
		return true
	}
	// Retry loop with exponential backoff to reduce contention
	// under high connection rates
	for attempts := 0; attempts < 10; attempts++ {
		cur := atomic.LoadInt64(&s.activeConns)
		if int(cur) >= s.maxConns {
			return false
		}
		if atomic.CompareAndSwapInt64(&s.activeConns, cur, cur+1) {
			return true
		}
		// Yield to other goroutines to reduce contention
		if attempts > 0 {
			runtime.Gosched()
		}
	}
	// Final attempt without yielding
	cur := atomic.LoadInt64(&s.activeConns)
	if int(cur) >= s.maxConns {
		return false
	}
	return atomic.CompareAndSwapInt64(&s.activeConns, cur, cur+1)
}

func (s *TCPServer) releaseConnSlot() {
	atomic.AddInt64(&s.activeConns, -1)
}

func (s *TCPServer) enqueueConn(conn net.Conn) {
	// Backpressure baseline: reject immediately when normal capacity exceeded.
	if !s.backpressure.TryAcquire() {
		atomic.AddInt64(&s.rejectedConnections, 1)
		s.releaseConnSlot()
		_ = conn.Close()
		return
	}

	// Bounded queue: fail-fast when queue is full.
	if err := s.connMailbox.Send(conn); err != nil {
		s.backpressure.Release()
		atomic.AddInt64(&s.rejectedConnections, 1)
		s.releaseConnSlot()
		_ = conn.Close()
		return
	}

	atomic.AddInt64(&s.queuedConnections, 1)
}

func (s *TCPServer) startConnWorkers() {
	s.startWorkersOnce.Do(func() {
		for i := 0; i < s.workers; i++ {
			task := concurrency.NewNamedTask(
				fmt.Sprintf("tcp-worker-%d", i),
				func(ctx context.Context) error {
					return s.processConnFromMailbox(ctx)
				},
			)
			if err := s.executor.Submit(task); err != nil {
				s.Logger().Error(fmt.Sprintf("failed to start tcp worker %d: %v", i, err))
			}
		}
	})
}

func (s *TCPServer) processConnFromMailbox(ctx context.Context) error {
	for {
		msg, err := s.connMailbox.Receive(ctx)
		if err != nil {
			return err
		}

		conn, ok := msg.(net.Conn)
		if !ok || conn == nil {
			// Fail-fast: unexpected mailbox payload.
			s.backpressure.Release()
			s.releaseConnSlot()
			continue
		}

		atomic.AddInt64(&s.queuedConnections, -1)

		s.mu.RLock()
		h := s.effective
		s.mu.RUnlock()

		// Per-connection timeouts (best-effort).
		_ = conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
		_ = conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))

		cctx := &ConnContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			Context:            ctx,
			Conn:               conn,
			GoCMD:              s.GoCMD(),
			EventBus:           s.EventBus(),
			LocalAddr:          conn.LocalAddr(),
			RemoteAddr:         conn.RemoteAddr(),
		}

		// Panic isolation must be per-connection; otherwise a panic would terminate
		// the worker goroutine and stop future processing.
		atomic.AddInt64(&s.handledConnections, 1)
		func() {
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt64(&s.errorConnections, 1)
					s.Logger().Error(fmt.Sprintf("panic in tcp handler (isolated): %v", r))
				}
			}()
			if err := h(cctx); err != nil {
				atomic.AddInt64(&s.errorConnections, 1)
				s.Logger().Error(fmt.Sprintf("tcp handler error: %v", err))
			}
		}()

		_ = conn.Close()
		s.backpressure.Release()
		s.releaseConnSlot()
	}
}
