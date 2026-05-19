package buckey_storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestBlobStorageStream_PutStream_GetStream(t *testing.T) {
	s := NewMemoryStorage()
	stream := NewBlobStorageStream(s)
	ctx := context.Background()
	data := []byte("hello stream")
	if err := stream.PutStream(ctx, "k1", bytes.NewReader(data)); err != nil {
		t.Fatalf("PutStream: %v", err)
	}
	rc, err := stream.GetStream(ctx, "k1")
	if err != nil {
		t.Fatalf("GetStream: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("GetStream: got %q", got)
	}
}
