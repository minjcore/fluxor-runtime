// Package airtable provides comprehensive Airtable integration for Fluxor.
//
// This package follows Fluxor's Premium Pattern architecture with BaseComponent
// lifecycle management, BaseConfig inheritance, and EventBus integration.
//
// Features:
//   - Full Airtable API support (Tables and Records operations)
//   - Automatic rate limiting (5 requests/second per Airtable limits)
//   - Retry mechanism with exponential backoff
//   - Context support for cancellation and timeouts
//   - EventBus integration for reactive patterns
//   - Fail-fast validation throughout
//
// Quick Start:
//
//	package main
//
//	import (
//	    "github.com/fluxorio/fluxor/pkg/connectors/airtable"
//	    "github.com/fluxorio/fluxor/pkg/core"
//	)
//
//	type MyVerticle struct {
//	    *core.BaseVerticle
//	    airtable *airtable.AirtableComponent
//	}
//
//	func (v *MyVerticle) Start(ctx core.FluxorContext) error {
//	    // Create and configure Airtable component
//	    config := airtable.DefaultConfig()
//	    config.APIKey = "keyXXXXXXXXXXXXXX"
//	    config.BaseID = "appXXXXXXXXXXXXXX"
//	    v.airtable = airtable.NewAirtableComponent(config)
//
//	    // Start the component
//	    if err := v.airtable.Start(ctx); err != nil {
//	        return err
//	    }
//
//	    // Use the Records client
//	    records, _ := v.airtable.Records()
//	    newRecord := &airtable.Record{
//	        Fields: map[string]interface{}{
//	            "Name":   "My Task",
//	            "Status": "In Progress",
//	        },
//	    }
//	    created, err := records.Create(ctx.Context(), "Tasks", newRecord)
//	    if err != nil {
//	        return err
//	    }
//
//	    log.Printf("Created record: %s", created.ID)
//	    return nil
//	}
//
// For complete documentation and examples, see README.md.
//
// Path: pkg/connectors/airtable
package airtable
