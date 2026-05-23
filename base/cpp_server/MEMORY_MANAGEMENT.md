# Memory Management Strategy

## Overview

This document explains how memory is managed in the C++ HTTP server, covering heap vs stack usage, memory pools, RAII patterns, and leak detection.

## Memory Allocation Strategies

### 1. Stack Allocation (Fast, Automatic)

**Used for:**
- Small, short-lived objects
- Local variables in functions
- Function parameters
- Automatic cleanup when scope exits

**Examples:**
```cpp
// Per-connection buffers sized at runtime (configured via --buffer-size).
// Allocated once when a TCPConnection is created (pool prealloc), then reused.
auto conn = std::make_unique<TCPConnection>(fd, g_config.buffer_size);
char* rbuf = conn->read_buffer();
char* wbuf = conn->write_buffer();
size_t cap = conn->buffer_capacity();
```

**Advantages:**
- Zero allocation overhead
- Automatic cleanup (no manual `delete`)
- Cache-friendly (L1/L2 cache locality)
- No fragmentation

**Limitations:**
- Limited size (typically 1-8MB per thread)
- Must know size at compile time
- Not suitable for large or variable-size allocations

### 2. Heap Allocation (Flexible, Slower)

**Used for:**
- Large objects (>8KB)
- Variable-size allocations
- Long-lived objects
- Objects shared across threads

**Examples:**
```cpp
// Heap-allocated connection pool
std::vector<std::unique_ptr<TCPConnection>> connections_;

// Heap-allocated buffers (when pool exhausted)
char* buffer = new char[buffer_size_];  // Fallback allocation
```

**Advantages:**
- Unlimited size (up to system limits)
- Dynamic sizing
- Can be shared across threads

**Disadvantages:**
- Allocation overhead (`malloc`/`new` syscalls)
- Manual cleanup required (or smart pointers)
- Memory fragmentation
- Cache misses (less locality)

### 3. Memory Pools (Best of Both Worlds)

**Strategy:** Pre-allocate fixed-size blocks, reuse them

**Implementation:**
- `MemoryPool<BlockSize, PoolSize>`: Fixed-size allocations
- `BufferPool`: Variable-size buffers (small/medium/large)
- `MemoryManager`: Centralized pool management

**Benefits:**
- Reduced allocation overhead (no syscalls in hot path)
- Better cache locality (pre-allocated blocks)
- Predictable memory usage
- Faster than heap allocation

**Trade-offs:**
- Fixed pool sizes (may need fallback to heap)
- Memory overhead (pre-allocated but unused)

## Connection Pool Architecture

### Thread-Safe Connection Management

```cpp
class TCPServer {
    // Thread-safe connection counter
    std::atomic<int> active_connections_;
    
    // Thread-safe connection pool
    mutable std::mutex connection_pool_mutex_;
    std::vector<std::unique_ptr<TCPConnection>> connections_;
    std::vector<TCPConnection*> free_connections_;
};
```

**Pre-allocation:**
- 500 connections pre-allocated at startup
- Reused across requests (no allocation in hot path)
- Thread-safe access via mutex

**Lifecycle:**
1. **Accept**: `allocate_connection()` - get from pool or allocate new
2. **Use**: Connection handles I/O operations
3. **Close**: `free_connection()` - return to pool for reuse

## RAII Patterns

### 1. Smart Pointers (`std::unique_ptr`)

**Automatic cleanup:**
```cpp
std::vector<std::unique_ptr<TCPConnection>> connections_;
// Automatically freed when TCPServer destructor runs
```

**Benefits:**
- No manual `delete` required
- Exception-safe (cleanup even if exception thrown)
- Clear ownership semantics

### 2. ScopedBuffer (RAII Wrapper)

```cpp
class ScopedBuffer {
    BufferPool* pool_;
    char* buffer_;
public:
    ~ScopedBuffer() {
        if (pool_ && buffer_) {
            pool_->release(buffer_);  // Automatic return to pool
        }
    }
};
```

**Usage:**
```cpp
{
    ScopedBuffer buf(pool, pool->acquire());
    // Use buffer...
    // Automatically returned to pool when scope exits
}
```

## Memory Leak Detection

### BufferPool Leak Tracking (DEBUG builds)

**Tracking:**
- Dynamically allocated buffers (from `new char[]` fallback)
- Set of active dynamic buffers
- Leak count on destruction

**Implementation:**
```cpp
#ifdef DEBUG
std::set<char*> dynamic_buffers_;  // Track fallback allocations
std::atomic<size_t> leaked_buffers_{0};  // Leak counter
#endif
```

**Detection:**
- Buffer not in `dynamic_buffers_` on `release()` → potential leak
- Unreleased buffers in destructor → leak detected

### External Tools

**Linux:**
```bash
valgrind --leak-check=full ./cpp_server
```

**macOS:**
```bash
leaks -atExit -- ./cpp_server
```

**AddressSanitizer (both platforms):**
```bash
g++ -fsanitize=address -g ...
```

## Memory Usage Patterns

### Hot Path (Request Handling)

**Goal:** Zero allocations in hot path

**Achieved by:**
1. Pre-allocated connection pool (500 connections)
2. Stack-allocated buffers in `TCPConnection` (8KB each)
3. Static HTTP responses (no allocation)
4. Memory pools for variable-size buffers

**Result:** ~16KB per connection (stack) + pool overhead

### Cold Path (Initialization)

**Allowed allocations:**
- Connection pool pre-allocation
- Buffer pool initialization
- Event loop setup (io_uring/kqueue)

**Result:** One-time setup cost, no runtime allocations

## Best Practices

### 1. Prefer Stack for Small Objects
```cpp
// Good: Stack allocation
char buffer[1024];

// Bad: Heap allocation for small objects
char* buffer = new char[1024];
```

### 2. Use Pools for Frequent Allocations
```cpp
// Good: Pool allocation
char* buf = buffer_pool->acquire();

// Bad: Frequent heap allocations
char* buf = new char[size];
```

### 3. RAII for Automatic Cleanup
```cpp
// Good: RAII wrapper
ScopedBuffer buf(pool, pool->acquire());

// Bad: Manual cleanup
char* buf = pool->acquire();
// ... use buffer ...
pool->release(buf);  // Easy to forget!
```

### 4. Atomic for Thread-Safe Counters
```cpp
// Good: Atomic counter
std::atomic<int> active_connections_;

// Bad: Non-atomic counter (race condition)
int active_connections_;
```

## Performance Considerations

### Allocation Overhead

**Stack allocation:** ~0 cycles (compile-time)
**Pool allocation:** ~10-50 cycles (pointer manipulation)
**Heap allocation:** ~100-1000 cycles (syscall + fragmentation)

### Cache Locality

**Stack:** Excellent (L1 cache)
**Pool:** Good (pre-allocated, sequential)
**Heap:** Poor (scattered, cache misses)

### Memory Fragmentation

**Stack:** None (sequential)
**Pool:** Minimal (fixed-size blocks)
**Heap:** High (variable-size, fragmentation)

## Summary

- **Stack**: Small, short-lived objects (buffers, local variables)
- **Heap**: Large, variable-size, long-lived objects
- **Pools**: Frequent allocations (connections, buffers)
- **RAII**: Automatic cleanup (smart pointers, ScopedBuffer)
- **Atomic**: Thread-safe counters (`active_connections_`)
- **Mutex**: Thread-safe pools (`connection_pool_mutex_`)

**Result:** Zero allocations in hot path, predictable memory usage, thread-safe operations.
