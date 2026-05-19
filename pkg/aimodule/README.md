# AI Module

The AI module (`pkg/aimodule`) provides a production-ready AI integration for Fluxor workflows with multi-provider support, tool calling, caching, and rate limiting.

## Features

- **Multi-provider support**: OpenAI, Anthropic, Ollama, Grok, Gemini, Cursor, and custom providers
- **Tool calling**: Full support for OpenAI function calling
- **Caching**: In-memory response caching with TTL
- **Rate limiting**: Token bucket rate limiter with per-minute and per-day limits
- **Workflow integration**: Seamless integration with Fluxor workflow engine
- **Component integration**: AIComponent for Fluxor component system with EventBus integration
- **Template support**: Prompt templating with `{{ $.input.field }}` syntax
- **Retry logic**: Automatic retry with exponential backoff
- **Simple API**: ChatSimple method for quick chat completions
- **Production ready**: Error handling, timeout management, and logging

## Quick Start

### Using in Workflows

```json
{
  "id": "ai-chat-workflow",
  "nodes": [
    {
      "id": "trigger",
      "type": "webhook"
    },
    {
      "id": "ai-chat",
      "type": "aimodule.chat",
      "config": {
        "provider": "openai",
        "model": "gpt-4o",
        "prompt": "Bạn là trợ lý hỗ trợ khách hàng MoMo. Trả lời: {{ $.input.query }}",
        "temperature": 0.7,
        "maxTokens": 500
      },
      "next": ["output"]
    },
    {
      "id": "output",
      "type": "respond"
    }
  ]
}
```

### Using AI Component (Recommended)

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/aimodule"
)

func main() {
    app, _ := entrypoint.NewMainVerticle("config.json")
    
    // Create and deploy AI component
    aiConfig := aimodule.Config{
        Provider:     aimodule.ProviderOpenAI,
        APIKey:       "your-api-key", // or use OPENAI_API_KEY env var
        DefaultModel: "gpt-3.5-turbo",
    }
    aiComponent := aimodule.NewAIComponent(aiConfig)
    
    // Deploy verticle that uses AI
    app.DeployVerticle(NewMyVerticle(aiComponent))
    app.Start()
}

type MyVerticle struct {
    *core.BaseVerticle
    aiComponent *aimodule.AIComponent
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Use AI component
    response, err := v.aiComponent.ChatSimple(ctx, "Hello, world!")
    if err != nil {
        return err
    }
    // Process response...
    return nil
}
```

### Using Client Directly

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/aimodule"
)

// Create client
config := aimodule.Config{
    Provider:     aimodule.ProviderOpenAI,
    APIKey:       "your-api-key", // or use OPENAI_API_KEY env var
    DefaultModel: "gpt-4o",
    Timeout:      60 * time.Second,
    MaxRetries:   3,
}

client, err := aimodule.NewClient(config)
if err != nil {
    log.Fatal(err)
}

// Simple chat
response, err := client.ChatSimple(context.Background(), "What is Fluxor?")
if err != nil {
    log.Fatal(err)
}
fmt.Println(response)

// Advanced chat with multiple messages
req := aimodule.ChatRequest{
    Model: "gpt-4o",
    Messages: []aimodule.Message{
        {Role: "user", Content: "Hello, world!"},
    },
    Temperature: 0.7,
}

resp, err := client.Chat(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

fmt.Println(resp.Choices[0].Message.Content)
```

## Configuration

### Environment Variables

- `OPENAI_API_KEY` - OpenAI API key
- `ANTHROPIC_API_KEY` - Anthropic API key
- `GROK_API_KEY` - Grok API key
- `GEMINI_API_KEY` - Gemini API key

### Config Options

```go
type Config struct {
    Provider     Provider        // AI provider
    APIKey       string          // API key (or use env var)
    BaseURL      string          // Base URL (optional)
    DefaultModel string          // Default model
    Model        string          // Alias for DefaultModel (backward compatibility)
    Timeout      time.Duration  // Request timeout
    MaxRetries   int            // Max retry attempts
    RateLimit    *RateLimitConfig // Rate limiting
    Cache        *CacheConfig   // Response caching
}
```

## Workflow Nodes

### `aimodule.chat`

Chat completion node with template support.

**Config:**
- `provider` - Provider name (default: "openai")
- `model` - Model name
- `prompt` - Prompt template (supports `{{ $.input.field }}`)
- `messages` - Array of messages (alternative to prompt)
- `temperature` - Temperature (0-2, default: 1.0)
- `maxTokens` - Max tokens
- `tools` - Array of tool definitions for function calling
- `toolChoice` - Tool choice ("auto", "none", or function name)
- `responseField` - Output field name (default: "response")

### `aimodule.embed`

Embedding generation node.

**Config:**
- `provider` - Provider name (default: "openai")
- `model` - Embedding model (default: "text-embedding-ada-002")
- `input` - Text or array of texts to embed
- `outputField` - Output field name (default: "embeddings")

### `aimodule.toolcall`

Tool calling node (uses chat with tools).

Same config as `aimodule.chat` with tools enabled.

## Examples

See `examples/aimodule-workflow/` for complete examples:
- `main.go` - Basic chat workflow
- `workflow-tool-calling.json` - Tool calling example
- `workflow-embedding.json` - Embedding example

## Parse Handler

The parse handler provides flexible parsing of AI model responses:

```go
import "github.com/fluxorio/fluxor/pkg/aimodule"

// Simple parsing - just get text
text, err := aimodule.ParseResponseSimple(resp)

// Custom parsing with options
result, err := aimodule.ParseResponseWithOptions(
    resp,
    aimodule.ProviderOpenAI,
    aimodule.WithExtractText(true),
    aimodule.WithExtractJSON(true),
    aimodule.WithResponseField("answer"),
    aimodule.WithIncludeMetadata(true),
)

// Parse tool calls
toolCalls, err := aimodule.ParseToolCalls(resp)

// Parse usage information
usage := aimodule.ParseUsage(resp)
```

### Parse Options

- `WithExtractText(bool)` - Extract plain text (default: true)
- `WithExtractJSON(bool)` - Extract and parse JSON from response
- `WithResponseField(string)` - Set output field name (default: "response")
- `WithIncludeMetadata(bool)` - Include usage, model, provider info
- `WithIncludeFullResponse(bool)` - Include full response object
- `WithCustomExtractor(func)` - Custom extraction function

## Architecture

```
pkg/aimodule/
├── types.go          - Core types and interfaces
├── client.go          - Client factory and registry
├── openai.go          - OpenAI implementation
├── component.go       - AIComponent for Fluxor component system
├── parse_handler.go   - Response parsing and extraction
├── cache.go           - Response caching
├── ratelimit.go       - Rate limiting
├── nodes_ai.go        - Workflow node handlers
├── config.go          - Configuration management
└── nodes.go           - Node registration (deprecated, handled in workflow)
```

## Future Enhancements

- Ollama local support
- Anthropic client implementation
- Vector search integration
- Redis cache backend
- Metrics and observability
- Streaming support

