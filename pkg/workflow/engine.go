package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/google/uuid"
)

// Engine implements WorkflowEngine using EventBus.
type Engine struct {
	eventBus   core.EventBus
	registry   NodeRegistry
	workflows  map[string]*WorkflowDefinition
	executions map[string]*ExecutionState
	mu         sync.RWMutex
	logger     core.Logger

	// Execution tracking
	mergeStates map[string]*mergeState // executionID:nodeID -> merge state
	mergeMu     sync.Mutex

	// Active node tracking for better completion detection
	activeNodes map[string]map[string]bool // executionID -> nodeID -> true
	activeMu    sync.Mutex

	// Context cancellation for executions
	execContexts map[string]context.CancelFunc // executionID -> cancel function
	execCtxMu    sync.Mutex
}

type mergeState struct {
	expectedInputs int
	receivedInputs int
	data           []interface{}
}

// NewEngine creates a new workflow engine.
func NewEngine(eventBus core.EventBus) *Engine {
	return &Engine{
		eventBus:     eventBus,
		registry:     NewNodeRegistry(),
		workflows:    make(map[string]*WorkflowDefinition),
		executions:   make(map[string]*ExecutionState),
		mergeStates:  make(map[string]*mergeState),
		activeNodes:  make(map[string]map[string]bool),
		execContexts: make(map[string]context.CancelFunc),
		logger:       core.NewDefaultLogger(),
	}
}

// RegisterNodeHandler registers a custom node handler.
func (e *Engine) RegisterNodeHandler(nodeType NodeType, handler NodeHandler) {
	e.registry.Register(nodeType, handler)
}

// Registry returns the node registry for external registration
func (e *Engine) Registry() NodeRegistry {
	return e.registry
}

// RegisterWorkflow registers a workflow definition.
func (e *Engine) RegisterWorkflow(def *WorkflowDefinition) error {
	if def.ID == "" {
		return fmt.Errorf("workflow ID is required")
	}
	if len(def.Nodes) == 0 {
		return fmt.Errorf("workflow must have at least one node")
	}

	// Validate node references
	nodeIDs := make(map[string]bool)
	for _, node := range def.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node ID is required")
		}
		nodeIDs[node.ID] = true
	}

	for _, node := range def.Nodes {
		for _, next := range node.Next {
			if !nodeIDs[next] {
				return fmt.Errorf("node %s references unknown node %s", node.ID, next)
			}
		}
		for _, next := range node.TrueNext {
			if !nodeIDs[next] {
				return fmt.Errorf("node %s references unknown node %s in trueNext", node.ID, next)
			}
		}
		for _, next := range node.FalseNext {
			if !nodeIDs[next] {
				return fmt.Errorf("node %s references unknown node %s in falseNext", node.ID, next)
			}
		}
	}

	e.mu.Lock()
	e.workflows[def.ID] = def
	e.mu.Unlock()

	// Register EventBus consumers for this workflow
	e.registerWorkflowConsumers(def)

	return nil
}

// registerWorkflowConsumers sets up EventBus consumers for workflow execution.
func (e *Engine) registerWorkflowConsumers(def *WorkflowDefinition) {
	// Consumer for workflow execution events
	address := fmt.Sprintf("workflow.%s.execute", def.ID)
	e.eventBus.Consumer(address).Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var execReq struct {
			ExecutionID string      `json:"executionId"`
			Input       interface{} `json:"input"`
		}
		if body, ok := msg.Body().([]byte); ok {
			if err := json.Unmarshal(body, &execReq); err != nil {
				return msg.Reply(map[string]interface{}{"error": err.Error()})
			}
		}

		execID, err := e.startExecution(ctx.Context(), def.ID, execReq.Input)
		if err != nil {
			return msg.Reply(map[string]interface{}{"error": err.Error()})
		}

		return msg.Reply(map[string]interface{}{"executionId": execID})
	})

	// Consumer for node completion events
	for _, node := range def.Nodes {
		nodeAddress := fmt.Sprintf("workflow.%s.node.%s", def.ID, node.ID)
		nodeDef := node // Capture for closure
		e.eventBus.Consumer(nodeAddress).Handler(func(ctx core.FluxorContext, msg core.Message) error {
			return e.handleNodeExecution(ctx.Context(), def, &nodeDef, msg)
		})
	}
}

// ExecuteWorkflow starts a workflow execution.
func (e *Engine) ExecuteWorkflow(ctx context.Context, workflowID string, input interface{}) (string, error) {
	return e.startExecution(ctx, workflowID, input)
}

func (e *Engine) startExecution(ctx context.Context, workflowID string, input interface{}) (string, error) {
	e.mu.RLock()
	def, ok := e.workflows[workflowID]
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("workflow not found: %s", workflowID)
	}

	executionID := uuid.New().String()

	// Create cancellable context for this execution
	execCtx, cancel := context.WithCancel(ctx)

	e.execCtxMu.Lock()
	e.execContexts[executionID] = cancel
	e.execCtxMu.Unlock()

	execCtxData := &ExecutionContext{
		WorkflowID:  workflowID,
		ExecutionID: executionID,
		StartTime:   time.Now(),
		Data:        make(map[string]interface{}),
		NodeOutputs: make(map[string]interface{}),
		Variables:   make(map[string]interface{}),
	}

	// Initialize variables from workflow definition
	if def.Variables != nil {
		for k, v := range def.Variables {
			execCtxData.Variables[k] = v
		}
	}

	// Store input
	if inputMap, ok := input.(map[string]interface{}); ok {
		execCtxData.Data = inputMap
	} else {
		execCtxData.Data["input"] = input
	}

	state := &ExecutionState{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		Status:      ExecutionStatusRunning,
		StartTime:   time.Now(),
		Context:     execCtxData,
	}

	e.mu.Lock()
	e.executions[executionID] = state
	e.mu.Unlock()

	// Initialize active nodes tracking
	e.activeMu.Lock()
	e.activeNodes[executionID] = make(map[string]bool)
	e.activeMu.Unlock()

	// Find and execute trigger/start nodes
	for _, node := range def.Nodes {
		if e.isStartNode(&node, def) {
			e.markNodeActive(executionID, node.ID)
			go e.executeNode(execCtx, def, &node, execCtxData, input)
		}
	}

	return executionID, nil
}

func (e *Engine) isStartNode(node *NodeDefinition, def *WorkflowDefinition) bool {
	// A start node is either a trigger type or has no incoming connections
	switch NodeType(node.Type) {
	case NodeTypeWebhook, NodeTypeSchedule, NodeTypeEvent, NodeTypeManual:
		return true
	}

	// Check if any node points to this node
	for _, n := range def.Nodes {
		for _, next := range n.Next {
			if next == node.ID {
				return false
			}
		}
		for _, next := range n.TrueNext {
			if next == node.ID {
				return false
			}
		}
		for _, next := range n.FalseNext {
			if next == node.ID {
				return false
			}
		}
	}

	return true
}

func (e *Engine) executeNode(ctx context.Context, def *WorkflowDefinition, node *NodeDefinition, execCtx *ExecutionContext, input interface{}) {
	// Check if execution was cancelled
	select {
	case <-ctx.Done():
		e.markNodeInactive(execCtx.ExecutionID, node.ID)
		return
	default:
	}

	nodeType := NodeType(node.Type)
	handler, ok := e.registry.Get(nodeType)
	if !ok {
		e.logger.Error(fmt.Sprintf("unknown node type: %s", node.Type))
		e.recordError(execCtx, node.ID, fmt.Sprintf("unknown node type: %s", node.Type))
		e.markNodeInactive(execCtx.ExecutionID, node.ID)
		return
	}

	// Apply timeout if configured
	nodeCtx := ctx
	if node.Timeout != "" {
		if timeout, err := time.ParseDuration(node.Timeout); err == nil {
			var cancel context.CancelFunc
			nodeCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
	}

	// Add engine to context for sub-workflow nodes
	nodeCtx = context.WithValue(nodeCtx, "workflow_engine", e)

	// Prepare input
	nodeInput := &NodeInput{
		Data:        input,
		Context:     execCtx,
		Config:      node.Config,
		TriggerData: execCtx.Data["input"],
	}

	// Execute with retry
	var output *NodeOutput
	var err error
	retries := node.RetryCount
	if retries == 0 {
		retries = 1
	}

	for i := 0; i < retries; i++ {
		// Check cancellation before each retry
		select {
		case <-ctx.Done():
			e.markNodeInactive(execCtx.ExecutionID, node.ID)
			return
		default:
		}

		output, err = handler(nodeCtx, nodeInput)
		if err == nil {
			break
		}
		if i < retries-1 {
			// Exponential backoff with jitter
			backoff := time.Duration(i+1) * time.Second
			select {
			case <-ctx.Done():
				e.markNodeInactive(execCtx.ExecutionID, node.ID)
				return
			case <-time.After(backoff):
			}
		}
	}

	// Mark node as completed
	defer e.markNodeInactive(execCtx.ExecutionID, node.ID)

	// Handle error
	if err != nil {
		e.recordError(execCtx, node.ID, err.Error())
		if len(node.OnError) > 0 {
			for _, nextID := range node.OnError {
				nextNode := e.findNode(def, nextID)
				if nextNode != nil {
					e.markNodeActive(execCtx.ExecutionID, nextID)
					go e.executeNode(ctx, def, nextNode, execCtx, input)
				}
			}
		} else {
			// No error handler - check if execution should complete
			e.checkExecutionComplete(execCtx.ExecutionID)
		}
		return
	}

	// Store output
	e.mu.Lock()
	execCtx.NodeOutputs[node.ID] = output.Data
	e.mu.Unlock()

	// Check if workflow should stop
	if output.Stop {
		e.completeExecution(execCtx.ExecutionID, nil)
		return
	}

	// Determine next nodes
	nextNodes := e.determineNextNodes(node, output)

	// Execute next nodes
	for _, nextID := range nextNodes {
		// Check cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		nextNode := e.findNode(def, nextID)
		if nextNode != nil {
			// Handle merge nodes
			if NodeType(nextNode.Type) == NodeTypeMerge {
				e.handleMergeInput(ctx, def, nextNode, execCtx, output.Data)
			} else {
				e.markNodeActive(execCtx.ExecutionID, nextID)
				go e.executeNode(ctx, def, nextNode, execCtx, output.Data)
			}
		}
	}

	// Check if execution is complete (no more nodes to execute)
	if len(nextNodes) == 0 {
		e.checkExecutionComplete(execCtx.ExecutionID)
	}
}

func (e *Engine) determineNextNodes(node *NodeDefinition, output *NodeOutput) []string {
	// If output specifies next nodes, use those
	if len(output.NextNodes) > 0 {
		return output.NextNodes
	}

	// Check for condition result
	if data, ok := output.Data.(map[string]interface{}); ok {
		if condResult, ok := data["_conditionResult"].(bool); ok {
			if condResult {
				return node.TrueNext
			}
			return node.FalseNext
		}
	}

	// Default to configured next nodes
	return node.Next
}

func (e *Engine) findNode(def *WorkflowDefinition, nodeID string) *NodeDefinition {
	for i := range def.Nodes {
		if def.Nodes[i].ID == nodeID {
			return &def.Nodes[i]
		}
	}
	return nil
}

func (e *Engine) handleMergeInput(ctx context.Context, def *WorkflowDefinition, node *NodeDefinition, execCtx *ExecutionContext, data interface{}) {
	key := fmt.Sprintf("%s:%s", execCtx.ExecutionID, node.ID)

	e.mergeMu.Lock()
	state, ok := e.mergeStates[key]
	if !ok {
		// Count expected inputs
		expected := 0
		for _, n := range def.Nodes {
			for _, next := range n.Next {
				if next == node.ID {
					expected++
				}
			}
			for _, next := range n.TrueNext {
				if next == node.ID {
					expected++
				}
			}
			for _, next := range n.FalseNext {
				if next == node.ID {
					expected++
				}
			}
		}
		state = &mergeState{
			expectedInputs: expected,
			data:           make([]interface{}, 0),
		}
		e.mergeStates[key] = state
	}

	state.data = append(state.data, data)
	state.receivedInputs++

	// Check mode from config
	mode := "waitAll"
	if m, ok := node.Config["mode"].(string); ok {
		mode = m
	}

	shouldProceed := false
	switch mode {
	case "waitAll":
		shouldProceed = state.receivedInputs >= state.expectedInputs
	case "waitAny":
		shouldProceed = state.receivedInputs >= 1
	}

	if shouldProceed {
		delete(e.mergeStates, key)
		e.mergeMu.Unlock()
		// Continue execution with merged data
		go e.executeNode(ctx, def, node, execCtx, state.data)
	} else {
		e.mergeMu.Unlock()
	}
}

func (e *Engine) handleNodeExecution(ctx context.Context, def *WorkflowDefinition, node *NodeDefinition, msg core.Message) error {
	var req struct {
		ExecutionID string      `json:"executionId"`
		Data        interface{} `json:"data"`
	}

	if body, ok := msg.Body().([]byte); ok {
		if err := json.Unmarshal(body, &req); err != nil {
			return err
		}
	}

	e.mu.RLock()
	state, ok := e.executions[req.ExecutionID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("execution not found: %s", req.ExecutionID)
	}

	go e.executeNode(ctx, def, node, state.Context, req.Data)
	return nil
}

func (e *Engine) recordError(execCtx *ExecutionContext, nodeID, message string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	execCtx.Errors = append(execCtx.Errors, ExecutionError{
		NodeID:    nodeID,
		Message:   message,
		Timestamp: time.Now(),
	})
}

func (e *Engine) completeExecution(executionID string, err error) {
	e.mu.Lock()
	state, ok := e.executions[executionID]
	if !ok {
		e.mu.Unlock()
		return
	}

	now := time.Now()
	state.EndTime = &now

	if err != nil {
		state.Status = ExecutionStatusFailed
		state.Error = err.Error()
	} else {
		state.Status = ExecutionStatusCompleted
	}
	e.mu.Unlock()

	// Clean up execution resources
	e.execCtxMu.Lock()
	delete(e.execContexts, executionID)
	e.execCtxMu.Unlock()

	e.activeMu.Lock()
	delete(e.activeNodes, executionID)
	e.activeMu.Unlock()

	// Clean up merge states for this execution
	e.mergeMu.Lock()
	for key := range e.mergeStates {
		if strings.HasPrefix(key, executionID+":") {
			delete(e.mergeStates, key)
		}
	}
	e.mergeMu.Unlock()
}

func (e *Engine) checkExecutionComplete(executionID string) {
	e.mu.RLock()
	state, ok := e.executions[executionID]
	e.mu.RUnlock()

	if !ok || state.Status != ExecutionStatusRunning {
		return
	}

	// Check if there are any active nodes
	e.activeMu.Lock()
	activeNodes := e.activeNodes[executionID]
	hasActiveNodes := len(activeNodes) > 0
	e.activeMu.Unlock()

	// Only complete if no active nodes remain
	if !hasActiveNodes {
		if len(state.Context.Errors) == 0 {
			e.completeExecution(executionID, nil)
		} else {
			e.completeExecution(executionID, fmt.Errorf("workflow had %d errors", len(state.Context.Errors)))
		}
	}
}

// markNodeActive marks a node as active in an execution.
func (e *Engine) markNodeActive(executionID, nodeID string) {
	e.activeMu.Lock()
	defer e.activeMu.Unlock()
	if e.activeNodes[executionID] == nil {
		e.activeNodes[executionID] = make(map[string]bool)
	}
	e.activeNodes[executionID][nodeID] = true
}

// markNodeInactive marks a node as inactive and checks completion.
func (e *Engine) markNodeInactive(executionID, nodeID string) {
	e.activeMu.Lock()
	if nodes, ok := e.activeNodes[executionID]; ok {
		delete(nodes, nodeID)
	}
	e.activeMu.Unlock()

	// Check completion after marking inactive
	e.checkExecutionComplete(executionID)
}

// GetExecution returns execution status.
func (e *Engine) GetExecution(executionID string) (*ExecutionContext, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	state, ok := e.executions[executionID]
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	return state.Context, nil
}

// CancelExecution cancels a running execution.
func (e *Engine) CancelExecution(executionID string) error {
	e.mu.Lock()
	state, ok := e.executions[executionID]
	if !ok {
		e.mu.Unlock()
		return fmt.Errorf("execution not found: %s", executionID)
	}

	if state.Status != ExecutionStatusRunning {
		e.mu.Unlock()
		return fmt.Errorf("execution is not running")
	}

	now := time.Now()
	state.EndTime = &now
	state.Status = ExecutionStatusCancelled
	e.mu.Unlock()

	// Cancel the execution context to stop all running nodes
	e.execCtxMu.Lock()
	cancel, ok := e.execContexts[executionID]
	if ok {
		cancel()
		delete(e.execContexts, executionID)
	}
	e.execCtxMu.Unlock()

	// Clean up active nodes tracking
	e.activeMu.Lock()
	delete(e.activeNodes, executionID)
	e.activeMu.Unlock()

	// Clean up merge states
	e.mergeMu.Lock()
	for key := range e.mergeStates {
		if strings.HasPrefix(key, executionID+":") {
			delete(e.mergeStates, key)
		}
	}
	e.mergeMu.Unlock()

	return nil
}

// ListWorkflows returns all registered workflows.
func (e *Engine) ListWorkflows() []*WorkflowDefinition {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*WorkflowDefinition, 0, len(e.workflows))
	for _, def := range e.workflows {
		result = append(result, def)
	}
	return result
}

// GetExecutionState returns the full execution state.
func (e *Engine) GetExecutionState(executionID string) (*ExecutionState, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	state, ok := e.executions[executionID]
	if !ok {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	return state, nil
}

// CleanupOldExecutions removes executions older than the specified duration.
// This helps prevent memory leaks from long-running workflows.
func (e *Engine) CleanupOldExecutions(maxAge time.Duration) int {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	cleaned := 0

	for execID, state := range e.executions {
		// Only clean up completed/failed/cancelled executions
		if state.Status != ExecutionStatusRunning && state.Status != ExecutionStatusPending {
			if state.EndTime != nil && now.Sub(*state.EndTime) > maxAge {
				delete(e.executions, execID)
				cleaned++

				// Clean up related resources
				e.execCtxMu.Lock()
				delete(e.execContexts, execID)
				e.execCtxMu.Unlock()

				e.activeMu.Lock()
				delete(e.activeNodes, execID)
				e.activeMu.Unlock()

				// Clean up merge states
				e.mergeMu.Lock()
				for key := range e.mergeStates {
					if strings.HasPrefix(key, execID+":") {
						delete(e.mergeStates, key)
					}
				}
				e.mergeMu.Unlock()
			}
		}
	}

	return cleaned
}
