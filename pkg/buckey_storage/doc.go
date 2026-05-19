// Package buckey_storage provides a blob storage abstraction with JSON-loadable
// configuration for the Fluxor framework.
//
// BlobStorage supports Put, Get, Delete, and List by key. Config can be
// loaded from a JSON file (e.g. buckey_storage.json) to select backend and options.
//
// Example: load config from JSON and use in-memory storage
//
//	import (
//	    "context"
//	    "encoding/json"
//	    "os"
//	    "github.com/fluxorio/fluxor/pkg/buckey_storage"
//	)
//
//	data, _ := os.ReadFile("buckey_storage.json")
//	var cfg buckey_storage.Config
//	_ = json.Unmarshal(data, &cfg)
//	s, _ := buckey_storage.NewFromConfig(&cfg)
//
//	ctx := context.Background()
//	_ = s.Put(ctx, "mykey", []byte("hello"))
//	blob, _ := s.Get(ctx, "mykey")
package buckey_storage
