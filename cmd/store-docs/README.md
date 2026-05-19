# store-docs

Stores `docs/*.md` (and nested `.md` files) into **buckey_storage** and optionally indexes them in **hyperspace-search**.

## Usage

From the repo root:

```bash
# 1. Store docs to buckey_storage (in-memory by default)
go run ./cmd/store-docs

# With a config file (e.g. FS or S3 backend)
go run ./cmd/store-docs -config=buckey_storage.json

# Custom docs directory
go run ./cmd/store-docs -docs=./docs
```

```bash
# 2. Index the same docs into hyperspace-search (Rust)
cd hyperspace-search && cargo run -- index-dir --dir ../docs
```

## Options

- `-docs=docs` — Directory containing `.md` files (default: `docs`)
- `-config=` — Path to `buckey_storage.json`; if empty, uses in-memory storage

## Config example

Save as `buckey_storage.json` for persistent storage:

```json
{
  "backend": "fs",
  "path": "/var/data/blobs",
  "prefix": "app/"
}
```

Or use `"backend": "memory"` for testing.
