# gitshare

A minimal service to securely view a single file from a private GitHub repository. Built using Fluxor's fast HTTP server, following the apps/quadgate-io structure.

## Endpoints

- GET `/file?owner=<org>&repo=<name>&path=<path>&ref=<branch>&download=1`
  - `owner` (required): GitHub org/user
  - `repo` (required): Repository name
  - `path` (required): File path in repo (e.g., `docs/readme.md`)
  - `ref` (optional): Branch/tag/commit (default: repo default branch)
  - `download` (optional): `1` or `true` to force attachment

- GET `/health` - Service health
- GET `/metrics` - Basic metrics

## Auth

Use a GitHub Personal Access Token with `repo` scope for private repos. Provide via:
- Environment: `GITHUB_TOKEN`
- Config: `github.token` in `config.json`

## Quick Start

```bash
cd cmd/gitshare
# Optional: create config.json and add token
cat > config.json <<JSON
{
  "server": {"addr": ":8088"},
  "github": {"token": "YOUR_PAT"}
}
JSON

# Build and run
go build .
./gitshare

# Fetch a file
curl "http://localhost:8088/file?owner=caokhang91&repo=playfluxor&path=README.MD&ref=main"
```

## Notes

- For raw content, we request `application/vnd.github.v3.raw` from GitHub.
- Errors return JSON (`400`) with a human-readable message.
- Ensure your PAT has access to the target private repository.
