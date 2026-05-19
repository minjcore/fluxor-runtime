package aimodule

import (
	"fmt"
	"os"
	"sync"
)

// ClientFactory creates LLM clients based on provider
type ClientFactory func(config Config) (LLMClient, error)

var (
	clientFactories map[Provider]ClientFactory
	factoryMu       sync.RWMutex
)

func init() {
	clientFactories = make(map[Provider]ClientFactory)
}

// RegisterClientFactory registers a client factory for a provider
func RegisterClientFactory(provider Provider, factory ClientFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	clientFactories[provider] = factory
}

// NewClient creates a new LLM client based on the configuration
func NewClient(config Config) (LLMClient, error) {
	// Fail-fast: Validate provider
	if config.Provider == "" {
		config.Provider = ProviderOpenAI
	}

	// Handle Model field (backward compatibility with pkg/ai)
	if config.Model != "" && config.DefaultModel == "" {
		config.DefaultModel = config.Model
	}

	// Get API key from config or environment
	if config.APIKey == "" {
		envVar := getProviderEnvVar(config.Provider)
		config.APIKey = os.Getenv(envVar)
		if config.APIKey == "" && config.Provider != ProviderOllama {
			return nil, fmt.Errorf("ai client requires 'apiKey' config or %s env var", envVar)
		}
	}

	// Get factory for provider
	factoryMu.RLock()
	factory, exists := clientFactories[config.Provider]
	factoryMu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	return factory(config)
}

// getProviderEnvVar returns the environment variable name for a provider's API key
func getProviderEnvVar(provider Provider) string {
	switch provider {
	case ProviderOpenAI:
		return "OPENAI_API_KEY"
	case ProviderAnthropic:
		return "ANTHROPIC_API_KEY"
	case ProviderOllama:
		return "" // Ollama doesn't need API key
	case ProviderGrok:
		return "GROK_API_KEY"
	case ProviderGemini:
		return "GEMINI_API_KEY"
	case ProviderCursor:
		return "CURSOR_API_KEY"
	default:
		return "AI_API_KEY"
	}
}
