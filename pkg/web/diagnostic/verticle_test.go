package diagnostic

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// testFluxorContext is a test helper that implements FluxorContext
type testFluxorContext struct {
	goCtx  context.Context
	gocmd  core.GoCMD
	config map[string]interface{}
}

func (c *testFluxorContext) Context() context.Context { return c.goCtx }
func (c *testFluxorContext) EventBus() core.EventBus { return c.gocmd.EventBus() }
func (c *testFluxorContext) GoCMD() core.GoCMD        { return c.gocmd }
func (c *testFluxorContext) Config() map[string]interface{} {
	if c.config == nil {
		c.config = make(map[string]interface{})
	}
	return c.config
}
func (c *testFluxorContext) SetConfig(key string, value interface{}) {
	if c.config == nil {
		c.config = make(map[string]interface{})
	}
	c.config[key] = value
}
func (c *testFluxorContext) Deploy(v core.Verticle) (string, error) {
	return c.gocmd.DeployVerticle(v)
}
func (c *testFluxorContext) Undeploy(id string) error {
	return c.gocmd.UndeployVerticle(id)
}
func (c *testFluxorContext) DeploymentID() string { return "" }

func newTestFluxorContext(gocmd core.GoCMD) core.FluxorContext {
	return &testFluxorContext{
		goCtx: context.Background(),
		gocmd: gocmd,
	}
}

func TestDefaultDiagnosticVerticleConfig(t *testing.T) {
	config := DefaultDiagnosticVerticleConfig()

	if config.Address != ":8080" {
		t.Errorf("Expected default address ':8080', got '%s'", config.Address)
	}

	if config.Prefix != "" {
		t.Errorf("Expected default prefix '', got '%s'", config.Prefix)
	}
}

func TestNewDiagnosticVerticle(t *testing.T) {
	verticle := NewDiagnosticVerticle()

	if verticle == nil {
		t.Fatal("NewDiagnosticVerticle() should not return nil")
	}

	if verticle.Name() != "diagnostic" {
		t.Errorf("Expected name 'diagnostic', got '%s'", verticle.Name())
	}
}

func TestNewDiagnosticVerticleWithConfig(t *testing.T) {
	config := DiagnosticVerticleConfig{
		Address: ":9090",
		Prefix:  "/admin",
	}

	verticle := NewDiagnosticVerticleWithConfig(config)

	if verticle == nil {
		t.Fatal("NewDiagnosticVerticleWithConfig() should not return nil")
	}

	if verticle.config.Address != ":9090" {
		t.Errorf("Expected address ':9090', got '%s'", verticle.config.Address)
	}

	if verticle.config.Prefix != "/admin" {
		t.Errorf("Expected prefix '/admin', got '%s'", verticle.config.Prefix)
	}
}

func TestDiagnosticVerticle_Start(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DiagnosticVerticleConfig{
		Address: ":0", // Use :0 for random port
		Prefix:  "",
	}

	verticle := NewDiagnosticVerticleWithConfig(config)
	fluxorCtx := newTestFluxorContext(gocmd)

	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	// Verify server was created
	if verticle.server == nil {
		t.Error("Server should be created on Start()")
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	verticle.Stop(fluxorCtx)
}

func TestDiagnosticVerticle_Start_DefaultAddress(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DiagnosticVerticleConfig{
		Address: "", // Empty address should use default
		Prefix:  "",
	}

	verticle := NewDiagnosticVerticleWithConfig(config)
	fluxorCtx := newTestFluxorContext(gocmd)

	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	// Cleanup
	verticle.Stop(fluxorCtx)
}

func TestDiagnosticVerticle_Start_WithPrefix(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DiagnosticVerticleConfig{
		Address: ":0",
		Prefix:  "/admin",
	}

	verticle := NewDiagnosticVerticleWithConfig(config)
	fluxorCtx := newTestFluxorContext(gocmd)

	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	// Cleanup
	verticle.Stop(fluxorCtx)
}

func TestDiagnosticVerticle_Stop(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DiagnosticVerticleConfig{
		Address: ":0",
		Prefix:  "",
	}

	verticle := NewDiagnosticVerticleWithConfig(config)
	fluxorCtx := newTestFluxorContext(gocmd)

	// Start first
	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Stop
	err = verticle.Stop(fluxorCtx)
	if err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

func TestDiagnosticVerticle_Stop_WithoutStart(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	verticle := NewDiagnosticVerticle()
	fluxorCtx := newTestFluxorContext(gocmd)

	// Stop without starting should not panic
	err := verticle.Stop(fluxorCtx)
	if err != nil {
		t.Logf("Stop() without start returned: %v (may be expected)", err)
	}
}

func TestDiagnosticVerticle_BaseVerticle(t *testing.T) {
	verticle := NewDiagnosticVerticle()

	if verticle.BaseVerticle == nil {
		t.Error("BaseVerticle should not be nil")
	}

	if verticle.Name() != "diagnostic" {
		t.Errorf("Expected name 'diagnostic', got '%s'", verticle.Name())
	}
}

func TestDiagnosticVerticle_RoutesRegistered(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := DiagnosticVerticleConfig{
		Address: ":0",
		Prefix:  "",
	}

	verticle := NewDiagnosticVerticleWithConfig(config)
	fluxorCtx := newTestFluxorContext(gocmd)

	err := verticle.Start(fluxorCtx)
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	// Verify server has router
	if verticle.server == nil {
		t.Fatal("Server should be created")
	}

	router := verticle.server.FastRouter()
	if router == nil {
		t.Fatal("Router should not be nil")
	}

	// Cleanup
	verticle.Stop(fluxorCtx)
}
