package aimodule

import (
	"os"
	"sync"
)

var (
	defaultClient LLMClient
	defaultMu     sync.RWMutex
	initialized   bool
	initMu        sync.Mutex
)

// Init initializes the AI module with default configuration
func Init() error {
	initMu.Lock()
	defer initMu.Unlock()

	if initialized {
		return nil
	}

	// Create default OpenAI client if API key is available
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		config := Config{
			Provider:     ProviderOpenAI,
			APIKey:       apiKey,
			DefaultModel: "gpt-3.5-turbo",
		}

		client, err := NewClient(config)
		if err != nil {
			return err
		}

		defaultMu.Lock()
		defaultClient = client
		defaultMu.Unlock()
	}

	initialized = true
	return nil
}

// DefaultClient returns the default AI client
func DefaultClient() LLMClient {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultClient
}

// SetDefaultClient sets the default AI client
func SetDefaultClient(client LLMClient) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultClient = client
}
