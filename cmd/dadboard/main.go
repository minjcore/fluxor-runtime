// Copyright (c) 2024-2028 Fluxor Framework
// All rights reserved.
//
// This source code is proprietary and confidential.
// Unauthorized copying, modification, distribution, or use of this software,
// via any medium is strictly prohibited without the express written permission
// of Fluxor Framework.
//
// This code is provided as an example for demonstration purposes only.
// Redistribution or sharing of this source code is not permitted.
//
// License: Proprietary - All Rights Reserved
// For licensing inquiries, please contact: caokhang91@gmail.com

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/dashboard"
)

// portFromEnv returns listen address like ":8080" from env key, or defaultAddr if unset/invalid.
func portFromEnv(envKey, defaultAddr string) string {
	s := os.Getenv(envKey)
	if s == "" {
		return defaultAddr
	}
	if _, err := strconv.Atoi(s); err != nil {
		return defaultAddr
	}
	return ":" + s
}

func main() {
	log.Println("============================================================")
	log.Println("  Fluxor Dashboard Example")
	log.Println("============================================================")
	log.Println("This example demonstrates:")
	log.Println("  1. Deploying DashboardVerticle (reusable component)")
	log.Println("  2. Creating HTTP server with metrics")
	log.Println("  3. Registering executors for metrics collection")
	log.Println("  4. Viewing real-time metrics in dashboard")
	log.Println("============================================================")

	// Create MainVerticle
	app, err := entrypoint.NewMainVerticle("")
	if err != nil {
		log.Fatalf("Failed to create main verticle: %v", err)
	}

	// Ports configurable via env (e.g. when 8080/9090 are in use)
	dashboardAddr := portFromEnv("DASHBOARD_PORT", ":8080")
	httpMetricsAddr := portFromEnv("HTTP_METRICS_PORT", ":9090")

	// Deploy a test verticle that demonstrates metrics collection
	testVerticle := NewTestMetricsVerticle()
	if _, err := app.DeployVerticle(testVerticle); err != nil {
		log.Fatalf("Failed to deploy test metrics verticle: %v", err)
	}

	// Deploy HTTP server verticle with metrics (use configured port)
	httpVerticle := NewHTTPMetricsVerticleWithAddr(httpMetricsAddr)
	if _, err := app.DeployVerticle(httpVerticle); err != nil {
		log.Fatalf("Failed to deploy HTTP metrics verticle: %v", err)
	}

	// Deploy dashboard verticle (reusable component)
	// Dashboard will automatically collect metrics from registered HTTP servers
	dashboardConfig := dashboard.DashboardVerticleConfig{
		Address: dashboardAddr,
		Prefix:  "", // No prefix (routes at root)
	}
	dashboardVerticle := dashboard.NewDashboardVerticleWithConfig(dashboardConfig)
	if _, err := app.DeployVerticle(dashboardVerticle); err != nil {
		log.Fatalf("Failed to deploy dashboard verticle: %v", err)
	}

	dashboardPort := dashboardAddr
	if dashboardPort[0] == ':' {
		dashboardPort = dashboardPort[1:]
	}
	httpMetricsPort := httpMetricsAddr
	if httpMetricsPort[0] == ':' {
		httpMetricsPort = httpMetricsPort[1:]
	}

	log.Println("")
	log.Println("============================================================")
	log.Println("  Dashboard Example is running")
	log.Println("============================================================")
	log.Printf("  Dashboard UI: http://localhost:%s/dashboard", dashboardPort)
	log.Printf("  Metrics API: http://localhost:%s/api/dashboard/metrics", dashboardPort)
	log.Printf("  Health API:  http://localhost:%s/api/dashboard/health", dashboardPort)
	log.Printf("  Test API:    http://localhost:%s/api/test", httpMetricsPort)
	log.Println("")
	log.Println("Press Ctrl+C to shutdown")
	log.Println("============================================================")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Channel to catch app errors
	errChan := make(chan error, 1)

	// Start app in background
	go func() {
		if err := app.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		log.Fatalf("Application error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received %v, shutting down...", sig)
	}

	if err := app.Stop(); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}

	log.Println("Dashboard Example shutdown complete")
}

// TestMetricsVerticle demonstrates how to register executors for metrics collection
// This simulates a real application verticle that uses executors
type TestMetricsVerticle struct {
	*core.BaseVerticle
	executor concurrency.Executor
}

// NewTestMetricsVerticle creates a new test metrics verticle
func NewTestMetricsVerticle() *TestMetricsVerticle {
	return &TestMetricsVerticle{
		BaseVerticle: core.NewBaseVerticle("test-metrics"),
	}
}

// Start sets up test executor with metrics collection
func (v *TestMetricsVerticle) Start(ctx core.FluxorContext) error {
	// Start base verticle first
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	logger := core.NewDefaultLogger()
	logger.Info("TestMetricsVerticle starting...")

	// Create executor config
	executorConfig := concurrency.DefaultExecutorConfig()
	executorConfig.Workers = 5
	executorConfig.QueueSize = 100

	// Create executor with automatic metrics registration using helper function
	v.executor = concurrency.NewExecutorWithMetrics(
		ctx.GoCMD().Context(),
		"dashboard-executor", // ID for metrics
		executorConfig,
	)

	logger.Info("Test executor created and registered for metrics collection")

	// Submit periodic tasks: short tasks to executor (for metrics), blocking work via SubmitBlocking.
	// Executor tasks must complete in < 20µs; use verticle SubmitBlocking() for blocking work.
	v.ExecuteOn(func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.GoCMD().Context().Done():
				return
			case <-ticker.C:
				// Short task to executor (< 20µs) - keeps executor metrics active
				shortTask := concurrency.TaskFunc(func(ctx context.Context) error {
					return nil
				})
				if err := v.executor.Submit(shortTask); err != nil {
					logger.Error("Failed to submit task: " + err.Error())
				}
				// Blocking work (e.g. I/O, CPU-intensive) on verticle worker pool via SubmitBlocking
				core.SubmitBlockingFunc(v.BaseVerticle, func() (struct{}, error) {
					time.Sleep(20 * time.Millisecond)
					return struct{}{}, nil
				})
			}
		}
	})

	logger.Info("TestMetricsVerticle started successfully")
	return nil
}

// Stop stops the executor
func (v *TestMetricsVerticle) Stop(ctx core.FluxorContext) error {
	logger := core.NewDefaultLogger()
	logger.Info("TestMetricsVerticle stopping...")

	// Stop executor if it exists
	if v.executor != nil {
		logger.Info("Shutting down executor...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := v.executor.Shutdown(shutdownCtx); err != nil {
			logger.Error("Error shutting down executor: " + err.Error())
		}
		concurrency.UnregisterExecutor("dashboard-executor")
		v.executor = nil
	}

	return v.BaseVerticle.Stop(ctx)
}

// HTTPMetricsVerticle demonstrates HTTP server with metrics
type HTTPMetricsVerticle struct {
	*core.BaseVerticle
	server    *web.FastHTTPServer
	listenAddr string
}

// NewHTTPMetricsVerticle creates a new HTTP metrics verticle (listens on :9090).
func NewHTTPMetricsVerticle() *HTTPMetricsVerticle {
	return NewHTTPMetricsVerticleWithAddr(":9090")
}

// NewHTTPMetricsVerticleWithAddr creates a new HTTP metrics verticle listening on the given address.
func NewHTTPMetricsVerticleWithAddr(addr string) *HTTPMetricsVerticle {
	if addr == "" {
		addr = ":9090"
	}
	return &HTTPMetricsVerticle{
		BaseVerticle: core.NewBaseVerticle("http-metrics"),
		listenAddr:   addr,
	}
}

// Start sets up HTTP server
func (v *HTTPMetricsVerticle) Start(ctx core.FluxorContext) error {
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	logger := core.NewDefaultLogger()
	logger.Info("HTTPMetricsVerticle starting...")

	// Create HTTP server config
	serverConfig := web.DefaultFastHTTPServerConfig(v.listenAddr)
	v.server = web.NewFastHTTPServer(ctx.GoCMD(), serverConfig)
	router := v.server.FastRouter()

	// Register test endpoint
	router.GETFast("/api/test", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{
			"status":  "ok",
			"message": "Test endpoint for metrics collection",
			"time":    time.Now().Unix(),
		})
	})

	// HTTP server will be automatically registered with dashboard
	// via auto-registration in dashboard package

	// Start server in background
	go func() {
		logger.Info("HTTP server starting on " + v.listenAddr)
		if err := v.server.Start(); err != nil {
			logger.Error(fmt.Sprintf("HTTP server error: %v", err))
		}
	}()

	logger.Info("HTTPMetricsVerticle started successfully")
	return nil
}

// Stop stops the HTTP server
func (v *HTTPMetricsVerticle) Stop(ctx core.FluxorContext) error {
	logger := core.NewDefaultLogger()
	logger.Info("HTTPMetricsVerticle stopping...")

	if v.server != nil {
		if err := v.server.Stop(); err != nil {
			logger.Error(fmt.Sprintf("Error stopping HTTP server: %v", err))
			return err
		}
	}

	return v.BaseVerticle.Stop(ctx)
}
