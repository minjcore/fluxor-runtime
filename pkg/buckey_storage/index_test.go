package buckey_storage

import (
	"context"
	"reflect"
	"sort"
	"testing"
)

func TestIndex_FullTextSearch(t *testing.T) {
	idx := NewIndex()
	ctx := context.Background()

	idx.IndexFullText(ctx, "doc1", "hello world")
	idx.IndexFullText(ctx, "doc2", "world foo")
	idx.IndexFullText(ctx, "doc3", "hello foo")

	// AND: both terms
	keys, err := idx.Search(ctx, "hello world")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(keys) != 1 || keys[0] != "doc1" {
		t.Errorf("Search hello world: got %v", keys)
	}

	// OR
	keys, err = idx.Search(ctx, "hello OR world")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	sort.Strings(keys)
	want := []string{"doc1", "doc2", "doc3"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("Search hello OR world: got %v, want %v", keys, want)
	}
}

func TestIndex_FilterField(t *testing.T) {
	idx := NewIndex()
	ctx := context.Background()

	idx.IndexRecord(ctx, "r1", map[string]string{"name": "alice", "city": "NYC"})
	idx.IndexRecord(ctx, "r2", map[string]string{"name": "bob", "city": "LA"})
	idx.IndexRecord(ctx, "r3", map[string]string{"name": "alice", "city": "LA"})

	keys, err := idx.FilterField(ctx, "name", "alice")
	if err != nil {
		t.Fatalf("FilterField: %v", err)
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"r1", "r3"}) {
		t.Errorf("FilterField name=alice: got %v", keys)
	}

	keys, err = idx.FilterField(ctx, "city", "LA")
	if err != nil {
		t.Fatalf("FilterField: %v", err)
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"r2", "r3"}) {
		t.Errorf("FilterField city=LA: got %v", keys)
	}
}

func TestIndex_GetFields(t *testing.T) {
	idx := NewIndex()
	ctx := context.Background()
	idx.IndexRecord(ctx, "k1", map[string]string{"a": "1", "b": "2"})

	f, ok := idx.GetFields(ctx, "k1")
	if !ok {
		t.Fatal("GetFields: not found")
	}
	if f["a"] != "1" || f["b"] != "2" {
		t.Errorf("GetFields: got %v", f)
	}

	_, ok = idx.GetFields(ctx, "missing")
	if ok {
		t.Error("GetFields: expected false for missing key")
	}
}

func TestIndex_RemoveKey(t *testing.T) {
	idx := NewIndex()
	ctx := context.Background()
	idx.IndexFullText(ctx, "d1", "hello")
	idx.IndexRecord(ctx, "d1", map[string]string{"x": "y"})

	idx.RemoveKey(ctx, "d1")

	keys, _ := idx.Search(ctx, "hello")
	if len(keys) != 0 {
		t.Errorf("Search after RemoveKey: got %v", keys)
	}
	_, ok := idx.GetFields(ctx, "d1")
	if ok {
		t.Error("GetFields after RemoveKey: expected false")
	}
}
