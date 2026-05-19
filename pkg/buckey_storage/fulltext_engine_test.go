package buckey_storage

import (
	"context"
	"strings"
	"testing"
)

func TestFullTextEngine_IndexFromStorage_SearchWithSnippets(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()
	idx := NewIndex()
	engine := NewFullTextEngine(s, idx)

	_ = s.Put(ctx, "doc/a.txt", []byte("Hello world and Cursor integration"))
	_ = s.Put(ctx, "doc/b.txt", []byte("Full-text search engine for docs"))
	_ = s.Put(ctx, "doc/d.txt", []byte("External content for Cursor"))
	_ = s.Put(ctx, "doc/c.bin", []byte{0x00, 0x01, 0x02}) // binary, skipped

	n, err := engine.IndexFromStorage(ctx, "doc/")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("indexed %d, want 3", n)
	}

	// "integration" appears only in doc/a.txt
	results, total, err := engine.SearchWithSnippets(ctx, "integration", QueryOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Errorf("total %d, want 1", total)
	}
	if len(results) != 1 {
		t.Fatalf("results %d, want 1", len(results))
	}
	if results[0].Key != "doc/a.txt" {
		t.Errorf("key %q, want doc/a.txt", results[0].Key)
	}
	if !strings.Contains(strings.ToLower(results[0].Snippet), "integration") {
		t.Errorf("snippet %q should contain integration", results[0].Snippet)
	}

	// OR: one doc has "search", another has "external"
	results2, total2, _ := engine.SearchWithSnippets(ctx, "search OR external", QueryOptions{Limit: 5})
	if total2 != 2 || len(results2) != 2 {
		t.Errorf("search OR: got %d results (total %d), want 2", len(results2), total2)
	}
}
