package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/fluxorio/fluxor/pkg/web"
)

// WorkflowVerticle provides workflow engine as a deployable verticle.
type WorkflowVerticle struct {
	engine           *Engine
	functionRegistry *FunctionRegistry
	server           *web.FastHTTPServer
	httpAddr         string
	executor         concurrency.Executor
}

// WorkflowVerticleConfig configures the workflow verticle.
type WorkflowVerticleConfig struct {
	// HTTPAddr enables HTTP API for workflow management (e.g., ":8081")
	HTTPAddr string

	// Workflows to register on start
	Workflows []*WorkflowDefinition

	// EventTriggers to set up on start
	EventTriggers []EventTriggerConfig
}

// NewWorkflowVerticle creates a new workflow verticle.
func NewWorkflowVerticle(config *WorkflowVerticleConfig) *WorkflowVerticle {
	v := &WorkflowVerticle{
		functionRegistry: NewFunctionRegistry(),
	}
	if config != nil {
		v.httpAddr = config.HTTPAddr
	}
	return v
}

// RegisterFunction registers a custom function for use in function nodes.
func (v *WorkflowVerticle) RegisterFunction(name string, fn func(data interface{}) (interface{}, error)) {
	v.functionRegistry.Register(name, func(_ context.Context, data interface{}) (interface{}, error) {
		return fn(data)
	})
}

// Engine returns the workflow engine.
func (v *WorkflowVerticle) Engine() *Engine {
	return v.engine
}

// Start implements core.Verticle.
func (v *WorkflowVerticle) Start(ctx core.FluxorContext) error {
	// Create workflow engine with EventBus
	v.engine = NewEngine(ctx.EventBus())

	// Create executor for blocking operations (e.g., ifthenelse inline nodes)
	executorConfig := concurrency.DefaultExecutorConfig()
	v.executor = concurrency.NewExecutor(ctx.Context(), executorConfig)

	// Register node handlers that require runtime dependencies
	v.engine.RegisterNodeHandler(NodeTypeHTTP, HTTPNodeHandler)
	v.engine.RegisterNodeHandler(NodeTypeOpenAI, OpenAINodeHandler)
	v.engine.RegisterNodeHandler(NodeTypeAI, AINodeHandler) // Generic AI node (supports Cursor, Anthropic, etc.)
	v.engine.RegisterNodeHandler(NodeTypeSubWorkflow, CreateSubWorkflowHandler(v.engine))
	v.engine.RegisterNodeHandler(NodeTypeDynamicLoop, DynamicLoopNodeHandler)
	v.engine.RegisterNodeHandler(NodeTypeEventBus, CreateEventBusHandler(ctx.EventBus()))
	v.engine.RegisterNodeHandler(NodeTypeFunction, CreateFunctionHandler(v.functionRegistry))
	v.engine.RegisterNodeHandler(NodeTypeCode, CodeNodeHandler)
	v.engine.RegisterNodeHandler(NodeTypeIfThenElse, CreateIfThenElseHandler(v.engine.Registry(), v.executor))
	v.engine.RegisterNodeHandler("filter", FilterNodeHandler)
	v.engine.RegisterNodeHandler("map", MapNodeHandler(v.functionRegistry))
	v.engine.RegisterNodeHandler("reduce", ReduceNodeHandler(v.functionRegistry))

	// Register AI module nodes
	v.engine.RegisterNodeHandler(NodeType("aimodule.chat"), AIChatNodeHandler)
	v.engine.RegisterNodeHandler(NodeType("aimodule.embed"), AIEmbedNodeHandler)
	v.engine.RegisterNodeHandler(NodeType("aimodule.toolcall"), AIChatNodeHandler)

	// Register connector-backed nodes
	v.engine.RegisterNodeHandler(NodeType("zalo.zns.send"), ZaloZNSNodeHandler)

	// Load workflows from config
	if workflows, ok := ctx.Config()["workflows"].([]interface{}); ok {
		for _, wf := range workflows {
			if wfMap, ok := wf.(map[string]interface{}); ok {
				wfJSON, _ := json.Marshal(wfMap)
				var def WorkflowDefinition
				if err := json.Unmarshal(wfJSON, &def); err != nil {
					return fmt.Errorf("failed to parse workflow: %w", err)
				}
				if err := v.engine.RegisterWorkflow(&def); err != nil {
					return fmt.Errorf("failed to register workflow %s: %w", def.ID, err)
				}
			}
		}
	}

	// Start HTTP API if configured
	if v.httpAddr != "" {
		if err := v.startHTTPAPI(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Stop implements core.Verticle.
func (v *WorkflowVerticle) Stop(ctx core.FluxorContext) error {
	// Shutdown executor gracefully
	if v.executor != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx.Context(), 10*time.Second)
		defer cancel()
		if err := v.executor.Shutdown(shutdownCtx); err != nil {
			// Log error but continue with server shutdown
			// Error might be timeout, which is acceptable during shutdown
		}
	}

	if v.server != nil {
		return v.server.Stop()
	}
	return nil
}

func (v *WorkflowVerticle) startHTTPAPI(ctx core.FluxorContext) error {
	config := web.DefaultFastHTTPServerConfig(v.httpAddr)
	v.server = web.NewFastHTTPServer(ctx.GoCMD(), config)
	router := v.server.FastRouter()

	// List workflows
	router.GETFast("/workflows", func(c *web.FastRequestContext) error {
		workflows := v.engine.ListWorkflows()
		return c.JSON(200, map[string]interface{}{
			"workflows": workflows,
		})
	})

	// Register workflow
	router.POSTFast("/workflows", func(c *web.FastRequestContext) error {
		var def WorkflowDefinition
		if err := c.BindJSON(&def); err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}
		if err := v.engine.RegisterWorkflow(&def); err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}
		return c.JSON(201, map[string]interface{}{
			"id":      def.ID,
			"message": "workflow registered",
		})
	})

	// Execute workflow
	router.POSTFast("/workflows/:id/execute", func(c *web.FastRequestContext) error {
		workflowID := c.Param("id")
		var input interface{}
		if len(c.RequestCtx.PostBody()) > 0 {
			if err := json.Unmarshal(c.RequestCtx.PostBody(), &input); err != nil {
				return c.JSON(400, map[string]interface{}{"error": "invalid JSON body"})
			}
		}

		execID, err := v.engine.ExecuteWorkflow(c.Context(), workflowID, input)
		if err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}

		return c.JSON(202, map[string]interface{}{
			"executionId": execID,
			"workflowId":  workflowID,
		})
	})

	// Get execution status
	router.GETFast("/executions/:id", func(c *web.FastRequestContext) error {
		execID := c.Param("id")
		state, err := v.engine.GetExecutionState(execID)
		if err != nil {
			return c.JSON(404, map[string]interface{}{"error": err.Error()})
		}
		return c.JSON(200, state)
	})

	// Cancel execution
	router.POSTFast("/executions/:id/cancel", func(c *web.FastRequestContext) error {
		execID := c.Param("id")
		if err := v.engine.CancelExecution(execID); err != nil {
			return c.JSON(400, map[string]interface{}{"error": err.Error()})
		}
		return c.JSON(200, map[string]interface{}{
			"message": "execution cancelled",
		})
	})

	// Health check
	router.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{
			"status":    "ok",
			"workflows": len(v.engine.ListWorkflows()),
		})
	})

	go v.server.Start()
	return nil
}

// Quick workflow builder helpers

// WorkflowBuilder helps build workflow definitions programmatically.
type WorkflowBuilder struct {
	def *WorkflowDefinition
}

// NewWorkflowBuilder creates a new workflow builder.
func NewWorkflowBuilder(id, name string) *WorkflowBuilder {
	return &WorkflowBuilder{
		def: &WorkflowDefinition{
			ID:    id,
			Name:  name,
			Nodes: make([]NodeDefinition, 0),
		},
	}
}

// AddNode adds a node to the workflow.
func (b *WorkflowBuilder) AddNode(id, nodeType string) *NodeBuilder {
	node := NodeDefinition{
		ID:   id,
		Type: nodeType,
	}
	b.def.Nodes = append(b.def.Nodes, node)
	return &NodeBuilder{
		workflow: b,
		nodeIdx:  len(b.def.Nodes) - 1,
	}
}

// Build returns the workflow definition.
func (b *WorkflowBuilder) Build() *WorkflowDefinition {
	return b.def
}

// NodeBuilder helps configure a node.
type NodeBuilder struct {
	workflow *WorkflowBuilder
	nodeIdx  int
}

func (n *NodeBuilder) node() *NodeDefinition {
	return &n.workflow.def.Nodes[n.nodeIdx]
}

// Name sets the node name.
func (n *NodeBuilder) Name(name string) *NodeBuilder {
	n.node().Name = name
	return n
}

// Config sets the node configuration.
func (n *NodeBuilder) Config(config map[string]interface{}) *NodeBuilder {
	n.node().Config = config
	return n
}

// Next sets the next nodes.
func (n *NodeBuilder) Next(nodeIDs ...string) *NodeBuilder {
	n.node().Next = nodeIDs
	return n
}

// OnError sets the error handling nodes.
func (n *NodeBuilder) OnError(nodeIDs ...string) *NodeBuilder {
	n.node().OnError = nodeIDs
	return n
}

// TrueNext sets the nodes to execute if condition is true.
func (n *NodeBuilder) TrueNext(nodeIDs ...string) *NodeBuilder {
	n.node().TrueNext = nodeIDs
	return n
}

// FalseNext sets the nodes to execute if condition is false.
func (n *NodeBuilder) FalseNext(nodeIDs ...string) *NodeBuilder {
	n.node().FalseNext = nodeIDs
	return n
}

// Retry sets the retry count.
func (n *NodeBuilder) Retry(count int) *NodeBuilder {
	n.node().RetryCount = count
	return n
}

// Timeout sets the execution timeout.
func (n *NodeBuilder) Timeout(d time.Duration) *NodeBuilder {
	n.node().Timeout = d.String()
	return n
}

// Done returns to the workflow builder.
func (n *NodeBuilder) Done() *WorkflowBuilder {
	return n.workflow
}
