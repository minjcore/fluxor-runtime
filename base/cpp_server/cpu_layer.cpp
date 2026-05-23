// CPU-Bound Layer Implementation

#include "cpu_layer.h"
#include <algorithm>
#include <cstring>

// CPUPool Implementation
CPUPool::CPUPool(size_t num_threads) : shutdown_(false), pending_tasks_(0) {
    if (num_threads == 0) {
        size_t hw_threads = std::thread::hardware_concurrency();
        num_threads = (hw_threads > 0) ? hw_threads : 1;
    }
    
    workers_.reserve(num_threads);
    for (size_t i = 0; i < num_threads; i++) {
        workers_.emplace_back(&CPUPool::worker_loop, this);
    }
}

CPUPool::~CPUPool() {
    shutdown();
}

void CPUPool::shutdown() {
    if (shutdown_.exchange(true)) {
        return; // Already shutting down
    }
    
    condition_.notify_all();
    
    for (auto& worker : workers_) {
        if (worker.joinable()) {
            worker.join();
        }
    }
}

void CPUPool::wait_for_completion() {
    while (pending_tasks_.load() > 0) {
        std::this_thread::yield();
    }
}

void CPUPool::submit(CPUTask task) {
    if (shutdown_.load()) {
        return; // Pool is shutting down
    }
    
    {
        std::lock_guard<std::mutex> lock(queue_mutex_);
        task_queue_.push(std::move(task));
        pending_tasks_++;
    }
    
    condition_.notify_one();
}

void CPUPool::worker_loop() {
    while (!shutdown_.load()) {
        CPUTask task;
        
        {
            std::unique_lock<std::mutex> lock(queue_mutex_);
            // Use wait_for with timeout to avoid spinning when idle
            // This reduces CPU usage when no tasks are available
            bool has_task = condition_.wait_for(lock, std::chrono::milliseconds(100), [this] {
                return !task_queue_.empty() || shutdown_.load();
            });
            
            if (shutdown_.load() && task_queue_.empty()) {
                break;
            }
            
            if (!task_queue_.empty()) {
                task = std::move(task_queue_.front());
                task_queue_.pop();
            } else if (!has_task) {
                // Timeout - no tasks available, continue loop to check shutdown
                continue;
            }
        }
        
        if (task) {
            task();
            pending_tasks_--;
        }
    }
}

// CPURequestProcessor Implementation
CPURequestProcessor::CPURequestProcessor(size_t pool_size) {
    cpu_pool_ = std::make_unique<CPUPool>(pool_size);
}

CPURequestProcessor::~CPURequestProcessor() {
    if (cpu_pool_) {
        cpu_pool_->wait_for_completion();
        cpu_pool_->shutdown();
    }
}

std::string CPURequestProcessor::process(const std::string& input) {
    // ICPUProcessor interface - synchronous processing
    // For now, return input as-is (can be enhanced with actual CPU-bound work)
    return input;
}

bool CPURequestProcessor::needs_cpu_processing(const char* request_data, size_t request_len) {
    // Check if request path indicates CPU-bound work
    // For now, check for specific paths that need CPU processing
    // Example: /compute, /process, /transform, etc.
    
    if (request_len < 10) return false;
    
    // Simple check for CPU-bound paths
    const char* cpu_paths[] = {
        "/compute",
        "/process",
        "/transform",
        "/calculate"
    };
    
    for (const char* path : cpu_paths) {
        size_t path_len = strlen(path);
        if (request_len >= path_len && 
            memcmp(request_data + 4, path, path_len) == 0) { // Skip "GET "
            return true;
        }
    }
    
    return false;
}

bool CPURequestProcessor::process_async(const char* request_data, size_t request_len,
                                        std::function<void(const char*, size_t)> on_complete) {
    if (!needs_cpu_processing(request_data, request_len)) {
        return true; // No CPU processing needed, handle synchronously
    }
    
    // Submit CPU-bound task
    cpu_pool_->submit([on_complete]() {
        // CPU-intensive work here
        // For now, just a placeholder - can be extended for actual processing
        // Example: JSON parsing, data transformation, computation, etc.
        
        // Simulate CPU work (replace with actual processing)
        // For demonstration, just return OK response
        const char* response = "HTTP/1.1 200 OK\r\n"
                              "Content-Length: 2\r\n"
                              "Connection: keep-alive\r\n"
                              "\r\n"
                              "OK";
        size_t response_len = strlen(response);
        
        // Call completion callback
        on_complete(response, response_len);
    });
    
    return false; // Processing is async
}
