# Google Sheets Connector

The `pkg/connectors/google/sheets` package provides comprehensive Google Sheets integration for Fluxor, following Fluxor's reactive patterns and fail-fast principles.

## Features

- ✅ **Full Google Sheets API v4 Integration** - Complete support for read, write, update, append, and clear operations
- ✅ **Connector Interface** - Implements standard `connectors.Connector` interface
- ✅ **EventBus Integration** - Google Sheets events published to Fluxor EventBus
- ✅ **Component Lifecycle** - Proper startup/shutdown with BaseComponent pattern
- ✅ **BaseConfig Inheritance** - Full configuration system with validation
- ✅ **Rate Limiting** - Built-in rate limiting (100 requests per second)
- ✅ **Retry Mechanism** - Automatic retries with exponential backoff
- ✅ **Fail-Fast Validation** - Immediate error detection
- ✅ **Context Support** - Full context support for cancellation and timeouts
- ✅ **Health Checking** - Built-in health check capability
- ✅ **Metadata & Capabilities** - Self-describing connector with capabilities
- ✅ **Batch Operations** - Support for batch read and write operations

## Quick Start

### Installation

The Google Sheets connector is part of the Fluxor framework. For full service account support, you may need to add:

```bash
go get google.golang.org/api/sheets/v4
go get google.golang.org/api/option
```

### Basic Usage

```go
package main

import (
    "context"
    "github.com/fluxorio/fluxor/pkg/connectors/google/sheets"
    "github.com/fluxorio/fluxor/pkg/core"
    "log"
)

type MyVerticle struct {
    *core.BaseVerticle
    sheets *sheets.SheetComponent
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Create Google Sheets component
    config := sheets.DefaultConfig()
    config.OAuth2Token = "your-oauth2-token"
    config.SpreadsheetID = "your-spreadsheet-id"
    v.sheets = sheets.NewSheetComponent(config)

    // Start component
    if err := v.sheets.Start(ctx); err != nil {
        return err
    }

    // Use the Sheets client
    client, err := v.sheets.Client()
    if err != nil {
        return err
    }

    // Read values
    values, err := client.Read(ctx.Context(), "Sheet1!A1:B10")
    if err != nil {
        return err
    }

    log.Printf("Read %d rows", len(values))
    return nil
}

func (v *MyVerticle) Stop(ctx core.FluxorContext) error {
    if v.sheets != nil {
        return v.sheets.Stop(ctx)
    }
    return nil
}
```

## Configuration

### Environment Variables

The package supports standard Google Sheets environment variables:

- `GOOGLE_APPLICATION_CREDENTIALS` - Path to service account credentials JSON file
- `GOOGLE_SHEETS_SPREADSHEET_ID` - Google Sheets spreadsheet ID (required)
- `GOOGLE_SHEETS_OAUTH2_TOKEN` - OAuth2 access token (alternative to service account)
- `GOOGLE_SHEETS_TIMEOUT` - Request timeout (default: 30s)
- `GOOGLE_SHEETS_MAX_RETRIES` - Maximum retries (default: 3)
- `GOOGLE_SHEETS_RATE_LIMIT` - Requests per second (default: 100)

### Programmatic Configuration

```go
config := sheets.Config{
    OAuth2Token:   "ya29.a0AfH6SMB...",
    SpreadsheetID: "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
    Timeout:       "30s",
    MaxRetries:    3,
    RateLimit:     100,
}

// Config also embeds BaseConfig for additional features
config.Service.Name = "sheets-service"
config.Server.Addr = ":8080"
```

### Getting API Credentials

#### OAuth2 Token (Quick Start)

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing one
3. Enable Google Sheets API
4. Create OAuth2 credentials
5. Get access token using OAuth2 flow

#### Service Account (Recommended for Production)

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Navigate to "IAM & Admin" > "Service Accounts"
3. Create a new service account
4. Download JSON key file
5. Share your Google Sheet with the service account email
6. Set `GOOGLE_APPLICATION_CREDENTIALS` environment variable

## API Reference

### Read Values

```go
client, _ := sheetComponent.Client()

// Read a range
values, err := client.Read(ctx, "Sheet1!A1:B10")
if err != nil {
    log.Fatal(err)
}

for i, row := range values {
    log.Printf("Row %d: %v", i+1, row)
}
```

### Write Values

```go
client, _ := sheetComponent.Client()

// Write values (overwrites existing)
values := [][]interface{}{
    {"Name", "Age", "City"},
    {"John", 30, "New York"},
    {"Jane", 25, "San Francisco"},
}

err := client.Write(ctx, "Sheet1!A1:C3", values)
if err != nil {
    log.Fatal(err)
}
```

### Update Values

```go
client, _ := sheetComponent.Client()

// Update specific cells
values := [][]interface{}{
    {"Updated Name"},
}

err := client.Update(ctx, "Sheet1!A2", values)
if err != nil {
    log.Fatal(err)
}
```

### Append Values

```go
client, _ := sheetComponent.Client()

// Append new rows
newRows := [][]interface{}{
    {"Bob", 35, "Chicago"},
    {"Alice", 28, "Boston"},
}

err := client.Append(ctx, "Sheet1!A:C", newRows)
if err != nil {
    log.Fatal(err)
}
```

### Clear Values

```go
client, _ := sheetComponent.Client()

// Clear a range
err := client.Clear(ctx, "Sheet1!A2:C10")
if err != nil {
    log.Fatal(err)
}
```

### Batch Operations

#### Batch Read

```go
client, _ := sheetComponent.Client()

// Read multiple ranges in one request
ranges := []string{
    "Sheet1!A1:B10",
    "Sheet2!A1:C5",
    "Sheet3!D1:E20",
}

results, err := client.BatchRead(ctx, ranges)
if err != nil {
    log.Fatal(err)
}

for range_, values := range results {
    log.Printf("Range %s: %d rows", range_, len(values))
}
```

#### Batch Write

```go
client, _ := sheetComponent.Client()

// Write to multiple ranges in one request
updates := []sheets.BatchUpdate{
    {
        Range_: "Sheet1!A1:B2",
        Values: [][]interface{}{
            {"Header1", "Header2"},
            {"Value1", "Value2"},
        },
    },
    {
        Range_: "Sheet2!A1:C1",
        Values: [][]interface{}{
            {"Col1", "Col2", "Col3"},
        },
    },
}

err := client.BatchWrite(ctx, updates)
if err != nil {
    log.Fatal(err)
}
```

### Get Spreadsheet Metadata

```go
client, _ := sheetComponent.Client()

// Get spreadsheet info
info, err := client.GetSpreadsheetInfo(ctx)
if err != nil {
    log.Fatal(err)
}

log.Printf("Spreadsheet: %s", info.Title)
log.Printf("URL: %s", info.URL)
log.Printf("Sheets: %d", len(info.Sheets))

for _, s := range info.Sheets {
    log.Printf("  - %s (%d rows x %d cols)", s.Title, s.RowCount, s.ColumnCount)
}
```

### Get Sheet Metadata

```go
client, _ := sheetComponent.Client()

// Get specific sheet info
sheetInfo, err := client.GetSheetInfo(ctx, "Sheet1")
if err != nil {
    log.Fatal(err)
}

log.Printf("Sheet: %s", sheetInfo.Title)
log.Printf("Size: %d rows x %d cols", sheetInfo.RowCount, sheetInfo.ColumnCount)
```

## EventBus Integration

The Google Sheets component publishes events to Fluxor's EventBus:

### Events

- `google.sheets.ready` - Published when Google Sheets component is ready
  ```json
  {
    "component": "google-sheets",
    "spreadsheetID": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"
  }
  ```

- `google.sheets.stopped` - Published when Google Sheets component is stopped
  ```json
  {
    "component": "google-sheets"
  }
  ```

### Listening to Events

```go
eventBus := gocmd.EventBus()

// Listen for Google Sheets ready event
eventBus.Consumer("google.sheets.ready").Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var data map[string]interface{}
    if err := msg.Decode(&data); err != nil {
        return err
    }

    spreadsheetID := data["spreadsheetID"].(string)
    log.Printf("Google Sheets component ready for spreadsheet: %s", spreadsheetID)
    return nil
})
```

## Connector Interface

The Google Sheets component implements the standard `connectors.Connector` interface, making it compatible with the Fluxor connector registry.

### Connector Metadata

```go
// Get connector metadata
metadata := sheetComponent.GetMetadata()

fmt.Printf("Name: %s\n", metadata.Name)                 // "google-sheets"
fmt.Printf("Display Name: %s\n", metadata.DisplayName)   // "Google Sheets"
fmt.Printf("Type: %s\n", metadata.Type)                  // "productivity"
fmt.Printf("Version: %s\n", metadata.Version)           // "1.0.0"
```

### Health Checking

```go
// Check if connector is healthy
healthy, err := sheetComponent.IsHealthy(ctx)
if err != nil {
    log.Printf("Health check failed: %v", err)
}

if healthy {
    log.Println("Google Sheets connector is healthy and operational")
}
```

### Connector Registry

```go
import "github.com/fluxorio/fluxor/pkg/connectors"

// Register globally
if err := connectors.Register(sheetComponent); err != nil {
    log.Fatal(err)
}

// Get connector by name
conn, exists := connectors.Get("google-sheets")
if exists {
    fmt.Printf("Retrieved connector: %s version %s\n", conn.Name(), conn.Version())
}
```

## Error Handling

All methods follow Fluxor's fail-fast pattern:

```go
client, err := sheetComponent.Client()
if err != nil {
    // Component not started or invalid
    return err
}

values, err := client.Read(ctx, "Sheet1!A1:B10")
if err != nil {
    // Handle Google Sheets API error
    if sheetsErr, ok := err.(*sheets.SheetsError); ok {
        log.Printf("Google Sheets error (%s): %s", sheetsErr.Code, sheetsErr.Message)
    }
    return err
}
```

### Common Error Types

- `ConfigError` - Configuration validation errors
- `SheetsError` - Google Sheets API errors (rate limits, not found, etc.)
- `EventBusError` - Component lifecycle errors

## Best Practices

1. **Always check errors** - All methods return errors that should be handled
2. **Use context** - Pass context for cancellation and timeouts
3. **Reuse component** - Create component once, reuse across verticles
4. **Batch operations** - Use batch read/write for multiple ranges to minimize API calls
5. **Rate limiting** - The connector automatically handles Google Sheets rate limits
6. **Service accounts** - Use service accounts for production applications
7. **Share sheets** - Remember to share your Google Sheet with the service account email

## Advanced Usage

### Custom Timeout

```go
config := sheets.DefaultConfig()
config.OAuth2Token = "your-token"
config.SpreadsheetID = "your-spreadsheet-id"
config.Timeout = "60s" // Longer timeout for slow operations
```

### Retry Configuration

```go
config := sheets.DefaultConfig()
config.MaxRetries = 5 // More retries for unreliable networks
```

### Context Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

client, _ := sheetComponent.Client()
values, err := client.Read(ctx, "Sheet1!A1:B10")
if err == context.DeadlineExceeded {
    log.Println("Operation timed out")
}
```

## Range Notation

Google Sheets uses A1 notation for ranges:

- Single cell: `"Sheet1!A1"`
- Range: `"Sheet1!A1:B10"`
- Entire column: `"Sheet1!A:A"`
- Entire row: `"Sheet1!1:1"`
- Multiple ranges: Use batch operations

## API Rate Limits

Google Sheets API enforces the following rate limits:

- **100 requests per 100 seconds per user** (enforced by connector)
- Consider caching frequently accessed data
- Use batch operations when available

## Troubleshooting

### Authentication Errors

```
Google Sheets error (UNAUTHENTICATED): Request had invalid authentication credentials
```

**Solution**: Check that your OAuth2 token is valid or service account credentials are correct.

### Rate Limit Errors

```
Google Sheets error (RESOURCE_EXHAUSTED): Rate limit exceeded
```

**Solution**: The connector automatically retries, but you may need to reduce request frequency.

### Not Found Errors

```
Google Sheets error (NOT_FOUND): Spreadsheet not found
```

**Solution**: Verify the spreadsheet ID exists and is accessible with your credentials.

### Permission Errors

```
Google Sheets error (PERMISSION_DENIED): The caller does not have permission
```

**Solution**: Share your Google Sheet with the service account email or ensure OAuth2 token has proper scopes.

## Roadmap

Future enhancements:

- [ ] Full service account authentication support with google.golang.org/api
- [ ] OAuth2 flow integration
- [ ] Cell formatting support
- [ ] Conditional formatting
- [ ] Charts and graphs
- [ ] Pivot tables
- [ ] Filtering and sorting
- [ ] Real-time collaboration features

## License

Part of Fluxor framework - see main LICENSE file.
