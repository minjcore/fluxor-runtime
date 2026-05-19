package buckey_storage

import (
	"context"
)

// PutMany writes multiple key-value pairs. Returns first error if any Put fails.
func PutMany(ctx context.Context, s BlobStorage, items map[string][]byte) error {
	ValidateContext(ctx)
	for key, data := range items {
		if err := s.Put(ctx, key, data); err != nil {
			return err
		}
	}
	return nil
}

// GetMany retrieves multiple keys. Result map contains only successfully fetched keys;
// missing keys are omitted (use Get for ErrNotFound per key).
func GetMany(ctx context.Context, s BlobStorage, keys []string) (map[string][]byte, error) {
	ValidateContext(ctx)
	out := make(map[string][]byte, len(keys))
	for _, key := range keys {
		if err := ValidateKey(key); err != nil {
			return nil, err
		}
		b, err := s.Get(ctx, key)
		if err != nil {
			continue
		}
		out[key] = b
	}
	return out, nil
}

// DeleteMany deletes multiple keys. Ignores per-key errors (Delete is idempotent).
func DeleteMany(ctx context.Context, s BlobStorage, keys []string) error {
	ValidateContext(ctx)
	for _, key := range keys {
		_ = s.Delete(ctx, key)
	}
	return nil
}

// ListPageResult holds a page of keys and total count.
type ListPageResult struct {
	Keys  []string
	Total int
}

// ListPage returns keys with prefix, limited and offset for pagination.
func ListPage(ctx context.Context, s BlobStorage, prefix string, limit, offset int) (ListPageResult, error) {
	ValidateContext(ctx)
	keys, err := s.List(ctx, prefix)
	if err != nil {
		return ListPageResult{}, err
	}
	total := len(keys)
	if offset > 0 {
		if offset >= len(keys) {
			keys = nil
		} else {
			keys = keys[offset:]
		}
	}
	if limit > 0 && len(keys) > limit {
		keys = keys[:limit]
	}
	return ListPageResult{Keys: keys, Total: total}, nil
}
