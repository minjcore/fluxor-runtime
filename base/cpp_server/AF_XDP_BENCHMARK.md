## AF_XDP (copy path) — UDP ping‑pong benchmark (đúng bài test)

Mục tiêu: **kernel-bypass “ăn phần ngon”**: AF_XDP **copy path**, UDP ping‑pong fixed payload, 1 core/1 queue, đo **p50/p99 + cycles/op**.

### Những gì có trong repo

- `xdp_redirect_kern.c`: XDP eBPF filter TCP/UDP dst port → redirect vào `XSKMAP`
- `afxdp_ring_consumer.c`: user-space AF_XDP consumer (mmap rings, poll RX/TX, parse L2/L3/L4, craft UDP response + checksum)
- `udp_pingpong_bench.c`: client benchmark ping‑pong (fixed payload), in p50/p99 + cycles/op

### 0) Requirements (Linux)

- Kernel + driver hỗ trợ **AF_XDP**
- Quyền: `CAP_NET_ADMIN` (attach XDP), `CAP_BPF`/`CAP_SYS_ADMIN` (tuỳ distro) để load/pin BPF
- Tools: `clang`, `bpftool`, `iproute2`
- Mount bpffs: `/sys/fs/bpf`

> macOS không chạy AF_XDP; code có `#ifdef __linux__`.

### 1) Build

Build XDP program (BPF object):

```bash
cd cpp-server
clang -O2 -g -target bpf -c xdp_redirect_kern.c -o xdp_redirect_kern.o
```

Build user-space consumer + benchmark client:

```bash
cd cpp-server
clang -O2 -Wall -Wextra -std=c11 afxdp_ring_consumer.c -o afxdp_consumer
clang -O3 -Wall -Wextra -std=c11 udp_pingpong_bench.c -o udp_pingpong_bench
```

### 2) Load + pin BPF, attach XDP

Mount bpffs (nếu chưa):

```bash
sudo mount -t bpf bpf /sys/fs/bpf || true
```

Load & pin program + maps (bpftool sẽ tạo folder và pin maps trong đó):

```bash
sudo rm -rf /sys/fs/bpf/fh || true
sudo mkdir -p /sys/fs/bpf/fh
sudo bpftool prog loadall xdp_redirect_kern.o /sys/fs/bpf/fh
```

Attach XDP vào NIC (chọn 1 trong 2 cách tuỳ hệ bạn):

```bash
# Cách A: ip link
sudo ip link set dev eth0 xdp pinned /sys/fs/bpf/fh/xdp_redirect_port

# Cách B: bpftool net (nếu có)
sudo bpftool net attach xdp pinned /sys/fs/bpf/fh/xdp_redirect_port dev eth0
```

Xác nhận map đã pin:

```bash
sudo bpftool map show pinned /sys/fs/bpf/fh/xsks_map
```

### 3) Run (1 queue, 1 core)

Pin 1 core (ví dụ core 2) + queue 0:

```bash
sudo taskset -c 2 ./afxdp_consumer --ifname eth0 --queue 0 --port 6380 --xskmap /sys/fs/bpf/fh/xsks_map
```

Ghi chú:
- `--xskmap` khiến consumer **tự update** `xsks_map[queue_id] = xsk_fd` (không cần loader riêng).
- Consumer sẽ reply UDP theo mini protocol `"FH"` (PING).

### 4) Benchmark đúng bài (không HTTP, không wrk)

Chạy client cùng core (hoặc core khác nhưng cố định), fixed payload, ping‑pong:

```bash
taskset -c 2 ./udp_pingpong_bench --host <IP-of-eth0> --port 6380 -n 1000000 --pin 2
```

Output:
- **latency p50 / p99** (ns + µs)
- **cycles/op** (x86 có rdtsc; non‑x86 sẽ báo unavailable)

### 5) Cleanup

Detach XDP:

```bash
sudo ip link set dev eth0 xdp off
```

Unpin artifacts:

```bash
sudo rm -rf /sys/fs/bpf/fh
```

### Notes / Pitfalls (phase 1)

- AF_XDP copy path không cần hugepages.
- Nếu không thấy traffic vào consumer: kiểm tra đúng **queue_id**, đúng **NIC**, đúng **dst port**, và `xsks_map` đã update.
- KPI meaningful nhất khi:
  - 1 core, 1 queue
  - fixed payload
  - tránh background noise (power-saving/c-states/irqbalance)

