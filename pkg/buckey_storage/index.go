package buckey_storage

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

// Index provides full-text (search-engine-like) and field/pattern (awk-like) search
// over keys. Compatible with in-memory and replicated storage: index by key and
// content or by key and record fields.
type Index struct {
	mu sync.RWMutex

	// Full-text: term -> set of keys (inverted index)
	inv map[string]map[string]struct{}
	// Raw text per key for phrase search
	fullTextRaw map[string]string

	// Record (awk-like): key -> field name -> value
	records map[string]map[string]string
}

// NewIndex creates an empty index for full-text and field search.
func NewIndex() *Index {
	return &Index{
		inv:         make(map[string]map[string]struct{}),
		fullTextRaw: make(map[string]string),
		records:     make(map[string]map[string]string),
	}
}

// tokenize splits text into lowercase tokens (words) for full-text index.
func tokenize(text string) []string {
	f := func(c rune) bool { return !unicode.IsLetter(c) && !unicode.IsNumber(c) }
	parts := strings.FieldsFunc(strings.ToLower(text), f)
	var out []string
	seen := make(map[string]bool)
	for _, p := range parts {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// IndexFullText adds a key with the given text to the inverted index and raw text for phrase search.
// Existing terms for this key are replaced (re-index).
func (x *Index) IndexFullText(ctx context.Context, key string, text string) {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return
	}
	tokens := tokenize(text)
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.fullTextRaw == nil {
		x.fullTextRaw = make(map[string]string)
	}
	x.fullTextRaw[key] = text
	// Remove key from old terms
	for term, keys := range x.inv {
		delete(keys, key)
		if len(keys) == 0 {
			delete(x.inv, term)
		}
	}
	for _, t := range tokens {
		if x.inv[t] == nil {
			x.inv[t] = make(map[string]struct{})
		}
		x.inv[t][key] = struct{}{}
	}
}

// IndexRecord stores a key with named fields (awk-like: key = row, fields = columns).
// Use FilterField to filter by pattern on a field.
func (x *Index) IndexRecord(ctx context.Context, key string, fields map[string]string) {
	ValidateContext(ctx)
	if err := ValidateKey(key); err != nil {
		return
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.records == nil {
		x.records = make(map[string]map[string]string)
	}
	x.records[key] = copyMap(fields)
}

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Search runs a simple query over the full-text index.
// Query syntax: space-separated = AND (all terms must appear). Use " OR " for OR.
// Example: "hello world" -> keys containing both "hello" and "world"
// Example: "hello OR world" -> keys containing "hello" or "world"
func (x *Index) Search(ctx context.Context, query string) ([]string, error) {
	ValidateContext(ctx)
	x.mu.RLock()
	defer x.mu.RUnlock()

	orParts := strings.Split(query, " OR ")
	var result map[string]struct{}
	for i, part := range orParts {
		terms := tokenize(strings.TrimSpace(part))
		if len(terms) == 0 {
			continue
		}
		var set map[string]struct{}
		for _, t := range terms {
			keys, ok := x.inv[t]
			if !ok {
				set = nil
				break
			}
			if set == nil {
				set = copyKeySet(keys)
				continue
			}
			// AND within part: intersect
			for k := range set {
				if _, has := keys[k]; !has {
					delete(set, k)
				}
			}
		}
		if set == nil {
			continue
		}
		if result == nil {
			result = set
		} else {
			// OR across parts: union
			for k := range set {
				result[k] = struct{}{}
			}
		}
		if i == 0 && len(orParts) > 1 {
			// first part done, result is our set; next parts union in
		}
	}
	return keysFromSet(result), nil
}

// SearchPhrase returns keys whose indexed text contains the exact phrase (substring, case-insensitive).
func (x *Index) SearchPhrase(ctx context.Context, phrase string) ([]string, error) {
	ValidateContext(ctx)
	phrase = strings.ToLower(strings.TrimSpace(phrase))
	if phrase == "" {
		return nil, nil
	}
	x.mu.RLock()
	defer x.mu.RUnlock()
	var out []string
	for key, text := range x.fullTextRaw {
		if strings.Contains(strings.ToLower(text), phrase) {
			out = append(out, key)
		}
	}
	return out, nil
}

// SearchPrefix returns keys that contain any term starting with the given prefix (case-insensitive).
func (x *Index) SearchPrefix(ctx context.Context, termPrefix string) ([]string, error) {
	ValidateContext(ctx)
	termPrefix = strings.ToLower(strings.TrimSpace(termPrefix))
	if termPrefix == "" {
		return nil, nil
	}
	x.mu.RLock()
	defer x.mu.RUnlock()
	var result map[string]struct{}
	for term, keys := range x.inv {
		if strings.HasPrefix(term, termPrefix) {
			if result == nil {
				result = copyKeySet(keys)
			} else {
				for k := range keys {
					result[k] = struct{}{}
				}
			}
		}
	}
	return keysFromSet(result), nil
}

// QueryOptions configures pagination and optional field filter for search.
type QueryOptions struct {
	Limit         int    // max keys to return (0 = no limit)
	Offset        int    // skip this many keys
	Field         string // if set, filter by this field
	Pattern       string // regex pattern for Field (required if Field set)
	SnippetMaxLen int    // max snippet length for SearchWithSnippets (0 = default 160)
}

// SearchPage runs Search with optional pagination (limit/offset) and optional field filter.
// If opts.Field and opts.Pattern are set, results are restricted to keys that also match the field regex.
func (x *Index) SearchPage(ctx context.Context, query string, opts QueryOptions) (keys []string, total int, err error) {
	ValidateContext(ctx)
	keys, err = x.Search(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	if opts.Field != "" && opts.Pattern != "" {
		re, reErr := regexp.Compile(opts.Pattern)
		if reErr != nil {
			return nil, 0, reErr
		}
		x.mu.RLock()
		var filtered []string
		for _, k := range keys {
			if fields, ok := x.records[k]; ok && re.MatchString(fields[opts.Field]) {
				filtered = append(filtered, k)
			}
		}
		x.mu.RUnlock()
		keys = filtered
	}
	total = len(keys)
	if opts.Offset > 0 {
		if opts.Offset >= len(keys) {
			keys = nil
		} else {
			keys = keys[opts.Offset:]
		}
	}
	if opts.Limit > 0 && len(keys) > opts.Limit {
		keys = keys[:opts.Limit]
	}
	return keys, total, nil
}

// IndexStats returns counts for terms, full-text docs, and records.
type IndexStats struct {
	TermCount   int
	DocCount    int
	RecordCount int
}

// Stats returns index statistics.
func (x *Index) Stats(ctx context.Context) IndexStats {
	ValidateContext(ctx)
	x.mu.RLock()
	defer x.mu.RUnlock()
	docSet := make(map[string]struct{})
	for _, keys := range x.inv {
		for k := range keys {
			docSet[k] = struct{}{}
		}
	}
	return IndexStats{
		TermCount:   len(x.inv),
		DocCount:    len(docSet),
		RecordCount: len(x.records),
	}
}

func copyKeySet(m map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(m))
	for k := range m {
		out[k] = struct{}{}
	}
	return out
}

func keysFromSet(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// FilterField returns keys whose given field matches the pattern (regex).
// Awk-like: filter records by pattern on one column.
func (x *Index) FilterField(ctx context.Context, field string, pattern string) ([]string, error) {
	ValidateContext(ctx)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	x.mu.RLock()
	defer x.mu.RUnlock()
	var out []string
	for key, fields := range x.records {
		if re.MatchString(fields[field]) {
			out = append(out, key)
		}
	}
	return out, nil
}

// GetFields returns the stored fields for a key (awk-like: get row/record).
func (x *Index) GetFields(ctx context.Context, key string) (map[string]string, bool) {
	ValidateContext(ctx)
	x.mu.RLock()
	defer x.mu.RUnlock()
	f, ok := x.records[key]
	if !ok {
		return nil, false
	}
	return copyMap(f), true
}

// GetFullText returns the raw text stored for a key (for snippet extraction). Used by FullTextEngine.
func (x *Index) GetFullText(ctx context.Context, key string) (string, bool) {
	ValidateContext(ctx)
	x.mu.RLock()
	defer x.mu.RUnlock()
	t, ok := x.fullTextRaw[key]
	return t, ok
}

// RemoveKey removes key from both full-text and record index.
func (x *Index) RemoveKey(ctx context.Context, key string) {
	ValidateContext(ctx)
	x.mu.Lock()
	defer x.mu.Unlock()
	for term, keys := range x.inv {
		delete(keys, key)
		if len(keys) == 0 {
			delete(x.inv, term)
		}
	}
	delete(x.fullTextRaw, key)
	delete(x.records, key)
}
