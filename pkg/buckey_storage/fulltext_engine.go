package buckey_storage

import (
	"context"
	"strings"
	"unicode/utf8"
)

// FullTextEngine provides a full-text search engine over BlobStorage for Cursor or external tools.
// It indexes blob content into an Index and exposes SearchWithSnippets so Cursor can refer to
// buckey_storage to search external content (e.g. docs, notes, code snippets stored in storage).
type FullTextEngine struct {
	storage BlobStorage
	index   *Index
}

// NewFullTextEngine creates a full-text engine backed by the given storage and index.
// Use IndexFromStorage to populate the index from existing blobs; then SearchWithSnippets for queries.
func NewFullTextEngine(storage BlobStorage, index *Index) *FullTextEngine {
	if index == nil {
		index = NewIndex()
	}
	return &FullTextEngine{storage: storage, index: index}
}

// IndexFromStorage indexes all blobs under prefix (or all keys if prefix is "") as UTF-8 text.
// Binary blobs are skipped (non-UTF-8 or control-heavy). Existing index entries for listed keys are replaced.
func (e *FullTextEngine) IndexFromStorage(ctx context.Context, prefix string) (indexed int, err error) {
	ValidateContext(ctx)
	keys, err := e.storage.List(ctx, prefix)
	if err != nil {
		return 0, err
	}
	for _, key := range keys {
		data, err := e.storage.Get(ctx, key)
		if err != nil {
			continue
		}
		if !isIndexableText(data) {
			continue
		}
		text := string(data)
		e.index.IndexFullText(ctx, key, text)
		indexed++
	}
	return indexed, nil
}

// isIndexableText returns true if data looks like UTF-8 text suitable for full-text indexing.
func isIndexableText(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if !utf8.Valid(data) {
		return false
	}
	// Reject if too many non-printable/control chars (likely binary)
	var control int
	for _, b := range data {
		if b < 0x20 && b != '\t' && b != '\n' && b != '\r' {
			control++
		}
	}
	return len(data) == 0 || control*10 < len(data)
}

// SearchResult is a single hit returned by SearchWithSnippets for Cursor/external use.
type SearchResult struct {
	Key     string // Storage key
	Snippet string // Short excerpt around the match (empty if no text stored)
}

// SearchWithSnippets runs a full-text query and returns keys plus snippets (excerpts) for Cursor.
// opts.Limit/Offset apply; SnippetMaxLen caps snippet length (0 = default ~160).
func (e *FullTextEngine) SearchWithSnippets(ctx context.Context, query string, opts QueryOptions) ([]SearchResult, int, error) {
	ValidateContext(ctx)
	keys, total, err := e.index.SearchPage(ctx, query, opts)
	if err != nil {
		return nil, 0, err
	}
	snippetLen := opts.SnippetMaxLen
	if snippetLen <= 0 {
		snippetLen = 160
	}
	results := make([]SearchResult, 0, len(keys))
	for _, key := range keys {
		text, _ := e.index.GetFullText(ctx, key)
		snippet := excerptAroundQuery(text, query, snippetLen)
		results = append(results, SearchResult{Key: key, Snippet: snippet})
	}
	return results, total, nil
}

// excerptAroundQuery returns a short excerpt of text around the first occurrence of a query term (case-insensitive).
func excerptAroundQuery(text, query string, maxLen int) string {
	text = strings.TrimSpace(text)
	if text == "" || maxLen <= 0 {
		return ""
	}
	lower := strings.ToLower(text)
	terms := tokenize(query)
	if len(terms) == 0 {
		return truncate(text, maxLen)
	}
	// Find first occurrence of any term
	firstIdx := -1
	for _, t := range terms {
		i := strings.Index(lower, t)
		if i >= 0 && (firstIdx < 0 || i < firstIdx) {
			firstIdx = i
		}
	}
	if firstIdx < 0 {
		return truncate(text, maxLen)
	}
	// Excerpt: center on firstIdx, half before half after
	start := firstIdx - maxLen/2
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(text) {
		end = len(text)
		start = end - maxLen
		if start < 0 {
			start = 0
		}
	}
	excerpt := text[start:end]
	if start > 0 {
		excerpt = "…" + excerpt
	}
	if end < len(text) {
		excerpt = excerpt + "…"
	}
	return excerpt
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// IndexBlob indexes a single blob under key (e.g. after Put). Call when syncing external content.
func (e *FullTextEngine) IndexBlob(ctx context.Context, key string, data []byte) error {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return err
	}
	if !isIndexableText(data) {
		return nil
	}
	e.index.IndexFullText(ctx, key, string(data))
	return nil
}

// RemoveFromIndex removes key from the index (e.g. after Delete). Call to keep index in sync.
func (e *FullTextEngine) RemoveFromIndex(ctx context.Context, key string) {
	ValidateContext(ctx)
	e.index.RemoveKey(ctx, key)
}

// Search runs the same full-text query as Index.Search (keys only, no snippets).
func (e *FullTextEngine) Search(ctx context.Context, query string) ([]string, error) {
	return e.index.Search(ctx, query)
}

// SearchPhrase runs phrase (substring) search; returns keys only.
func (e *FullTextEngine) SearchPhrase(ctx context.Context, phrase string) ([]string, error) {
	return e.index.SearchPhrase(ctx, phrase)
}
