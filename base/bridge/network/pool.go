// Copyright (c) 2024-2028 Fluxor Framework
// Package network — Pool is a bounded connection pool for the C++ bridge server.
// Each slot holds a dedicated TCP connection (http.Transport with MaxConnsPerHost=1).
//
// Usage:
//
//	pool := network.NewPool("http://localhost:9090", 8, 30*time.Second)
//	defer pool.Close()
//
//	client, release, err := pool.Get(ctx)
//	if err != nil { ... }
//	defer release()
//	v, err := client.Add(1, 2)
package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// PoolStats is a snapshot of pool health metrics.
type PoolStats struct {
	MaxSize      int
	Size         int   // total allocated slots (active + idle)
	Active       int64 // slots currently checked-out
	Idle         int64 // slots waiting in pool
	TotalCreated int64 // lifetime new connections created
	TotalWaits   int64 // times Get had to block for a slot
}

// Pool is a bounded connection pool for the C++ bridge HTTP server.
type Pool struct {
	baseURL     string
	maxSize     int
	idleTimeout time.Duration
	rpcTimeout  time.Duration

	free   chan *poolConn // idle connections ready to check out
	mu     sync.Mutex
	size   int // total allocated, ≤ maxSize
	closed bool

	totalCreated atomic.Int64
	totalWaits   atomic.Int64
	activeConns  atomic.Int64
}

// poolConn is one slot: its own http.Transport ensures a single TCP connection.
type poolConn struct {
	tr       *http.Transport
	client   *Client
	lastUsed time.Time
}

// NewPool creates a pool of up to maxSize dedicated connections to baseURL.
// idleTimeout is how long an idle connection is kept before eviction.
func NewPool(baseURL string, maxSize int, idleTimeout time.Duration) *Pool {
	if maxSize < 1 {
		maxSize = 1
	}
	p := &Pool{
		baseURL:     baseURL,
		maxSize:     maxSize,
		idleTimeout: idleTimeout,
		rpcTimeout:  30 * time.Second,
		free:        make(chan *poolConn, maxSize),
	}
	go p.evictLoop()
	return p
}

// WithRPCTimeout sets the per-request timeout (default 30s). Returns p for chaining.
func (p *Pool) WithRPCTimeout(d time.Duration) *Pool {
	p.rpcTimeout = d
	return p
}

// Get checks out a connection from the pool.
// Blocks until one is available or ctx is cancelled.
// The caller MUST invoke the returned release func exactly once when done.
func (p *Pool) Get(ctx context.Context) (*Client, func(), error) {
	p.totalWaits.Add(1)
	for {
		// Fast path: take idle connection without blocking.
		select {
		case conn := <-p.free:
			if p.isExpired(conn) {
				p.retire(conn)
				continue
			}
			p.activeConns.Add(1)
			return conn.client, p.makeRelease(conn), nil
		default:
		}

		// No idle connection. Try to allocate a new slot.
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			return nil, nil, fmt.Errorf("bridge pool: closed")
		}
		if p.size < p.maxSize {
			conn := p.newConn()
			p.size++
			p.mu.Unlock()
			p.totalCreated.Add(1)
			p.activeConns.Add(1)
			return conn.client, p.makeRelease(conn), nil
		}
		p.mu.Unlock()

		// Pool full — wait for a release or context cancellation.
		select {
		case conn := <-p.free:
			if p.isExpired(conn) {
				p.retire(conn)
				continue
			}
			p.activeConns.Add(1)
			return conn.client, p.makeRelease(conn), nil
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("bridge pool: %w", ctx.Err())
		}
	}
}

// Close drains idle connections. In-use connections are closed when released.
func (p *Pool) Close() {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	for {
		select {
		case conn := <-p.free:
			p.closeConn(conn)
		default:
			return
		}
	}
}

// Stats returns a snapshot of pool metrics.
func (p *Pool) Stats() PoolStats {
	p.mu.Lock()
	size := p.size
	p.mu.Unlock()
	active := p.activeConns.Load()
	return PoolStats{
		MaxSize:      p.maxSize,
		Size:         size,
		Active:       active,
		Idle:         int64(size) - active,
		TotalCreated: p.totalCreated.Load(),
		TotalWaits:   p.totalWaits.Load(),
	}
}

// ── internals ────────────────────────────────────────────────────────────────

func (p *Pool) newConn() *poolConn {
	tr := &http.Transport{
		// Force exactly one TCP connection per pool slot.
		MaxIdleConnsPerHost: 1,
		MaxConnsPerHost:     1,
		IdleConnTimeout:     p.idleTimeout,
		DisableCompression:  true,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	return &poolConn{
		tr: tr,
		client: &Client{
			BaseURL: p.baseURL,
			HTTPClient: &http.Client{
				Transport: tr,
				Timeout:   p.rpcTimeout,
			},
		},
		lastUsed: time.Now(),
	}
}

func (p *Pool) makeRelease(conn *poolConn) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			p.activeConns.Add(-1)
			conn.lastUsed = time.Now()

			p.mu.Lock()
			closed := p.closed
			p.mu.Unlock()

			if closed {
				p.closeConn(conn)
				return
			}
			select {
			case p.free <- conn:
			default:
				// free channel full — this shouldn't happen since cap == maxSize.
				p.retire(conn)
			}
		})
	}
}

// retire closes and deallocates a slot.
func (p *Pool) retire(conn *poolConn) {
	p.closeConn(conn)
	p.mu.Lock()
	p.size--
	p.mu.Unlock()
}

func (p *Pool) closeConn(conn *poolConn) {
	conn.tr.CloseIdleConnections()
}

func (p *Pool) isExpired(conn *poolConn) bool {
	return p.idleTimeout > 0 && time.Since(conn.lastUsed) > p.idleTimeout
}

// evictLoop closes connections that have been idle past idleTimeout.
func (p *Pool) evictLoop() {
	if p.idleTimeout <= 0 {
		return
	}
	tick := time.NewTicker(p.idleTimeout / 2)
	defer tick.Stop()
	for range tick.C {
		p.mu.Lock()
		closed := p.closed
		p.mu.Unlock()
		if closed {
			return
		}
		p.evictIdle()
	}
}

func (p *Pool) evictIdle() {
	var keep []*poolConn
	for {
		select {
		case conn := <-p.free:
			if p.isExpired(conn) {
				p.closeConn(conn)
				p.mu.Lock()
				p.size--
				p.mu.Unlock()
			} else {
				keep = append(keep, conn)
			}
		default:
			goto done
		}
	}
done:
	for _, conn := range keep {
		select {
		case p.free <- conn:
		default:
			p.retire(conn)
		}
	}
}
