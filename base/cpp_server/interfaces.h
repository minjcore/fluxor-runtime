// Interface Definitions for Dependency Injection and Testing
// These interfaces allow for easy mocking and swapping of implementations

#pragma once

#include <string>
#include <string_view>
#include <cstddef>

// Forward declarations
class TCPConnection;

// HTTP Handler Interface
class IHTTPHandler {
public:
    virtual ~IHTTPHandler() = default;
    
    // Handle HTTP request
    // Returns true if request was handled successfully
    virtual bool handle_request(TCPConnection* conn, const char* data, size_t len) = 0;
    
    // Get response data and length
    virtual const char* response_data() const = 0;
    virtual size_t response_len() const = 0;
    
    // Check if response is static (for zero-copy optimization)
    virtual bool is_static_response() const = 0;
};

// Cache Layer Interface
class ICacheLayer {
public:
    virtual ~ICacheLayer() = default;
    
    // Get cached value by key
    virtual std::string get(const std::string_view& key) = 0;
    
    // Put value into cache
    virtual void put(const std::string_view& key, const char* value, size_t len) = 0;
    
    // Evict entries if cache is full
    virtual void evict_if_needed() = 0;
    
    // Cache statistics
    virtual size_t current_size() const = 0;
    virtual size_t max_size() const = 0;
    virtual size_t hits() const = 0;
    virtual size_t misses() const = 0;
};

// CPU Processor Interface
class ICPUProcessor {
public:
    virtual ~ICPUProcessor() = default;
    
    // Process CPU-bound task
    // Returns result as string
    virtual std::string process(const std::string& input) = 0;
    
    // Check if processor is available
    virtual bool is_available() const = 0;
};
