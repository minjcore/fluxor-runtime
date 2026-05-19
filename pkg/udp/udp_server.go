package udp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/concurrency"
)

// UDPServer implements a fail-fast, backpressured UDP server.
// It mirrors pkg/tcp.TCPServer's structure: BaseServer + Mailbox + Executor + Backpressure.
// UDP is connectionless, so it handles packets instead of connections.
type UDPServer struct {
	*core.BaseServer

	addr   string
	config *UDPServerConfig

	mu       sync.RWMutex
	conn     net.PacketConn
	stopping int32

	packetMailbox concurrency.Mailbox
	executor      concurrency.Executor
	workers       int
	maxQueue      int

	startWorkersOnce sync.Once

	handler      PacketHandler
	middlewares []Middleware
	effective    PacketHandler
	backpressure *concurrency.BackpressureController
	rateLimiter  *RateLimiter // MaxPPS rate limiting (nil when maxPPS=0)
	maxPPS       int
	activePackets int64 // atomic: in-flight (queued + processing)

	// Metrics (atomic for thread-safety)
	queuedPackets   int64
	droppedPackets  int64
	totalReceived   int64
	handledPackets  int64
	errorPackets    int64
}

// UDPServerConfig configures the UDP server.
type UDPServerConfig struct {
	Addr string

	// Backpressure: bounded queue + worker pool.
	MaxQueue int
	Workers  int
	// MaxPPS bounds packets per second (rate limiting).
	// 0 means unlimited.
	MaxPPS int

	// ReadGoroutines controls the number of goroutines reading packets.
	// Default: 1 (single read goroutine, recommended for most cases).
	// Multiple read goroutines can improve packet receive rate under extreme load
	// but may cause out-of-order processing - use with caution.
	// Recommended: 1-4, typically 1 or runtime.GOMAXPROCS(0)
	ReadGoroutines int

	// BufferSize is the UDP read buffer size in bytes.
	// Default: 65507 (max UDP packet size).
	BufferSize int

	// ReadTimeout is the timeout for reading packets.
	// 0 means no timeout.
	ReadTimeout time.Duration

	// StopTimeout is the timeout for graceful shutdown (waiting for workers to finish).
	// Default: 5 seconds.
	StopTimeout time.Duration
}

// DefaultUDPServerConfig returns a sensible default configuration.
func DefaultUDPServerConfig(addr string) *UDPServerConfig {
	if addr == "" {
		addr = ":9001"
	}
	return &UDPServerConfig{
		Addr:           addr,
		MaxQueue:       1000,
		Workers:        50,
		MaxPPS:          0,
		ReadGoroutines: 1, // Single read goroutine (recommended)
		BufferSize:     65507, // Max UDP packet size
		ReadTimeout:    0,    // No timeout by default
		StopTimeout:    5 * time.Second,
	}
}

// HighThroughputIOBoundConfig returns configuration optimized for IO-bound workloads.
// Designed for high packet rate scenarios where CPU utilization is low (10-30%).
// Key optimizations:
//   - High worker count (IO-bound: many goroutines OK, not limited by CPU cores)
//   - Large queue size to handle packet bursts
//   - High packet rate limits
//   - Multiple read goroutines for maximum receive rate (runtime.GOMAXPROCS aware)
//   - Target: 100k+ packets/second for IO-bound operations
//
// Usage:
//
//	config := HighThroughputIOBoundConfig(":9001")
//	server := udp.NewUDPServer(gocmd, config)
func HighThroughputIOBoundConfig(addr string) *UDPServerConfig {
	if addr == "" {
		addr = ":9001"
	}
	// For IO-bound: Workers can be much higher than CPU cores
	// CPU idle during I/O → many goroutines OK
	workers := 2000 // High worker count for IO-bound

	// Large queue to handle packet bursts
	// Queue size should accommodate peak load
	queueSize := 50000

	// High packet rate limit
	maxPPS := 100000

	// Use multiple read goroutines for maximum receive rate
	// Limit to GOMAXPROCS to avoid excessive context switching
	readGoroutines := runtime.GOMAXPROCS(0)
	if readGoroutines > 4 {
		readGoroutines = 4 // Cap at 4 to prevent out-of-order issues
	}
	if readGoroutines < 1 {
		readGoroutines = 1
	}

	return &UDPServerConfig{
		Addr:           addr,
		MaxQueue:       queueSize,
		Workers:        workers,
		MaxPPS:         maxPPS,
		ReadGoroutines: readGoroutines,
		BufferSize:     65507,
		ReadTimeout:    0,
	}
}

// CPUBoundConfig returns configuration optimized for CPU-bound workloads.
// Designed for scenarios where packet handling involves CPU-intensive operations
// (crypto, data processing, computation).
// Key optimizations:
//   - Limited workers (CPU-bound: workers ≤ CPU cores)
//   - Moderate queue size
//
// Usage:
//
//	config := CPUBoundConfig(":9001")
//	server := udp.NewUDPServer(gocmd, config)
func CPUBoundConfig(addr string) *UDPServerConfig {
	if addr == "" {
		addr = ":9001"
	}
	// For CPU-bound: Workers should be ≤ CPU cores
	// Too many workers = context switching overhead, cache thrashing
	numCPU := runtime.NumCPU()
	workers := numCPU // 1 worker per CPU core

	// Moderate queue size (CPU-bound work is slower, don't need huge queue)
	queueSize := 1000

	// MaxPPS: Limited by CPU capacity
	// CPU-bound can't handle as many packets per second
	maxPPS := workers * 100 // 100 packets per second per worker

	return &UDPServerConfig{
		Addr:           addr,
		MaxQueue:       queueSize,
		Workers:        workers,
		MaxPPS:         maxPPS,
		ReadGoroutines: 1, // Single read goroutine for CPU-bound
		BufferSize:     65507,
		ReadTimeout:    0,
	}
}

// NewUDPServer creates a new UDP server.
func NewUDPServer(gocmd core.GoCMD, config *UDPServerConfig) *UDPServer {
	if config == nil {
		config = DefaultUDPServerConfig(":9001")
	}
	if config.Addr == "" {
		config.Addr = ":9001"
	}
	if config.MaxQueue < 1 {
		config.MaxQueue = 100
	}
	if config.Workers < 1 {
		config.Workers = 1
	}
	if config.MaxPPS < 0 {
		config.MaxPPS = 0
	}
	if config.ReadGoroutines < 1 {
		config.ReadGoroutines = 1
	}
	if config.ReadGoroutines > runtime.GOMAXPROCS(0)*2 {
		// Cap read goroutines to prevent excessive context switching
		config.ReadGoroutines = runtime.GOMAXPROCS(0)
		if config.ReadGoroutines < 1 {
			config.ReadGoroutines = 1
		}
	}
	if config.BufferSize < 1 {
		config.BufferSize = 65507 // Max UDP packet size
	}

	normalCapacity := config.MaxQueue + config.Workers
	packetMailbox := concurrency.NewBoundedMailbox(config.MaxQueue)
	executor := concurrency.NewExecutor(gocmd.Context(), concurrency.ExecutorConfig{
		Workers:   config.Workers,
		QueueSize: config.MaxQueue,
	})

	var rateLimiter *RateLimiter
	if config.MaxPPS > 0 {
		rateLimiter = NewRateLimiter(config.MaxPPS)
	}
	s := &UDPServer{
		BaseServer:    core.NewBaseServer("udp-server", gocmd),
		addr:           config.Addr,
		config:         config,
		packetMailbox:  packetMailbox,
		executor:       executor,
		workers:        config.Workers,
		maxQueue:       config.MaxQueue,
		maxPPS:         config.MaxPPS,
		rateLimiter:    rateLimiter,
		backpressure:   concurrency.NewBackpressureController(normalCapacity, 60),
		handler:        defaultPacketHandler,
	}
	s.effective = s.handler

	// Wire BaseServer hooks (template method pattern).
	s.BaseServer.SetHooks(s.doStart, s.doStop)

	// Start workers once (constructor and Start() both might be called by different patterns).
	s.startPacketWorkers()

	return s
}

func defaultPacketHandler(ctx *PacketContext) error {
	// Default: do nothing. Packet is received but not processed.
	return nil
}

// SetHandler sets the packet handler (fail-fast on nil).
func (s *UDPServer) SetHandler(handler PacketHandler) {
	if handler == nil {
		panic("udp handler cannot be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = handler
	s.rebuildHandlerLocked()
}

// Use adds middleware to the UDP server. Best practice: call before Start().
// Fail-fast: panics if any middleware is nil.
func (s *UDPServer) Use(mw ...Middleware) {
	if len(mw) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range mw {
		if m == nil {
			panic("udp middleware cannot be nil")
		}
		s.middlewares = append(s.middlewares, m)
	}
	s.rebuildHandlerLocked()
}

func (s *UDPServer) rebuildHandlerLocked() {
	h := s.handler
	// Wrap like TCP middleware: last added runs outermost.
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		h = s.middlewares[i](h)
	}
	s.effective = h
}

// ListeningAddr returns the actual listening address (useful when Addr is ":0").
// Returns empty string if not currently listening.
func (s *UDPServer) ListeningAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.conn == nil {
		return ""
	}
	return s.conn.LocalAddr().String()
}

// PacketConn returns the underlying UDP PacketConn.
// Returns nil if server is not started.
// This allows other services (e.g., HTTP/3) to reuse the same UDP socket.
func (s *UDPServer) PacketConn() net.PacketConn {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conn
}

// doStart is called by BaseServer.Start() - implements hook method.
// Note: Like TCPServer, Start() is a blocking call.
func (s *UDPServer) doStart() error {
	// Ensure workers are running.
	s.startPacketWorkers()

	conn, err := net.ListenPacket("udp", s.addr)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	// Start read goroutines (1 or more for maximum receive rate)
	readGoroutines := s.config.ReadGoroutines
	if readGoroutines < 1 {
		readGoroutines = 1
	}

	var wg sync.WaitGroup
	errCh := make(chan error, readGoroutines)

	// Launch multiple read goroutines for maximum receive rate
	for i := 0; i < readGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			s.readPacketsLoop(conn, id, errCh)
		}(i)
	}

	// Wait for all read goroutines to finish
	wg.Wait()

	// Check for errors
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

// readPacketsLoop reads packets from the connection and enqueues them.
func (s *UDPServer) readPacketsLoop(conn net.PacketConn, id int, errCh chan error) {
	buffer := make([]byte, s.config.BufferSize)

	for {
		// Check if stopping
		if atomic.LoadInt32(&s.stopping) == 1 {
			return
		}

		// Set read timeout if configured
		if s.config.ReadTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
		}

		// Read packet
		n, remoteAddr, err := conn.ReadFrom(buffer)
		if err != nil {
			// If we're stopping, treat "closed" as clean shutdown.
			if atomic.LoadInt32(&s.stopping) == 1 {
				return
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			// ReadTimeout: continue on timeout (no traffic), don't fail server
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			// Other errors: report and return
			if id == 0 {
				select {
				case errCh <- err:
				default:
				}
			}
			return
		}

		atomic.AddInt64(&s.totalReceived, 1)

		// Check MaxPPS rate limit (if configured)
		if s.rateLimiter != nil && !s.rateLimiter.TryAcquire() {
			atomic.AddInt64(&s.droppedPackets, 1)
			continue
		}

		// Check backpressure (normal capacity)
		if !s.backpressure.TryAcquire() {
			atomic.AddInt64(&s.droppedPackets, 1)
			continue
		}

		// Copy packet data (buffer is reused)
		packetData := make([]byte, n)
		copy(packetData, buffer[:n])

		// Create packet context
		packet := &packetMessage{
			data:       packetData,
			remoteAddr: remoteAddr,
		}

		// Enqueue packet (bounded queue: fail-fast when queue is full)
		if err := s.packetMailbox.Send(packet); err != nil {
			s.backpressure.Release()
			atomic.AddInt64(&s.droppedPackets, 1)
			continue
		}

		atomic.AddInt64(&s.queuedPackets, 1)
	}
}

// packetMessage represents a UDP packet to be processed.
type packetMessage struct {
	data       []byte
	remoteAddr net.Addr
}

// doStop is called by BaseServer.Stop() - implements hook method.
func (s *UDPServer) doStop() error {
	atomic.StoreInt32(&s.stopping, 1)

	s.mu.Lock()
	conn := s.conn
	s.conn = nil
	s.mu.Unlock()

	// Close connection to break ReadFrom().
	if conn != nil {
		_ = conn.Close()
	}

	// Close mailbox and shutdown executor.
	s.packetMailbox.Close()

	timeout := s.config.StopTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.executor.Shutdown(ctx)
}

// Metrics returns current server metrics.
func (s *UDPServer) Metrics() ServerMetrics {
	bp := s.backpressure.GetMetrics()

	queued := atomic.LoadInt64(&s.queuedPackets)
	queueUtil := float64(queued) / float64(s.maxQueue) * 100
	if s.maxQueue <= 0 {
		queueUtil = 0
	} else if queueUtil > 100.0 {
		queueUtil = 100.0
	}

	return ServerMetrics{
		QueuedPackets:      queued,
		DroppedPackets:     atomic.LoadInt64(&s.droppedPackets),
		QueueCapacity:      s.maxQueue,
		Workers:            s.workers,
		QueueUtilization:   queueUtil,
		NormalPPS:          int(bp.NormalCapacity),
		CurrentPPS:         int(bp.CurrentLoad),
		PPSUtilization:     bp.Utilization,
		TotalReceived:      atomic.LoadInt64(&s.totalReceived),
		HandledPackets:     atomic.LoadInt64(&s.handledPackets),
		ErrorPackets:       atomic.LoadInt64(&s.errorPackets),
		ActivePackets:      atomic.LoadInt64(&s.activePackets),
		MaxPacketsPerSecond: s.maxPPS,
	}
}

func (s *UDPServer) tryAcquirePacketSlot() bool {
	// Track active packets for metrics
	atomic.AddInt64(&s.activePackets, 1)
	return true
}

func (s *UDPServer) releasePacketSlot() {
	atomic.AddInt64(&s.activePackets, -1)
}

func (s *UDPServer) startPacketWorkers() {
	s.startWorkersOnce.Do(func() {
		for i := 0; i < s.workers; i++ {
			task := concurrency.NewNamedTask(
				fmt.Sprintf("udp-worker-%d", i),
				func(ctx context.Context) error {
					return s.processPacketFromMailbox(ctx)
				},
			)
			if err := s.executor.Submit(task); err != nil {
				s.Logger().Error(fmt.Sprintf("failed to start udp worker %d: %v", i, err))
			}
		}
	})
}

func (s *UDPServer) processPacketFromMailbox(ctx context.Context) error {
	for {
		msg, err := s.packetMailbox.Receive(ctx)
		if err != nil {
			return err
		}

		packet, ok := msg.(*packetMessage)
		if !ok || packet == nil {
			// Fail-fast: unexpected mailbox payload.
			s.backpressure.Release()
			s.releasePacketSlot()
			continue
		}

		atomic.AddInt64(&s.queuedPackets, -1)

		s.mu.RLock()
		conn := s.conn
		h := s.effective
		s.mu.RUnlock()

		// Skip if server stopped (conn nil) - avoid nil Conn in handler
		if conn == nil {
			s.backpressure.Release()
			s.releasePacketSlot()
			continue
		}

		// Create packet context with conn captured under lock
		pctx := &PacketContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			Context:            ctx,
			Conn:               conn,
			GoCMD:              s.GoCMD(),
			EventBus:           s.EventBus(),
			Data:               packet.data,
			RemoteAddr:         packet.remoteAddr,
			LocalAddr:          conn.LocalAddr(),
		}

		// Panic isolation must be per-packet; otherwise a panic would terminate
		// the worker goroutine and stop future processing.
		atomic.AddInt64(&s.handledPackets, 1)
		func() {
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt64(&s.errorPackets, 1)
					s.Logger().Error(fmt.Sprintf("panic in udp handler (isolated): %v", r))
				}
			}()
			if err := h(pctx); err != nil {
				atomic.AddInt64(&s.errorPackets, 1)
				s.Logger().Error(fmt.Sprintf("udp handler error: %v", err))
			}
		}()

		s.backpressure.Release()
		s.releasePacketSlot()
	}
}
