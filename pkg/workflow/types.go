// Package workflow provides an n8n-like workflow engine using EventBus.
//
// Workflows are defined as JSON and executed via event-driven nodes.
// Each node communicates through EventBus, enabling distributed execution.
//
// Example workflow:
//
//	{
//	  "id": "order-processing",
//	  "nodes": [
//	    {"id": "trigger", "type": "webhook", "next": ["validate"]},
//	    {"id": "validate", "type": "function", "next": ["check-stock"]},
//	    {"id": "check-stock", "type": "condition", "true": ["process"], "false": ["notify-oos"]},
//	    {"id": "process", "type": "function", "next": ["notify-success"]},
//	    {"id": "notify-oos", "type": "http", "next": []},
//	    {"id": "notify-success", "type": "http", "next": []}
//	  ]
//	}
package workflow

import (
	"context"
	"time"
)

// WorkflowDefinition defines a workflow in JSON-serializable format.
type WorkflowDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Nodes       []NodeDefinition       `json:"nodes"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"` // Workflow-level variables
}

// NodeDefinition defines a single node in the workflow.
type NodeDefinition struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
	Next       []string               `json:"next,omitempty"`       // Next nodes on success
	OnError    []string               `json:"onError,omitempty"`    // Next nodes on error
	TrueNext   []string               `json:"trueNext,omitempty"`   // For condition nodes
	FalseNext  []string               `json:"falseNext,omitempty"`  // For condition nodes
	RetryCount int                    `json:"retryCount,omitempty"` // Retry on failure
	Timeout    string                 `json:"timeout,omitempty"`    // Execution timeout
}

// NodeType represents the type of workflow node.
type NodeType string

const (
	// Trigger nodes - start the workflow
	NodeTypeWebhook  NodeType = "webhook"  // HTTP webhook trigger
	NodeTypeSchedule NodeType = "schedule" // Cron/interval trigger
	NodeTypeEvent    NodeType = "event"    // EventBus trigger
	NodeTypeManual   NodeType = "manual"   // Manual trigger

	// Action nodes - perform operations
	NodeTypeFunction NodeType = "function" // Custom function
	NodeTypeHTTP     NodeType = "http"     // HTTP request
	NodeTypeOpenAI   NodeType = "openai"   // OpenAI API request
	NodeTypeAI       NodeType = "ai"       // Generic AI API (OpenAI, Cursor, Anthropic, etc.)
	NodeTypeEventBus NodeType = "eventbus" // Send to EventBus
	NodeTypeSet      NodeType = "set"      // Set variables
	NodeTypeVariable NodeType = "variable" // Define variables
	NodeTypeCode     NodeType = "code"     // Execute code

	// Flow control nodes
	NodeTypeCondition   NodeType = "condition"   // If/else branching
	NodeTypeIfThenElse  NodeType = "ifthenelse"  // Structured IF-THEN-ELSE with inline branches
	NodeTypeSplit       NodeType = "split"       // Parallel execution
	NodeTypeMerge       NodeType = "merge"       // Wait for multiple inputs
	NodeTypeLoop        NodeType = "loop"        // Iterate over items
	NodeTypeDynamicLoop NodeType = "dynamicloop" // Dynamic loop based on data
	NodeTypeSwitch      NodeType = "switch"      // Multi-way branching
	NodeTypeWait        NodeType = "wait"        // Delay execution
	NodeTypeSubWorkflow NodeType = "subworkflow" // Execute nested workflow

	// Utility nodes
	NodeTypeNoOp    NodeType = "noop"    // Pass-through
	NodeTypeError   NodeType = "error"   // Throw error
	NodeTypeRespond NodeType = "respond" // Respond to trigger
)

// ExecutionContext holds the context for a workflow execution.
type ExecutionContext struct {
	WorkflowID  string                 `json:"workflowId"`
	ExecutionID string                 `json:"executionId"`
	StartTime   time.Time              `json:"startTime"`
	Data        map[string]interface{} `json:"data"`        // Workflow-level data
	NodeOutputs map[string]interface{} `json:"nodeOutputs"` // Output from each node
	Variables   map[string]interface{} `json:"variables"`   // User-defined variables
	Errors      []ExecutionError       `json:"errors,omitempty"`
}

// ExecutionError represents an error during execution.
type ExecutionError struct {
	NodeID    string    `json:"nodeId"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Retried   bool      `json:"retried"`
}

// NodeInput is passed to each node during execution.
type NodeInput struct {
	Data        interface{}            `json:"data"`        // Input data from previous node
	Context     *ExecutionContext      `json:"context"`     // Execution context
	Config      map[string]interface{} `json:"config"`      // Node configuration
	TriggerData interface{}            `json:"triggerData"` // Original trigger data
}

// NodeOutput is returned from each node after execution.
type NodeOutput struct {
	Data      interface{} `json:"data"`                // Output data
	Error     error       `json:"error,omitempty"`     // Error if any
	NextNodes []string    `json:"nextNodes,omitempty"` // Override next nodes
	Stop      bool        `json:"stop,omitempty"`      // Stop workflow
}

// NodeHandler is the function signature for node execution.
type NodeHandler func(ctx context.Context, input *NodeInput) (*NodeOutput, error)

// NodeRegistry stores registered node handlers.
type NodeRegistry interface {
	Register(nodeType NodeType, handler NodeHandler)
	Get(nodeType NodeType) (NodeHandler, bool)
}

// WorkflowEngine executes workflows.
type WorkflowEngine interface {
	// RegisterWorkflow registers a workflow definition
	RegisterWorkflow(def *WorkflowDefinition) error

	// ExecuteWorkflow starts a workflow execution
	ExecuteWorkflow(ctx context.Context, workflowID string, input interface{}) (string, error)

	// GetExecution returns execution status
	GetExecution(executionID string) (*ExecutionContext, error)

	// CancelExecution cancels a running execution
	CancelExecution(executionID string) error

	// ListWorkflows returns all registered workflows
	ListWorkflows() []*WorkflowDefinition
}

// ExecutionStatus represents the status of a workflow execution.
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// ExecutionState tracks the state of a workflow execution.
type ExecutionState struct {
	ExecutionID string            `json:"executionId"`
	WorkflowID  string            `json:"workflowId"`
	Status      ExecutionStatus   `json:"status"`
	StartTime   time.Time         `json:"startTime"`
	EndTime     *time.Time        `json:"endTime,omitempty"`
	Context     *ExecutionContext `json:"context"`
	Error       string            `json:"error,omitempty"`
}
