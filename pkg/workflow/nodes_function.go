package workflow

import (
	"context"
	"fmt"
	"sync"
)

// FunctionRegistry stores custom functions that can be called by function nodes.
type FunctionRegistry struct {
	functions map[string]func(ctx context.Context, data interface{}) (interface{}, error)
	mu        sync.RWMutex
}

// NewFunctionRegistry creates a new function registry.
func NewFunctionRegistry() *FunctionRegistry {
	return &FunctionRegistry{
		functions: make(map[string]func(ctx context.Context, data interface{}) (interface{}, error)),
	}
}

// Register registers a function with a name.
func (r *FunctionRegistry) Register(name string, fn func(ctx context.Context, data interface{}) (interface{}, error)) {
	if name == "" {
		panic("function name cannot be empty")
	}
	if fn == nil {
		panic("function cannot be nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.functions[name] = fn
}

// Get returns a function by name.
func (r *FunctionRegistry) Get(name string) (func(ctx context.Context, data interface{}) (interface{}, error), bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.functions[name]
	return fn, ok
}

// CreateFunctionHandler creates a function node handler with the given registry.
func CreateFunctionHandler(registry *FunctionRegistry) NodeHandler {
	return func(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
		// Config:
		// - "function": name of the registered function
		// - "inline": inline function definition (for simple transformations)

		functionName, ok := input.Config["function"].(string)
		if ok && functionName != "" {
			fn, exists := registry.Get(functionName)
			if !exists {
				return nil, fmt.Errorf("function not found: %s", functionName)
			}

			result, err := fn(ctx, input.Data)
			if err != nil {
				return nil, err
			}

			return &NodeOutput{Data: result}, nil
		}

		// Handle inline transformations
		if inline, ok := input.Config["inline"].(map[string]interface{}); ok {
			return handleInlineTransform(input.Data, inline)
		}

		// Pass through if no function specified
		return &NodeOutput{Data: input.Data}, nil
	}
}

func handleInlineTransform(data interface{}, transform map[string]interface{}) (*NodeOutput, error) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		dataMap = map[string]interface{}{"value": data}
	}

	result := make(map[string]interface{})

	for key, value := range transform {
		switch v := value.(type) {
		case string:
			// Check if it's a field reference (starts with $)
			if len(v) > 0 && v[0] == '$' {
				fieldName := v[1:]
				result[key] = dataMap[fieldName]
			} else {
				result[key] = processTemplate(v, dataMap)
			}
		case map[string]interface{}:
			// Nested transformation
			nested, err := handleInlineTransform(dataMap, v)
			if err != nil {
				return nil, err
			}
			result[key] = nested.Data
		default:
			result[key] = v
		}
	}

	return &NodeOutput{Data: result}, nil
}

// CodeNodeHandler handles code execution nodes.
// WARNING: Use with caution - executes arbitrary code.
func CodeNodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config:
	// - "language": "json" (only JSON transformation supported for safety)
	// - "transform": transformation rules

	language, _ := input.Config["language"].(string)
	if language != "json" && language != "" {
		return nil, fmt.Errorf("unsupported code language: %s (only 'json' supported)", language)
	}

	transform, ok := input.Config["transform"].(map[string]interface{})
	if !ok {
		return &NodeOutput{Data: input.Data}, nil
	}

	return handleInlineTransform(input.Data, transform)
}

// MapNodeHandler maps over an array and transforms each item.
func MapNodeHandler(registry *FunctionRegistry) NodeHandler {
	return func(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
		// Config:
		// - "items": field containing array (or use input if array)
		// - "transform": transformation to apply to each item
		// - "function": function to apply to each item

		var items []interface{}

		if itemsField, ok := input.Config["items"].(string); ok {
			if dataMap, ok := input.Data.(map[string]interface{}); ok {
				if arr, ok := dataMap[itemsField].([]interface{}); ok {
					items = arr
				}
			}
		} else if arr, ok := input.Data.([]interface{}); ok {
			items = arr
		}

		if len(items) == 0 {
			return &NodeOutput{Data: []interface{}{}}, nil
		}

		results := make([]interface{}, len(items))

		// Check for function
		if fnName, ok := input.Config["function"].(string); ok && fnName != "" {
			fn, exists := registry.Get(fnName)
			if !exists {
				return nil, fmt.Errorf("function not found: %s", fnName)
			}

			for i, item := range items {
				result, err := fn(ctx, item)
				if err != nil {
					return nil, fmt.Errorf("map function error at index %d: %w", i, err)
				}
				results[i] = result
			}
		} else if transform, ok := input.Config["transform"].(map[string]interface{}); ok {
			for i, item := range items {
				output, err := handleInlineTransform(item, transform)
				if err != nil {
					return nil, fmt.Errorf("map transform error at index %d: %w", i, err)
				}
				results[i] = output.Data
			}
		} else {
			results = items
		}

		return &NodeOutput{Data: results}, nil
	}
}

// FilterNodeHandler filters an array based on a condition.
func FilterNodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config:
	// - "items": field containing array
	// - "field": field to check
	// - "operator": comparison operator
	// - "value": value to compare

	var items []interface{}

	if itemsField, ok := input.Config["items"].(string); ok {
		if dataMap, ok := input.Data.(map[string]interface{}); ok {
			if arr, ok := dataMap[itemsField].([]interface{}); ok {
				items = arr
			}
		}
	} else if arr, ok := input.Data.([]interface{}); ok {
		items = arr
	}

	if len(items) == 0 {
		return &NodeOutput{Data: []interface{}{}}, nil
	}

	field, _ := input.Config["field"].(string)
	operator, _ := input.Config["operator"].(string)
	expectedValue := input.Config["value"]

	results := make([]interface{}, 0)

	for _, item := range items {
		var actualValue interface{}

		if itemMap, ok := item.(map[string]interface{}); ok {
			actualValue = itemMap[field]
		} else if field == "" {
			actualValue = item
		}

		if evaluateCondition(actualValue, operator, expectedValue) {
			results = append(results, item)
		}
	}

	return &NodeOutput{Data: results}, nil
}

// ReduceNodeHandler reduces an array to a single value.
func ReduceNodeHandler(registry *FunctionRegistry) NodeHandler {
	return func(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
		// Config:
		// - "items": field containing array
		// - "function": reduce function name
		// - "initial": initial value
		// - "operation": simple operation ("sum", "count", "concat", "first", "last")

		var items []interface{}

		if itemsField, ok := input.Config["items"].(string); ok {
			if dataMap, ok := input.Data.(map[string]interface{}); ok {
				if arr, ok := dataMap[itemsField].([]interface{}); ok {
					items = arr
				}
			}
		} else if arr, ok := input.Data.([]interface{}); ok {
			items = arr
		}

		if len(items) == 0 {
			return &NodeOutput{Data: input.Config["initial"]}, nil
		}

		// Handle simple operations
		if op, ok := input.Config["operation"].(string); ok {
			switch op {
			case "sum":
				sum := 0.0
				for _, item := range items {
					sum += toFloat(item)
				}
				return &NodeOutput{Data: sum}, nil

			case "count":
				return &NodeOutput{Data: len(items)}, nil

			case "concat":
				result := ""
				for _, item := range items {
					result += fmt.Sprintf("%v", item)
				}
				return &NodeOutput{Data: result}, nil

			case "first":
				return &NodeOutput{Data: items[0]}, nil

			case "last":
				return &NodeOutput{Data: items[len(items)-1]}, nil
			}
		}

		// Default: return items as-is
		return &NodeOutput{Data: items}, nil
	}
}
