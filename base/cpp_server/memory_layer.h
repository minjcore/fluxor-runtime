// Memory Layer - Efficient Memory Management
// Handles: Memory pools, buffer management, object pooling, caching
// Reduces allocation overhead and improves cache locality

#pragma once

#include <cstddef>
#include <cstdint>
#include <vector>
#include <memory>
#include <mutex>
#include <atomic>
#include <set>

// Memory pool for fixed-size allocations
template<size_t BlockSize, size_t PoolSize = 1024>
class MemoryPool {
public:
    MemoryPool();
    ~MemoryPool();
    
    // Allocate a block
    void* allocate();
    
    // Deallocate a block
    void deallocate(void* ptr);
    
    // Get pool statistics
    size_t allocated_count() const { return allocated_count_.load(); }
    size_t free_count() const;
    
private:
    struct Block {
        alignas(BlockSize) char data[BlockSize];
        Block* next;
    };
    
    std::vector<Block> blocks_;
    Block* free_list_;
    mutable std::mutex mutex_;
    std::atomic<size_t> allocated_count_;
    size_t next_free_index_;
};

// Buffer pool for variable-size buffers
class BufferPool {
public:
    BufferPool(size_t buffer_size, size_t pool_size = 100);
    ~BufferPool();
    
    // Get a buffer from pool
    char* acquire();
    
    // Return buffer to pool
    void release(char* buffer);
    
    // Get buffer size
    size_t buffer_size() const { return buffer_size_; }
    
    // Get pool statistics
    size_t available_count() const;
    size_t total_count() const;
    
    #ifdef DEBUG
    // Memory leak tracking (only in DEBUG builds)
    size_t leaked_buffers() const { return leaked_buffers_.load(); }
    #endif
    
private:
    size_t buffer_size_;
    std::vector<std::unique_ptr<char[]>> buffers_;
    std::vector<char*> free_buffers_;
    mutable std::mutex mutex_;
    
    #ifdef DEBUG
    // Track dynamically allocated buffers (from fallback new char[])
    std::set<char*> dynamic_buffers_;
    std::atomic<size_t> leaked_buffers_{0};
    #endif
};

// Memory manager - central memory management
class MemoryManager {
public:
    static MemoryManager& instance();
    
    // Get buffer pool for a specific size
    BufferPool* get_buffer_pool(size_t size);
    
    // Allocate aligned memory
    void* allocate_aligned(size_t size, size_t alignment = 64);
    
    // Deallocate aligned memory
    void deallocate_aligned(void* ptr);
    
    // Get memory statistics
    size_t total_allocated() const { return total_allocated_.load(); }
    size_t total_deallocated() const { return total_deallocated_.load(); }
    
private:
    MemoryManager();
    ~MemoryManager();
    
    // Buffer pools for common sizes
    static constexpr size_t SMALL_BUFFER = 1024;
    static constexpr size_t MEDIUM_BUFFER = 4096;
    static constexpr size_t LARGE_BUFFER = 8192;
    
    std::unique_ptr<BufferPool> small_pool_;
    std::unique_ptr<BufferPool> medium_pool_;
    std::unique_ptr<BufferPool> large_pool_;
    
    std::mutex mutex_;
    std::atomic<size_t> total_allocated_;
    std::atomic<size_t> total_deallocated_;
};

// RAII wrapper for buffer pool
class ScopedBuffer {
public:
    ScopedBuffer(BufferPool* pool, char* buffer);
    ~ScopedBuffer();
    
    char* get() { return buffer_; }
    const char* get() const { return buffer_; }
    
    // Non-copyable
    ScopedBuffer(const ScopedBuffer&) = delete;
    ScopedBuffer& operator=(const ScopedBuffer&) = delete;
    
    // Movable
    ScopedBuffer(ScopedBuffer&& other) noexcept;
    ScopedBuffer& operator=(ScopedBuffer&& other) noexcept;
    
private:
    BufferPool* pool_;
    char* buffer_;
};
