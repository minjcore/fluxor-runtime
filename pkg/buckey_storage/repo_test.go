package buckey_storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRepoToStorage(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	// Create a minimal repo-like tree
	_ = os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	_ = os.MkdirAll(filepath.Join(dir, "pkg", "foo"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello repo"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "pkg", "foo", "bar.go"), []byte("package foo\nfunc Bar() {}"), 0644)
	_ = os.WriteFile(filepath.Join(dir, ".git", "config"), []byte("[core]"), 0644)

	s := NewMemoryStorage()
	n, err := StoreRepoToStorage(ctx, s, dir, StoreRepoOptions{
		KeyPrefix: "repos/myrepo",
		TextOnly:  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	// .git is skipped, so 2 files: README.md, pkg/foo/bar.go
	if n != 2 {
		t.Errorf("stored %d, want 2", n)
	}

	keys, _ := s.List(ctx, "repos/myrepo/")
	if len(keys) != 2 {
		t.Errorf("list len %d, want 2", len(keys))
	}
	data, err := s.Get(ctx, "repos/myrepo/pkg/foo/bar.go")
	if err != nil || string(data) != "package foo\nfunc Bar() {}" {
		t.Errorf("get bar.go: err=%v data=%q", err, string(data))
	}
}

func TestStoreRepoToStorage_IncludeSuffixes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Doc"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("notes"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "sub", "guide.md"), []byte("# Guide"), 0644)

	s := NewMemoryStorage()
	n, err := StoreRepoToStorage(ctx, s, dir, StoreRepoOptions{
		KeyPrefix:       "docs/",
		IncludeSuffixes: []string{".md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("stored %d, want 2 (.md only)", n)
	}
	_, err = s.Get(ctx, "docs/README.md")
	if err != nil {
		t.Errorf("docs/README.md: %v", err)
	}
	_, err = s.Get(ctx, "docs/notes.txt")
	if err == nil {
		t.Error("docs/notes.txt should not be stored")
	}
}
