#!/usr/bin/env bash
set -euo pipefail

# Auto-sweep wrk parameters and summarize results.
#
# Typical usage:
#   cd cpp-server
#   ./benchmark_sweep.sh
#
# Options:
#   --url URL                 (default: http://127.0.0.1:8083/test)
#   --duration SECONDS        (default: 30)
#   --threads "8 16"          (default: "8 16")
#   --conns "256 512 1024"    (default: "64 128 256 512 1024 2000 4000")
#   --server-cmd "..."        (default: ./cpp_server -w 0 -p 8083 --backlog 65535)
#   --warmup SECONDS          (default: 2)
#   --pin-server "0-7"        (optional) taskset CPU list for server
#   --pin-client "8-15"       (optional) taskset CPU list for wrk
#   --out FILE.csv            (default: benchmark_sweep_YYYYmmdd_HHMMSS.csv)
#   --dry-run                 Print commands only
#
# Notes:
# - If you want ulimit to apply to this terminal, do:
#     source ./increase_limits.sh

URL="http://127.0.0.1:8083/test"
DURATION="30"
THREADS_LIST="8 16"
CONNS_LIST="64 128 256 512 1024 2000 4000"
SERVER_CMD="./cpp_server -w 0 -p 8083 --backlog 65535"
WARMUP="2"
PIN_SERVER=""
PIN_CLIENT=""
OUT=""
DRY_RUN=0

usage() { sed -n '1,120p' "$0"; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --url) URL="$2"; shift 2 ;;
    --duration) DURATION="$2"; shift 2 ;;
    --threads) THREADS_LIST="$2"; shift 2 ;;
    --conns) CONNS_LIST="$2"; shift 2 ;;
    --server-cmd) SERVER_CMD="$2"; shift 2 ;;
    --warmup) WARMUP="$2"; shift 2 ;;
    --pin-server) PIN_SERVER="$2"; shift 2 ;;
    --pin-client) PIN_CLIENT="$2"; shift 2 ;;
    --out) OUT="$2"; shift 2 ;;
    --dry-run) DRY_RUN=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *)
      echo "Unknown arg: $1" >&2
      usage
      exit 2
      ;;
  esac
done

ts() { date +"%Y%m%d_%H%M%S"; }
if [[ -z "$OUT" ]]; then
  OUT="benchmark_sweep_$(ts).csv"
fi

run() {
  if [[ "$DRY_RUN" == "1" ]]; then
    echo "+ $*"
    return 0
  fi
  eval "$@"
}

need_bin() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing required tool: $1" >&2
    exit 1
  }
}

need_bin wrk

TASKSET_PREFIX_SERVER=""
TASKSET_PREFIX_CLIENT=""
if [[ -n "$PIN_SERVER" || -n "$PIN_CLIENT" ]]; then
  need_bin taskset
fi
if [[ -n "$PIN_SERVER" ]]; then
  TASKSET_PREFIX_SERVER="taskset -c $PIN_SERVER"
fi
if [[ -n "$PIN_CLIENT" ]]; then
  TASKSET_PREFIX_CLIENT="taskset -c $PIN_CLIENT"
fi

SERVER_PID=""
cleanup() {
  if [[ -n "${SERVER_PID}" ]]; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

wait_port() {
  # best-effort wait for localhost:port to accept connections
  local host port
  host="$(echo "$URL" | sed -E 's#^https?://([^/:]+).*#\1#')"
  port="$(echo "$URL" | sed -E 's#^https?://[^/:]+:([0-9]+).*#\1#')"
  if [[ "$port" == "$URL" ]]; then
    # no explicit port in URL -> default
    if [[ "$URL" =~ ^https:// ]]; then port="443"; else port="80"; fi
  fi

  local i
  for i in {1..100}; do
    if (echo >"/dev/tcp/${host}/${port}") >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.05
  done
  return 1
}

echo "== OS / host info =="
uname -a 2>/dev/null || true
if command -v nproc >/dev/null 2>&1; then echo "CPUs: $(nproc)"; elif [[ "$(uname -s)" == Darwin ]]; then echo "CPUs: $(sysctl -n hw.ncpu 2>/dev/null)"; fi
if [[ "$(uname -s)" == Darwin ]]; then sysctl -n machdep.cpu.brand_string 2>/dev/null || true; fi
echo

echo "== Config =="
echo "URL        : $URL"
echo "Duration   : ${DURATION}s"
echo "Warmup     : ${WARMUP}s"
echo "Threads    : $THREADS_LIST"
echo "Connections: $CONNS_LIST"
echo "Server cmd : $SERVER_CMD"
if [[ -n "$PIN_SERVER" ]]; then echo "Pin server : $PIN_SERVER"; fi
if [[ -n "$PIN_CLIENT" ]]; then echo "Pin client : $PIN_CLIENT"; fi
echo "Output CSV : $OUT"
echo

echo "== Starting server =="
run "${TASKSET_PREFIX_SERVER} ${SERVER_CMD} >/tmp/cpp_server_sweep.log 2>&1 & echo \$! > /tmp/cpp_server_sweep.pid"
if [[ "$DRY_RUN" == "0" ]]; then
  SERVER_PID="$(cat /tmp/cpp_server_sweep.pid)"
fi

if [[ "$DRY_RUN" == "0" ]]; then
  if ! wait_port; then
    echo "Server did not open port in time. Log: /tmp/cpp_server_sweep.log" >&2
    exit 1
  fi
fi

echo "== Warmup =="
run "${TASKSET_PREFIX_CLIENT} wrk -t1 -c10 -d${WARMUP}s \"$URL\" >/dev/null 2>&1 || true"
echo

echo "== Running sweep =="
echo "threads,connections,requests_sec,lat_avg,lat_p50,lat_p90,lat_p99,connect_err,read_err,write_err,timeout_err" >"$OUT"

best_rps="0"
best_key=""

print_row() {
  local t c rps lat_avg lat50 lat90 lat99 ce re we te
  t="$1"; c="$2"; rps="$3"; lat_avg="$4"; lat50="$5"; lat90="$6"; lat99="$7"
  ce="$8"; re="$9"; we="${10}"; te="${11}"
  printf "t=%-3s c=%-5s  rps=%-12s  avg=%-10s p50=%-10s p90=%-10s p99=%-10s  conn_err=%s\n" \
    "$t" "$c" "$rps" "$lat_avg" "$lat50" "$lat90" "$lat99" "$ce"
}

for t in $THREADS_LIST; do
  for c in $CONNS_LIST; do
    echo "--- wrk -t${t} -c${c} -d${DURATION}s --latency ---"
    if [[ "$DRY_RUN" == "1" ]]; then
      echo "+ ${TASKSET_PREFIX_CLIENT} wrk -t${t} -c${c} -d${DURATION}s --latency \"$URL\""
      continue
    fi

    tmp="$(mktemp)"
    set +e
    ${TASKSET_PREFIX_CLIENT} wrk -t"${t}" -c"${c}" -d"${DURATION}s" --latency "$URL" | tee "$tmp"
    wrk_rc="${PIPESTATUS[0]}"
    set -e

    # Parse results (best-effort)
    rps="$(awk '/Requests\/sec:/ {print $2}' "$tmp" | tail -n 1)"
    lat_avg="$(awk '/^[[:space:]]*Latency[[:space:]]/ {print $2}' "$tmp" | head -n 1)"
    lat50="$(awk '/^[[:space:]]*50%/ {print $2}' "$tmp" | head -n 1)"
    lat90="$(awk '/^[[:space:]]*90%/ {print $2}' "$tmp" | head -n 1)"
    lat99="$(awk '/^[[:space:]]*99%/ {print $2}' "$tmp" | head -n 1)"

    # Socket errors line may not exist
    sock_line="$(awk '/Socket errors:/ {print}' "$tmp" | tail -n 1 || true)"
    ce="0"; re="0"; we="0"; te="0"
    if [[ -n "$sock_line" ]]; then
      ce="$(echo "$sock_line" | sed -n 's/.*connect \([0-9]\+\).*/\1/p')"
      re="$(echo "$sock_line" | sed -n 's/.*read \([0-9]\+\).*/\1/p')"
      we="$(echo "$sock_line" | sed -n 's/.*write \([0-9]\+\).*/\1/p')"
      te="$(echo "$sock_line" | sed -n 's/.*timeout \([0-9]\+\).*/\1/p')"
      ce="${ce:-0}"; re="${re:-0}"; we="${we:-0}"; te="${te:-0}"
    fi

    echo "${t},${c},${rps},${lat_avg},${lat50},${lat90},${lat99},${ce},${re},${we},${te}" >>"$OUT"
    print_row "$t" "$c" "$rps" "$lat_avg" "$lat50" "$lat90" "$lat99" "$ce" "$re" "$we" "$te"

    # Track best RPS (numeric compare)
    rps_num="$(printf '%s' "$rps" | awk '{printf "%.0f\n", $1}')"
    if [[ -n "$rps_num" ]] && [[ "$rps_num" -gt "$best_rps" ]]; then
      best_rps="$rps_num"
      best_key="t=${t} c=${c}"
    fi

    rm -f "$tmp"

    # If wrk failed hard, keep going but note it
    if [[ "$wrk_rc" -ne 0 ]]; then
      echo "WARN: wrk exit code $wrk_rc for t=$t c=$c (continuing)" >&2
    fi
    echo
  done
done

echo "== Done =="
echo "Best RPS : ${best_rps} @ ${best_key}"
echo "CSV      : ${OUT}"
