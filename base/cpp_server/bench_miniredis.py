#!/usr/bin/env python3
"""
Minimal benchmark for mini_redis_server (RESP) without external deps.

Measures client-observed ops/sec for a fixed command mix.

Defaults: PING with pipelining (fastest sanity check).
"""

import argparse
import os
import socket
import time
from multiprocessing import Process, Queue


PING_CMD = b"*1\r\n$4\r\nPING\r\n"
PING_RESP = b"+PONG\r\n"
OK_RESP = b"+OK\r\n"


def build_pipeline(cmd: bytes, n: int) -> bytes:
    return cmd * n


def resp_cmd(*parts: bytes) -> bytes:
    # Build RESP array of bulk strings.
    out = [f"*{len(parts)}\r\n".encode()]
    for p in parts:
        out.append(f"${len(p)}\r\n".encode())
        out.append(p)
        out.append(b"\r\n")
    return b"".join(out)


def recv_exact_bytes(sock: socket.socket, nbytes: int) -> bytes:
    remaining = nbytes
    chunks = []
    while remaining > 0:
        try:
            data = sock.recv(remaining)
        except socket.timeout:
            break
        if not data:
            break
        chunks.append(data)
        remaining -= len(data)
    return b"".join(chunks)


def recv_exact(sock: socket.socket, nbytes: int) -> bool:
    return len(recv_exact_bytes(sock, nbytes)) == nbytes


class XorShift64:
    __slots__ = ("x",)

    def __init__(self, seed: int) -> None:
        self.x = seed & 0xFFFFFFFFFFFFFFFF or 0x9E3779B97F4A7C15

    def next_u64(self) -> int:
        x = self.x
        x ^= (x << 13) & 0xFFFFFFFFFFFFFFFF
        x ^= (x >> 7) & 0xFFFFFFFFFFFFFFFF
        x ^= (x << 17) & 0xFFFFFFFFFFFFFFFF
        self.x = x & 0xFFFFFFFFFFFFFFFF
        return self.x


def build_get_resp(value: bytes) -> bytes:
    return b"$" + str(len(value)).encode() + b"\r\n" + value + b"\r\n"


def prefill_keyspace(sock: socket.socket, key_prefix: bytes, keyspace: int, value: bytes, pipeline: int) -> None:
    # Prefill keys so GET always hits and response length is stable.
    i = 0
    while i < keyspace:
        batch = min(pipeline, keyspace - i)
        payload_parts = []
        for j in range(batch):
            k = key_prefix + str(i + j).encode()
            payload_parts.append(resp_cmd(b"SET", k, value))
        payload = b"".join(payload_parts)
        sock.sendall(payload)
        if not recv_exact(sock, len(OK_RESP) * batch):
            raise RuntimeError("prefill: server closed connection")
        i += batch


def build_setget_pipeline(
    rng: XorShift64,
    key_prefix: bytes,
    keyspace: int,
    value: bytes,
    pipeline: int,
    get_pct: int,
) -> tuple[bytes, int, int, int]:
    # Returns (payload, expected_bytes, get_ops, set_ops)
    get_resp = build_get_resp(value)
    expected = 0
    gets = 0
    sets = 0
    parts = []
    for _ in range(pipeline):
        r = rng.next_u64() % 100
        k = key_prefix + str(rng.next_u64() % keyspace).encode()
        if r < get_pct:
            parts.append(resp_cmd(b"GET", k))
            expected += len(get_resp)
            gets += 1
        else:
            parts.append(resp_cmd(b"SET", k, value))
            expected += len(OK_RESP)
            sets += 1
    return b"".join(parts), expected, gets, sets


def build_get_pipeline(
    rng: XorShift64,
    key_prefix: bytes,
    keyspace: int,
    value: bytes,
    pipeline: int,
) -> tuple[bytes, int]:
    get_resp = build_get_resp(value)
    expected = len(get_resp) * pipeline
    parts = []
    for _ in range(pipeline):
        k = key_prefix + str(rng.next_u64() % keyspace).encode()
        parts.append(resp_cmd(b"GET", k))
    return b"".join(parts), expected


def build_set_pipeline(
    rng: XorShift64,
    key_prefix: bytes,
    keyspace: int,
    value: bytes,
    pipeline: int,
) -> tuple[bytes, int]:
    expected = len(OK_RESP) * pipeline
    parts = []
    for _ in range(pipeline):
        k = key_prefix + str(rng.next_u64() % keyspace).encode()
        parts.append(resp_cmd(b"SET", k, value))
    return b"".join(parts), expected


def worker(
    host: str,
    port: int,
    conns: int,
    pipeline: int,
    duration_s: float,
    inflight: int,
    mode: str,
    keyspace: int,
    value_size: int,
    get_pct: int,
    verify: bool,
    q: Queue,
) -> None:
    socks = []
    try:
        for _ in range(conns):
            s = socket.create_connection((host, port), timeout=2.0)
            s.setsockopt(socket.IPPROTO_TCP, socket.TCP_NODELAY, 1)
            # Under write-heavy workloads, per-roundtrip latency can exceed 2s.
            s.settimeout(max(2.0, float(duration_s)))
            socks.append(s)

        seed = (os.getpid() << 32) ^ int(time.time_ns() & 0xFFFFFFFFFFFFFFFF)
        rng = XorShift64(seed)

        start = time.perf_counter()
        end = start + duration_s
        ops = 0
        get_ops = 0
        set_ops = 0

        # Prep workload
        value = (b"x" * value_size) if value_size > 0 else b""
        if mode in ("setget", "get", "set"):
            # Prefill using first connection to minimize overhead.
            key_prefix = b"k:"
            if mode in ("setget", "get"):
                prefill_keyspace(socks[0], key_prefix, keyspace, value, pipeline=max(1, min(256, pipeline)))

            # Precompute a small ring of pipelines to reduce per-iteration overhead.
            ring = []
            if mode == "setget":
                for _ in range(256):
                    payload, expected, g, s = build_setget_pipeline(
                        rng, key_prefix, keyspace, value, pipeline, get_pct
                    )
                    ring.append((payload, expected, g, s))
            elif mode == "get":
                for _ in range(256):
                    payload, expected = build_get_pipeline(rng, key_prefix, keyspace, value, pipeline)
                    ring.append((payload, expected, pipeline, 0))
            else:  # mode == "set"
                for _ in range(256):
                    payload, expected = build_set_pipeline(rng, key_prefix, keyspace, value, pipeline)
                    ring.append((payload, expected, 0, pipeline))
        else:
            payload = build_pipeline(PING_CMD, pipeline)
            expected = len(PING_RESP) * pipeline

        # Round-robin over connections.
        i = 0
        j = 0
        while True:
            now = time.perf_counter()
            if now >= end:
                break
            s = socks[i]
            i += 1
            if i >= len(socks):
                i = 0

            # "Real" pipelining: send N requests before reading responses (in-flight).
            # This keeps the connection busy and typically yields higher throughput.
            batch = []
            for _ in range(max(1, inflight)):
                if mode in ("setget", "get", "set"):
                    payload, expected, g, ss = ring[j]
                    j += 1
                    if j >= len(ring):
                        j = 0
                    s.sendall(payload)
                    batch.append((expected, g, ss))
                else:
                    s.sendall(payload)
                    batch.append((expected, 0, 0))

            for expected, g, ss in batch:
                data = recv_exact_bytes(s, expected)
                if len(data) != expected:
                    batch = None
                    break
                if verify and mode in ("setget", "get", "set"):
                    # Lightweight verification: ensure each response starts with '+' or '$'
                    # (does not fully parse, but catches obvious desync).
                    cnt = 0
                    idx = 0
                    while idx < len(data) and cnt < pipeline:
                        b0 = data[idx : idx + 1]
                        if b0 == b"+":
                            idx = data.find(b"\r\n", idx) + 2
                        elif b0 == b"$":
                            eol = data.find(b"\r\n", idx)
                            ln = int(data[idx + 1 : eol])
                            idx = eol + 2 + ln + 2
                        else:
                            raise RuntimeError(f"verify failed at byte {idx}: {b0!r}")
                        cnt += 1
                ops += pipeline
                get_ops += g
                set_ops += ss
            if batch is None:
                break

        elapsed = time.perf_counter() - start
        q.put((ops, elapsed, get_ops, set_ops))
    finally:
        for s in socks:
            try:
                s.close()
            except Exception:
                pass


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--port", type=int, default=6380)
    ap.add_argument("--mode", choices=["ping", "setget", "get", "set"], default="ping")
    ap.add_argument("-c", "--connections", type=int, default=50, help="connections per process")
    ap.add_argument("-P", "--pipeline", type=int, default=16, help="commands per request (pipeline)")
    ap.add_argument("--inflight", type=int, default=1, help="requests in-flight per connection (send N before recv)")
    ap.add_argument("-d", "--duration", type=float, default=10.0, help="seconds")
    ap.add_argument("-p", "--processes", type=int, default=max(1, (os.cpu_count() or 1) // 2))
    ap.add_argument("--keyspace", type=int, default=10000, help="keys for set/get mode (prefilled)")
    ap.add_argument("--value-size", type=int, default=16, help="value size bytes for set/get mode")
    ap.add_argument("--get-pct", type=int, default=80, help="GET percent in set/get mix (0-100)")
    ap.add_argument("--verify", action="store_true", help="lightweight response verification (slower)")
    args = ap.parse_args()

    q: Queue = Queue()
    procs = []
    for _ in range(args.processes):
        p = Process(
            target=worker,
            args=(
                args.host,
                args.port,
                args.connections,
                args.pipeline,
                args.duration,
                args.inflight,
                args.mode,
                args.keyspace,
                args.value_size,
                args.get_pct,
                args.verify,
                q,
            ),
            daemon=True,
        )
        p.start()
        procs.append(p)

    total_ops = 0
    total_get = 0
    total_set = 0
    max_elapsed = 0.0
    for _ in procs:
        ops, elapsed, g, s = q.get()
        total_ops += int(ops)
        total_get += int(g)
        total_set += int(s)
        if elapsed > max_elapsed:
            max_elapsed = float(elapsed)

    for p in procs:
        p.join(timeout=1.0)

    rps = total_ops / max_elapsed if max_elapsed > 0 else 0.0
    if args.mode in ("setget", "get", "set"):
        mode_label = {
            "setget": "SET/GET mix",
            "get": "GET-only",
            "set": "SET-only",
        }[args.mode]
        extra = ""
        if args.mode == "setget":
            extra = f" get_pct={args.get_pct}"
        print(
            f"mini_redis_server {mode_label}: procs={args.processes} conns/proc={args.connections} "
            f"pipeline={args.pipeline} inflight={args.inflight} duration={args.duration:.1f}s keyspace={args.keyspace} "
            f"value_size={args.value_size}B{extra}"
        )
        if total_ops > 0:
            print(
                f"Mix: GET={total_get} ({(100.0*total_get/total_ops):.1f}%)  "
                f"SET={total_set} ({(100.0*total_set/total_ops):.1f}%)"
            )
    else:
        print(
            f"mini_redis_server PING: procs={args.processes} conns/proc={args.connections} "
            f"pipeline={args.pipeline} inflight={args.inflight} duration={args.duration:.1f}s"
        )
    print(f"Total ops: {total_ops}  Elapsed(max): {max_elapsed:.3f}s  Ops/sec: {rps:,.0f}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

