package entrypoint

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
)

type cfgCheckVerticle struct{}

func (v *cfgCheckVerticle) Start(ctx core.FluxorContext) error {
	if got := ctx.Config()["foo"]; got != "bar" {
		return &core.EventBusError{Code: "CONFIG_MISSING", Message: "expected foo=bar in context config"}
	}
	return nil
}

func (v *cfgCheckVerticle) Stop(ctx core.FluxorContext) error { return nil }

func TestMainVerticle_LoadConfig_AndInjectOnDeploy(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"foo":"bar"}`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	app, err := NewMainVerticle(cfgPath)
	if err != nil {
		t.Fatalf("NewMainVerticle: %v", err)
	}
	defer app.Stop()

	if _, err := app.DeployVerticle(&cfgCheckVerticle{}); err != nil {
		t.Fatalf("DeployVerticle: %v", err)
	}
}

func TestNewMainVerticle_FailFast_ConfigLoadError(t *testing.T) {
	// Non-existent config path should fail immediately.
	_, err := NewMainVerticle("does-not-exist.json")
	if err == nil {
		t.Fatalf("expected error for missing config file")
	}
}

func TestMainVerticle_DeployVerticle_FailFast_NilVerticle(t *testing.T) {
	app, err := NewMainVerticle("")
	if err != nil {
		t.Fatalf("NewMainVerticle: %v", err)
	}
	defer app.Stop()

	_, err = app.DeployVerticle(nil)
	if err == nil {
		t.Fatalf("expected error for nil verticle")
	}
	if ce, ok := err.(*core.EventBusError); ok {
		if ce.Code != "INVALID_INPUT" {
			t.Fatalf("error code = %q, want %q", ce.Code, "INVALID_INPUT")
		}
	}
}

func TestNewMainVerticleWithOptions_FailFast_EventBusFactoryError(t *testing.T) {
	_, err := NewMainVerticleWithOptions("", MainVerticleOptions{
		EventBusFactory: func(ctx context.Context, gocmd core.GoCMD, cfg map[string]any) (core.EventBus, error) {
			return nil, &core.EventBusError{Code: "TEST", Message: "boom"}
		},
	})
	if err == nil {
		t.Fatalf("expected error when EventBusFactory returns error")
	}
}
