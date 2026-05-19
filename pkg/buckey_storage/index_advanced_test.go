package buckey_storage

import (
	"context"
	"reflect"
	"sort"
	"testing"
)

func TestIndex_SearchPhrase(t *testing.T) {
	idx := NewIndex()
	ctx := context.Background()
	idx.IndexFullText(ctx, "d1", "hello world from fluxor")
	idx.IndexFullText(ctx, "d2", "world only")
	idx.IndexFullText(ctx, "d3", "hello fluxor")

	keys, err := idx.SearchPhrase(ctx, "world from")
	if err != nil {
		t.Fatalf("SearchPhrase: %v", err)
	}
	if len(keys) != 1 || keys[0] != "d1" {
		t.Errorf("SearchPhrase world from: got %v", keys)
	}

	keys, _ = idx.SearchPhrase(ctx, "HELLO")
	if len(keys) != 2 {
		t.Errorf("SearchPhrase HELLO: got %v", keys)
	}
}

func TestIndex_SearchPrefix(t *testing.T) {
	idx := NewIndex()
	ctx := context.Background()
	idx.IndexFullText(ctx, "a", "hello world")
	idx.IndexFullText(ctx, "b", "world foo")
	idx.IndexFullText(ctx, "c", "hell")

	keys, err := idx.SearchPrefix(ctx, "hell")
	if err != nil {
		t.Fatalf("SearchPrefix: %v", err)
	}
	sort.Strings(keys)
	want := []string{"a", "c"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("SearchPrefix hell: got %v, want %v", keys, want)
	}
}

func TestIndex_SearchPage(t *testing.T) {
	idx := NewIndex()
	ctx := context.Background()
	idx.IndexFullText(ctx, "d1", "a b")
	idx.IndexFullText(ctx, "d2", "a b")
	idx.IndexFullText(ctx, "d3", "a b")
	idx.IndexRecord(ctx, "d1", map[string]string{"tag": "x"})
	idx.IndexRecord(ctx, "d2", map[string]string{"tag": "y"})
	idx.IndexRecord(ctx, "d3", map[string]string{"tag": "x"})

	keys, total, err := idx.SearchPage(ctx, "a", QueryOptions{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("SearchPage: %v", err)
	}
	if total != 3 || len(keys) != 2 {
		t.Errorf("SearchPage limit=2: got %d keys, total %d", len(keys), total)
	}

	keys, total, err = idx.SearchPage(ctx, "a", QueryOptions{Field: "tag", Pattern: "x"})
	if err != nil {
		t.Fatalf("SearchPage with filter: %v", err)
	}
	sort.Strings(keys)
	if total != 2 || !reflect.DeepEqual(keys, []string{"d1", "d3"}) {
		t.Errorf("SearchPage tag=x: got %v total %d", keys, total)
	}
}

func TestIndex_Stats(t *testing.T) {
	idx := NewIndex()
	ctx := context.Background()
	idx.IndexFullText(ctx, "k1", "hello world")
	idx.IndexRecord(ctx, "k1", map[string]string{"a": "1"})

	st := idx.Stats(ctx)
	if st.TermCount < 2 || st.DocCount != 1 || st.RecordCount != 1 {
		t.Errorf("Stats: got %+v", st)
	}
}
