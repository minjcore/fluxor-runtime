# Kernel-Userspace Architecture

## Overview

This document explains the kernel-userspace boundary and how the high-performance HTTP server interacts with the Linux kernel (io_uring) and macOS kernel (kqueue).

## Kernel vs Userspace

### Userspace (Application Layer)
- **Location**: Application memory space (cpp_server process)
- **Responsibilities**:
  - HTTP parsing and routing
  - Request/response handling
  - Connection state management
  - Business logic
- **Limitations**: Cannot directly access hardware or kernel data structures

### Kernel Space
- **Location**: Protected kernel memory
- **Responsibilities**:
  - Network stack (TCP/IP)
  - Socket management
  - I/O operations (read/write)
  - Event notification
  - System resource management
- **Access**: Only through system calls (syscalls)

## System Call Boundary

```
┌─────────────────────────────────────┐
│      Userspace (cpp_server)         │
│  ┌───────────────────────────────┐ │
│  │  Application Code             │ │
│  │  - HTTP parsing               │ │
│  │  - Request handling           │ │
│  │  - Connection management      │ │
│  └───────────┬───────────────────┘ │
│              │ syscalls             │
└──────────────┼──────────────────────┘
               │
               ▼
┌──────────────┼──────────────────────┐
│             │ Kernel Space              │
│  ┌──────────▼──────────────────────┐  │
│  │  Network Stack                   │  │
│  │  - TCP/IP processing             │  │
│  │  - Socket buffers                │  │
│  │  - Connection state              │  │
│  └──────────────────────────────────┘  │
│  ┌──────────────────────────────────┐  │
│  │  I/O Subsystem                  │  │
│  │  - io_uring (Linux)             │  │
│  │  - kqueue (macOS)               │  │
│  │  - Event notification           │  │
│  └──────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

## Linux: io_uring Architecture

### How io_uring Works

io_uring provides a **shared memory ring buffer** between userspace and kernel:

```
Userspace                          Kernel Space
┌──────────────┐                  ┌──────────────┐
│  SQ (Submit) │ ────submit──────> │  io_uring    │
│  Ring Buffer │                  │  Kernel      │
└──────────────┘                  │  Processing  │
                                  │              │
┌──────────────┐                  │              │
│  CQ (Complete)│ <───complete──── │              │
│  Ring Buffer │                  └──────────────┘
└──────────────┘
```

### Key Components

1. **Submission Queue (SQ)**
   - Userspace writes I/O requests here
   - Kernel reads from this queue
   - Shared memory buffer (mmap'd)

2. **Completion Queue (CQ)**
   - Kernel writes completion events here
   - Userspace reads from this queue
   - Shared memory buffer (mmap'd)

3. **System Calls**
   - `io_uring_setup()`: Initialize io_uring instance
   - `io_uring_enter()`: Submit requests and wait for completions
   - `io_uring_register()`: Register buffers/files (optional optimization)

### Code Flow (Linux)

```cpp
// 1. Setup io_uring (userspace -> kernel)
io_uring_queue_init(ENTRIES, &ring, 0);

// 2. Submit I/O request (userspace -> kernel)
io_uring_prep_read(sqe, fd, buf, len, offset);
io_uring_submit(&ring);  // syscall: io_uring_enter()

// 3. Kernel processes I/O (kernel space)
// - Reads from socket buffer
// - Copies data to userspace buffer
// - Writes completion to CQ

// 4. Get completion (kernel -> userspace)
io_uring_wait_cqe(&ring, &cqe);  // syscall: io_uring_enter()
// Process completion
io_uring_cqe_seen(&ring, cqe);
```

### Performance Benefits

- **Batched syscalls**: Submit multiple I/O ops with one syscall
- **Zero-copy**: Can use registered buffers (kernel bypass)
- **Async I/O**: No blocking, kernel notifies via CQ
- **Shared memory**: No data copying for queue operations

## macOS: kqueue Architecture

### How kqueue Works

kqueue is a **kernel event notification mechanism**:

```
Userspace                          Kernel Space
┌──────────────┐                  ┌──────────────┐
│  kqueue fd   │                  │  Kernel      │
│  (file desc) │                  │  Event       │
└──────┬───────┘                  │  Filter      │
       │                          │              │
       │───kevent(register)──────>│              │
       │                          │  Monitors:   │
       │                          │  - Sockets   │
       │                          │  - Files     │
       │                          │  - Signals   │
       │<───kevent(events)────────│              │
       │                          └──────────────┘
```

### Key Components

1. **kqueue file descriptor**
   - Created with `kqueue()` syscall
   - Used to register and receive events

2. **Event Filters (EVFILT_*)**
   - `EVFILT_READ`: Socket has data to read
   - `EVFILT_WRITE`: Socket ready for writing
   - `EVFILT_TIMER`: Timer events

3. **System Calls**
   - `kqueue()`: Create event queue
   - `kevent()`: Register filters and receive events

### Code Flow (macOS)

```cpp
// 1. Create kqueue (userspace -> kernel)
int kq = kqueue();  // syscall

// 2. Register event filter (userspace -> kernel)
struct kevent ev;
EV_SET(&ev, fd, EVFILT_READ, EV_ADD | EV_ONESHOT, 0, 0, NULL);
kevent(kq, &ev, 1, NULL, 0, NULL);  // syscall

// 3. Kernel monitors socket (kernel space)
// - Watches for readable data
// - When data arrives, marks event ready

// 4. Wait for events (kernel -> userspace)
struct kevent events[256];
int n = kevent(kq, NULL, 0, events, 256, &timeout);  // syscall

// 5. Process events (userspace)
for (int i = 0; i < n; i++) {
    if (events[i].filter == EVFILT_READ) {
        // Read data from socket
        read(events[i].ident, buf, len);  // syscall
    }
}
```

### Performance Characteristics

- **Event-driven**: Kernel notifies when I/O is ready
- **Efficient**: Single syscall can handle multiple events
- **One-shot mode**: Event auto-disables after trigger (EV_ONESHOT)
- **Timeout support**: Can wait with timeout

## Comparison: io_uring vs kqueue

| Feature | io_uring (Linux) | kqueue (macOS) |
|---------|------------------|----------------|
| **Architecture** | Shared memory rings | Event notification |
| **Syscall Model** | Batched submit/wait | Per-event registration |
| **Zero-copy** | Yes (registered buffers) | No |
| **Async I/O** | True async | Event-driven |
| **Batch Operations** | Excellent | Good |
| **Performance** | Very high (low syscall overhead) | High (efficient events) |

## System Call Overhead

### Traditional I/O (blocking)
```
read() syscall → kernel → wait for data → copy to userspace → return
```
- **1 syscall per I/O operation**
- **Blocking**: Thread waits for I/O
- **High overhead**: Context switch per operation

### io_uring (Linux)
```
Submit batch → kernel processes → completions → handle batch
```
- **1 syscall for multiple operations**
- **Non-blocking**: Kernel notifies via completion queue
- **Low overhead**: Shared memory, batched operations

### kqueue (macOS)
```
Register filters → kernel monitors → kevent() → process events
```
- **1 syscall per event batch**
- **Non-blocking**: Kernel notifies when ready
- **Medium overhead**: Event-driven, efficient filtering

## Memory Mapping

### io_uring Shared Memory

```cpp
// Kernel creates shared memory regions
struct io_uring_sq {
    unsigned *khead, *ktail, *kring_mask, *kring_entries;
    unsigned *kflags, *kdropped;
    unsigned *array;  // SQE array
    struct io_uring_sqe *sqes;  // Submission queue entries
};

struct io_uring_cq {
    unsigned *khead, *ktail, *kring_mask, *kring_entries;
    unsigned *kflags, *koverflow;
    struct io_uring_cqe *cqes;  // Completion queue entries
};

// Userspace and kernel both access these via mmap()
```

### kqueue Event Structure

```cpp
// Events passed through syscall (not shared memory)
struct kevent {
    uintptr_t ident;      // File descriptor
    short filter;         // Event filter (EVFILT_READ, etc.)
    unsigned short flags; // Event flags (EV_ADD, EV_ONESHOT, etc.)
    unsigned int fflags;  // Filter-specific flags
    intptr_t data;        // Filter-specific data
    void *udata;          // User data pointer
};
```

## Performance Implications

### Context Switches

**Traditional I/O**: High context switch overhead
- Each `read()`/`write()` requires:
  1. Userspace → Kernel (syscall)
  2. Kernel → Userspace (return)
  3. Potentially multiple switches if blocking

**io_uring**: Minimal context switches
- Batch submit: 1 syscall for N operations
- Batch completion: 1 syscall for N completions
- Shared memory eliminates some copies

**kqueue**: Moderate context switches
- 1 syscall per event batch
- Efficient event filtering in kernel

### CPU Cache Locality

**io_uring**:
- Shared memory rings stay in cache
- Batch processing improves cache hit rate

**kqueue**:
- Event structures passed through syscall
- Less cache-friendly than shared memory

## Security Considerations

### Kernel-Userspace Boundary

1. **System Call Validation**
   - Kernel validates all syscall parameters
   - Prevents invalid memory access
   - Enforces permissions

2. **Memory Protection**
   - Userspace cannot access kernel memory
   - Kernel cannot directly access userspace (must copy)
   - MMU enforces protection

3. **Resource Limits**
   - Kernel enforces file descriptor limits
   - Memory limits per process
   - CPU time limits

## Debugging Kernel-Userspace Interaction

### Tools

1. **strace** (Linux): Trace system calls
   ```bash
   strace -e trace=io_uring_enter ./cpp_server
   ```

2. **dtruss** (macOS): Trace system calls
   ```bash
   sudo dtruss ./cpp_server
   ```

3. **perf** (Linux): Profile kernel and userspace
   ```bash
   perf record -e syscalls:sys_enter_io_uring_enter ./cpp_server
   ```

### Common Issues

1. **Syscall Errors**
   - Check `errno` after syscalls
   - Common: `EAGAIN`, `EMFILE`, `ENFILE`

2. **Memory Issues**
   - Invalid pointers passed to kernel
   - Buffer overflows
   - Use-after-free

3. **Performance Issues**
   - Too many syscalls (not batching)
   - Context switch overhead
   - Cache misses

## Best Practices

1. **Batch Operations**
   - Submit multiple I/O ops together
   - Process completions in batches

2. **Minimize Syscalls**
   - Use shared memory when possible
   - Batch event processing

3. **Error Handling**
   - Always check syscall return values
   - Handle `EAGAIN`/`EWOULDBLOCK` gracefully

4. **Resource Management**
   - Close file descriptors properly
   - Clean up kernel resources on shutdown

## References

- Linux: `man 2 io_uring_setup`, `man 2 io_uring_enter`
- macOS: `man 2 kqueue`, `man 2 kevent`
- io_uring: https://kernel.dk/io_uring.pdf
- kqueue: https://www.freebsd.org/cgi/man.cgi?query=kqueue
