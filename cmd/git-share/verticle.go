package main

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

// GitShareVerticle serves a single-file view from private GitHub repos.
type GitShareVerticle struct {
	*core.BaseVerticle
	server     *web.FastHTTPServer
	router     *web.FastRouter
	cfg        *AppConfig
	startTime  time.Time
	reqCounter int64
}

func NewGitShareVerticle(cfg *AppConfig) *GitShareVerticle {
	return &GitShareVerticle{BaseVerticle: core.NewBaseVerticle("git-share"), cfg: cfg}
}

func (v *GitShareVerticle) Start(ctx core.FluxorContext) error {
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}
	v.startTime = time.Now()
	gocmd := ctx.GoCMD()

	serverCfg := &web.FastHTTPServerConfig{
		Addr:            v.cfg.Server.Addr,
		MaxQueue:        v.cfg.Server.MaxQueue,
		Workers:         v.cfg.Server.Workers,
		ReadTimeout:     time.Duration(v.cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout:    time.Duration(v.cfg.Server.WriteTimeoutSec) * time.Second,
		MaxConns:        v.cfg.Server.MaxConns,
		ReadBufferSize:  v.cfg.Server.ReadBufferSize,
		WriteBufferSize: v.cfg.Server.WriteBufferSize,
	}

	if v.cfg.TLS.Enabled {
		tlsCfg, err := web.NewTLSConfigFromFiles(v.cfg.TLS.CertFile, v.cfg.TLS.KeyFile)
		if err != nil {
			return fmt.Errorf("TLS config error: %w", err)
		}
		if v.cfg.TLS.CAFile != "" {
			tlsCfg.CAFile = v.cfg.TLS.CAFile
		}
		serverCfg.TLSConfig = tlsCfg
	}

	v.server = web.NewFastHTTPServer(gocmd, serverCfg)
	v.router = v.server.FastRouter()
	v.registerRoutes()

	if v.cfg.Metrics.Enabled {
		go v.reportMetrics(ctx)
	}
	return v.server.Start()
}

func (v *GitShareVerticle) registerRoutes() {
	// GET /file?owner=...&repo=...&path=...&ref=main&download=1
	v.router.GETFast("/file", func(c *web.FastRequestContext) error {
		atomic.AddInt64(&v.reqCounter, 1)
		q := c.RequestCtx.URI().QueryArgs()
		owner := string(q.Peek("owner"))
		repo := string(q.Peek("repo"))
		path := string(q.Peek("path"))
		ref := string(q.Peek("ref"))
		download := string(q.Peek("download"))

		data, ct, err := FetchFileRaw(v.cfg, owner, repo, path, ref)
		if err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}

		if download == "1" || download == "true" {
			c.RequestCtx.Response.Header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path))
		}
		c.RequestCtx.SetContentType(ct)
		c.RequestCtx.SetStatusCode(200)
		_, werr := c.RequestCtx.Write(data)
		return werr
	})

	// Health
	v.router.GETFast("/health", func(c *web.FastRequestContext) error {
		atomic.AddInt64(&v.reqCounter, 1)
		return c.JSON(200, map[string]interface{}{
			"status":  "healthy",
			"service": v.cfg.Service.Name,
			"version": v.cfg.Service.Version,
			"uptime":  time.Since(v.startTime).String(),
		})
	})

	// Metrics
	v.router.GETFast("/metrics", func(c *web.FastRequestContext) error {
		m := v.server.Metrics()
		return c.JSON(200, map[string]interface{}{
			"server": map[string]interface{}{
				"totalRequests": atomic.LoadInt64(&v.reqCounter),
				"workers":       m.Workers,
				"queued":        m.QueuedRequests,
				"rejected":      m.RejectedRequests,
				"currentCCU":    m.CurrentCCU,
				"ccuUtil":       fmt.Sprintf("%.1f%%", m.CCUUtilization),
			},
			"runtime": map[string]interface{}{
				"goroutines": runtime.NumGoroutine(),
				"numCPU":     runtime.NumCPU(),
				"goMaxProcs": runtime.GOMAXPROCS(0),
			},
		})
	})
}

func (v *GitShareVerticle) reportMetrics(ctx core.FluxorContext) {}
