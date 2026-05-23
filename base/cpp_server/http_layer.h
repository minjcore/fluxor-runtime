// HTTP Layer 7 - Request/Response Handling
// Handles: HTTP parsing, request validation, response generation

#pragma once

#include <cstddef>
#include <cstdint>
#include <string_view>
#include "interfaces.h"

// Forward declaration
class TCPConnection;

// HTTP Request Parser (Layer 7)
class HTTPParser {
public:
    // Check if request is complete (RFC 7230: headers terminated by \r\n\r\n)
    // Uses SIMD if available, falls back to scalar code
    static bool is_complete_request(const char* buf, size_t len);
    
    // SIMD-optimized version (SSE4.2)
    static bool is_complete_request_simd(const char* buf, size_t len);
    
    // Scalar fallback version
    static inline bool is_complete_request_scalar(const char* buf, size_t len) {
        if (len < 4) return false;
        
        // Optimized manual search - compiler will vectorize this
        // Unroll loop for better performance
        size_t i = 0;
        size_t limit = len - 3;
        
        // Process 4 bytes at a time with unrolled checks
        for (; i + 7 < limit; i += 4) {
            // Check 4 positions at once
            if ((buf[i] == '\r' && buf[i+1] == '\n' && buf[i+2] == '\r' && buf[i+3] == '\n') ||
                (buf[i+1] == '\r' && buf[i+2] == '\n' && buf[i+3] == '\r' && buf[i+4] == '\n') ||
                (buf[i+2] == '\r' && buf[i+3] == '\n' && buf[i+4] == '\r' && buf[i+5] == '\n') ||
                (buf[i+3] == '\r' && buf[i+4] == '\n' && buf[i+5] == '\r' && buf[i+6] == '\n')) {
                return true;
            }
        }
        
        // Check remaining bytes
        for (; i < limit; i++) {
            if (buf[i] == '\r' && buf[i+1] == '\n' && 
                buf[i+2] == '\r' && buf[i+3] == '\n') {
                return true;
            }
        }
        return false;
    }
    
    // Runtime SIMD detection
    static bool has_simd_support();
    
    // Parse request line (returns method, path, version)
    static bool parse_request_line(const char* buf, size_t len, 
                                   std::string_view& method,
                                   std::string_view& path,
                                   std::string_view& version);
    
    // Find header value
    static bool find_header(const char* buf, size_t len, 
                           const char* header_name,
                           std::string_view& value);
};

// HTTP Response Builder (Layer 7)
class HTTPResponse {
public:
    // Pre-built responses (zero-allocation)
    static constexpr const char OK_RESPONSE[] = 
        "HTTP/1.1 200 OK\r\n"
        "Content-Length: 2\r\n"
        "Connection: keep-alive\r\n"
        "\r\n"
        "OK";
    
    static constexpr size_t OK_RESPONSE_LEN = sizeof(OK_RESPONSE) - 1;
    
    // Error response (compile-time constant to avoid strlen)
    static constexpr const char BAD_REQUEST_RESPONSE[] = 
        "HTTP/1.1 400 Bad Request\r\n"
        "Content-Length: 11\r\n"
        "Connection: close\r\n"
        "\r\n"
        "Bad Request";
    
    static constexpr size_t BAD_REQUEST_RESPONSE_LEN = sizeof(BAD_REQUEST_RESPONSE) - 1;
    
    // Build responses
    static void build_ok_response(char* buffer, size_t buffer_size, 
                                  const char* body, size_t body_len);
    static void build_error_response(char* buffer, size_t buffer_size, 
                                    int status_code, const char* message);
    
    // Get pre-built OK response
    static const char* get_ok_response() { return OK_RESPONSE; }
    static size_t get_ok_response_len() { return OK_RESPONSE_LEN; }
    
    // Get pre-built error response
    static const char* get_bad_request_response() { return BAD_REQUEST_RESPONSE; }
    static size_t get_bad_request_response_len() { return BAD_REQUEST_RESPONSE_LEN; }
};

// Forward declaration
class CPURequestProcessor;
class CacheLayer;

// HTTP Handler (Layer 7) - Processes HTTP requests
class HTTPHandler : public IHTTPHandler {
public:
    HTTPHandler();
    ~HTTPHandler();
    
    // IHTTPHandler interface
    bool handle_request(TCPConnection* conn, const char* data, size_t len) override;
    // Inline getters for hot path
    inline const char* response_data() const override { return response_data_; }
    inline size_t response_len() const override { return response_len_; }
    inline bool is_static_response() const override { return response_data_ == HTTPResponse::get_ok_response(); }
    
    // Process incoming data and generate response (legacy method)
    // Returns: true if request is complete and response is ready
    bool process_request(TCPConnection* conn, ssize_t bytes_read);
    
    // Get response to send (zero-copy: returns pointer to static data)
    // Inline for hot path
    inline const char* get_response() const { return response_data_; }
    inline size_t get_response_len() const { return response_len_; }
    
    // Set CPU processor for CPU-bound tasks
    void set_cpu_processor(CPURequestProcessor* processor) { cpu_processor_ = processor; }
    
    // Set cache layer for response caching
    void set_cache_layer(CacheLayer* cache) { cache_layer_ = cache; }
    
private:
    // Request processing
    bool handle_request(const char* data, size_t len);
    
    // Response data (points to static buffer for zero-copy)
    const char* response_data_;
    size_t response_len_;
    
    // CPU-bound processor (optional)
    CPURequestProcessor* cpu_processor_;
    
    // Cache layer (optional)
    CacheLayer* cache_layer_;
};
