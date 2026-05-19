// Package sheets provides comprehensive Google Sheets integration for Fluxor.
//
// This package follows Fluxor's Premium Pattern architecture with BaseComponent
// lifecycle management, BaseConfig inheritance, and EventBus integration.
//
// Features:
//   - Full Google Sheets API v4 support (read, write, update operations)
//   - Automatic rate limiting and retry mechanism
//   - Context support for cancellation and timeouts
//   - EventBus integration for reactive patterns
//   - Fail-fast validation throughout
//   - OAuth2 and Service Account authentication support
//
// Quick Start:
//
//	package main
//
//	import (
//	    "github.com/fluxorio/fluxor/pkg/connectors/google/sheets"
//	    "github.com/fluxorio/fluxor/pkg/core"
//	)
//
//	type MyVerticle struct {
//	    *core.BaseVerticle
//	    sheets *sheets.SheetComponent
//	}
//
//	func (v *MyVerticle) Start(ctx core.FluxorContext) error {
//	    // Create and configure Google Sheets component
//	    config := sheets.DefaultConfig()
//	    config.CredentialsPath = "/path/to/credentials.json"
//	    config.SpreadsheetID = "your-spreadsheet-id"
//	    v.sheets = sheets.NewSheetComponent(config)
//
//	    // Start the component
//	    if err := v.sheets.Start(ctx); err != nil {
//	        return err
//	    }
//
//	    // Use the Sheets client
//	    client, _ := v.sheets.Client()
//	    values, err := client.Read(ctx.Context(), "Sheet1!A1:B10")
//	    if err != nil {
//	        return err
//	    }
//
//	    log.Printf("Read %d rows", len(values))
//	    return nil
//	}
//
// For complete documentation and examples, see README.md.
//
// Path: pkg/connectors/google/sheets
package sheets
