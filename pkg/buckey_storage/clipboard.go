package buckey_storage

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

const clipboardPrefix = "clipboard/"

// ClipboardItem is a stored clipboard entry with title and text.
type ClipboardItem struct {
	Title string
	Text  string
}

// StoreClipboard saves text under the given title (e.g. "không bac91 buộc").
// Title can contain any character; it is encoded in the storage key.
func StoreClipboard(ctx context.Context, s BlobStorage, title, text string) error {
	ValidateContext(ctx)
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("buckey_storage: clipboard title cannot be empty")
	}
	key := clipboardPrefix + base64.URLEncoding.EncodeToString([]byte(title))
	return s.Put(ctx, key, []byte(text))
}

// GetClipboard returns the text stored under the given title.
// Returns ErrNotFound if no clipboard entry exists for that title.
func GetClipboard(ctx context.Context, s BlobStorage, title string) (string, error) {
	ValidateContext(ctx)
	key := clipboardPrefix + base64.URLEncoding.EncodeToString([]byte(title))
	b, err := s.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DeleteClipboard removes the clipboard entry for the given title.
func DeleteClipboard(ctx context.Context, s BlobStorage, title string) error {
	ValidateContext(ctx)
	key := clipboardPrefix + base64.URLEncoding.EncodeToString([]byte(title))
	return s.Delete(ctx, key)
}

// ListClipboard returns all stored clipboard items (title + text).
func ListClipboard(ctx context.Context, s BlobStorage) ([]ClipboardItem, error) {
	ValidateContext(ctx)
	keys, err := s.List(ctx, clipboardPrefix)
	if err != nil {
		return nil, err
	}
	var out []ClipboardItem
	for _, k := range keys {
		if !strings.HasPrefix(k, clipboardPrefix) {
			continue
		}
		encoded := k[len(clipboardPrefix):]
		titleBytes, err := base64.URLEncoding.DecodeString(encoded)
		if err != nil {
			continue
		}
		title := string(titleBytes)
		b, err := s.Get(ctx, k)
		if err != nil {
			continue
		}
		out = append(out, ClipboardItem{Title: title, Text: string(b)})
	}
	return out, nil
}
