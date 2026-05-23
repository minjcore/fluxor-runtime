// Cache Layer - LRU Cache for HTTP Responses
// Handles: Response caching, eviction, thread-safe operations

#pragma once

#include <string>
#include <string_view>
#include <unordered_map>
#include <list>
#include <mutex>
#include <shared_mutex>
#include <atomic>
#include <cstddef>
#include "interfaces.h"

// LRU Cache for HTTP responses
class CacheLayer : public ICacheLayer {
public:
    // Cache entry: response data and metadata
    struct CacheEntry {
        std::string response;  // HTTP response (headers + body)
        size_t size;           // Size in bytes
        size_t hits;           // Number of cache hits
        
        CacheEntry(std::string resp, size_t sz) 
            : response(std::move(resp)), size(sz), hits(0) {}
    };
    
    CacheLayer(size_t max_size = 10 * 1024 * 1024);  // Default: 10MB
    ~CacheLayer();
    
    // ICacheLayer interface
    std::string get(const std::string_view& key) override;
    void put(const std::string_view& key, const char* value, size_t len) override;
    void evict_if_needed() override;
    size_t current_size() const override { return current_size_.load(); }
    size_t max_size() const override { return max_size_; }
    size_t hits() const override { return total_hits_.load(); }
    size_t misses() const override { return total_misses_.load(); }
    
    // Zero-copy get: returns string_view pointing to cache entry
    // WARNING: Caller must ensure cache entry is not evicted during use
    // Use shared_lock for read-only access (multiple concurrent readers)
    std::string_view get_view(const std::string_view& key) const;
    
    // Additional methods
    // Evict entries until size is below max_size
    void evict(size_t max_size);
    
    // Clear all cache entries
    void clear();
    
    // Get cache statistics
    size_t size() const { return current_size_.load(); }
    size_t entries() const { 
        std::shared_lock<std::shared_mutex> lock(mutex_);
        return cache_.size(); 
    }
    
private:
    // LRU data structures
    using KeyList = std::list<std::string>;  // LRU order (most recent at front)
    using KeyMap = std::unordered_map<std::string, 
                                      std::pair<CacheEntry, KeyList::iterator>>;
    
    KeyMap cache_;              // Key -> (Entry, iterator in LRU list)
    KeyList lru_list_;          // LRU order (front = most recent)
    mutable std::shared_mutex mutex_;  // Thread-safe access (read-write lock)
    
    size_t max_size_;                    // Maximum cache size (bytes)
    std::atomic<size_t> current_size_;   // Current cache size (bytes)
    std::atomic<size_t> total_hits_;     // Total cache hits
    std::atomic<size_t> total_misses_;   // Total cache misses
    
    // Evict least recently used entry
    void evict_lru();
};
