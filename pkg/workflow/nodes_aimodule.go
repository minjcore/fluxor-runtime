package workflow

import (
	"context"

	aimodule "github.com/fluxorio/fluxor/pkg/aimodule"
)

// AIChatNodeHandler handles AI chat completion nodes using aimodule
func AIChatNodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Convert workflow types to aimodule types
	aimoduleInput := &aimodule.NodeInput{
		Data:        input.Data,
		Context:     input.Context,
		Config:      input.Config,
		TriggerData: input.TriggerData,
	}

	aimoduleOutput, err := aimodule.AIChatNodeHandler(ctx, aimoduleInput)
	if err != nil {
		return nil, err
	}

	// Convert aimodule types back to workflow types
	return &NodeOutput{
		Data:      aimoduleOutput.Data,
		Error:     aimoduleOutput.Error,
		NextNodes: aimoduleOutput.NextNodes,
		Stop:      aimoduleOutput.Stop,
	}, nil
}

// AIEmbedNodeHandler handles AI embedding nodes using aimodule
func AIEmbedNodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Convert workflow types to aimodule types
	aimoduleInput := &aimodule.NodeInput{
		Data:        input.Data,
		Context:     input.Context,
		Config:      input.Config,
		TriggerData: input.TriggerData,
	}

	aimoduleOutput, err := aimodule.AIEmbedNodeHandler(ctx, aimoduleInput)
	if err != nil {
		return nil, err
	}

	// Convert aimodule types back to workflow types
	return &NodeOutput{
		Data:      aimoduleOutput.Data,
		Error:     aimoduleOutput.Error,
		NextNodes: aimoduleOutput.NextNodes,
		Stop:      aimoduleOutput.Stop,
	}, nil
}
