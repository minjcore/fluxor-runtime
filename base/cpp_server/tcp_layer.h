// TCP Layer 4 - Connection Management and I/O
// Handles: socket operations, connection lifecycle, event loop (io_uring/kqueue)

#pragma once

#include <cstddef>
#include <cstdint>
#include <memory>
#include <vector>
#include <map>
#include <functional>
#include <atomic>
#include <mutex>
#include "event_loop.h"
#include "platform.h"

// Platform-specific includes
#if PLATFORM_MACOS
#include <sys/event.h>
#elif PLATFORM_LINUX
#include <liburing.h>
#endif

// Forward declarations
class TCPConnection;
class TCPServer;

// TCP Connection (Layer 4)
// Cache-aligned for optimal cache line usage (64 bytes per cache line)
class __attribute__((aligned(64))) TCPConnection {
public:
    static constexpr size_t DEFAULT_BUFFER_SIZE = 8192;

    TCPConnection(int fd, size_t buffer_capacity);
    ~TCPConnection();
    
    int fd() const { return fd_; }
    bool is_active() const { return fd_ >= 0; }
    
    // Buffer management
    char* read_buffer() { return read_buffer_.get(); }
    char* write_buffer() { return write_buffer_.get(); }
    size_t buffer_capacity() const { return buffer_capacity_; }
    size_t read_pos() const { return read_pos_; }
    size_t write_pos() const { return write_pos_; }
    size_t write_len() const { return write_len_; }
    
    void set_read_pos(size_t pos) { read_pos_ = pos; }
    void set_write_pos(size_t pos) { write_pos_ = pos; }
    void set_write_len(size_t len) { write_len_ = len; }
    void advance_read_pos(size_t n) { read_pos_ += n; }
    void advance_write_pos(size_t n) { write_pos_ += n; }
    void reset() { 
        read_pos_ = 0; 
        write_pos_ = 0; 
        write_len_ = 0;
        reading_ = false;
        writing_ = false;
    }
    
    // I/O state
    bool is_reading() const { return reading_; }
    bool is_writing() const { return writing_; }
    void set_reading(bool val) { reading_ = val; }
    void set_writing(bool val) { writing_ = val; }
    
    // Close connection
    void close();
    
private:
    int fd_;
    size_t buffer_capacity_;
    std::unique_ptr<char[]> read_buffer_;
    std::unique_ptr<char[]> write_buffer_;
    size_t read_pos_;
    size_t write_pos_;
    size_t write_len_;  // Total length of data to write
    bool reading_;
    bool writing_;
    
    friend class TCPServer;
};

// TCP Server (Layer 4) - Manages connections and event loop
class TCPServer {
public:
    static constexpr int MAX_CONNECTIONS = 10000;
    
    TCPServer(int thread_id);
    ~TCPServer();
    
    // Server lifecycle
    bool setup(int port);
    void run();
    void shutdown();
    
    // Connection management
    TCPConnection* accept_connection();
    void close_connection(TCPConnection* conn);
    int active_connections() const { return active_connections_; }
    
    // I/O operations
    void submit_read(TCPConnection* conn);
    void submit_write(TCPConnection* conn, const void* data, size_t len);
    void submit_write_static(TCPConnection* conn, const void* data, size_t len);  // Zero-copy for static data
    void submit_write_continue(TCPConnection* conn);  // Continue existing write
    
    // Event callbacks (called by event loop)
    using ReadCallback = std::function<void(TCPServer* server, TCPConnection* conn, ssize_t bytes_read)>;
    using WriteCallback = std::function<void(TCPServer* server, TCPConnection* conn, ssize_t bytes_written)>;
    using AcceptCallback = std::function<void(TCPServer* server, TCPConnection* conn)>;
    using ErrorCallback = std::function<void(TCPServer* server, TCPConnection* conn, int error)>;
    
    void set_read_callback(ReadCallback cb) { read_callback_ = cb; }
    void set_write_callback(WriteCallback cb) { write_callback_ = cb; }
    void set_accept_callback(AcceptCallback cb) { accept_callback_ = cb; }
    void set_error_callback(ErrorCallback cb) { error_callback_ = cb; }
    
    int thread_id() const { return thread_id_; }
    
    // Global running flag (accessible for signal handler)
    static std::atomic<bool> global_running_;
    
private:
    // Platform-independent event loop
    std::unique_ptr<EventLoop> event_loop_;
    
    // Bridge EventLoop callbacks to TCPServer callbacks
    void handle_event(const EventData& event);
    
    // Connection management
    TCPConnection* allocate_connection(int fd);
    void free_connection(TCPConnection* conn);
    
    // Socket operations
    int create_listener(int port);
    bool set_nonblocking(int fd);
    
    // Event loop
    void event_loop();

    // Listener accept backpressure (pause accepts when at capacity)
    void update_accept_backpressure();
    bool accept_paused_ = false;
    
    // Callbacks
    ReadCallback read_callback_ = nullptr;
    WriteCallback write_callback_ = nullptr;
    AcceptCallback accept_callback_ = nullptr;
    ErrorCallback error_callback_ = nullptr;
    
    // State
    int thread_id_;
    int listener_fd_;
    std::vector<std::unique_ptr<TCPConnection>> connections_;
    std::vector<TCPConnection*> free_connections_;
    mutable std::mutex connection_pool_mutex_;  // Protects connections_ and free_connections_
    std::atomic<int> active_connections_;  // Thread-safe connection counter
    bool running_;
    
    // Connection pool optimization - pre-allocate connections
    // Size comes from g_config.initial_pool_size
    void preallocate_connections();
};
