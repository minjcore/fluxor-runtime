// Event Loop Abstraction - Platform-independent interface
// Allows switching between io_uring (Linux) and kqueue (macOS) implementations

#pragma once

#include <cstddef>
#include <functional>
#include <memory>
#include <sys/types.h>

// Forward declarations
class TCPConnection;
class TCPServer;

// Event types
enum class EventType {
    READ,
    WRITE,
    ACCEPT,
    ERROR
};

// Event data structure
struct EventData {
    TCPConnection* conn;
    EventType type;
    ssize_t result;  // Bytes read/written, or error code
    int error_code;  // errno for errors
};

// Abstract Event Loop Interface
class EventLoop {
public:
    virtual ~EventLoop() = default;
    
    // Initialize event loop (platform-specific setup)
    virtual bool setup() = 0;
    
    // Cleanup event loop (platform-specific cleanup)
    virtual void cleanup() = 0;
    
    // Register listener socket for accept events
    virtual bool register_listener(int listener_fd) = 0;

    // Enable/disable accepting new connections (listener backpressure).
    // Default implementation: no-op (always accepting).
    virtual bool set_accepting(bool enable) { (void)enable; return true; }
    
    // Register connection for read events
    virtual bool register_read(TCPConnection* conn) = 0;
    
    // Register connection for write events
    virtual bool register_write(TCPConnection* conn) = 0;
    
    // Unregister connection (cleanup)
    virtual void unregister_connection(TCPConnection* conn) = 0;
    
    // Run event loop (blocking, until shutdown)
    virtual void run() = 0;
    
    // Shutdown event loop
    virtual void shutdown() = 0;
    
    // Callbacks (set by TCPServer)
    using EventCallback = std::function<void(TCPServer* server, const EventData& event)>;
    void set_event_callback(EventCallback cb) { event_callback_ = cb; }
    
    // Set server pointer for callbacks
    void set_server(TCPServer* server) { server_ = server; }
    
protected:
    EventCallback event_callback_;
    TCPServer* server_ = nullptr;
    bool running_ = false;
};

// Factory function to create platform-specific event loop
std::unique_ptr<EventLoop> create_event_loop();
