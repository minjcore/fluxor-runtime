# C++ io_uring HTTP Server

High-performance HTTP server using Linux io_uring, targeting **lower CPU usage than Nginx** while maintaining 80k+ RPS.

## Features

- **io_uring**: Zero-copy, batched I/O operations
- **Zero-allocation hot path**: Pre-allocated responses
- **Event-driven**: Single-threaded, non-blocking
- **Low CPU overhead**: No runtime, no GC, minimal syscalls

## Prerequisites

- **Linux kernel 5.1+** (Ubuntu 20.04+, Debian 11+)
- **liburing-dev**: `sudo apt-get install liburing-dev`
- **g++ with C++20 support**

## Build

```bash
# Install dependencies (skip update to avoid PPA issues)
sudo apt-get install -y liburing-dev
# OR use make (may fail if package not in cache)
make install-deps

# Build
make

# Or manually
g++ -O3 -std=c++20 -march=native main.cpp -luring -pthread -o cpp_server
```

## Run

```bash
# Default port 8083
./cpp_server

# Custom port
./cpp_server 8084
```

## Benchmark

```bash
# Quick benchmark
make benchmark

# Manual benchmark
./cpp_server &
wrk -c10 -t2 -d10 http://localhost:8083/test

# Compare with Nginx
wrk -c10 -t2 -d10 http://localhost:80
```

## Expected Performance

| Metric | Nginx | C++ Server (Target) |
|--------|-------|---------------------|
| **RPS** | 80k | 80k+ |
| **CPU** | 5% | **<5%** (target) |
| **Latency** | ~100μs | ~100μs |

## Architecture

Xem bản “tách layer + sơ đồ kiến trúc” tại: `ARCHITECTURE.md`.
Nguyên tắc “safe-first” (correctness trước, tối ưu sau) tại: `SAFE_FIRST.md`.

### io_uring Benefits

1. **Batched Operations**: Submit 1000s of I/O ops in one syscall
2. **Kernel Polling**: Kernel polls for completions (zero syscalls)
3. **Zero-Copy**: Direct memory access
4. **Lower CPU**: Fewer context switches than epoll

### Why C++?

- **No runtime overhead**: No GC, no scheduler
- **Direct system calls**: Minimal abstraction
- **Zero-allocation**: Pre-allocated buffers
- **Native performance**: Compiler optimizations

## Comparison with Go Server

| Aspect | Go Server | C++ Server |
|--------|-----------|------------|
| **Runtime** | Go runtime (scheduler, GC) | None |
| **CPU Overhead** | 22-67% | **<5%** (target) |
| **I/O Model** | epoll via fasthttp | io_uring (native) |
| **Memory** | GC-managed | Manual (zero-allocation) |

## Optimization Techniques

1. **Pre-allocated responses**: `OK_RESPONSE` constant
2. **Connection pooling**: Reuse connection objects
3. **Minimal HTTP parsing**: Fast path for `/test` endpoint
4. **Batched I/O**: Multiple operations per syscall
5. **Kernel polling**: Enable if kernel 5.11+

## Future Enhancements

- [ ] Multi-threaded worker model (SO_REUSEPORT)
- [ ] HTTP/1.1 pipelining support
- [ ] Zero-copy sendfile for static files
- [ ] Connection keep-alive optimization
- [ ] Request batching

## Troubleshooting

### "liburing not found"
```bash
sudo apt-get install liburing-dev
```

### "io_uring not supported"
```bash
# Check kernel version
uname -r  # Need 5.1+

# Check io_uring support
ls /sys/kernel/config/io_uring
```

### Performance issues
- Ensure kernel 5.1+ (5.11+ for kernel polling)
- Check CPU governor: `cpupower frequency-set -g performance`
- Increase ring size if needed (RING_SIZE in main.cpp)

## References

- [io_uring Documentation](https://kernel.dk/io_uring.pdf)
- [liburing Examples](https://github.com/axboe/liburing)
- [Nginx Architecture](https://www.nginx.com/blog/inside-nginx-how-we-designed-for-performance-scale/)
