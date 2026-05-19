package web

import (
	"context"

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
