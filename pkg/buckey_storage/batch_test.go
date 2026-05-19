package buckey_storage

import (
	"context"
	"testing"
)

func TestPutMany_GetMany_DeleteMany(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()

	items := map[string][]byte{"a": []byte("1"), "b": []byte("2"), "c": []byte("3")}
	if err := PutMany(ctx, s, items); err != nil {
		t.Fatalf("PutMany: %v", err)
	}

	got, err := GetMany(ctx, s, []string{"a", "b", "c", "missing"})
	if err != nil {
		t.Fatalf("GetMany: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("GetMany: got %d keys, want 3", len(got))
	}
	if string(got["a"]) != "1" {
		t.Errorf("GetMany a: got %q", got["a"])
	}
	if _, ok := got["missing"]; ok {
		t.Error("GetMany: should not have missing")
	}

	if err := DeleteMany(ctx, s, []string{"a", "b"}); err != nil {
		t.Fatalf("DeleteMany: %v", err)
	}
	_, err = s.Get(ctx, "a")
	if err != ErrNotFound {
		t.Errorf("after DeleteMany: got %v", err)
	}
	data, _ := s.Get(ctx, "c")
	if string(data) != "3" {
		t.Errorf("c should remain: got %q", data)
	}
}

func TestListPage(t *testing.T) {
	s := NewMemoryStorage()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		s.Put(ctx, string(rune('a'+i)), []byte("x"))
	}

	res, err := ListPage(ctx, s, "", 2, 0)
	if err != nil {
		t.Fatalf("ListPage: %v", err)
	}
	if res.Total != 5 || len(res.Keys) != 2 {
		t.Errorf("ListPage limit=2: total=%d keys=%d", res.Total, len(res.Keys))
	}

	res, _ = ListPage(ctx, s, "", 10, 3)
	if res.Total != 5 || len(res.Keys) != 2 {
		t.Errorf("ListPage offset=3: total=%d keys=%d", res.Total, len(res.Keys))
	}
}
