package buckey_storage

import (
	"context"
	"testing"
)

func TestStoreClipboard_GetClipboard(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	title := "không bac91 buộc"
	text := "clipboard content here"
	if err := StoreClipboard(ctx, s, title, text); err != nil {
		t.Fatalf("StoreClipboard: %v", err)
	}
	got, err := GetClipboard(ctx, s, title)
	if err != nil {
		t.Fatalf("GetClipboard: %v", err)
	}
	if got != text {
		t.Errorf("GetClipboard: got %q, want %q", got, text)
	}
}

func TestGetClipboard_NotFound(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	_, err := GetClipboard(ctx, s, "missing title")
	if err != ErrNotFound {
		t.Errorf("GetClipboard missing: got %v, want ErrNotFound", err)
	}
}

func TestDeleteClipboard(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	StoreClipboard(ctx, s, "title1", "text1")
	if err := DeleteClipboard(ctx, s, "title1"); err != nil {
		t.Fatalf("DeleteClipboard: %v", err)
	}
	_, err := GetClipboard(ctx, s, "title1")
	if err != ErrNotFound {
		t.Errorf("after DeleteClipboard: got %v", err)
	}
}

func TestListClipboard(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	StoreClipboard(ctx, s, "không bac91 buộc", "content 1")
	StoreClipboard(ctx, s, "title two", "content 2")

	items, err := ListClipboard(ctx, s)
	if err != nil {
		t.Fatalf("ListClipboard: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("ListClipboard: got %d items, want 2", len(items))
	}
	var found bool
	for _, it := range items {
		if it.Title == "không bac91 buộc" && it.Text == "content 1" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListClipboard: did not find 'không bac91 buộc' with content 1")
	}
}
