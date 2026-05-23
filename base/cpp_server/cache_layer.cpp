// Cache Layer Implementation

#include "cache_layer.h"
#include <algorithm>

CacheLayer::CacheLayer(size_t max_size) 
    : max_size_(max_size), current_size_(0), total_hits_(0), total_misses_(0) {
}

CacheLayer::~CacheLayer() {
    clear();
}

std::string CacheLayer::get(const std::string_view& key) {
    // Use shared_lock for read, then upgrade to unique_lock for LRU update
    std::shared_lock<std::shared_mutex> shared_lock(mutex_);
    
    std::string key_str(key);
    auto it = cache_.find(key_str);
    
    if (it == cache_.end()) {
        total_misses_.fetch_add(1);
        return std::string();  // Not found
    }
    
    // Cache hit - need to update LRU, so upgrade to unique lock
    shared_lock.unlock();
    std::unique_lock<std::shared_mutex> unique_lock(mutex_);
    
    // Re-check after lock upgrade (another thread might have evicted)
    it = cache_.find(key_str);
    if (it == cache_.end()) {
        total_misses_.fetch_add(1);
        return std::string();
    }
    
    CacheEntry& entry = it->second.first;
    entry.hits++;
    total_hits_.fetch_add(1);
    
    // Update LRU order: move to front
    lru_list_.erase(it->second.second);
    lru_list_.push_front(key_str);
    it->second.second = lru_list_.begin();
    
    return entry.response;
}

std::string_view CacheLayer::get_view(const std::string_view& key) const {
    std::shared_lock<std::shared_mutex> lock(mutex_);
    
    // Convert key to string for lookup (required by unordered_map)
    std::string key_str(key);
    auto it = cache_.find(key_str);
    
    if (it == cache_.end()) {
        // Can't update misses in const method - skip statistics for zero-copy path
        return std::string_view();  // Not found - empty string_view
    }
    
    // Cache hit - return view into cached response (zero-copy)
    // Note: Caller must ensure cache entry is not evicted during use
    // We don't update hits/misses or LRU here to keep it fast and truly read-only
    // Statistics are only updated in non-const get() method
    const CacheEntry& entry = it->second.first;
    
    return std::string_view(entry.response.data(), entry.response.size());
}

void CacheLayer::evict_if_needed() {
    evict(max_size_);
}

void CacheLayer::put(const std::string_view& key, const char* value, size_t len) {
    if (len == 0) return;
    
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    std::string key_str(key);
    std::string response(value, len);
    
    // Check if key already exists
    auto it = cache_.find(key_str);
    if (it != cache_.end()) {
        // Update existing entry
        size_t old_size = it->second.first.size;
        it->second.first.response = std::move(response);
        it->second.first.size = len;
        
        // Update LRU order
        lru_list_.erase(it->second.second);
        lru_list_.push_front(key_str);
        it->second.second = lru_list_.begin();
        
        // Update size
        current_size_.fetch_sub(old_size);
        current_size_.fetch_add(len);
        
        // Evict if necessary
        if (current_size_.load() > max_size_) {
            evict(max_size_);
        }
        return;
    }
    
    // New entry - check if we need to evict
    while (current_size_.load() + len > max_size_ && !cache_.empty()) {
        evict_lru();
    }
    
    // Add new entry
    lru_list_.push_front(key_str);
    cache_.emplace(key_str, std::make_pair(
        CacheEntry(std::move(response), len),
        lru_list_.begin()
    ));
    
    current_size_.fetch_add(len);
}

void CacheLayer::evict(size_t target_size) {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    while (current_size_.load() > target_size && !cache_.empty()) {
        evict_lru();
    }
}

void CacheLayer::evict_lru() {
    if (lru_list_.empty()) return;
    
    // Remove least recently used (back of list)
    std::string lru_key = lru_list_.back();
    lru_list_.pop_back();
    
    auto it = cache_.find(lru_key);
    if (it != cache_.end()) {
        size_t entry_size = it->second.first.size;
        cache_.erase(it);
        current_size_.fetch_sub(entry_size);
    }
}

void CacheLayer::clear() {
    std::unique_lock<std::shared_mutex> lock(mutex_);
    
    cache_.clear();
    lru_list_.clear();
    current_size_.store(0);
}
