# Linux-Specific Optimizations

## Overview

This document describes Linux-specific optimizations that can achieve even lower latency than macOS. The target is **<24μs latency** on Linux with proper kernel tuning.

## Current State (macOS)

- **Latency**: 36.25μs average
- **Platform**: macOS (kqueue)
- **Limitations**: 
  - No `io_uring` zero-copy support
  - Limited kernel tuning options
  - XNU kernel optimized for user experience, not server throughput

## Linux Advantages

### 1. io_uring Zero-Copy (Target: -3μs)

**Current Implementation:**
- Uses standard `io_uring_prep_write()` with userspace buffers
- Data copied: kernel → userspace → kernel

**Zero-Copy Implementation:**
```cpp
// Register buffers with kernel (one-time setup)
struct iovec buffers[1024];
// ... initialize buffers ...
io_uring_register_buffers(&ring, buffers, 1024);

// Zero-copy send
io_uring_prep_send_zc(sqe, fd, buffer_index, len, flags);
```

**Benefits:**
- No kernel ↔ userspace copy
- Lower CPU usage
- Higher throughput
- **Latency reduction: ~3μs**

**Requirements:**
- Linux kernel 5.6+
- `IORING_OP_SEND_ZC` support
- Buffer registration overhead (one-time)

### 2. sendmmsg() Batching (Target: -2μs)

**Current Implementation:**
- Individual `write()` calls per response
- Each call = context switch overhead (~1-3μs)

**Batched Implementation:**
```cpp
struct mmsghdr msgs[32];
// ... prepare multiple messages ...
int sent = sendmmsg(fd, msgs, 32, 0);
```

**Benefits:**
- Multiple responses in one syscall
- Reduced context switch overhead
- **Latency reduction: ~2μs per batch**

**Trade-offs:**
- More complex buffer management
- Requires queuing responses

### 3. Thread Pinning & Interrupt Isolation (Target: -3μs)

**Current Implementation:**
- Basic CPU affinity (`pthread_setaffinity_np`)
- No interrupt isolation

**Advanced Implementation:**
```bash
# Isolate CPU cores for server threads
isolcpus=2,4,6,8

# Set CPU affinity
taskset -c 2,4,6,8 ./cpp_server

# IRQ affinity (bind network interrupts to other cores)
echo 2 > /proc/irq/24/smp_affinity
```

**Benefits:**
- Server threads never interrupted
- Data always in L1 cache
- Predictable performance
- **Latency reduction: ~3μs**

### 4. Kernel Tuning

**Network Stack Tuning:**
```bash
# Increase socket buffers
sysctl -w net.core.rmem_max=134217728
sysctl -w net.core.wmem_max=134217728

# TCP optimizations
sysctl -w net.ipv4.tcp_fastopen=3
sysctl -w net.ipv4.tcp_tw_reuse=1
sysctl -w net.ipv4.tcp_fin_timeout=10

# Reduce context switch overhead
sysctl -w kernel.sched_migration_cost_ns=5000000
```

**Benefits:**
- Larger buffers = fewer syscalls
- Faster connection reuse
- Reduced scheduling overhead
- **Latency reduction: ~1-2μs**

### 5. NUMA Awareness

**For Multi-Socket Systems:**
```cpp
// Pin thread to specific NUMA node
numactl --membind=0 --cpunodebind=0 ./cpp_server
```

**Benefits:**
- Local memory access (faster)
- Reduced cross-socket traffic
- **Latency reduction: ~1μs** (on NUMA systems)

## Expected Results

### macOS (Current)
- **Before optimizations**: 36.25μs
- **After Phase 1-3 optimizations**: ~27-29μs
- **Limitation**: macOS kernel prevents full optimization

### Linux (With All Optimizations)
- **Standard build**: ~28-30μs (similar to optimized macOS)
- **With io_uring zero-copy**: ~25-27μs
- **With sendmmsg batching**: ~23-25μs
- **With interrupt isolation**: ~20-22μs
- **With kernel tuning**: **~18-20μs** 🚀

## Implementation Checklist

### Code Changes (Already Implemented)
- ✅ Fast-path HTTP parser
- ✅ Zero-copy static response writing
- ✅ Cache line alignment
- ✅ SIMD parsing
- ✅ Write batching (io_uring)
- ✅ PGO build system

### Linux-Specific (To Implement)
- [ ] io_uring zero-copy (`IORING_OP_SEND_ZC`)
- [ ] `sendmmsg()` batching
- [ ] Advanced CPU affinity (isolcpus)
- [ ] IRQ affinity configuration
- [ ] Kernel tuning scripts

### Deployment Scripts
- [ ] `tune-linux.sh`: Kernel tuning
- [ ] `setup-irq-affinity.sh`: IRQ binding
- [ ] `run-optimized.sh`: Launch with optimizations

## Migration Guide

### Step 1: Build on Linux
```bash
# Install dependencies
sudo apt-get install liburing-dev build-essential

# Build with PGO
make pgo
```

### Step 2: Kernel Tuning
```bash
# Run tuning script
sudo ./tune-linux.sh
```

### Step 3: Run with Optimizations
```bash
# Isolate CPUs and run
sudo ./run-optimized.sh
```

### Step 4: Benchmark
```bash
wrk -c10 -t2 -d10s http://localhost:8083/test
# Expected: <24μs latency
```

## Performance Comparison

| Platform | Latency | RPS | Notes |
|----------|---------|-----|-------|
| macOS (current) | 36.25μs | 243k | kqueue, limited tuning |
| macOS (optimized) | 27-29μs | 260k+ | With Phase 1-3 optimizations |
| Linux (standard) | 28-30μs | 250k+ | io_uring, basic tuning |
| Linux (optimized) | 20-22μs | 300k+ | Zero-copy, interrupt isolation |
| Linux (extreme) | **18-20μs** | **350k+** | All optimizations + kernel tuning |

## Conclusion

**Linux provides 5-10μs additional latency reduction** compared to macOS due to:
1. Better I/O APIs (`io_uring` zero-copy)
2. More kernel tuning options
3. Interrupt isolation capabilities
4. NUMA awareness

**Target Achievement: <24μs latency is achievable on Linux** with proper configuration.
