# Fluxor Workflow Engine

An n8n-like workflow engine for Fluxor using EventBus for node communication.

## Features

- **JSON-defined workflows** - Define workflows in JSON, similar to n8n
- **Event-driven execution** - Nodes communicate via EventBus
- **Built-in node types** - HTTP, conditions, loops, transforms, and more
- **Custom functions** - Register your own Go functions as nodes
- **Parallel execution** - Split/merge patterns for concurrent processing
- **Error handling** - Retry, fallback, and error nodes
- **HTTP API** - RESTful API for workflow management

## Quick Start

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/entrypoint"
    "github.com/fluxorio/fluxor/pkg/workflow"
)

func main() {
    app, _ := entrypoint.NewMainVerticle("")

    // Create workflow verticle with HTTP API
    wfVerticle := workflow.NewWorkflowVerticle(&workflow.WorkflowVerticleConfig{
        HTTPAddr: ":8081",
    })

    // Register custom functions
    wfVerticle.RegisterFunction("myFunction", func(data interface{}) (interface{}, error) {
        // Process data
        return data, nil
    })

    app.DeployVerticle(wfVerticle)
    app.Start()
}
```

## Workflow Definition

```json
{
  "id": "order-processing",
  "name": "Order Processing",
  "nodes": [
    {
      "id": "start",
      "type": "manual",
      "next": ["validate"]
    },
    {
      "id": "validate",
      "type": "condition",
      "config": {
        "field": "amount",
        "operator": "gt",
        "value": 0
      },
      "trueNext": ["process"],
      "falseNext": ["reject"]
    },
    {
      "id": "process",
      "type": "function",
      "config": {"function": "processOrder"},
      "next": ["notify"]
    },
    {
      "id": "notify",
      "type": "http",
      "config": {
        "url": "https://api.example.com/notify",
        "method": "POST"
      }
    },
    {
      "id": "reject",
      "type": "error",
      "config": {"message": "Invalid order amount"}
    }
  ]
}
```

## Node Types

### Trigger Nodes

| Type | Description |
|------|-------------|
| `manual` | Manual trigger (API call) |
| `webhook` | HTTP webhook trigger |
| `schedule` | Cron/interval trigger |
| `event` | EventBus event trigger |

### Action Nodes

| Type | Description | Config |
|------|-------------|--------|
| `function` | Execute registered function | `function`: function name |
| `http` | HTTP request | `url`, `method`, `headers`, `body`, `timeout` |
| `openai` | OpenAI API request | `apiKey`, `model`, `prompt`, `temperature`, `maxTokens` |
| `ai` | Generic AI API (OpenAI, Cursor, Anthropic) | `provider`, `apiKey`, `model`, `prompt`, `temperature` |
| `eventbus` | Send to EventBus | `address`, `action` (publish/send/request) |
| `set` | Set variables | `values`: map of key-value pairs, `storeInVariables`: bool (optional) |
| `variable` | Define variables | `values`: map of key-value pairs |
| `code` | Transform data | `transform`: transformation rules |
| `subworkflow` | Execute nested workflow | `workflowId`, `inputField`, `outputField` |

### Flow Control Nodes

| Type | Description | Config |
|------|-------------|--------|
| `condition` | If/else branch | `field`, `operator`, `value` |
| `ifthenelse` | Structured IF-THEN-ELSE with inline branches | `condition`: {field, operator, value}, `then`: array of node definitions, `else`: array of node definitions |
| `switch` | Multi-way branch | `field`, `cases`, `default` |
| `split` | Parallel execution | (uses all `next` nodes) |
| `merge` | Wait for inputs | `mode`: waitAll/waitAny |
| `loop` | Iterate array | `items`: field name |
| `dynamicloop` | Dynamic loop with custom next node | `itemsField`, `nextNode`, `batchSize` |
| `wait` | Delay | `duration`: e.g., "5s" |
| `subworkflow` | Execute nested workflow | `workflowId`, `inputField`, `outputField` |

### Utility Nodes

| Type | Description |
|------|-------------|
| `noop` | Pass-through |
| `error` | Throw error |
| `filter` | Filter array |
| `map` | Transform array items |
| `reduce` | Reduce array |

## Condition Operators

- `eq`, `==`, `equals` - Equal
- `ne`, `!=`, `notEquals` - Not equal
- `gt`, `>` - Greater than
- `lt`, `<` - Less than
- `gte`, `>=` - Greater than or equal
- `lte`, `<=` - Less than or equal
- `contains` - Contains
- `exists` - Not null
- `empty` - Is empty
- `notEmpty` - Is not empty

## Structured IF-THEN-ELSE Node

The `ifthenelse` node allows defining conditional branches inline within a single node configuration. Unlike the `condition` node which requires separate nodes connected via `trueNext`/`falseNext`, the `ifthenelse` node executes inline node definitions from the "then" or "else" branches sequentially.

### Configuration

- `condition`: Object with `field`, `operator`, and `value` (same as condition node)
- `then`: Array of node definitions to execute if condition is true
- `else`: Array of node definitions to execute if condition is false (optional)

### Example

```json
{
  "id": "conditional-processing",
  "type": "ifthenelse",
  "config": {
    "condition": {
      "field": "amount",
      "operator": "gt",
      "value": 100
    },
    "then": [
      {
        "type": "set",
        "config": {
          "values": {"status": "premium", "tier": "gold"}
        }
      },
      {
        "type": "http",
        "config": {
          "url": "https://api.example.com/premium",
          "method": "POST"
        }
      }
    ],
    "else": [
      {
        "type": "set",
        "config": {
          "values": {"status": "standard", "tier": "silver"}
        }
      }
    ]
  },
  "next": ["notify"]
}
```

Nodes in the `then` and `else` branches execute sequentially, with each node's output becoming the input to the next node. The final output from the branch is returned as the node's output.

## Variables

Variables provide a way to store values that persist throughout workflow execution and are accessible to all nodes via `input.Context.Variables`.

### Variable Definition Methods

1. **Workflow-level variables**: Define variables at the workflow definition level
2. **set node with `storeInVariables`**: Use the `set` node with `storeInVariables: true` option
3. **variable node**: Dedicated node type for defining variables

### Workflow-Level Variables

Define variables in the workflow definition JSON:

```json
{
  "id": "my-workflow",
  "name": "My Workflow",
  "variables": {
    "apiBaseUrl": "https://api.example.com",
    "timeout": 30,
    "apiKey": "secret-key"
  },
  "nodes": [...]
}
```

These variables are initialized when the workflow execution starts and are available to all nodes.

### Set Node with Variables

The `set` node can store values in `ExecutionContext.Variables` using the `storeInVariables` option:

```json
{
  "id": "set-vars",
  "type": "set",
  "config": {
    "values": {
      "userId": "123",
      "status": "active"
    },
    "storeInVariables": true
  },
  "next": ["process"]
}
```

When `storeInVariables` is `true`, values are stored in `ExecutionContext.Variables` instead of being merged into output data. When `false` or not set (default), values are merged into output data as before (backward compatible).

### Variable Node

The `variable` node is a dedicated node type for defining variables:

```json
{
  "id": "define-vars",
  "type": "variable",
  "config": {
    "values": {
      "apiKey": "secret-key",
      "retryCount": 3,
      "maxRetries": 5
    }
  },
  "next": ["use-vars"]
}
```

The `variable` node stores values in `ExecutionContext.Variables` and returns input data unchanged (variables persist in context, not in data flow).

### Accessing Variables

Variables are accessible to all node handlers via `input.Context.Variables`:

```go
func myNodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
    // Access variables
    if input.Context != nil && input.Context.Variables != nil {
        apiKey := input.Context.Variables["apiKey"]
        timeout := input.Context.Variables["timeout"]
        // Use variables...
    }
    // ...
}
```

Variables persist throughout the workflow execution and are shared across all nodes in the same execution.

## Template Variables

Use `{{field}}` syntax in strings to reference data:

```json
{
  "type": "http",
  "config": {
    "url": "https://api.example.com/users/{{userId}}",
    "headers": {
      "Authorization": "Bearer {{token}}"
    }
  }
}
```

## Programmatic Workflow Building

```go
wf := workflow.NewWorkflowBuilder("my-workflow", "My Workflow").
    AddNode("start", "manual").
    Next("process").
    Done().
    AddNode("process", "function").
    Config(map[string]interface{}{"function": "myFunc"}).
    Next("end").
    Retry(3).
    Timeout(30 * time.Second).
    Done().
    AddNode("end", "noop").
    Done().
    Build()

engine.RegisterWorkflow(wf)
```

## HTTP API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/workflows` | GET | List all workflows |
| `/workflows` | POST | Register workflow |
| `/workflows/:id/execute` | POST | Execute workflow |
| `/executions/:id` | GET | Get execution status |
| `/executions/:id/cancel` | POST | Cancel execution |
| `/health` | GET | Health check |

## Event-Driven Execution

Workflows use EventBus internally:

```
workflow.{workflowId}.execute     → Start workflow
workflow.{workflowId}.node.{id}   → Execute specific node
```

You can also trigger workflows via EventBus:

```go
// Register event trigger
workflow.RegisterEventTrigger(eventBus, engine, workflow.EventTriggerConfig{
    Address:    "orders.new",
    WorkflowID: "order-processing",
})

// Trigger from anywhere
eventBus.Publish("orders.new", orderData)
```

## Generic AI Node (OpenAI, Cursor, Anthropic, etc.)

The generic AI node supports multiple AI providers including OpenAI, Cursor, Anthropic, and any OpenAI-compatible API.

### Supported Providers

- **openai** - OpenAI API (default)
- **cursor** - Cursor AI (uses OpenAI-compatible API)
- **anthropic** - Anthropic Claude API
- **custom** - Custom provider with custom baseURL

### Configuration

```json
{
  "id": "ai-node",
  "type": "ai",
  "config": {
    "provider": "cursor",  // or "openai", "anthropic", "custom"
    "apiKey": "sk-...",    // Or use $CURSOR_API_KEY env var
    "baseURL": "https://api.openai.com/v1",  // Optional, provider-specific default
    "model": "gpt-4",
    "prompt": "{{ $.input.message }}",
    "temperature": 0.7,
    "maxTokens": 2000
  }
}
```

### Cursor Example

```json
{
  "id": "cursor-workflow",
  "nodes": [
    {
      "id": "start",
      "type": "manual",
      "next": ["cursor"]
    },
    {
      "id": "cursor",
      "type": "ai",
      "config": {
        "provider": "cursor",
        "model": "gpt-4",
        "prompt": "{{ $.input.prompt }}",
        "temperature": 0.7,
        "maxTokens": 2000
      },
      "next": ["format"]
    }
  ]
}
```

### Environment Variables

- `OPENAI_API_KEY` - For OpenAI provider
- `CURSOR_API_KEY` - For Cursor provider (or use OPENAI_API_KEY if compatible)
- `ANTHROPIC_API_KEY` - For Anthropic provider

### Template Syntax

Same as OpenAI node:
- `{{ field }}` - Access field from input data
- `{{ $.input.field }}` - Access field with explicit input prefix
- `{{ $.field }}` - Access root-level field
- `{{ $.input.nested.field }}` - Access nested fields

## OpenAI Node

The OpenAI node allows you to call OpenAI's API directly from workflows with template support.

### Configuration

```json
{
  "id": "openai-node",
  "type": "openai",
  "config": {
    "apiKey": "sk-...",  // Or use $OPENAI_API_KEY env var
    "model": "gpt-3.5-turbo",
    "prompt": "{{ $.input.message }}",
    "temperature": 0.7,
    "maxTokens": 500
  }
}
```

### Template Syntax

The OpenAI node supports template syntax for dynamic prompts:

- `{{ field }}` - Access field from input data
- `{{ $.input.field }}` - Access field with explicit input prefix
- `{{ $.field }}` - Access root-level field
- `{{ $.input.nested.field }}` - Access nested fields

### Example

```json
{
  "id": "chat-workflow",
  "nodes": [
    {
      "id": "start",
      "type": "manual",
      "next": ["openai"]
    },
    {
      "id": "openai",
      "type": "openai",
      "config": {
        "model": "gpt-3.5-turbo",
        "prompt": "You are a helpful assistant. User says: {{ $.input.message }}",
        "temperature": 0.7,
        "maxTokens": 500
      },
      "next": ["format"]
    },
    {
      "id": "format",
      "type": "set",
      "config": {
        "values": {
          "success": true
        }
      }
    }
  ]
}
```

### Chat Completions

For chat-based models, use the `messages` config:

```json
{
  "id": "openai-chat",
  "type": "openai",
  "config": {
    "model": "gpt-4",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful assistant."
      },
      {
        "role": "user",
        "content": "{{ $.input.question }}"
      }
    ],
    "temperature": 0.7
  }
}
```

## Nested Workflows (Sub-Workflows)

Execute nested workflows from within a workflow for modularity and reusability.

### Configuration

```json
{
  "id": "subworkflow-node",
  "type": "subworkflow",
  "config": {
    "workflowId": "data-processing",
    "inputField": "data",
    "outputField": "processed_data",
    "waitForCompletion": true
  }
}
```

### Example

```json
{
  "id": "main-workflow",
  "nodes": [
    {
      "id": "start",
      "type": "manual",
      "next": ["process"]
    },
    {
      "id": "process",
      "type": "subworkflow",
      "config": {
        "workflowId": "data-processing",
        "inputField": "items"
      },
      "next": ["format"]
    }
  ]
}
```

## Dynamic Loops

Execute nodes dynamically for each item in an array with custom next node.

### Configuration

```json
{
  "id": "dynamic-loop",
  "type": "dynamicloop",
  "config": {
    "itemsField": "items",
    "nextNode": "process-item",
    "itemField": "item",
    "indexField": "index",
    "batchSize": 5
  }
}
```

### Example

```json
{
  "id": "batch-processing",
  "nodes": [
    {
      "id": "start",
      "type": "manual",
      "next": ["loop"]
    },
    {
      "id": "loop",
      "type": "dynamicloop",
      "config": {
        "itemsField": "items",
        "nextNode": "process-item"
      }
    },
    {
      "id": "process-item",
      "type": "function",
      "config": {
        "function": "processItem"
      }
    }
  ]
}
```

## Example: Order Processing Pipeline

```
┌──────────┐
│  Start   │
└────┬─────┘
     │
     ▼
┌──────────────┐     ┌──────────┐
│   Validate   │────►│  Reject  │
│  amount > 0  │ no  └──────────┘
└──────┬───────┘
       │ yes
       ▼
┌──────────────┐
│   Process    │
│  (function)  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Calculate   │
│   Discount   │
└──────┬───────┘
       │
       ▼
┌──────────────────┐     ┌──────────┐
│  Check Amount    │────►│ Standard │
│  > 100 (premium) │ no  └────┬─────┘
└──────┬───────────┘          │
       │ yes                  │
       ▼                      │
┌──────────┐                  │
│ Premium  │                  │
└────┬─────┘                  │
     │                        │
     └────────────┬───────────┘
                  │
                  ▼
            ┌──────────┐
            │  Format  │
            │ Response │
            └──────────┘
```

## See Also

- [examples/workflow-demo](../../examples/workflow-demo/) - Full working example
- [PRIMARY_PATTERN.md](../../docs/PRIMARY_PATTERN.md) - Fluxor patterns
