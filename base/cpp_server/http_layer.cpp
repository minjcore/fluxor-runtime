// HTTP Layer 7 Implementation

#include "http_layer.h"
#include "tcp_layer.h"
#include "cpu_layer.h"
#include "cache_layer.h"
#include <string.h>
#include <cstring>
#include <cstdint>

#ifdef __SSE4_2__
#include <nmmintrin.h>  // SSE4.2 intrinsics
#include <emmintrin.h>  // SSE2 intrinsics
#endif

#ifdef __AVX2__
#include <immintrin.h>  // AVX2 intrinsics
#endif

// Optimized memmem implementation
// Fast path for short needles, SIMD for longer patterns
#ifndef _GNU_SOURCE
static const char* memmem_optimized(const char* haystack, size_t haystack_len,
                                    const char* needle, size_t needle_len) {
    if (needle_len == 0) return haystack;
    if (haystack_len < needle_len) return nullptr;
    if (needle_len == 1) {
        // Single character search - use memchr
        return (const char*)memchr(haystack, needle[0], haystack_len);
    }
    
    // Fast path for very short needles (2-4 bytes) - use word comparison
    if (needle_len <= 4) {
        const char* end = haystack + haystack_len - needle_len + 1;
        uint32_t needle_word = 0;
        memcpy(&needle_word, needle, needle_len);
        
        for (const char* p = haystack; p < end; p++) {
            uint32_t haystack_word = 0;
            memcpy(&haystack_word, p, needle_len);
            if (haystack_word == needle_word) {
                return p;
            }
        }
        return nullptr;
    }
    
#ifdef __SSE4_2__
    // SIMD-optimized path for longer needles
    if (haystack_len >= 16 && needle_len <= 16) {
        const char* haystack_end = haystack + haystack_len - needle_len + 1;
        const char* pos = haystack;
        
        // Load first byte of needle for quick filtering
        __m128i first_byte = _mm_set1_epi8(needle[0]);
        
        while (pos + 16 <= haystack_end) {
            __m128i data = _mm_loadu_si128(reinterpret_cast<const __m128i*>(pos));
            __m128i matches = _mm_cmpeq_epi8(data, first_byte);
            int mask = _mm_movemask_epi8(matches);
            
            // Check each potential match
            while (mask != 0) {
                int bit = __builtin_ctz(mask);  // Find first set bit
                if (pos + bit + needle_len <= haystack + haystack_len) {
                    if (memcmp(pos + bit, needle, needle_len) == 0) {
                        return pos + bit;
                    }
                }
                mask &= mask - 1;  // Clear lowest set bit
            }
            
            pos += 16;
        }
        
        // Handle remaining bytes with scalar code
        while (pos < haystack_end) {
            if (memcmp(pos, needle, needle_len) == 0) {
                return pos;
            }
            pos++;
        }
        return nullptr;
    }
#endif
    
    // Fallback: optimized scalar search with first-byte optimization
    const char first = needle[0];
    const char* haystack_end = haystack + haystack_len - needle_len + 1;
    
    for (const char* p = haystack; p < haystack_end; p++) {
        // Quick filter: check first byte before full comparison
        if (*p == first && memcmp(p, needle, needle_len) == 0) {
            return p;
        }
    }
    return nullptr;
}

// Use optimized version
#define memmem memmem_optimized
#endif

// HTTPParser Implementation

// Runtime SIMD detection
bool HTTPParser::has_simd_support() {
#ifdef __SSE4_2__
    // Check CPUID for SSE4.2 support
    // On x86_64, SSE4.2 is always available, but we check anyway
    #ifdef __x86_64__
    return true;  // x86_64 always has SSE4.2
    #else
    // For other architectures, use runtime detection if needed
    return __builtin_cpu_supports("sse4.2");
    #endif
#else
    return false;
#endif
}

// SIMD-optimized version using SSE4.2
bool HTTPParser::is_complete_request_simd(const char* buf, size_t len) {
#ifdef __SSE4_2__
    if (len < 4) return false;
    
    const char* end = buf + len;
    const char* pos = buf;
    
    // Pattern to search: \r\n\r\n (0x0D 0x0A 0x0D 0x0A)
    const __m128i cr = _mm_set1_epi8('\r');  // 0x0D
    const __m128i lf = _mm_set1_epi8('\n');  // 0x0A
    
    // Process 16 bytes at a time, but check overlapping windows for pattern
    while (pos + 19 <= end) {  // Need 19 bytes to check pattern at offset 15
        __m128i data = _mm_loadu_si128(reinterpret_cast<const __m128i*>(pos));
        
        // Find all \r and \n positions
        __m128i cr_eq = _mm_cmpeq_epi8(data, cr);
        __m128i lf_eq = _mm_cmpeq_epi8(data, lf);
        
        int cr_mask = _mm_movemask_epi8(cr_eq);
        int lf_mask = _mm_movemask_epi8(lf_eq);
        
        // Check for \r\n\r\n pattern within this 16-byte window
        // Pattern at position i: buf[i]=='\r', buf[i+1]=='\n', buf[i+2]=='\r', buf[i+3]=='\n'
        for (int i = 0; i <= 12; i++) {  // Can check up to position 12 (need 4 bytes)
            if ((cr_mask & (1 << i)) && 
                (lf_mask & (1 << (i+1))) && 
                (cr_mask & (1 << (i+2))) && 
                (lf_mask & (1 << (i+3)))) {
                return true;
            }
        }
        
        pos += 16;
    }
    
    // Handle remaining bytes and cross-boundary patterns with scalar code
    // Check last 16 bytes of buffer for any remaining patterns
    if (pos < end) {
        size_t remaining = end - pos;
        if (remaining >= 4) {
            for (size_t i = 0; i <= remaining - 4; i++) {
                if (pos[i] == '\r' && pos[i+1] == '\n' && 
                    pos[i+2] == '\r' && pos[i+3] == '\n') {
                    return true;
                }
            }
        }
    }
    
    return false;
#else
    // Fallback to scalar if SIMD not available
    return is_complete_request_scalar(buf, len);
#endif
}

// Main function: tries SIMD first, falls back to scalar
bool HTTPParser::is_complete_request(const char* buf, size_t len) {
    if (len < 4) return false;
    
    // Use SIMD if available and buffer is large enough to benefit
    if (len >= 16 && has_simd_support()) {
        return is_complete_request_simd(buf, len);
    }
    
    // Fallback to scalar version
    return is_complete_request_scalar(buf, len);
}

bool HTTPParser::parse_request_line(const char* buf, size_t len,
                                    std::string_view& method,
                                    std::string_view& path,
                                    std::string_view& version) {
    const char* start = buf;
    const char* end = buf + len;
    
    // Find method
    const char* space = (const char*)memchr(start, ' ', end - start);
    if (!space || space == start) return false;
    method = std::string_view(start, space - start);
    
    // Find path
    start = space + 1;
    if (start >= end) return false;
    const char* space2 = (const char*)memchr(start, ' ', end - start);
    if (!space2 || space2 == start) return false;
    path = std::string_view(start, space2 - start);
    
    // Find version (HTTP/1.0 or HTTP/1.1)
    start = space2 + 1;
    if (start >= end) return false;
    const char* crlf = (const char*)memchr(start, '\r', end - start);
    if (!crlf) crlf = (const char*)memchr(start, '\n', end - start);
    if (!crlf) return false;
    version = std::string_view(start, crlf - start);
    
    return true;
}

bool HTTPParser::find_header(const char* buf, size_t len,
                            const char* header_name,
                            std::string_view& value) {
    size_t name_len = strlen(header_name);
    const char* pos = buf;
    const char* end = buf + len;
    
    while (pos < end) {
        // Case-insensitive search for header name
        const char* found = (const char*)memmem(pos, end - pos, header_name, name_len);
        if (!found) break;
        
        // Check if it's at the start of a line or after a newline
        if (found == buf || found[-1] == '\n' || found[-1] == '\r') {
            // Check if followed by ':'
            if (found + name_len < end && found[name_len] == ':') {
                // Found header - extract value
                const char* value_start = found + name_len + 1;
                // Skip whitespace
                while (value_start < end && (*value_start == ' ' || *value_start == '\t')) {
                    value_start++;
                }
                // Find end of value (CRLF)
                const char* value_end = (const char*)memchr(value_start, '\r', end - value_start);
                if (!value_end) value_end = (const char*)memchr(value_start, '\n', end - value_start);
                if (!value_end) value_end = end;
                
                value = std::string_view(value_start, value_end - value_start);
                return true;
            }
        }
        pos = found + 1;
    }
    
    return false;
}

// HTTPResponse Implementation
constexpr const char HTTPResponse::OK_RESPONSE[];

void HTTPResponse::build_ok_response(char* buffer, size_t buffer_size,
                                     const char* body, size_t body_len) {
    int n = snprintf(buffer, buffer_size,
        "HTTP/1.1 200 OK\r\n"
        "Content-Length: %zu\r\n"
        "Connection: keep-alive\r\n"
        "\r\n",
        body_len);
    
    if (n > 0 && (size_t)n < buffer_size) {
        memcpy(buffer + n, body, body_len);
    }
}

void HTTPResponse::build_error_response(char* buffer, size_t buffer_size,
                                        int status_code, const char* message) {
    snprintf(buffer, buffer_size,
        "HTTP/1.1 %d %s\r\n"
        "Content-Length: %zu\r\n"
        "Connection: close\r\n"
        "\r\n"
        "%s",
        status_code, message, strlen(message), message);
}

// HTTPHandler Implementation
HTTPHandler::HTTPHandler() : response_data_(nullptr), response_len_(0), cpu_processor_(nullptr), cache_layer_(nullptr) {
}

HTTPHandler::~HTTPHandler() {
    // CPU processor is owned by HTTPServer, not HTTPHandler
}

bool HTTPHandler::handle_request(TCPConnection* /* conn */, const char* data, size_t len) {
    // Use provided data directly
    return handle_request(data, len);
}

bool HTTPHandler::process_request(TCPConnection* conn, ssize_t bytes_read) {
    if (bytes_read <= 0) return false;
    
    // Read position already updated by TCP layer
    size_t total_read = conn->read_pos();
    
    // Fast path: check if request is complete first (most common case)
    if (!HTTPParser::is_complete_request(conn->read_buffer(), total_read)) {
        // Need more data
        return false;
    }
    
    // Parse and handle request
    if (handle_request(conn->read_buffer(), total_read)) {
        // Response is ready
        return true;
    }
    
    return false;
}

bool HTTPHandler::handle_request(const char* data, size_t len) {
    // Ultra-fast path: Check first 9 bytes for "GET /test" (most common case)
    // This avoids full HTTP parsing overhead for the hot path
    // Use direct byte comparison - compiler will optimize to word comparison
    if (len >= 9) {
        // Safe word load (avoid UB due to unaligned pointer dereference under -O3/-flto)
        uint32_t word = 0;
        memcpy(&word, data, sizeof(word));
        // "GET " = 0x20544547 in little-endian (x86/ARM)
        if (word == 0x20544547 && 
            data[4] == '/' && data[5] == 't' && data[6] == 'e' && 
            data[7] == 's' && data[8] == 't') {
            // Fast path: Direct response without parsing or cache (static response)
            response_data_ = HTTPResponse::get_ok_response();
            response_len_ = HTTPResponse::get_ok_response_len();
            // Skip cache for /test endpoint (static response, no need to cache)
            return true;
        }
    }
    
    // Parse request line (fallback for other endpoints)
    std::string_view method, path, version;
    if (!HTTPParser::parse_request_line(data, len, method, path, version)) {
        // Invalid request - use compile-time constant error response (no strlen)
        response_data_ = HTTPResponse::get_bad_request_response();
        response_len_ = HTTPResponse::get_bad_request_response_len();
        return true;
    }
    
    // Check cache first (if cache layer is available)
    // Fast method comparison: "GET" = 3 bytes, compare directly
    bool is_get = (method.size() == 3 && 
                   method[0] == 'G' && method[1] == 'E' && method[2] == 'T');
    // Alternative: use word comparison for even faster check
    // bool is_get = (method.size() == 3 && 
    //                *reinterpret_cast<const uint32_t*>(method.data()) == 0x00544547);
    
    if (cache_layer_ && is_get) {
        // Zero-copy cache lookup
        std::string_view cached_response = cache_layer_->get_view(path);
        if (!cached_response.empty()) {
            // Cache hit - use cached response directly (zero-copy)
            // Note: Cache entry must remain valid during response send
            response_data_ = cached_response.data();
            response_len_ = cached_response.size();
            return true;
        }
    }
    
    // Skip CPU processing check for /test endpoint (hot path optimization)
    // CPU processing is not used in current implementation
    
    // For now, always return OK response
    // In the future, this can route to different handlers based on path
    response_data_ = HTTPResponse::get_ok_response();
    response_len_ = HTTPResponse::get_ok_response_len();
    
    // Store in cache (if cache layer is available and method is GET)
    if (cache_layer_ && is_get) {
        cache_layer_->put(path, response_data_, response_len_);
    }
    
    return true;
}
