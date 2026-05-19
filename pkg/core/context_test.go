package core

import (
	"context"
	"testing"
	"time"
)

func TestNewFluxorContext(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	fluxorCtx := newFluxorContext(ctx, gocmd)

	if fluxorCtx == nil {
		t.Error("newFluxorContext() should not return nil")
	}

	if fluxorCtx.Context() == nil {
		t.Error("Context() should not return nil")
	}

	if fluxorCtx.GoCMD() == nil {
		t.Error("GoCMD() should not return nil")
	}

	if fluxorCtx.EventBus() == nil {
		t.Error("EventBus() should not return nil")
	}
}

func TestFluxorContext_Config(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	fluxorCtx := newFluxorContext(ctx, gocmd)

	// Test setting config
	fluxorCtx.SetConfig("key1", "value1")
	fluxorCtx.SetConfig("key2", 42)

	config := fluxorCtx.Config()
	if config["key1"] != "value1" {
		t.Errorf("Config() key1 = %v, want value1", config["key1"])
	}

	if config["key2"] != 42 {
		t.Errorf("Config() key2 = %v, want 42", config["key2"])
	}
}

func TestFluxorContext_Deploy(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	fluxorCtx := newFluxorContext(ctx, gocmd)

	verticle := &testVerticle{}
	deploymentID, err := fluxorCtx.Deploy(verticle)

	if err != nil {
		t.Errorf("Deploy() error = %v", err)
	}

	if deploymentID == "" {
		t.Error("Deploy() returned empty deployment ID")
	}

	// Wait for async start to complete
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if verticle.isStarted() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !verticle.isStarted() {
		t.Error("Verticle should be started")
	}
}

func TestFluxorContext_Undeploy(t *testing.T) {
	ctx := context.Background()
	gocmd := NewGoCMD(ctx)
	defer gocmd.Close()

	fluxorCtx := newFluxorContext(ctx, gocmd)

	verticle := &testVerticle{}
	deploymentID, err := fluxorCtx.Deploy(verticle)
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	// Wait for async start to complete before undeploying
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if verticle.isStarted() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	err = fluxorCtx.Undeploy(deploymentID)
	if err != nil {
		t.Errorf("Undeploy() error = %v", err)
	}

	// Wait for async stop to complete
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if verticle.isStopped() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !verticle.isStopped() {
		t.Error("Verticle should be stopped")
	}
}
