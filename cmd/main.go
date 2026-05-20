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
	"reflect"
	"syscall"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
	"github.com/fluxorio/fluxor/pkg/fx"
	"github.com/fluxorio/fluxor/pkg/observability/prometheus"
	"github.com/fluxorio/fluxor/pkg/web"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Fluxor application with dependency injection
	app, err := fx.New(ctx,
		fx.Provide(fx.NewValueProvider("example-config")), // Optional custom providers
		fx.Invoke(fx.NewInvoker(setupApplication)),
	)
	if err != nil {
		log.Fatalf("Failed to create Fluxor app: %v", err)
	}

	// Start the application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start Fluxor app: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")

	if err := app.Stop(); err != nil {
		log.Fatalf("Error stopping app: %v", err)
	}
}

// setupApplication initializes the application components
func setupApplication(deps map[reflect.Type]interface{}) error {
	gocmd := deps[reflect.TypeOf((*core.GoCMD)(nil)).Elem()].(core.GoCMD)
	eventBus := deps[reflect.TypeOf((*core.EventBus)(nil)).Elem()].(core.EventBus)

	// Deploy example verticle
	verticle := &ExampleVerticle{eventBus: eventBus}
	deploymentID, err := gocmd.DeployVerticle(verticle)
	if err != nil {
		return fmt.Errorf("failed to deploy verticle: %w", err)
	}
	log.Printf("Deployed verticle: %s", deploymentID)

	// Create and start FastHTTP server with 67% utilization target
	// Normal runtime: Operates at 67% capacity (3350 CCU for 5000 max)
	// Headroom: 33% available for traffic spikes
	// Overflow: Requests beyond normal capacity get 503 (fail-fast backpressure)
	// This prevents system crash and maintains stable resource utilization
	maxCCU := 5000
	utilizationPercent := 67
	config := web.CCUBasedConfigWithUtilization(":8080", maxCCU, utilizationPercent)
	normalCapacity := config.MaxQueue + config.Workers
	log.Printf("Server configured: max=%d CCU, normal=%d CCU (%.0f%% utilization), workers=%d, queue=%d",
		maxCCU, normalCapacity, float64(utilizationPercent), config.Workers, config.MaxQueue)

	server := web.NewFastHTTPServer(gocmd, config)

	// Setup routes with JSON as default using fast router
	router := server.FastRouter()

	// Simple JSON response
	router.GETFast("/", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{
			"message": "Hello from Fluxor!",
			"format":  "json",
		})
	})

	// Status endpoint with JSON
	router.GETFast("/api/status", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{
			"status": "ok",
			"time":   time.Now().Unix(),
			"rps":    "100k target",
		})
	})

	// Add metrics middleware to router
	router.UseFast(prometheus.FastHTTPMetricsMiddleware())

	// Prometheus metrics endpoint — serves from custom registry (includes fluxor_* + go_* metrics)
	router.GETFast("/metrics", prometheus.FastHTTPHandler())

	// Start metrics updater
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			prometheus.UpdateServerMetrics(server)
		}
	}()

	// Liveness check endpoint (formerly /health)
	router.GETFast("/live", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{
			"status":    "up",
			"timestamp": time.Now().Unix(),
		})
	})

	// Keep /health as alias for /live for backward compatibility
	router.GETFast("/health", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{
			"status":    "up",
			"timestamp": time.Now().Unix(),
		})
	})

	// Readiness check endpoint
	router.GETFast("/ready", func(ctx *web.FastRequestContext) error {
		metrics := server.Metrics()
		// Consider ready if queue utilization is below 90% and backpressure not active
		// Also check gocmd deployment count > 0 (example check)
		gocmd := ctx.GoCMD
		ready := metrics.QueueUtilization < 90.0 && metrics.CCUUtilization < 90.0 && gocmd.DeploymentCount() > 0
		statusCode := 200
		status := "ready"
		if !ready {
			statusCode = 503
			status = "not_ready"
		}
		return ctx.JSON(statusCode, map[string]interface{}{
			"status":            status,
			"ready":             ready,
			"queue_utilization": metrics.QueueUtilization,
			"ccu_utilization":   metrics.CCUUtilization,
			"verticle_count":    gocmd.DeploymentCount(),
		})
	})

	// API metrics endpoint (JSON) - preserved for compatibility but discouraged in favor of /metrics
	router.GETFast("/api/metrics", func(ctx *web.FastRequestContext) error {
		metrics := server.Metrics()
		return ctx.JSON(200, map[string]interface{}{
			"queued_requests":     metrics.QueuedRequests,
			"rejected_requests":   metrics.RejectedRequests,
			"queue_capacity":      metrics.QueueCapacity,
			"queue_utilization":   fmt.Sprintf("%.2f%%", metrics.QueueUtilization),
			"workers":             metrics.Workers,
			"normal_ccu":          metrics.NormalCCU,
			"current_ccu":         metrics.CurrentCCU,
			"ccu_utilization":     fmt.Sprintf("%.2f%%", metrics.CCUUtilization),
			"backpressure_active": metrics.CCUUtilization >= 100.0,
			"total_requests":      metrics.TotalRequests,
			"successful_requests": metrics.SuccessfulRequests,
			"error_requests":      metrics.ErrorRequests,
			"request_id":          ctx.RequestID(),
		})
	})

	// Echo endpoint - demonstrates JSON request/response
	router.POSTFast("/api/echo", func(ctx *web.FastRequestContext) error {
		var data map[string]interface{}
		if err := ctx.BindJSON(&data); err != nil {
			return ctx.JSON(400, map[string]interface{}{
				"error": "invalid json",
			})
		}

		return ctx.JSON(200, map[string]interface{}{
			"echo":    data,
			"message": "Echo successful",
		})
	})

	// Set default handler for unmatched routes (like nginx default_server)
	router.SetDefaultHandler(func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{
			"message": "Welcome to Fluxor",
			"status":  "ok",
			"server":  "fluxor",
			"version": "1.0.0",
		})
	})

	// Start server in goroutine
	go func() {
		log.Printf("Starting FastHTTP server on %s for 100k RPS", config.Addr)
		if err := server.Start(); err != nil {
			log.Printf("FastHTTP server error: %v", err)
		}
	}()

	// Create runtime and execute example task
	// TODO: Fix undefined fluxor.NewRuntime - this code is commented out until the correct implementation is available
	// runtime := fluxor.NewRuntime(context.Background())
	// task := &ExampleTask{name: "example-task"}
	// if err := runtime.Execute(task); err != nil {
	// 	log.Printf("Failed to execute task: %v", err)
	// }

	// Example workflow
	workflow := createExampleWorkflow()
	if err := workflow.Execute(context.Background()); err != nil {
		log.Printf("Workflow execution error: %v", err)
	}

	// Example reactive pattern
	reactiveExample(eventBus)

	return nil
}

// ExampleVerticle is an example verticle
type ExampleVerticle struct {
	eventBus core.EventBus
}

func (v *ExampleVerticle) Start(ctx core.FluxorContext) error {
	log.Println("ExampleVerticle started")

	// Register event bus consumer
	consumer := ctx.EventBus().Consumer("example.address")
	consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
		log.Printf("Received message: %v", msg.Body())
		return msg.Reply("acknowledged")
	})

	return nil
}

func (v *ExampleVerticle) Stop(ctx core.FluxorContext) error {
	log.Println("ExampleVerticle stopped")
	return nil
}

// ExampleTask is an example task
type ExampleTask struct {
	name string
}

func (t *ExampleTask) Name() string {
	return t.name
}

func (t *ExampleTask) Execute(ctx context.Context) error {
	log.Printf("Executing task: %s", t.name)
	time.Sleep(100 * time.Millisecond)
	log.Printf("Task %s completed", t.name)
	return nil
}

// createExampleWorkflow creates an example workflow
func createExampleWorkflow() entrypoint.Workflow {
	step1 := entrypoint.NewStep("step1", func(ctx context.Context, data interface{}) (interface{}, error) {
		log.Println("Executing step 1")
		return "step1-result", nil
	})

	step2 := entrypoint.NewStep("step2", func(ctx context.Context, data interface{}) (interface{}, error) {
		log.Printf("Executing step 2 with data: %v", data)
		return "step2-result", nil
	})

	return entrypoint.NewWorkflow("example-workflow", step1, step2)
}

// reactiveExample demonstrates reactive patterns
func reactiveExample(eventBus core.EventBus) {
	// Create a future
	future := entrypoint.NewFuture()

	// Register handlers
	future.OnSuccess(func(result interface{}) {
		log.Printf("Future succeeded with: %v", result)
	})

	future.OnFailure(func(err error) {
		log.Printf("Future failed: %v", err)
	})

	// Simulate async operation
	go func() {
		time.Sleep(100 * time.Millisecond)
		future.Complete("async-result")
	}()

	// Example promise
	promise := entrypoint.NewPromise()
	promise.OnSuccess(func(result interface{}) {
		log.Printf("Promise succeeded: %v", result)
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		promise.Complete("promise-result")
	}()
}
