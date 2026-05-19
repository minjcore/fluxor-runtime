package buckey_storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileToStorage(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(fpath, []byte("content from file"), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewMemoryStorage()
	ctx := context.Background()
	if err := CopyFileToStorage(ctx, s, "mykey", fpath); err != nil {
		t.Fatalf("CopyFileToStorage: %v", err)
	}
	b, err := s.Get(ctx, "mykey")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(b) != "content from file" {
		t.Errorf("Get: got %q", b)
	}
}

func TestCopyFileToClipboard(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(fpath, []byte("clipboard text from file"), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewMemoryStorage()
	ctx := context.Background()
	title := "không bac91 buộc"
	if err := CopyFileToClipboard(ctx, s, title, fpath); err != nil {
		t.Fatalf("CopyFileToClipboard: %v", err)
	}
	text, err := GetClipboard(ctx, s, title)
	if err != nil {
		t.Fatalf("GetClipboard: %v", err)
	}
	if text != "clipboard text from file" {
		t.Errorf("GetClipboard: got %q", text)
	}
}

func TestCopyFileToStorageAsKey(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "auto.txt")
	if err := os.WriteFile(fpath, []byte("auto key content"), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewMemoryStorage()
	ctx := context.Background()
	if err := CopyFileToStorageAsKey(ctx, s, "", fpath); err != nil {
		t.Fatalf("CopyFileToStorageAsKey: %v", err)
	}
	b, err := s.Get(ctx, "auto.txt")
	if err != nil {
		t.Fatalf("Get auto.txt: %v", err)
	}
	if string(b) != "auto key content" {
		t.Errorf("Get: got %q", b)
	}
}
