// Package elasticsearch provides Elasticsearch integration for Fluxor.
//
// This package implements the Connector interface and provides a high-level
// API for Elasticsearch operations including index management, document operations,
// and search functionality.
//
// Example usage:
//
//	// Create Elasticsearch component with configuration
//	config := elasticsearch.DefaultConfig()
//	config.Addresses = []string{"http://localhost:9200"}
//	config.Username = "elastic"
//	config.Password = "changeme"
//
//	component := elasticsearch.NewElasticsearchComponent(config)
//
//	// Start the component
//	ctx := core.NewFluxorContext(...)
//	if err := component.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer component.Stop(ctx)
//
//	// Create an index
//	indices, _ := component.Indices()
//	if err := indices.Create(context.Background(), "my-index", nil); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Index a document
//	documents, _ := component.Documents()
//	doc := map[string]interface{}{
//	    "title": "Example Document",
//	    "content": "This is an example",
//	}
//	if err := documents.Index(context.Background(), "my-index", "doc-1", doc); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Search documents
//	search, _ := component.Search()
//	query := map[string]interface{}{
//	    "query": map[string]interface{}{
//	        "match": map[string]interface{}{
//	            "title": "Example",
//	        },
//	    },
//	}
//	results, err := search.Search(context.Background(), "my-index", query)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get a document
//	doc, err := documents.Get(context.Background(), "my-index", "doc-1")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Features:
//   - Index management (create, delete, exists, list, settings, mapping)
//   - Document operations (index, get, update, delete, exists)
//   - Search operations (search, count)
//   - Bulk operations for high throughput
//   - Support for authentication (basic auth, API key, Cloud ID)
//   - Retry mechanism with exponential backoff
//   - Rate limiting
//
// Configuration:
//   - ELASTICSEARCH_ADDRESSES: Comma-separated list of Elasticsearch node addresses (required)
//   - ELASTICSEARCH_USERNAME: Username for basic authentication (optional)
//   - ELASTICSEARCH_PASSWORD: Password for basic authentication (optional)
//   - ELASTICSEARCH_API_KEY: API key for authentication (optional)
//   - ELASTICSEARCH_CLOUD_ID: Elastic Cloud ID (optional)
//   - ELASTICSEARCH_DEFAULT_INDEX: Default index name (optional)
//   - ELASTICSEARCH_TIMEOUT: Timeout for API calls (default: 30s)
//   - ELASTICSEARCH_MAX_RETRIES: Maximum retries (default: 3)
//   - ELASTICSEARCH_DEBUG: Enable debug logging (default: false)
package elasticsearch
