package tcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// SimpleTCPServer is a lightweight TCP server: net.Listen + acceptLoop + goroutine per conn.
// No backpressure, no worker pool. Suitable for RTMP, simple protocols.
// TCP tuning (NoDelay, buffers, optional SocketTuning) applied on accept.
type SimpleTCPServer struct {
	*core.BaseServer

	addr     string
	config   *SimpleTCPServerConfig
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc

	handler ConnectionHandler

	// Metrics
	totalAccepted int64
	handledConns  int64
	errorConns    int64
	activeConns   int64
}

// SimpleTCPServerConfig configures SimpleTCPServer.
type SimpleTCPServerConfig struct {
	Addr string

	// TCP tuning (0 = use defaults)
	ReadBuffer  int
	WriteBuffer int
	NoDelay     bool // TCP_NODELAY, default true

	// Optional: platform-specific socket tuning (SO_KEEPALIVE, TCP_QUICKACK, etc.)
	SocketTuning func(conn net.Conn)

	// StopTimeout for graceful shutdown. 0 = 5s default.
	StopTimeout time.Duration
}

// DefaultSimpleTCPServerConfig returns defaults for RTMP-style streaming.
func DefaultSimpleTCPServerConfig(addr string) *SimpleTCPServerConfig {
	if addr == "" {
		addr = ":1935"
	}
	return &SimpleTCPServerConfig{
		Addr:        addr,
		ReadBuffer:  512 * 1024,  // 512KB
		WriteBuffer: 2 * 1024 * 1024, // 2MB
		NoDelay:     true,
		StopTimeout: 5 * time.Second,
	}
}

// NewSimpleTCPServer creates a simple TCP server.
func NewSimpleTCPServer(gocmd core.GoCMD, config *SimpleTCPServerConfig) *SimpleTCPServer {
	if config == nil {
		config = DefaultSimpleTCPServerConfig("")
	}
	return &SimpleTCPServer{
		BaseServer: core.NewBaseServer("simple-tcp-server", gocmd),
		addr:       config.Addr,
		config:     config,
	}
}

// SetHandler sets the connection handler (fail-fast on nil).
func (s *SimpleTCPServer) SetHandler(handler ConnectionHandler) {
	if handler == nil {
		panic("tcp handler cannot be nil")
	}
	s.handler = handler
}

// Start starts the server (blocking). Typically run in a goroutine.
func (s *SimpleTCPServer) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}
	s.listener = listener

	s.ctx, s.cancel = context.WithCancel(s.GoCMD().Context())

	s.acceptLoop()
	return nil
}

// Stop stops the server gracefully.
func (s *SimpleTCPServer) Stop() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.listener != nil {
		s.listener.Close()
	}
	// Give handlers time to finish (best-effort)
	timeout := s.config.StopTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	<-time.After(timeout)
	return nil
}

// acceptLoop accepts connections and spawns a goroutine per conn.
func (s *SimpleTCPServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				if !isClosed(err) {
					s.Logger().Error(fmt.Sprintf("Accept error: %v", err))
				}
				continue
			}
		}

		// TCP tuning
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			if s.config.NoDelay {
				tcpConn.SetNoDelay(true)
			}
			if s.config.ReadBuffer > 0 {
				tcpConn.SetReadBuffer(s.config.ReadBuffer)
			}
			if s.config.WriteBuffer > 0 {
				tcpConn.SetWriteBuffer(s.config.WriteBuffer)
			}
		}
		if s.config.SocketTuning != nil {
			s.config.SocketTuning(conn)
		}

		atomic.AddInt64(&s.totalAccepted, 1)
		atomic.AddInt64(&s.activeConns, 1)

		cctx := &ConnContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			Context:            s.ctx,
			Conn:               conn,
			GoCMD:              s.GoCMD(),
			EventBus:           s.EventBus(),
			LocalAddr:          conn.LocalAddr(),
			RemoteAddr:         conn.RemoteAddr(),
		}

		go func() {
			defer func() {
				atomic.AddInt64(&s.activeConns, -1)
				conn.Close()
			}()
			atomic.AddInt64(&s.handledConns, 1)
			if err := s.handler(cctx); err != nil {
				atomic.AddInt64(&s.errorConns, 1)
				s.Logger().Error(fmt.Sprintf("handler error for %s: %v", conn.RemoteAddr(), err))
			}
		}()
	}
}

func isClosed(err error) bool {
	return errors.Is(err, net.ErrClosed)
}

// Metrics returns current server metrics.
func (s *SimpleTCPServer) Metrics() ServerMetrics {
	return ServerMetrics{
		QueuedConnections:   0,
		RejectedConnections: 0,
		QueueCapacity:       0,
		Workers:             0,
		QueueUtilization:    0,
		NormalCCU:           0,
		CurrentCCU:           0,
		CCUUtilization:      0,
		TotalAccepted:       atomic.LoadInt64(&s.totalAccepted),
		HandledConnections:  atomic.LoadInt64(&s.handledConns),
		ErrorConnections:    atomic.LoadInt64(&s.errorConns),
		ActiveConnections:   atomic.LoadInt64(&s.activeConns),
		MaxConns:            0,
	}
}
