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
	clusterbus "github.com/fluxorio/fluxor/pkg/core/cluster/eventbus"
	"context"
	"fmt"
	"os"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/entrypoint"

	"github.com/fluxorio/fluxor/examples/fluxor-project/common/contracts"
)

// FlowListenerVerticle listens to events from flox service
type FlowListenerVerticle struct {
	*core.BaseVerticle
	bus core.EventBus
}

func NewFlowListenerVerticle() *FlowListenerVerticle {
	return &FlowListenerVerticle{
		BaseVerticle: core.NewBaseVerticle("flow-listener"),
	}
}

func (v *FlowListenerVerticle) Start(ctx core.FluxorContext) error {
	// Call base Start first to setup event loop
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	logger := core.NewDefaultLogger()
	logger.Info("FlowListenerVerticle Started")

	v.bus = ctx.EventBus()

	// Listen to flox.process events
	logger.Info("Starting EventBus consumer on address:", contracts.AddressFloxProcess)
	consumer := v.Consumer(contracts.AddressFloxProcess)
	consumer.Handler(func(c core.FluxorContext, msg core.Message) error {
		body, ok := msg.Body().([]byte)
		if !ok {
			logger.Error("Invalid payload type")
			return msg.Reply(contracts.FloxProcessReply{OK: false, Error: "invalid_request"})
		}

		var req contracts.FloxProcessRequest
		if err := core.JSONDecode(body, &req); err != nil {
			logger.Error("Invalid JSON", "error", err)
			return msg.Reply(contracts.FloxProcessReply{OK: false, Error: "invalid_request"})
		}

		// Process the flow event
		logger.Info("FlowListener received event", "requestId", req.RequestID, "data", req.Data)

		// Reply with success
		return msg.Reply(contracts.FloxProcessReply{
			OK:        true,
			RequestID: req.RequestID,
			Result:    "processed_by_flow_listener",
		})
	})

	// Listen to logs events
	logger.Info("Starting EventBus consumer on address:", contracts.AddressLogs)
	logsConsumer := v.Consumer(contracts.AddressLogs)
	logsConsumer.Handler(func(c core.FluxorContext, msg core.Message) error {
		body, ok := msg.Body().([]byte)
		if !ok {
			return nil // Ignore invalid logs
		}

		var logEvent contracts.LogEvent
		if err := core.JSONDecode(body, &logEvent); err != nil {
			return nil // Ignore invalid logs
		}

		logger.Info("FlowListener received log", "service", logEvent.Service, "message", logEvent.Message)
		return nil
	})

	logger.Info("FlowListenerVerticle ready - listening to events")
	return nil
}

func (v *FlowListenerVerticle) Stop(ctx core.FluxorContext) error {
	logger := core.NewDefaultLogger()
	logger.Info("FlowListenerVerticle Stopped")
	return v.BaseVerticle.Stop(ctx)
}

func main() {
	logger := core.NewDefaultLogger()
	logger.Info("Starting Fluxor Stream...")

	// Create runtime with embedded NATS server and NATS EventBus
	app, err := entrypoint.NewMainVerticleWithOptions("config.json", entrypoint.WithOptions(entrypoint.MainVerticleOptions{
		// BootstrapHook starts embedded NATS server before EventBus creation
		BootstrapHook: entrypoint.StartEmbeddedNATS,
		// EventBusFactory reads nats_url from config (set by BootstrapHook)
		EventBusFactory: func(ctx context.Context, gocmd core.GoCMD, cfg map[string]any) (core.EventBus, error) {
			natsCfg, _ := cfg["nats"].(map[string]any)
			prefix, _ := natsCfg["prefix"].(string)
			if prefix == "" {
				prefix = "fluxor.demo"
			}
			// Read nats_url from config (set by BootstrapHook)
			natsURL, _ := cfg["nats_url"].(string)
			eventBus, err := clusterbus.NewClusterEventBusJetStream(ctx, gocmd, clusterbus.ClusterJetStreamConfig{
				URL:     natsURL,
				Prefix:  prefix,
				Service: "fluxor",
			})
			if err == nil {
				core.Info(fmt.Sprintf("Embedded NATS started at %s (prefix: %s)", natsURL, prefix))
			}
			return eventBus, err
		},
	}))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create runtime: %v", err))
		os.Exit(1)
	}

	// Deploy FlowListenerVerticle
	logger.Info("Deploying FlowListenerVerticle...")
	_, err = app.DeployVerticle(NewFlowListenerVerticle())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deploy FlowListenerVerticle: %v", err))
		os.Exit(1)
	}
	logger.Info("FlowListenerVerticle deployed successfully")

	// Start application
	logger.Info("Starting application...")
	if err := app.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to start application: %v", err))
		os.Exit(1)
	}
	logger.Info("Fluxor Stream started successfully - listening to flow events")
}
