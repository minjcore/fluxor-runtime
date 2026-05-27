-- postgres-demo JWT read benchmark
-- Target: ≥13,500 RPS on /api/wallet/balance with valid JWT
--
-- Usage:
--   TOKEN=$(curl -s -XPOST http://localhost:8081/api/auth/login \
--     -H 'Content-Type: application/json' \
--     -d '{"username":"admin","password":"admin123"}' \
--     | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
--
--   wrk -t4 -c100 -d30s -s load-test/cases/postgres_demo_jwt.lua \
--     --env TOKEN=$TOKEN http://localhost:8081
--
-- Or set TOKEN in environment before running wrk.

local token = os.getenv("TOKEN") or ""

if token == "" then
  io.stderr:write("[postgres_demo_jwt] WARNING: TOKEN env var not set — requests will 401\n")
end

wrk.method  = "GET"
wrk.path    = "/api/wallet/balance"
wrk.headers = {
  ["Authorization"] = "Bearer " .. token,
  ["Connection"]    = "keep-alive",
}

-- Track latency histogram buckets
local ok_count   = 0
local err_count  = 0
local lat_total  = 0
local lat_p99    = 0

response = function(status, headers, body)
  if status == 200 then
    ok_count = ok_count + 1
  else
    err_count = err_count + 1
    if err_count <= 5 then
      io.stderr:write(string.format("[ERR] HTTP %d: %s\n", status, body or ""))
    end
  end
end

done = function(summary, latency, requests)
  local rps = summary.requests / (summary.duration / 1e6)
  local p50 = latency:percentile(50)
  local p95 = latency:percentile(95)
  local p99 = latency:percentile(99)

  io.write(string.format("\n=== postgres-demo JWT read benchmark ===\n"))
  io.write(string.format("  RPS:          %.0f  (target ≥13,500)\n", rps))
  io.write(string.format("  Latency p50:  %.2f ms\n", p50 / 1000))
  io.write(string.format("  Latency p95:  %.2f ms\n", p95 / 1000))
  io.write(string.format("  Latency p99:  %.2f ms\n", p99 / 1000))
  io.write(string.format("  OK:           %d\n", ok_count))
  io.write(string.format("  Errors:       %d\n", err_count))

  if rps >= 13500 then
    io.write("  RESULT: PASS\n")
  else
    io.write(string.format("  RESULT: FAIL (%.0f < 13,500)\n", rps))
  end
  io.write("========================================\n")
end
