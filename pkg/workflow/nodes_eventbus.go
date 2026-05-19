package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// EventBusNodeConfig holds EventBus configuration.
type EventBusNodeConfig struct {
	EventBus core.EventBus
}

// CreateEventBusHandler creates an EventBus node handler with the given EventBus.
func CreateEventBusHandler(eventBus core.EventBus) NodeHandler {
	return func(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
		// Config:
		// - "address": EventBus address (required)
		// - "action": "publish", "send", "request" (default: "send")
		// - "timeout": timeout for request action (default: 5s)

		address, ok := input.Config["address"].(string)
		if !ok || address == "" {
			return nil, fmt.Errorf("eventbus node requires 'address' config")
		}

		// Process address template
		address = processTemplate(address, input.Data)

		action := "send"
		if a, ok := input.Config["action"].(string); ok {
			action = a
		}

		// Prepare message body
		body := input.Data

		switch action {
		case "publish":
			if err := eventBus.Publish(address, body); err != nil {
				return nil, fmt.Errorf("publish failed: %w", err)
			}
			return &NodeOutput{Data: input.Data}, nil

		case "send":
			if err := eventBus.Send(address, body); err != nil {
				return nil, fmt.Errorf("send failed: %w", err)
			}
			return &NodeOutput{Data: input.Data}, nil

		case "request":
			timeout := 5 * time.Second
			if t, ok := input.Config["timeout"].(string); ok {
				if d, err := time.ParseDuration(t); err == nil {
					timeout = d
				}
			}

			reply, err := eventBus.Request(address, body, timeout)
			if err != nil {
				return nil, fmt.Errorf("request failed: %w", err)
			}

			var responseData interface{}
			if bodyBytes, ok := reply.Body().([]byte); ok {
				if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
					responseData = string(bodyBytes)
				}
			} else {
				responseData = reply.Body()
			}

			return &NodeOutput{
				Data: map[string]interface{}{
					"response": responseData,
					"_input":   input.Data,
				},
			}, nil

		default:
			return nil, fmt.Errorf("unknown eventbus action: %s", action)
		}
	}
}

// EventTriggerConfig configures event-based workflow triggers.
type EventTriggerConfig struct {
	Address    string `json:"address"`
	WorkflowID string `json:"workflowId"`
}

// RegisterEventTrigger registers an EventBus consumer that triggers a workflow.
func RegisterEventTrigger(eventBus core.EventBus, engine *Engine, config EventTriggerConfig) error {
	consumer := eventBus.Consumer(config.Address)
	consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
		var input interface{}
		if bodyBytes, ok := msg.Body().([]byte); ok {
			if err := json.Unmarshal(bodyBytes, &input); err != nil {
				input = string(bodyBytes)
			}
		} else {
			input = msg.Body()
		}

		execID, err := engine.ExecuteWorkflow(ctx.Context(), config.WorkflowID, input)
		if err != nil {
			return err
		}

		// If the message expects a reply, send the execution ID
		return msg.Reply(map[string]interface{}{
			"executionId": execID,
			"workflowId":  config.WorkflowID,
		})
	})

	return nil
}
