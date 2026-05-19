package elasticsearch

import (
	"context"
	"sync"

	"github.com/fluxorio/fluxor/pkg/connectors"
	"github.com/fluxorio/fluxor/pkg/core"
)

// ElasticsearchComponent provides Elasticsearch integration with Fluxor
type ElasticsearchComponent struct {
	*core.BaseComponent
	config  Config
	client  Client
	mu      sync.RWMutex
	started bool
}

// NewElasticsearchComponent creates a new Elasticsearch component
func NewElasticsearchComponent(config Config) *ElasticsearchComponent {
	return &ElasticsearchComponent{
		BaseComponent: core.NewBaseComponent("elasticsearch"),
		config:        config,
	}
}

func (c *ElasticsearchComponent) Start(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return &core.EventBusError{Code: "ALREADY_STARTED", Message: "component already started"}
	}

	if err := c.doStart(ctx); err != nil {
		return err
	}

	c.started = true
	return nil
}

func (c *ElasticsearchComponent) Stop(ctx core.FluxorContext) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	if err := c.doStop(ctx); err != nil {
		return err
	}

	c.started = false
	return nil
}

func (c *ElasticsearchComponent) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

func (c *ElasticsearchComponent) doStart(ctx core.FluxorContext) error {
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	if err := c.config.Validate(); err != nil {
		return &core.EventBusError{Code: "CONFIG_VALIDATION_ERROR", Message: err.Error()}
	}

	client, err := NewClient(c.config)
	if err != nil {
		return &core.EventBusError{Code: "ELASTICSEARCH_CLIENT_ERROR", Message: err.Error()}
	}

	c.client = client

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("elasticsearch.ready", map[string]interface{}{
			"component": "elasticsearch",
			"addresses": c.config.Addresses,
		})
	}

	return nil
}

func (c *ElasticsearchComponent) doStop(ctx core.FluxorContext) error {
	c.client = nil

	if eventBus := c.EventBus(); eventBus != nil {
		_ = eventBus.Publish("elasticsearch.stopped", map[string]interface{}{
			"component": "elasticsearch",
		})
	}

	return nil
}

func (c *ElasticsearchComponent) Client() (Client, error) {
	if !c.IsStarted() {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "Elasticsearch component is not started"}
	}
	if c.client == nil {
		return nil, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Elasticsearch client is not initialized"}
	}
	return c.client, nil
}

func (c *ElasticsearchComponent) Indices() (IndicesClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Indices(), nil
}

func (c *ElasticsearchComponent) Documents() (DocumentsClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Documents(), nil
}

func (c *ElasticsearchComponent) Search() (SearchClient, error) {
	client, err := c.Client()
	if err != nil {
		return nil, err
	}
	return client.Search(), nil
}

func (c *ElasticsearchComponent) Name() string {
	return "elasticsearch"
}

func (c *ElasticsearchComponent) Type() connectors.ConnectorType {
	return connectors.TypeDatabase
}

func (c *ElasticsearchComponent) Version() string {
	return "1.0.0"
}

func (c *ElasticsearchComponent) IsHealthy(ctx context.Context) (bool, error) {
	if !c.IsStarted() {
		return false, &core.EventBusError{Code: "NOT_STARTED", Message: "Elasticsearch component is not started"}
	}
	if c.client == nil {
		return false, &core.EventBusError{Code: "CLIENT_NOT_INITIALIZED", Message: "Elasticsearch client is not initialized"}
	}

	// Try to list indices as a health check
	indices, err := c.client.Indices().List(ctx)
	if err != nil {
		return false, err
	}

	// If we can list indices, we're healthy
	_ = indices
	return true, nil
}

func (c *ElasticsearchComponent) GetMetadata() connectors.ConnectorMetadata {
	return connectors.ConnectorMetadata{
		Name:        "elasticsearch",
		DisplayName: "Elasticsearch",
		Description: "Elasticsearch connector for search, indexing, and document management",
		Version:     "1.0.0",
		Type:        connectors.TypeDatabase,
		Author:      "Fluxor Team",
		DocsURL:     "https://github.com/fluxorio/fluxor/tree/main/pkg/connectors/elasticsearch",
		Capabilities: []connectors.Capability{
			{Name: "indices", Description: "Create, delete, and manage indices", Enabled: true},
			{Name: "documents", Description: "Index, get, update, delete documents", Enabled: true},
			{Name: "search", Description: "Search documents with queries", Enabled: true},
			{Name: "bulk", Description: "Bulk operations for high throughput", Enabled: true},
		},
		AuthMethods: []string{"basic_auth", "api_key", "cloud_id"},
		Tags:        []string{"database", "search", "nosql", "elasticsearch", "lucene"},
	}
}
