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
	"os"
	"time"

	"github.com/fluxorio/fluxor/pkg/cache"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
	"github.com/fluxorio/fluxor/pkg/queue"
)

// QueueRPCGRPCVerticle provides gRPC server for queue RPC operations
type QueueRPCGRPCVerticle struct {
	*core.BaseVerticle
	cache  cache.Cache
	server *queue.GRPCServer
}

func NewQueueRPCGRPCVerticle() *QueueRPCGRPCVerticle {
	return &QueueRPCGRPCVerticle{
		BaseVerticle: core.NewBaseVerticle("queue-rpc-grpc-server"),
	}
}

func (v *QueueRPCGRPCVerticle) Start(ctx core.FluxorContext) error {
	// Call base Start first to setup event loop
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	logger := core.NewDefaultLogger()
	logger.Info("QueueRPCGRPCVerticle Started")

	// Setup cache
	v.cache = cache.NewMemoryCache()

	// Pre-populate cache with some test data
	testCtx := context.Background()
	testData := map[string][]byte{
		"user:123":   []byte(`{"id": 123, "name": "John Doe", "email": "john@example.com"}`),
		"config:app": []byte(`{"app_name": "fluxor", "version": "1.0.0"}`),
		"data:test":  []byte(`{"key": "test", "value": "test_data"}`),
	}

	for key, data := range testData {
		if err := v.cache.Set(testCtx, key, data, 10*time.Minute); err != nil {
			logger.Error(fmt.Sprintf("Failed to set cache for %s: %v", key, err))
		} else {
			logger.Info(fmt.Sprintf("Pre-populated cache: %s", key))
		}
	}

	// Setup gRPC server
	address := getEnv("GRPC_ADDRESS", "localhost:50051")
	server, err := queue.NewGRPCServer(v.cache, address)
	if err != nil {
		return fmt.Errorf("failed to create gRPC server: %w", err)
	}
	v.server = server

	// Start gRPC server
	serverCtx := context.Background()
	if err := server.Start(serverCtx); err != nil {
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	logger.Info(fmt.Sprintf("gRPC Server started on %s", address))
	logger.Info("QueueRPCGRPCVerticle ready - listening to gRPC requests")
	return nil
}

func (v *QueueRPCGRPCVerticle) Stop(ctx core.FluxorContext) error {
	logger := core.NewDefaultLogger()
	logger.Info("QueueRPCGRPCVerticle Stopped")

	if v.server != nil {
		if err := v.server.Close(); err != nil {
			logger.Error(fmt.Sprintf("Error closing gRPC server: %v", err))
		}
	}

	return v.BaseVerticle.Stop(ctx)
}

func main() {
	logger := core.NewDefaultLogger()
	logger.Info("Starting Queue RPC gRPC runtime...")

	// Create runtime
	app, err := entrypoint.NewMainVerticle("")
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create runtime: %v", err))
		os.Exit(1)
	}

	// Deploy QueueRPCGRPCVerticle
	logger.Info("Deploying QueueRPCGRPCVerticle...")
	_, err = app.DeployVerticle(NewQueueRPCGRPCVerticle())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deploy QueueRPCGRPCVerticle: %v", err))
		os.Exit(1)
	}
	logger.Info("QueueRPCGRPCVerticle deployed successfully")

	// Start application
	logger.Info("Starting application...")
	if err := app.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to start application: %v", err))
		os.Exit(1)
	}

	logger.Info("Queue RPC gRPC runtime started successfully - listening on gRPC port")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
