package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiskCache_New(t *testing.T) {
	tmpDir := t.TempDir()
	maxSize := int64(1024 * 1024 * 1024) // 1GB

	dc := NewDiskCache(tmpDir, maxSize)

	if dc.root != tmpDir {
		t.Errorf("Expected root %s, got %s", tmpDir, dc.root)
	}

	if dc.maxSize != maxSize {
		t.Errorf("Expected maxSize %d, got %d", maxSize, dc.maxSize)
	}

	if dc.currentSize != 0 {
		t.Errorf("Expected currentSize 0, got %d", dc.currentSize)
	}
}

func TestDiskCache_Add(t *testing.T) {
	tmpDir := t.TempDir()
	dc := NewDiskCache(tmpDir, 1024*1024*1024)

	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("test data")
	os.WriteFile(testFile, testData, 0644)

	dc.Add(testFile, int64(len(testData)))

	if dc.currentSize != int64(len(testData)) {
		t.Errorf("Expected currentSize %d, got %d", len(testData), dc.currentSize)
	}

	if dc.FileCount() != 1 {
		t.Errorf("Expected file count 1, got %d", dc.FileCount())
	}
}

func TestDiskCache_Add_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	dc := NewDiskCache(tmpDir, 1024*1024*1024)

	testFile := filepath.Join(tmpDir, "test.txt")

	// Add first time
	dc.Add(testFile, 100)
	if dc.currentSize != 100 {
		t.Errorf("Expected currentSize 100, got %d", dc.currentSize)
	}

	// Update with larger size
	dc.Add(testFile, 200)
	if dc.currentSize != 200 {
		t.Errorf("Expected currentSize 200, got %d", dc.currentSize)
	}

	if dc.FileCount() != 1 {
		t.Errorf("Expected file count 1, got %d", dc.FileCount())
	}
}

func TestDiskCache_Eviction(t *testing.T) {
	tmpDir := t.TempDir()
	maxSize := int64(100) // Small size to trigger eviction
	dc := NewDiskCache(tmpDir, maxSize)

	// Add files that exceed max size
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for i, name := range files {
		path := filepath.Join(tmpDir, name)
		data := []byte("test data longer than 100 bytes total")
		os.WriteFile(path, data, 0644)

		dc.Add(path, int64(len(data)))

		// After adding enough data, eviction should occur
		if i == 2 && dc.currentSize > maxSize {
			t.Errorf("Expected size <= %d after eviction, got %d", maxSize, dc.currentSize)
		}
	}
}

func TestDiskCache_Touch(t *testing.T) {
	tmpDir := t.TempDir()
	dc := NewDiskCache(tmpDir, 1024*1024*1024)

	testFile := filepath.Join(tmpDir, "test.txt")
	dc.Add(testFile, 100)

	time.Sleep(10 * time.Millisecond)

	dc.Touch(testFile)

	// File should be moved to back of list (most recently used)
	if dc.fileList.Back() == nil {
		t.Error("Expected file to be in list")
	}

	entry := dc.fileList.Back().Value.(*CacheEntry)
	if entry.Path != testFile {
		t.Errorf("Expected last file to be %s, got %s", testFile, entry.Path)
	}
}

func TestDiskCache_Size(t *testing.T) {
	tmpDir := t.TempDir()
	dc := NewDiskCache(tmpDir, 1024*1024*1024)

	if dc.Size() != 0 {
		t.Errorf("Expected size 0, got %d", dc.Size())
	}

	dc.Add(filepath.Join(tmpDir, "test.txt"), 100)

	if dc.Size() != 100 {
		t.Errorf("Expected size 100, got %d", dc.Size())
	}
}

func TestDiskCache_FileCount(t *testing.T) {
	tmpDir := t.TempDir()
	dc := NewDiskCache(tmpDir, 1024*1024*1024)

	if dc.FileCount() != 0 {
		t.Errorf("Expected file count 0, got %d", dc.FileCount())
	}

	dc.Add(filepath.Join(tmpDir, "test1.txt"), 100)
	dc.Add(filepath.Join(tmpDir, "test2.txt"), 100)

	if dc.FileCount() != 2 {
		t.Errorf("Expected file count 2, got %d", dc.FileCount())
	}
}

func TestDiskCache_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	dc := NewDiskCache(tmpDir, 1024*1024*1024)

	// Add some files
	for i := 0; i < 5; i++ {
		path := filepath.Join(tmpDir, "test"+string(rune(i))+".txt")
		os.WriteFile(path, []byte("data"), 0644)
		dc.Add(path, 4)
	}

	if dc.FileCount() == 0 {
		t.Error("Expected files to be added")
	}

	err := dc.Clear()
	if err != nil {
		t.Errorf("Clear failed: %v", err)
	}

	if dc.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", dc.Size())
	}

	if dc.FileCount() != 0 {
		t.Errorf("Expected file count 0 after clear, got %d", dc.FileCount())
	}
}

func TestDiskCache_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	maxSize := int64(1000)
	dc := NewDiskCache(tmpDir, maxSize)

	dc.Add(filepath.Join(tmpDir, "test.txt"), 500)

	stats := dc.Stats()

	if stats["total_size"].(int64) != 500 {
		t.Errorf("Expected total_size 500, got %v", stats["total_size"])
	}

	if stats["max_size"].(int64) != maxSize {
		t.Errorf("Expected max_size %d, got %v", maxSize, stats["max_size"])
	}

	if stats["file_count"].(int) != 1 {
		t.Errorf("Expected file_count 1, got %v", stats["file_count"])
	}

	usagePercent := stats["usage_percent"].(float64)
	if usagePercent != 50.0 {
		t.Errorf("Expected usage_percent 50.0, got %f", usagePercent)
	}
}

func BenchmarkDiskCache_Add(b *testing.B) {
	tmpDir := b.TempDir()
	dc := NewDiskCache(tmpDir, 1024*1024*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := filepath.Join(tmpDir, "bench.txt")
		dc.Add(path, 100)
	}
}

func BenchmarkDiskCache_Touch(b *testing.B) {
	tmpDir := b.TempDir()
	dc := NewDiskCache(tmpDir, 1024*1024*1024)

	testFile := filepath.Join(tmpDir, "test.txt")
	dc.Add(testFile, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dc.Touch(testFile)
	}
}
