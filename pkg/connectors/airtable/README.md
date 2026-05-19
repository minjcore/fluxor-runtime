# Airtable Connector

The `pkg/connectors/airtable` package provides comprehensive Airtable integration for Fluxor, following Fluxor's reactive patterns and fail-fast principles.

## Features

- ✅ **Full Airtable API Integration** - Complete support for Tables and Records operations
- ✅ **Connector Interface** - Implements standard `connectors.Connector` interface
- ✅ **EventBus Integration** - Airtable events published to Fluxor EventBus
- ✅ **Component Lifecycle** - Proper startup/shutdown with BaseComponent pattern
- ✅ **BaseConfig Inheritance** - Full configuration system with validation
- ✅ **Rate Limiting** - Built-in rate limiting (5 requests/second per Airtable limits)
- ✅ **Retry Mechanism** - Automatic retries with exponential backoff
- ✅ **Fail-Fast Validation** - Immediate error detection
- ✅ **Context Support** - Full context support for cancellation and timeouts
- ✅ **Health Checking** - Built-in health check capability
- ✅ **Metadata & Capabilities** - Self-describing connector with capabilities

## Quick Start

### Installation

The Airtable connector is part of the Fluxor framework. No additional dependencies required beyond the standard library.

### Basic Usage

```go
package main

import (
    "context"
    "github.com/fluxorio/fluxor/pkg/connectors/airtable"
    "github.com/fluxorio/fluxor/pkg/core"
    "log"
)

type MyVerticle struct {
    *core.BaseVerticle
    airtable *airtable.AirtableComponent
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Create Airtable component
    config := airtable.DefaultConfig()
    config.APIKey = "keyXXXXXXXXXXXXXX"
    config.BaseID = "appXXXXXXXXXXXXXX"
    v.airtable = airtable.NewAirtableComponent(config)

    // Start component
    if err := v.airtable.Start(ctx); err != nil {
        return err
    }

    // Use Records client
    records, err := v.airtable.Records()
    if err != nil {
        return err
    }

    // Create a record
    newRecord := &airtable.Record{
        Fields: map[string]interface{}{
            "Name":   "My Task",
            "Status": "In Progress",
        },
    }

    created, err := records.Create(ctx.Context(), "Tasks", newRecord)
    if err != nil {
        return err
    }

    log.Printf("Created record: %s", created.ID)
    return nil
}

func (v *MyVerticle) Stop(ctx core.FluxorContext) error {
    if v.airtable != nil {
        return v.airtable.Stop(ctx)
    }
    return nil
}
```

## Configuration

### Environment Variables

The package supports standard Airtable environment variables:

- `AIRTABLE_API_KEY` - Airtable API key (required)
- `AIRTABLE_BASE_ID` - Airtable base ID (required)
- `AIRTABLE_TIMEOUT` - Request timeout (default: 30s)
- `AIRTABLE_MAX_RETRIES` - Maximum retries (default: 3)
- `AIRTABLE_RATE_LIMIT` - Requests per second (default: 5, max: 5)

### Programmatic Configuration

```go
config := airtable.Config{
    APIKey:     "keyXXXXXXXXXXXXXX",
    BaseID:     "appXXXXXXXXXXXXXX",
    Timeout:    "30s",
    MaxRetries: 3,
    RateLimit:  5,
}

// Config also embeds BaseConfig for additional features
config.Service.Name = "airtable-service"
config.Server.Addr = ":8080"
```

### Getting API Credentials

1. Go to https://airtable.com/account
2. Generate an API key under "API" section
3. Find your base ID in the API documentation for your base (starts with "app")

## API Reference

### Tables

#### List Tables

```go
tables, _ := airtableComponent.Tables()

// List all tables in the base
tablesList, err := tables.List(ctx)
if err != nil {
    log.Fatal(err)
}

for _, table := range tablesList {
    log.Printf("Table: %s (ID: %s)", table.Name, table.ID)
    log.Printf("  Primary Field: %s", table.PrimaryField)
    for _, field := range table.Fields {
        log.Printf("  Field: %s (Type: %s)", field.Name, field.Type)
    }
}
```

#### Get Table Metadata

```go
tables, _ := airtableComponent.Tables()

// Get by ID
table, err := tables.Get(ctx, "tblXXXXXXXXXXXXXX")

// Or get by name
table, err := tables.Get(ctx, "Tasks")
```

### Records

#### Create Record

```go
records, _ := airtableComponent.Records()

newRecord := &airtable.Record{
    Fields: map[string]interface{}{
        "Name":        "Complete project",
        "Status":      "In Progress",
        "Priority":    "High",
        "Due Date":    "2024-12-31",
        "Assignee":    []string{"recUserID123"},
        "Description": "Finish the implementation",
    },
}

created, err := records.Create(ctx, "Tasks", newRecord)
if err != nil {
    log.Fatal(err)
}

log.Printf("Created record ID: %s", created.ID)
log.Printf("Created at: %s", created.CreatedTime)
```

#### Get Record

```go
records, _ := airtableComponent.Records()

record, err := records.Get(ctx, "Tasks", "recXXXXXXXXXXXXXX")
if err != nil {
    log.Fatal(err)
}

log.Printf("Record: %+v", record.Fields)
```

#### Update Record

```go
records, _ := airtableComponent.Records()

updateRecord := &airtable.Record{
    Fields: map[string]interface{}{
        "Status": "Completed",
    },
}

updated, err := records.Update(ctx, "Tasks", "recXXXXXXXXXXXXXX", updateRecord)
if err != nil {
    log.Fatal(err)
}

log.Printf("Updated record: %s", updated.ID)
```

#### Delete Record

```go
records, _ := airtableComponent.Records()

err := records.Delete(ctx, "Tasks", "recXXXXXXXXXXXXXX")
if err != nil {
    log.Fatal(err)
}

log.Println("Record deleted successfully")
```

#### List Records

```go
records, _ := airtableComponent.Records()

// Simple list
allRecords, err := records.List(ctx, "Tasks", airtable.ListParams{})

// With filtering and sorting
params := airtable.ListParams{
    MaxRecords:      100,
    PageSize:        20,
    View:            "My View",
    FilterByFormula: "AND({Status} = 'In Progress', {Priority} = 'High')",
    Sort: []airtable.SortParam{
        {Field: "Due Date", Direction: "asc"},
        {Field: "Priority", Direction: "desc"},
    },
}

filteredRecords, err := records.List(ctx, "Tasks", params)
if err != nil {
    log.Fatal(err)
}

for _, record := range filteredRecords {
    log.Printf("Record %s: %+v", record.ID, record.Fields)
}
```

#### Pagination

```go
records, _ := airtableComponent.Records()

params := airtable.ListParams{
    PageSize: 100,
}

allRecords := []airtable.Record{}

for {
    pageRecords, err := records.List(ctx, "Tasks", params)
    if err != nil {
        log.Fatal(err)
    }

    allRecords = append(allRecords, pageRecords...)

    // Check if there are more pages
    // Note: Current implementation returns records from single page
    // For full pagination support, you would check the offset in the response
    break
}

log.Printf("Total records fetched: %d", len(allRecords))
```

## EventBus Integration

The Airtable component publishes events to Fluxor's EventBus:

### Events

- `airtable.ready` - Published when Airtable component is ready
  ```json
  {
    "component": "airtable",
    "baseID": "appXXXXXXXXXXXXXX"
  }
  ```

- `airtable.stopped` - Published when Airtable component is stopped
  ```json
  {
    "component": "airtable"
  }
  ```

### Listening to Events

```go
eventBus := gocmd.EventBus()

// Listen for Airtable ready event
eventBus.Consumer("airtable.ready").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var data map[string]interface{}
    if err := msg.Decode(&data); err != nil {
        return err
    }

    baseID := data["baseID"].(string)
    log.Printf("Airtable component ready for base: %s", baseID)
    return nil
})

// Listen for Airtable stopped event
eventBus.Consumer("airtable.stopped").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    log.Println("Airtable component stopped")
    return nil
})
```

## Connector Interface

The Airtable component implements the standard `connectors.Connector` interface, making it compatible with the Fluxor connector registry and providing standardized metadata and health checking.

### Connector Metadata

```go
// Get connector metadata
metadata := airtableComponent.GetMetadata()

fmt.Printf("Name: %s\n", metadata.Name)                 // "airtable"
fmt.Printf("Display Name: %s\n", metadata.DisplayName)  // "Airtable"
fmt.Printf("Type: %s\n", metadata.Type)                 // "productivity"
fmt.Printf("Version: %s\n", metadata.Version)           // "1.0.0"

// Check capabilities
for _, cap := range metadata.Capabilities {
    fmt.Printf("- %s: %s (enabled: %v)\n", cap.Name, cap.Description, cap.Enabled)
}
// Output:
// - read: Read tables and records from Airtable (enabled: true)
// - write: Create and update records in Airtable (enabled: true)
// - delete: Delete records from Airtable (enabled: true)
// - metadata: Access table and field metadata (enabled: true)

// Check rate limits
fmt.Printf("Rate Limit: %d req/sec\n", metadata.RateLimits.RequestsPerSecond) // 5
```

### Health Checking

```go
// Check if connector is healthy
healthy, err := airtableComponent.IsHealthy(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)
}

if healthy {
    log.Println("Airtable connector is healthy and operational")
} else {
    log.Println("Airtable connector is not healthy")
}
```

### Connector Registry

Register the Airtable connector in the global registry for discovery:

```go
import "github.com/fluxorio/fluxor/pkg/connectors"

// Register globally
if err := connectors.Register(airtableComponent); err != nil {
    log.Fatal(err)
}

// List all registered connectors
allConnectors := connectors.List()
for _, conn := range allConnectors {
    fmt.Printf("Found connector: %s (%s)\n", conn.Name(), conn.Type())
}

// Get connector by name
conn, exists := connectors.Get("airtable")
if exists {
    fmt.Printf("Retrieved connector: %s version %s\n", conn.Name(), conn.Version())
}

// List productivity connectors
productivityConns := connectors.ListByType(connectors.TypeProductivity)
for _, conn := range productivityConns {
    fmt.Printf("Productivity connector: %s\n", conn.Name())
}
```

### Using Registry in Verticles

```go
type MyVerticle struct {
    *core.BaseVerticle
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Discover Airtable connector
    conn, exists := connectors.Get("airtable")
    if !exists {
        return fmt.Errorf("airtable connector not found")
    }

    // Check if it's healthy
    healthy, err := conn.IsHealthy(ctx.Context())
    if err != nil || !healthy {
        return fmt.Errorf("airtable connector is not healthy: %w", err)
    }

    // Cast to AirtableComponent to access specific methods
    airtable := conn.(*airtable.AirtableComponent)
    tables, _ := airtable.Tables()

    // Use the connector...
    return nil
}
```

## Error Handling

All methods follow Fluxor's fail-fast pattern:

```go
records, err := airtableComponent.Records()
if err != nil {
    // Component not started or invalid
    return err
}

record, err := records.Get(ctx, "Tasks", "recXXX")
if err != nil {
    // Handle Airtable API error
    if airtableErr, ok := err.(*airtable.AirtableError); ok {
        log.Printf("Airtable error (%s): %s", airtableErr.Error.Type, airtableErr.Error.Message)
    }
    return err
}
```

### Common Error Types

- `ConfigError` - Configuration validation errors
- `AirtableError` - Airtable API errors (rate limits, not found, etc.)
- `EventBusError` - Component lifecycle errors

## Best Practices

1. **Always check errors** - All methods return errors that should be handled
2. **Use context** - Pass context for cancellation and timeouts
3. **Reuse component** - Create component once, reuse across verticles
4. **EventBus integration** - Listen to Airtable events for reactive patterns
5. **Rate limiting** - The connector automatically handles Airtable's rate limits
6. **Batch operations** - For bulk operations, consider batching to minimize API calls

## Advanced Usage

### Custom Timeout

```go
config := airtable.DefaultConfig()
config.APIKey = "keyXXXXXXXXXXXXXX"
config.BaseID = "appXXXXXXXXXXXXXX"
config.Timeout = "60s" // Longer timeout for slow operations
```

### Retry Configuration

```go
config := airtable.DefaultConfig()
config.MaxRetries = 5 // More retries for unreliable networks
```

### Context Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

records, _ := airtableComponent.Records()
record, err := records.Get(ctx, "Tasks", "recXXX")
if err == context.DeadlineExceeded {
    log.Println("Operation timed out")
}
```

## Integration with Workflows

Airtable connector can be used in Fluxor workflows:

```json
{
  "id": "airtable-workflow",
  "nodes": [
    {
      "id": "trigger",
      "type": "webhook"
    },
    {
      "id": "create-record",
      "type": "function",
      "config": {
        "code": "records, _ := airtable.Records(); records.Create(ctx, 'Tasks', record)"
      }
    },
    {
      "id": "notify",
      "type": "eventbus",
      "config": {
        "address": "task.created"
      }
    }
  ]
}
```

## Testing

The connector includes comprehensive tests:

```bash
cd pkg/connectors/airtable
go test -v ./...
```

### Mock Testing

```go
// Use httptest to mock Airtable API responses
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Mock response
    json.NewEncoder(w).Encode(mockRecord)
}))
defer server.Close()
```

## Roadmap

Future enhancements:

- [ ] Batch create/update operations
- [ ] Attachment upload support
- [ ] Webhook integration
- [ ] Field type validation
- [ ] Automatic pagination helper
- [ ] Bulk delete operations
- [ ] Real-time collaboration features
- [ ] Integration with Airtable Automations

## API Rate Limits

Airtable enforces the following rate limits:

- **5 requests per second per base** (enforced by connector)
- Consider caching frequently accessed data
- Use batch operations when available

## Troubleshooting

### Authentication Errors

```
airtable error (AUTHENTICATION_REQUIRED): Authentication required
```

**Solution**: Check that your API key is correct and set in config or environment variable.

### Rate Limit Errors

```
airtable error (RATE_LIMIT): Rate limit exceeded
```

**Solution**: The connector automatically retries, but you may need to reduce request frequency or increase RateLimit config (max 5).

### Not Found Errors

```
airtable error (NOT_FOUND): Record not found
```

**Solution**: Verify the record/table ID exists in your base.

## License

Part of Fluxor framework - see main LICENSE file.
