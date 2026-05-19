# Buckey Storage Package

Blob storage abstraction for the Fluxor framework with JSON-loadable configuration.

## Features

- **BlobStorage interface**: Put, Get, Delete, List by key
- **JSON config**: Load backend and options from a file (e.g. `buckey_storage.json`)
- **In-memory backend**: Built-in implementation for tests and simple use
- **Replicated backend**: 3 replicas on one server (in-memory), compatible with in-memory semantics
- **S3 backend**: AWS S3 (or compatible) via `pkg/cloud/aws`; config: `backend`, `bucket`, `prefix`, `region`, credentials
- **FS backend**: Local filesystem; config: `backend`, `path`, `prefix`
- **TTL**: Optional expiry via `NewTTLStorage(s, defaultTTL)` and `PutWithTTL(ctx, key, data, ttl)`
- **Streaming**: `BlobStorageStreaming` with `PutStream`/`GetStream`; wrap any store with `NewBlobStorageStream(s)`
- **Index/search**: Full-text inverted index (search-engine-like) and field/pattern filter (awk-like)
- **Fail-fast validation**: Keys and context validated before operations
- **Context support**: All operations accept `context.Context`

## Quick Start

### Load config from JSON

```json
{
  "backend": "memory",
  "prefix": "app/"
}
```

Save as `buckey_storage.json` and load:

```go
import (
    "context"
    "encoding/json"
    "os"
    "github.com/fluxorio/fluxor/pkg/buckey_storage"
)

data, err := os.ReadFile("buckey_storage.json")
if err != nil {
    log.Fatal(err)
}
var cfg buckey_storage.Config
if err := json.Unmarshal(data, &cfg); err != nil {
    log.Fatal(err)
}

s, err := buckey_storage.NewFromConfig(&cfg)
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()
if err := s.Put(ctx, "mykey", []byte("hello")); err != nil {
    log.Fatal(err)
}
blob, err := s.Get(ctx, "mykey")
if err != nil {
    log.Fatal(err)
}
// blob == []byte("hello")
```

### In-memory storage without config file

```go
s := buckey_storage.NewMemoryStorage()
ctx := context.Background()
s.Put(ctx, "key", []byte("value"))
data, _ := s.Get(ctx, "key")
keys, _ := s.List(ctx, "")
s.Delete(ctx, "key")
```

### 3 replicas on one server (replicated)

```json
{ "backend": "replicated", "replicas": 3 }
```

```go
cfg := buckey_storage.Config{Backend: buckey_storage.BackendReplicated, Replicas: 3}
s := buckey_storage.NewReplicated(&cfg)
// Put writes to all 3 in-memory copies; Get reads from first successful replica
s.Put(ctx, "key", []byte("value"))
data, _ := s.Get(ctx, "key")
```

### S3 backend

```json
{ "backend": "s3", "bucket": "my-bucket", "prefix": "app/", "region": "us-east-1" }
```

Uses `pkg/cloud/aws` (env `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, or IAM). Or implement `buckey_storage.S3Ops` and use `buckey_storage.NewS3(ops, bucket, prefix)`.

### FS (filesystem) backend

```json
{ "backend": "fs", "path": "/var/data/blobs", "prefix": "app/" }
```

```go
s, err := buckey_storage.NewFS("/var/data/blobs", "app/")
```

### TTL (time-to-live)

Wrap any store to expire keys after a duration:

```go
s := buckey_storage.NewMemoryStorage()
ttl := buckey_storage.NewTTLStorage(s, 5*time.Minute)
ttl.Put(ctx, "k", []byte("v"))           // expires in 5m
ttl.PutWithTTL(ctx, "k2", []byte("v2"), 1*time.Hour)  // per-key TTL
data, err := ttl.Get(ctx, "k")           // ErrNotFound after expiry
```

### Streaming (PutStream / GetStream)

For large blobs, use `BlobStorageStreaming`:

```go
s := buckey_storage.NewMemoryStorage()
stream := buckey_storage.NewBlobStorageStream(s)
stream.PutStream(ctx, "large", reader)
rc, _ := stream.GetStream(ctx, "large")
defer rc.Close()
io.Copy(dest, rc)
```

### Index and search (full-text + awk-like)

**Full-text (search-engine-like):** inverted index, query with AND/OR.

```go
idx := buckey_storage.NewIndex()
idx.IndexFullText(ctx, "doc1", "hello world")
idx.IndexFullText(ctx, "doc2", "world foo")
keys, _ := idx.Search(ctx, "hello world")   // AND: keys with both terms
keys, _ := idx.Search(ctx, "hello OR world") // OR: keys with either term
```

**Awk-like:** index by fields, filter by regex on a column.

```go
idx.IndexRecord(ctx, "r1", map[string]string{"name": "alice", "city": "NYC"})
idx.IndexRecord(ctx, "r2", map[string]string{"name": "bob", "city": "LA"})
keys, _ := idx.FilterField(ctx, "name", "alice")  // keys where name matches
fields, _ := idx.GetFields(ctx, "r1")              // get row/record
```

## Advanced

### Replicated: quorum read and repair

Read from a majority of replicas and repair divergent copies:

```go
r := buckey_storage.NewReplicated(&buckey_storage.Config{Backend: buckey_storage.BackendReplicated, Replicas: 3})
r.Put(ctx, "key", []byte("v1"))
// Quorum read: require at least 2 replicas; return majority value; repair minority
data, err := r.GetQuorum(ctx, "key", 2)
// Force repair: make all replicas agree to majority value
data, err = r.RepairKey(ctx, "key")
```

- `GetQuorum(ctx, key, minReads)`: read from at least `minReads` replicas, return majority value, overwrite minority replicas (repair).
- `RepairKey(ctx, key)`: same as GetQuorum with minReads=1 (repair only).
- `ErrDivergent`: when no majority can be determined (e.g. 3 different values).

### Full-text search engine for Cursor / external tools

Use `FullTextEngine` to index blob content and run full-text search with snippets so **Cursor** (or an MCP server / HTTP API) can refer to buckey_storage to search external content (docs, notes, code).

```go
s, _ := buckey_storage.NewFromConfig(&cfg)
idx := buckey_storage.NewIndex()
engine := buckey_storage.NewFullTextEngine(s, idx)

// Index existing blobs (e.g. after syncing external files into storage)
n, _ := engine.IndexFromStorage(ctx, "docs/")   // prefix "" = all keys

// Search with snippets for Cursor / MCP / API
results, total, _ := engine.SearchWithSnippets(ctx, "Cursor integration", buckey_storage.QueryOptions{
    Limit: 20, Offset: 0, SnippetMaxLen: 200,
})
// results[i].Key, results[i].Snippet

// Sync: after Put, index the blob; after Delete, remove from index
engine.IndexBlob(ctx, "newkey", data)
engine.RemoveFromIndex(ctx, "deletedkey")
```

- **IndexFromStorage(ctx, prefix)**: list keys under prefix, get each blob as UTF-8 text, index into the engine (binary blobs skipped).
- **SearchWithSnippets(ctx, query, opts)**: full-text query (same syntax as Index: space = AND, ` OR ` = OR) with pagination and optional snippet length; returns `[]SearchResult{Key, Snippet}`.
- **IndexBlob / RemoveFromIndex**: keep index in sync when storage changes.

### Index: phrase search, prefix search, pagination, stats

```go
idx.IndexFullText(ctx, "doc1", "hello world from fluxor")
keys, _ := idx.SearchPhrase(ctx, "world from")   // exact phrase (substring, case-insensitive)
keys, _ := idx.SearchPrefix(ctx, "hell")         // any term starting with prefix

// Pagination + combined full-text and field filter
keys, total, _ := idx.SearchPage(ctx, "hello", buckey_storage.QueryOptions{
    Limit: 10, Offset: 0,
    Field: "tag", Pattern: "x",
})

st := idx.Stats(ctx)  // TermCount, DocCount, RecordCount
```

### Clipboard (text by title)

Store and retrieve clipboard-style text under a title (e.g. `"không bac91 buộc"`). Title can contain any character (Unicode, spaces).

```go
// Store
buckey_storage.StoreClipboard(ctx, s, "không bac91 buộc", "clipboard text here")

// Get
text, err := buckey_storage.GetClipboard(ctx, s, "không bac91 buộc")

// List all
items, _ := buckey_storage.ListClipboard(ctx, s)  // []ClipboardItem{Title, Text}

// Delete
buckey_storage.DeleteClipboard(ctx, s, "không bac91 buộc")
```

### Store a local GitHub repo (source code)

Walk a local clone and store each file under a key prefix. Skips `.git`, `node_modules`, `vendor`, etc.

```go
// Store repo into BlobStorage (keys = prefix + relative path, e.g. "repos/myproject/pkg/foo.go")
n, err := buckey_storage.StoreRepoToStorage(ctx, s, "/path/to/cloned-repo", buckey_storage.StoreRepoOptions{
    KeyPrefix:   "repos/myproject",
    TextOnly:    true,   // skip binary files
    MaxFileSize: 1 << 20, // 1 MiB per file (0 = no limit)
})
// n = number of files stored

// Store repo and index for full-text search (Cursor / FullTextEngine)
engine := buckey_storage.NewFullTextEngine(s, buckey_storage.NewIndex())
n, err = buckey_storage.StoreRepoToStorageWithIndex(ctx, engine, "/path/to/cloned-repo", buckey_storage.StoreRepoOptions{
    KeyPrefix: "repos/myproject",
    TextOnly:  true,
})
// Then: results, total, _ := engine.SearchWithSnippets(ctx, "func Bar", opts)
```

- **StoreRepoToStorage**: stores files only; use **IndexFromStorage(ctx, "repos/myproject/")** later to index.
- **StoreRepoToStorageWithIndex**: stores and indexes in one pass so you can search immediately.
- **DefaultRepoSkipDirs**: `.git`, `node_modules`, `vendor`, `__pycache__`, `dist`, `build`, `.next`, `.idea`, `.vscode`, `.cursor`, etc. Override with **StoreRepoOptions.SkipDirs**.

### Copy from file to storage

Read an original file and store its contents in storage (by key or as clipboard).

```go
// Store file contents under a key
buckey_storage.CopyFileToStorage(ctx, s, "mykey", "/path/to/file.txt")

// Store file contents as clipboard entry (title = e.g. filename or "không bac91 buộc")
buckey_storage.CopyFileToClipboard(ctx, s, "không bac91 buộc", "/path/to/file.txt")

// Use file base name as key when key is empty
buckey_storage.CopyFileToStorageAsKey(ctx, s, "", "/path/to/note.txt")  // stored under "note.txt"
```

### Seat limit (max N concurrent per scope)

Limit concurrent "seats" per scope (e.g. global or per-org). Acquire/Release by holder ID (user or session).

```go
mgr := buckey_storage.NewSeatManager(s)

// Acquire one seat; returns ErrNoSeat if limit reached
err := mgr.Acquire(ctx, "global", "user-123", 10)
if errors.Is(err, buckey_storage.ErrNoSeat) {
    // at capacity
}
_ = mgr.Release(ctx, "global", "user-123")

// Per-org seats
_ = mgr.Acquire(ctx, "org:abc", "session-xyz", 5)
used, _ := mgr.Usage(ctx, "org:abc")
```

- **Acquire(ctx, scope, holderID, limit)**: add holderID if used &lt; limit; idempotent for same holderID.
- **Release(ctx, scope, holderID)**: remove holderID.
- **Usage(ctx, scope)**: current number of holders. Keys stored under `seats/{scope}`.

### Batch and list pagination

```go
// Batch ops (work with any BlobStorage)
buckey_storage.PutMany(ctx, s, map[string][]byte{"a": []byte("1"), "b": []byte("2")})
got, _ := buckey_storage.GetMany(ctx, s, []string{"a", "b", "missing"})  // got has only found keys
buckey_storage.DeleteMany(ctx, s, []string{"a", "b"})

// List with limit/offset
page, _ := buckey_storage.ListPage(ctx, s, "prefix/", 20, 0)  // page.Keys, page.Total
```

## API

### BlobStorage interface

```go
type BlobStorage interface {
    Put(ctx context.Context, key string, data []byte) error
    Get(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string) ([]string, error)
}
```

### Config (JSON)

| Field    | Description                                          |
| -------- | ---------------------------------------------------- |
| backend  | `"memory"`, `"replicated"`, `"s3"`, `"fs"`             |
| prefix   | Optional key prefix for all keys                      |
| replicas | For `replicated`: copies per key on one server (default 3) |
| bucket   | For `s3`: S3 bucket name (required)                   |
| path     | For `fs`: root directory (required)                  |
| region, accessKeyID, secretAccessKey | For `s3` (or use env) |
| defaultTTLSeconds | Optional TTL when using `NewTTLStorage` (0 = no expiry) |

### Errors

- `buckey_storage.ErrNotFound`: returned by `Get` when the key does not exist.
- `buckey_storage.ErrDivergent`: returned by `GetQuorum` when replicas disagree and no majority exists.

## Validation

- Keys: non-empty, max 1024 characters
- Context: must be non-nil (panic if nil)
- Config: `Validate()` checks backend and returns error for empty or unsupported backend
