package eureka_test

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/eureka"
	"github.com/fluxorio/fluxor/pkg/entrypoint"
)

// ExampleServer demonstrates how to start a Eureka server
func ExampleServer() {
	// Create main application
	app, _ := entrypoint.NewMainVerticle("config.json")

	// Create Eureka server configuration
	serverConfig := eureka.DefaultServerConfig()
	serverConfig.Address = ":8761"

	// Create and deploy server verticle
	server := eureka.NewServerVerticle(serverConfig)
	app.DeployVerticle(server)

	// Start application
	app.Start()
	defer app.Stop()
}

// ExampleClient demonstrates how to register a service
func ExampleClient() {
	// Create service instance
	instance := &eureka.ServiceInstance{
		ServiceName: "my-service",
		Host:        "localhost",
		Port:        8080,
		Status:      eureka.InstanceStatusUp,
		Metadata: map[string]string{
			"version": "1.0.0",
		},
	}

	// Create client
	clientConfig := eureka.DefaultClientConfig("http://localhost:8761", instance)
	client := eureka.NewClient(clientConfig)

	// Register
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Register(ctx); err != nil {
		fmt.Printf("Registration failed: %v\n", err)
		return
	}

	// Start heartbeat
	heartbeatCtx := context.Background()
	if err := client.StartHeartbeat(heartbeatCtx); err != nil {
		fmt.Printf("Failed to start heartbeat: %v\n", err)
		return
	}

	// Later, unregister on shutdown
	defer func() {
		client.StopHeartbeat()
		unregisterCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Unregister(unregisterCtx)
	}()
}

// ExampleServiceVerticle demonstrates a complete service verticle
type ExampleServiceVerticle struct {
	*core.BaseVerticle
	client *eureka.Client
}

func NewExampleServiceVerticle() *ExampleServiceVerticle {
	instance := &eureka.ServiceInstance{
		ServiceName: "example-service",
		Host:        "localhost",
		Port:        8080,
		Status:      eureka.InstanceStatusUp,
	}

	clientConfig := eureka.DefaultClientConfig("http://localhost:8761", instance)
	client := eureka.NewClient(clientConfig)

	return &ExampleServiceVerticle{
		BaseVerticle: core.NewBaseVerticle("example-service"),
		client:       client,
	}
}

func (v *ExampleServiceVerticle) Start(ctx core.FluxorContext) error {
	if err := v.BaseVerticle.Start(ctx); err != nil {
		return err
	}

	// Register with Eureka
	registerCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := v.client.Register(registerCtx); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	// Start heartbeat
	if err := v.client.StartHeartbeat(ctx.GoCMD().Context()); err != nil {
		return fmt.Errorf("failed to start heartbeat: %w", err)
	}

	return nil
}

func (v *ExampleServiceVerticle) Stop(ctx core.FluxorContext) error {
	// Stop heartbeat
	v.client.StopHeartbeat()

	// Unregister
	unregisterCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = v.client.Unregister(unregisterCtx)

	return v.BaseVerticle.Stop(ctx)
}
