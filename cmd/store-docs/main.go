// Package main stores docs/*.md into buckey_storage. Run from repo root:
//
//	go run ./cmd/store-docs [-docs=docs] [-config=buckey_storage.json]
//
// With no -config, uses in-memory storage (for testing). With -config, loads
// backend from JSON (e.g. memory, fs, s3). After storing, index with hyperspace-search:
//
//	hyperspace-search index-dir --dir docs
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fluxorio/fluxor/pkg/buckey_storage"
)

func main() {
	docsDir := flag.String("docs", "docs", "Directory containing .md files to store")
	configPath := flag.String("config", "", "Path to buckey_storage.json (optional; default: memory)")
	flag.Parse()

	ctx := context.Background()

	absDocs, err := filepath.Abs(*docsDir)
	if err != nil {
		log.Fatalf("docs path: %v", err)
	}
	if info, err := os.Stat(absDocs); err != nil || !info.IsDir() {
		log.Fatalf("docs directory %q missing or not a directory: %v", absDocs, err)
	}

	var s buckey_storage.BlobStorage
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			log.Fatalf("read config: %v", err)
		}
		var cfg buckey_storage.Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			log.Fatalf("parse config: %v", err)
		}
		if err := cfg.Validate(); err != nil {
			log.Fatalf("config: %v", err)
		}
		s, err = buckey_storage.NewFromConfig(&cfg)
		if err != nil {
			log.Fatalf("storage from config: %v", err)
		}
	} else {
		s = buckey_storage.NewMemoryStorage()
	}

	n, err := buckey_storage.StoreRepoToStorage(ctx, s, absDocs, buckey_storage.StoreRepoOptions{
		KeyPrefix:       "docs/",
		IncludeSuffixes: []string{".md"},
	})
	if err != nil {
		log.Fatalf("store docs: %v", err)
	}
	fmt.Printf("Stored %d .md files to buckey_storage under docs/\n", n)
	fmt.Println("Index with: hyperspace-search index-dir --dir docs")
}
