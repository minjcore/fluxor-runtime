# llmwiki-inventory

Builds the **[LLMWiki](../LLMWiki/README.md) warehouse** (`LLMWiki/inventory.json`): one `items[]` list for every source — the default way to map documentation across the repo (and merged externals). The built-in provider scans repo `.md` files as `source: local_markdown`. You can **merge** rows from Jira, Confluence, web crawlers, etc. via JSON (`-merge`). Generates `LLMWiki/INDEX.md` (no content copy).

Schema and field meanings: [LLMWiki/SCHEMA.md](../../LLMWiki/SCHEMA.md).

## Usage

From the repository root:

```bash
go run ./cmd/llmwiki-inventory
```

With merged external catalogs:

```bash
go run ./cmd/llmwiki-inventory -merge=./imports/jira-items.json,./imports/confluence-items.json
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-root` | `.` | Directory to scan for `local_markdown` |
| `-out` | `LLMWiki` | Where to write `inventory.json` and `INDEX.md` |
| `-skip` | _(empty)_ | Extra comma-separated directory **base names** to skip (e.g. `build,coverage`) |
| `-merge` | _(empty)_ | Comma-separated JSON files, each `{"items":[...]}` (same shape as in SCHEMA.md) |

Skipped by default: `node_modules`, `vendor`, `dist`, `.git`, `.cursor`, `target`.

The output directory tree is excluded from the scan so generated files are not listed as sources.
