// TCP Layer 4 Implementation
// This layer manages the kernel-userspace boundary:
// - Linux: Uses io_uring (shared memory rings) for async I/O
// - macOS: Uses kqueue (event notification) for async I/O
// All syscalls cross the kernel-userspace boundary

#include "tcp_layer.h"
#include "event_loop.h"
#include "config.h"
#include <sys/socket.h>
#include <netinet/in.h>
#include <netinet/tcp.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <fcntl.h>
#include <string.h>
#include <errno.h>
#include <stdio.h>
#include <signal.h>
#include <atomic>
#include <algorithm>

// Global running flag
std::atomic<bool> TCPServer::global_running_{true};

// TCPConnection Implementation
TCPConnection::TCPConnection(int fd, size_t buffer_capacity)
    : fd_(fd),
      buffer_capacity_(buffer_capacity ? buffer_capacity : TCPConnection::DEFAULT_BUFFER_SIZE),
      read_buffer_(std::make_unique<char[]>(buffer_capacity_)),
      write_buffer_(std::make_unique<char[]>(buffer_capacity_)),
      read_pos_(0),
      write_pos_(0),
      write_len_(0),
      reading_(false),
      writing_(false) {}

TCPConnection::~TCPConnection() {
    close();
}

void TCPConnection::close() {
    if (fd_ >= 0) {
        ::close(fd_);  // SYSCALL: close() - kernel closes file descriptor
        fd_ = -1;
    }
    reading_ = false;
    writing_ = false;
    read_pos_ = 0;
    write_pos_ = 0;
}

// TCPServer Implementation
TCPServer::TCPServer(int thread_id) 
    : thread_id_(thread_id), listener_fd_(-1), active_connections_(0), running_(true) {
    // Create platform-specific event loop
    event_loop_ = create_event_loop();
    if (event_loop_) {
        event_loop_->set_server(this);
        event_loop_->set_event_callback([this](TCPServer* /* server */, const EventData& event) {
            this->handle_event(event);
        });
    }
}

TCPServer::~TCPServer() {
    shutdown();
}

bool TCPServer::setup(int port) {
    if (!event_loop_) {
        return false;
    }
    
    // Setup event loop (platform-specific initialization)
    if (!event_loop_->setup()) {
        return false;
    }
    
    // Pre-allocate connection pool for better performance
    preallocate_connections();
    
    listener_fd_ = create_listener(port);
    if (listener_fd_ < 0) {
        event_loop_->cleanup();
        return false;
    }
    
    // Register listener with event loop
    if (!event_loop_->register_listener(listener_fd_)) {
        event_loop_->cleanup();
        return false;
    }
    
    return true;
}

void TCPServer::shutdown() {
    running_ = false;
    
    if (event_loop_) {
        event_loop_->shutdown();
    }
    
    // Close all connections
    for (auto& conn : connections_) {
        if (conn && conn->is_active()) {
            conn->close();
        }
    }
    connections_.clear();
    free_connections_.clear();
    
    // Close listener
    if (listener_fd_ >= 0) {
        ::close(listener_fd_);
        listener_fd_ = -1;
    }
    
    if (event_loop_) {
        event_loop_->cleanup();
    }
}

TCPConnection* TCPServer::accept_connection() {
    // Use atomic load for thread-safe check
    if (active_connections_.load() >= (int)g_config.max_connections) {
        // At capacity - accept and immediately close
        int new_fd = accept(listener_fd_, nullptr, nullptr);
        if (new_fd >= 0) {
            ::close(new_fd);
        }
        return nullptr;
    }
    
    int new_fd = accept(listener_fd_, nullptr, nullptr);
    if (new_fd < 0) {
        // Handle different error cases
        if (errno == EAGAIN || errno == EWOULDBLOCK) {
            // No more connections available - normal case
            return nullptr;
        } else if (errno == ECONNABORTED) {
            // Connection aborted by client - ignore and continue
            return nullptr;
        } else if (errno == EMFILE || errno == ENFILE) {
            // Too many file descriptors - log but don't spam
            static int error_count = 0;
            if (error_count++ % 100 == 0) {
                fprintf(stderr, "Thread %d: File descriptor limit reached (errno: %d)\n", 
                        thread_id_, errno);
            }
            return nullptr;
        } else {
            // Other errors - log but don't spam callbacks
            static int error_count = 0;
            if (error_count++ % 100 == 0 && error_callback_) {
                error_callback_(this, nullptr, errno);
            }
            return nullptr;
        }
    }
    
    set_nonblocking(new_fd);
    // Set TCP_NODELAY for low latency (disable Nagle's algorithm)
    int flag = 1;
    setsockopt(new_fd, IPPROTO_TCP, TCP_NODELAY, &flag, sizeof(flag));

    // Optional busy-polling to reduce latency on Linux (micro-bench friendly).
    // Requires kernel support; if unsupported, the setsockopt will fail harmlessly.
    #if PLATFORM_LINUX
    #ifdef SO_BUSY_POLL
    if (g_config.busy_poll_us > 0) {
        int us = g_config.busy_poll_us;
        (void)setsockopt(new_fd, SOL_SOCKET, SO_BUSY_POLL, &us, sizeof(us));
    }
    #endif
    #endif
    TCPConnection* conn = allocate_connection(new_fd);
    active_connections_.fetch_add(1);  // Atomic increment
    
    return conn;
}

void TCPServer::close_connection(TCPConnection* conn) {
    if (!conn || !conn->is_active()) return;
    
    // Unregister from event loop
    if (event_loop_) {
        event_loop_->unregister_connection(conn);
    }
    
    conn->close();
    active_connections_.fetch_sub(1);  // Atomic decrement
    free_connection(conn);
    update_accept_backpressure();
}

void TCPServer::update_accept_backpressure() {
    if (!event_loop_) return;
    const int maxc = (int)g_config.max_connections;
    const int cur = active_connections_.load();
    const int resume_threshold = std::max(0, maxc - 64); // hysteresis

    if (!accept_paused_ && cur >= maxc) {
        accept_paused_ = true;
        (void)event_loop_->set_accepting(false);
    } else if (accept_paused_ && cur <= resume_threshold) {
        accept_paused_ = false;
        (void)event_loop_->set_accepting(true);
    }
}

void TCPServer::submit_read(TCPConnection* conn) {
    // Backpressure: do not read new request data while a write is in-flight.
    if (!conn || !conn->is_active() || conn->is_reading() || conn->is_writing() || !event_loop_) return;
    
    // Use event loop abstraction
    event_loop_->register_read(conn);
}

void TCPServer::submit_write_static(TCPConnection* conn, const void* data, size_t len) {
    if (!conn || !conn->is_active()) return;
    
    // If already writing, this is a continue operation
    if (conn->is_writing()) {
        submit_write_continue(conn);
        return;
    }
    
    // Zero-copy: Use static data directly without copying to write_buffer
    // For now, we still need to handle this specially
    // TODO: Enhance EventLoop to support zero-copy writes
    if (len <= conn->buffer_capacity()) {
        // Small enough - copy to buffer and use normal path
        // Use __builtin_memcpy for better compiler optimization
        __builtin_memcpy(conn->write_buffer(), data, len);
        conn->set_write_pos(0);
        conn->set_write_len(len);
        conn->set_writing(true);
        if (event_loop_) {
            event_loop_->register_write(conn);
        }
    } else {
        // Too large - fallback to normal write
        submit_write(conn, data, len);
    }
}

void TCPServer::submit_write(TCPConnection* conn, const void* data, size_t len) {
    if (!conn || !conn->is_active() || !event_loop_) return;
    
    // If already writing, this is a continue operation
    if (conn->is_writing()) {
        submit_write_continue(conn);
        return;
    }
    
    // Copy data to write buffer
    // Use __builtin_memcpy for better compiler optimization
    if (len > conn->buffer_capacity()) len = conn->buffer_capacity();
    __builtin_memcpy(conn->write_buffer(), data, len);
    conn->set_write_pos(0);
    conn->set_write_len(len);
    conn->set_writing(true);
    
    // Use event loop abstraction
    event_loop_->register_write(conn);
}

void TCPServer::submit_write_continue(TCPConnection* conn) {
    if (!conn || !conn->is_active() || !conn->is_writing() || !event_loop_) return;
    
    // Use event loop abstraction
    event_loop_->register_write(conn);
}

void TCPServer::run() {
    event_loop();
}

TCPConnection* TCPServer::allocate_connection(int fd) {
    std::lock_guard<std::mutex> lock(connection_pool_mutex_);
    
    TCPConnection* conn = nullptr;
    
    if (!free_connections_.empty()) {
        conn = free_connections_.back();
        free_connections_.pop_back();
        // Reuse connection - reset it efficiently
        conn->fd_ = fd;
        conn->read_pos_ = 0;
        conn->write_pos_ = 0;
        conn->write_len_ = 0;
        conn->reading_ = false;
        conn->writing_ = false;
    } else {
        // Allocate new connection
        connections_.emplace_back(std::make_unique<TCPConnection>(fd, g_config.buffer_size));
        conn = connections_.back().get();
    }
    
    return conn;
}

void TCPServer::preallocate_connections() {
    std::lock_guard<std::mutex> lock(connection_pool_mutex_);
    
    // Pre-allocate connections to avoid allocation in hot path
    size_t pool_size = g_config.initial_pool_size;
    for (size_t i = 0; i < pool_size; i++) {
        connections_.emplace_back(std::make_unique<TCPConnection>(-1, g_config.buffer_size));
        free_connections_.push_back(connections_.back().get());
    }
}

void TCPServer::free_connection(TCPConnection* conn) {
    if (conn) {
        // Fast reset - only reset what's necessary (avoid clearing buffers)
        conn->fd_ = -1;
        conn->read_pos_ = 0;
        conn->write_pos_ = 0;
        conn->write_len_ = 0;
        conn->reading_ = false;
        conn->writing_ = false;
        
        // Don't clear buffers - they'll be overwritten on next use
        {
            std::lock_guard<std::mutex> lock(connection_pool_mutex_);
            free_connections_.push_back(conn);
        }
    }
}

int TCPServer::create_listener(int port) {
    // SYSCALL: socket() - kernel creates socket file descriptor
    // Kernel allocates socket structure, TCP state machine, buffers
    int fd = socket(AF_INET, SOCK_STREAM, 0);
    
    if (fd < 0) {
        perror("socket");
        return -1;
    }
    
    // Set non-blocking (required for event loop)
    if (!set_nonblocking(fd)) {
        perror("fcntl");
        ::close(fd);
        return -1;
    }
    
    // Set TCP_NODELAY for low latency (disable Nagle's algorithm)
    int flag = 1;
    setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &flag, sizeof(flag));
    
    int opt = 1;
    setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));
    #ifdef SO_REUSEPORT
    setsockopt(fd, SOL_SOCKET, SO_REUSEPORT, &opt, sizeof(opt));
    #endif
    
    struct sockaddr_in addr;
    memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = INADDR_ANY;
    addr.sin_port = htons(port);
    
    // SYSCALL: bind() - kernel binds socket to address/port
    // Kernel sets up network interface binding
    if (bind(fd, (struct sockaddr*)&addr, sizeof(addr)) < 0) {
        perror("bind");
        ::close(fd);
        return -1;
    }
    
    // SYSCALL: listen() - kernel starts listening for connections
    // Kernel manages accept queue
    int backlog = g_config.listen_backlog;
    if (listen(fd, backlog) < 0) {
        perror("listen");
        ::close(fd);
        return -1;
    }
    
    return fd;
}

bool TCPServer::set_nonblocking(int fd) {
    // SYSCALL: fcntl() - kernel sets file descriptor flags
    // O_NONBLOCK makes I/O operations non-blocking
    int flags = fcntl(fd, F_GETFL, 0);
    if (flags < 0) return false;
    return fcntl(fd, F_SETFL, flags | O_NONBLOCK) == 0;
}

// Platform-specific code moved to event_loop_uring.cpp and event_loop_kqueue.cpp

void TCPServer::handle_event(const EventData& event) {
    TCPConnection* conn = event.conn;
    
    switch (event.type) {
        case EventType::ACCEPT: {
            // Accept completion - create connection with new fd
            if (event.result >= 0) {
                // Check connection limit
                if (active_connections_.load() >= (int)g_config.max_connections) {
                    // At capacity - close immediately
                    ::close(event.result);
                    update_accept_backpressure();
                    return;
                }
                
                TCPConnection* new_conn = allocate_connection(event.result);
                if (new_conn) {
                    set_nonblocking(new_conn->fd());
                    // Set TCP_NODELAY for low latency
                    int flag = 1;
                    setsockopt(new_conn->fd(), IPPROTO_TCP, TCP_NODELAY, &flag, sizeof(flag));

                    // Optional busy-polling (Linux only)
                    #if PLATFORM_LINUX
                    #ifdef SO_BUSY_POLL
                    if (g_config.busy_poll_us > 0) {
                        int us = g_config.busy_poll_us;
                        (void)setsockopt(new_conn->fd(), SOL_SOCKET, SO_BUSY_POLL, &us, sizeof(us));
                    }
                    #endif
                    #endif
                    active_connections_.fetch_add(1);
                    update_accept_backpressure();
                    if (accept_callback_) {
                        accept_callback_(this, new_conn);
                    }
                } else {
                    // Failed to allocate - close fd
                    ::close(event.result);
                }
            }
            break;
        }
        case EventType::READ: {
            if (conn && event.result > 0) {
                // IMPORTANT: io_uring/kqueue read() appends at read_pos().
                // The HTTP layer expects read_pos() to reflect total bytes available.
                conn->advance_read_pos((size_t)event.result);
                // Inline common case: read complete, process immediately
                if (read_callback_) {
                    read_callback_(this, conn, event.result);
                }
            } else if (conn && event.result == 0) {
                // EOF - close immediately
                close_connection(conn);
            } else if (conn) {
                // Error - close immediately
                close_connection(conn);
            }
            break;
        }
        case EventType::WRITE: {
            // io_uring updates write_pos() in IOUringEventLoop::handle_completion().
            // When the write is fully done, we must reset state and schedule next read
            // to support HTTP keep-alive.
            if (conn && event.result > 0) {
                if (conn->write_pos() < conn->write_len()) {
                    // Partial write - continue
                    if (write_callback_) {
                        write_callback_(this, conn, event.result);
                    }
                } else {
                    // Write complete - reset for next request and re-arm read
                    // Notify application (allows protocols like Redis QUIT to close after reply).
                    if (write_callback_) {
                        write_callback_(this, conn, event.result);
                    }
                    // Callback may have closed/unregistered the connection.
                    if (!conn->is_active()) {
                        return;
                    }
                    conn->set_writing(false);
                    conn->set_write_pos(0);
                    conn->set_write_len(0);
                    conn->set_read_pos(0);
                    submit_read(conn);
                }
            } else if (conn) {
                // Error or zero bytes
                conn->set_writing(false);
                if (error_callback_ && event.result < 0) {
                    error_callback_(this, conn, event.error_code);
                }
                close_connection(conn);
            }
            break;
        }
        case EventType::ERROR: {
            if (conn) {
                if (error_callback_) {
                    error_callback_(this, conn, event.error_code);
                }
            }
            break;
        }
    }
}

void TCPServer::event_loop() {
    if (!event_loop_) return;
    
    // Run event loop (blocking until shutdown)
    // All platform-specific code is now in event_loop_uring.cpp / event_loop_kqueue.cpp
    event_loop_->run();
}
