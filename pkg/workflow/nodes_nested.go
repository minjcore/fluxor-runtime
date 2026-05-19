package workflow

import (
	"context"
	"fmt"
)

// CreateSubWorkflowHandler creates a sub-workflow handler with engine reference.
func CreateSubWorkflowHandler(engine *Engine) NodeHandler {
	return func(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
		return subWorkflowNodeHandler(ctx, input, engine)
	}
}

// subWorkflowNodeHandler executes a nested workflow.
func subWorkflowNodeHandler(ctx context.Context, input *NodeInput, engine *Engine) (*NodeOutput, error) {
	// Config:
	// - "workflowId": ID of workflow to execute (required)
	// - "inputField": Field from input data to pass to sub-workflow (default: use entire input)
	// - "outputField": Field name for sub-workflow output (default: "subworkflow_output")
	// - "waitForCompletion": Wait for sub-workflow to complete (default: true)

	workflowID, ok := input.Config["workflowId"].(string)
	if !ok || workflowID == "" {
		return nil, fmt.Errorf("subworkflow node requires 'workflowId' config")
	}

	// Use engine from parameter (passed via CreateSubWorkflowHandler)
	// Fallback to context if needed
	if engine == nil {
		if e, ok := ctx.Value("workflow_engine").(*Engine); ok {
			engine = e
		} else {
			return nil, fmt.Errorf("workflow engine not available")
		}
	}

	// Get input data for sub-workflow
	var subWorkflowInput interface{} = input.Data
	if inputField, ok := input.Config["inputField"].(string); ok && inputField != "" {
		if data, ok := input.Data.(map[string]interface{}); ok {
			if fieldValue, ok := data[inputField]; ok {
				subWorkflowInput = fieldValue
			}
		}
	}

	// Execute sub-workflow
	execID, err := engine.ExecuteWorkflow(ctx, workflowID, subWorkflowInput)
	if err != nil {
		return nil, fmt.Errorf("failed to execute sub-workflow %s: %w", workflowID, err)
	}

	// Wait for completion
	waitForCompletion := true
	if wait, ok := input.Config["waitForCompletion"].(bool); ok {
		waitForCompletion = wait
	}

	var subWorkflowOutput interface{}
	if waitForCompletion {
		// Wait for sub-workflow to complete
		// This is a simplified version - in production, you'd want proper async handling
		execCtx, err := engine.GetExecution(execID)
		if err != nil {
			return nil, fmt.Errorf("failed to get sub-workflow execution: %w", err)
		}

		// Get output from last node or all node outputs
		subWorkflowOutput = execCtx.NodeOutputs
	} else {
		// Return execution ID for async handling
		subWorkflowOutput = map[string]interface{}{
			"executionId": execID,
			"workflowId":  workflowID,
		}
	}

	// Merge sub-workflow output into result
	output := make(map[string]interface{})
	if data, ok := input.Data.(map[string]interface{}); ok {
		for k, v := range data {
			output[k] = v
		}
	}

	outputField := "subworkflow_output"
	if of, ok := input.Config["outputField"].(string); ok && of != "" {
		outputField = of
	}

	output[outputField] = subWorkflowOutput
	output["_subworkflow_executionId"] = execID

	return &NodeOutput{Data: output}, nil
}

// DynamicLoopNodeHandler executes next nodes for each item in an array dynamically.
func DynamicLoopNodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config:
	// - "itemsField": Field name containing array to iterate (default: use input data if array)
	// - "itemField": Field name for current item in output (default: "item")
	// - "indexField": Field name for current index (default: "index")
	// - "batchSize": Process items in batches (default: 1, process all in parallel if 0)
	// - "nextNode": Node ID to execute for each item (required)

	nextNodeID, ok := input.Config["nextNode"].(string)
	if !ok || nextNodeID == "" {
		return nil, fmt.Errorf("dynamicloop node requires 'nextNode' config")
	}

	// Get items to iterate
	var items []interface{}
	itemsField, _ := input.Config["itemsField"].(string)

	if itemsField != "" {
		if data, ok := input.Data.(map[string]interface{}); ok {
			if arr, ok := data[itemsField].([]interface{}); ok {
				items = arr
			}
		}
	} else if arr, ok := input.Data.([]interface{}); ok {
		items = arr
	}

	if len(items) == 0 {
		// No items to process, return input as-is
		return &NodeOutput{Data: input.Data}, nil
	}

	// Get field names
	itemField := "item"
	if ifield, ok := input.Config["itemField"].(string); ok && ifield != "" {
		itemField = ifield
	}

	indexField := "index"
	if ifield, ok := input.Config["indexField"].(string); ok && ifield != "" {
		indexField = ifield
	}

	// Get batch size
	batchSize := 1
	if bs, ok := input.Config["batchSize"].(float64); ok {
		batchSize = int(bs)
	} else if bs, ok := input.Config["batchSize"].(int); ok {
		batchSize = bs
	}

	// Process items
	results := make([]interface{}, 0, len(items))
	for i, item := range items {
		// Create context for this item
		itemData := make(map[string]interface{})
		if data, ok := input.Data.(map[string]interface{}); ok {
			for k, v := range data {
				itemData[k] = v
			}
		}
		itemData[itemField] = item
		itemData[indexField] = i
		itemData["_loopTotal"] = len(items)
		itemData["_loopIndex"] = i

		// Store result (in real implementation, this would execute the next node)
		results = append(results, itemData)

		// If batchSize > 0, process in batches
		if batchSize > 0 && (i+1)%batchSize == 0 {
			// Process batch (would execute nextNode for each item in batch)
		}
	}

	// Return results
	output := make(map[string]interface{})
	if data, ok := input.Data.(map[string]interface{}); ok {
		for k, v := range data {
			output[k] = v
		}
	}
	output["_loopResults"] = results
	output["_loopCount"] = len(results)

	return &NodeOutput{
		Data:      output,
		NextNodes: []string{nextNodeID}, // Signal to execute next node for each item
	}, nil
}
