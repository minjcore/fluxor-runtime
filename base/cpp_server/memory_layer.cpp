// Memory Layer Implementation

#include "memory_layer.h"
#include <cstdlib>
#include <cstring>
#include <algorithm>

// MemoryPool Implementation
template<size_t BlockSize, size_t PoolSize>
MemoryPool<BlockSize, PoolSize>::MemoryPool() 
    : free_list_(nullptr), allocated_count_(0), next_free_index_(0) {
    blocks_.resize(PoolSize);
    
    // Initialize free list
    for (size_t i = 0; i < PoolSize - 1; i++) {
        blocks_[i].next = reinterpret_cast<Block*>(&blocks_[i + 1]);
    }
    blocks_[PoolSize - 1].next = nullptr;
    free_list_ = reinterpret_cast<Block*>(&blocks_[0]);
}

template<size_t BlockSize, size_t PoolSize>
MemoryPool<BlockSize, PoolSize>::~MemoryPool() {
    // All blocks should be deallocated
}

template<size_t BlockSize, size_t PoolSize>
void* MemoryPool<BlockSize, PoolSize>::allocate() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (free_list_ == nullptr) {
        return nullptr; // Pool exhausted
    }
    
    Block* block = free_list_;
    free_list_ = block->next;
    allocated_count_++;
    
    return block->data;
}

template<size_t BlockSize, size_t PoolSize>
void MemoryPool<BlockSize, PoolSize>::deallocate(void* ptr) {
    if (ptr == nullptr) return;
    
    std::lock_guard<std::mutex> lock(mutex_);
    
    // Calculate block address from data pointer
    // Since data is first member, we can cast directly
    Block* block = reinterpret_cast<Block*>(
        reinterpret_cast<char*>(ptr) - offsetof(Block, data));
    block->next = free_list_;
    free_list_ = block;
    allocated_count_--;
}

// MemoryPool free_count implementation
template<size_t BlockSize, size_t PoolSize>
size_t MemoryPool<BlockSize, PoolSize>::free_count() const {
    std::lock_guard<std::mutex> lock(const_cast<std::mutex&>(mutex_));
    size_t count = 0;
    Block* current = free_list_;
    while (current) {
        count++;
        current = current->next;
    }
    return count;
}

// Explicit template instantiations for common sizes
template class MemoryPool<64, 1024>;
template class MemoryPool<128, 1024>;
template class MemoryPool<256, 512>;

// BufferPool Implementation
BufferPool::BufferPool(size_t buffer_size, size_t pool_size)
    : buffer_size_(buffer_size) {
    buffers_.reserve(pool_size);
    free_buffers_.reserve(pool_size);
    
    // Pre-allocate buffers
    for (size_t i = 0; i < pool_size; i++) {
        buffers_.emplace_back(std::make_unique<char[]>(buffer_size));
        free_buffers_.push_back(buffers_.back().get());
    }
}

BufferPool::~BufferPool() {
    #ifdef DEBUG
    // Check for leaked dynamically allocated buffers
    if (!dynamic_buffers_.empty()) {
        leaked_buffers_.store(dynamic_buffers_.size());
        // In production, we might want to log this
        // For now, we just track it
    }
    #endif
    // Buffers will be automatically freed
}

char* BufferPool::acquire() {
    std::lock_guard<std::mutex> lock(mutex_);
    
    if (free_buffers_.empty()) {
        // Pool exhausted - allocate new buffer (will be freed on release)
        char* buffer = new char[buffer_size_];
        #ifdef DEBUG
        dynamic_buffers_.insert(buffer);
        #endif
        return buffer;
    }
    
    char* buffer = free_buffers_.back();
    free_buffers_.pop_back();
    return buffer;
}

void BufferPool::release(char* buffer) {
    if (buffer == nullptr) return;
    
    std::lock_guard<std::mutex> lock(mutex_);
    
    // Check if buffer belongs to this pool
    bool belongs_to_pool = false;
    for (const auto& owned_buffer : buffers_) {
        if (owned_buffer.get() == buffer) {
            belongs_to_pool = true;
            break;
        }
    }
    
    if (belongs_to_pool) {
        free_buffers_.push_back(buffer);
    } else {
        #ifdef DEBUG
        // Check if this was a dynamically allocated buffer
        auto it = dynamic_buffers_.find(buffer);
        if (it != dynamic_buffers_.end()) {
            dynamic_buffers_.erase(it);
        } else {
            // Buffer not tracked - potential leak or double-free
            leaked_buffers_.fetch_add(1);
        }
        #endif
        // Buffer was dynamically allocated, delete it
        delete[] buffer;
    }
}

size_t BufferPool::available_count() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return free_buffers_.size();
}

size_t BufferPool::total_count() const {
    return buffers_.size();
}

// MemoryManager Implementation
MemoryManager& MemoryManager::instance() {
    static MemoryManager inst;
    return inst;
}

MemoryManager::MemoryManager() 
    : total_allocated_(0), total_deallocated_(0) {
    small_pool_ = std::make_unique<BufferPool>(SMALL_BUFFER, 200);
    medium_pool_ = std::make_unique<BufferPool>(MEDIUM_BUFFER, 100);
    large_pool_ = std::make_unique<BufferPool>(LARGE_BUFFER, 50);
}

MemoryManager::~MemoryManager() {
    // Pools will be automatically destroyed
}

BufferPool* MemoryManager::get_buffer_pool(size_t size) {
    if (size <= SMALL_BUFFER) {
        return small_pool_.get();
    } else if (size <= MEDIUM_BUFFER) {
        return medium_pool_.get();
    } else if (size <= LARGE_BUFFER) {
        return large_pool_.get();
    }
    return nullptr; // Too large for pools
}

void* MemoryManager::allocate_aligned(size_t size, size_t alignment) {
    // Use aligned_alloc if available, otherwise fallback to malloc
    #ifdef _ISOC11_SOURCE
    void* ptr = aligned_alloc(alignment, size);
    #else
    void* ptr = nullptr;
    if (posix_memalign(&ptr, alignment, size) != 0) {
        return nullptr;
    }
    #endif
    
    if (ptr) {
        total_allocated_ += size;
    }
    return ptr;
}

void MemoryManager::deallocate_aligned(void* ptr) {
    if (ptr) {
        free(ptr);
        // Note: We can't track exact size here, so we just increment counter
        total_deallocated_++;
    }
}

// ScopedBuffer Implementation
ScopedBuffer::ScopedBuffer(BufferPool* pool, char* buffer)
    : pool_(pool), buffer_(buffer) {
}

ScopedBuffer::~ScopedBuffer() {
    if (pool_ && buffer_) {
        pool_->release(buffer_);
    }
}

ScopedBuffer::ScopedBuffer(ScopedBuffer&& other) noexcept
    : pool_(other.pool_), buffer_(other.buffer_) {
    other.pool_ = nullptr;
    other.buffer_ = nullptr;
}

ScopedBuffer& ScopedBuffer::operator=(ScopedBuffer&& other) noexcept {
    if (this != &other) {
        if (pool_ && buffer_) {
            pool_->release(buffer_);
        }
        pool_ = other.pool_;
        buffer_ = other.buffer_;
        other.pool_ = nullptr;
        other.buffer_ = nullptr;
    }
    return *this;
}
