package appendlog

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFSStore_AppendRead_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := NewFSStore(FSStoreConfig{
		Dir:              dir,
		MaxSegmentBytes:  1 << 20,
		MaxBufferedBytes: 1 << 20,
		Durability:       DurabilityFsync,
	})
	if err != nil {
		t.Fatalf("NewFSStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	o1, err := s.Append([]byte("a"))
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	o2, err := s.Append([]byte("b"))
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if o2 <= o1 {
		t.Fatalf("expected monotonic offsets, got %d then %d", o1, o2)
	}

	recs, err := s.Read(o1, 10)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(recs) < 2 {
		t.Fatalf("expected at least 2 records, got %d", len(recs))
	}
	if !bytes.Equal(recs[0].Data, []byte("a")) {
		t.Fatalf("unexpected rec0: %q", recs[0].Data)
	}
	if !bytes.Equal(recs[1].Data, []byte("b")) {
		t.Fatalf("unexpected rec1: %q", recs[1].Data)
	}
}

func TestFSStore_RotateBySize_CreatesMultipleSegments(t *testing.T) {
	dir := t.TempDir()
	s, err := NewFSStore(FSStoreConfig{
		Dir:              dir,
		MaxSegmentBytes:  64, // tiny to force rotation
		MaxBufferedBytes: 1 << 20,
		Durability:       DurabilityFsync,
	})
	if err != nil {
		t.Fatalf("NewFSStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	for i := 0; i < 50; i++ {
		if _, err := s.Append(bytes.Repeat([]byte("x"), 8)); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	// Ensure everything is flushed.
	if err := s.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	var logs int
	for _, e := range ents {
		if filepath.Ext(e.Name()) == ".log" {
			logs++
		}
	}
	if logs < 2 {
		t.Fatalf("expected >=2 segments, got %d", logs)
	}
}

func TestFSStore_Recovery_ReopensAndReads(t *testing.T) {
	dir := t.TempDir()

	cfg := FSStoreConfig{
		Dir:              dir,
		MaxSegmentBytes:  64,
		MaxBufferedBytes: 1 << 20,
		Durability:       DurabilityFsync,
	}

	s1, err := NewFSStore(cfg)
	if err != nil {
		t.Fatalf("NewFSStore: %v", err)
	}
	o1, _ := s1.Append([]byte("one"))
	_, _ = s1.Append([]byte("two"))
	_ = s1.Sync()
	_ = s1.Close()

	s2, err := NewFSStore(cfg)
	if err != nil {
		t.Fatalf("NewFSStore reopen: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })

	recs, err := s2.Read(o1, 10)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(recs) < 2 {
		t.Fatalf("expected >=2 records after recovery, got %d", len(recs))
	}
	if string(recs[0].Data) != "one" {
		t.Fatalf("unexpected: %q", recs[0].Data)
	}
	if string(recs[1].Data) != "two" {
		t.Fatalf("unexpected: %q", recs[1].Data)
	}
}

func TestFSStore_FailFast_Backpressure(t *testing.T) {
	dir := t.TempDir()
	s, err := NewFSStore(FSStoreConfig{
		Dir:              dir,
		MaxSegmentBytes:  1 << 20,
		MaxBufferedBytes: 64, // tiny to force reject
		Durability:       DurabilityMemory,
	})
	if err != nil {
		t.Fatalf("NewFSStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	payload := bytes.Repeat([]byte("a"), 128)
	_, err = s.Append(payload)
	if err == nil {
		// buffered bytes accounting is async-decremented; give it a moment and retry.
		time.Sleep(50 * time.Millisecond)
		_, err = s.Append(payload)
	}
	if err != ErrBackpressure {
		t.Fatalf("expected ErrBackpressure, got %v", err)
	}
}
