#!/usr/bin/env bash
set -euo pipefail

PORT="${1:-8083}"

cd "$(dirname "$0")/.."

make -s

./cpp_server -w 1 -p "$PORT" --backlog 65535 >/tmp/cpp_server_smoke.log 2>&1 &
pid=$!

cleanup() {
  kill "$pid" >/dev/null 2>&1 || true
  wait "$pid" >/dev/null 2>&1 || true
}
trap cleanup EXIT

sleep 0.2

body="$(curl -sS "http://127.0.0.1:${PORT}/test")"
if [[ "$body" != "OK" ]]; then
  echo "FAIL: expected OK, got: $body"
  exit 1
fi

echo "OK: smoke_http"

