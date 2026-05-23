// CPU-Bound Layer - Thread Pool for CPU-Intensive Tasks
// Handles: CPU-bound processing, computation, data transformation
// Separates CPU work from I/O threads to maintain high throughput

#pragma once

#include <functional>
#include <thread>
#include <vector>
#include <queue>
#include <mutex>
#include <condition_variable>
#include <atomic>
#include <future>
#include <memory>
#include <string>
#include "interfaces.h"

// CPU-bound task type
using CPUTask = std::function<void()>;

// CPU-bound task result type
template<typename T>
using CPUTaskResult = std::function<T()>;

// CPU-bound task pool - manages worker threads for CPU-intensive work
class CPUPool {
public:
    CPUPool(size_t num_threads = 0);
    ~CPUPool();
    
    // Submit a CPU-bound task (fire-and-forget)
    void submit(CPUTask task);
    
    // Submit a CPU-bound task and get future result
    template<typename T>
    std::future<T> submit_with_result(CPUTaskResult<T> task);
    
    // Get number of worker threads
    size_t size() const { return workers_.size(); }
    
    // Shutdown the pool
    void shutdown();
    
    // Wait for all pending tasks to complete
    void wait_for_completion();
    
private:
    void worker_loop();
    
    std::vector<std::thread> workers_;
    std::queue<CPUTask> task_queue_;
    std::mutex queue_mutex_;
    std::condition_variable condition_;
    std::atomic<bool> shutdown_;
    std::atomic<size_t> pending_tasks_;
};

// CPU-bound request processor
// Processes HTTP requests that require CPU-intensive work
class CPURequestProcessor : public ICPUProcessor {
public:
    CPURequestProcessor(size_t pool_size = 0);
    ~CPURequestProcessor();
    
    // ICPUProcessor interface
    std::string process(const std::string& input) override;
    bool is_available() const override { return cpu_pool_ != nullptr; }
    
    // Additional methods
    // Process request asynchronously (CPU-bound work)
    // Returns true if processing is complete synchronously, false if async
    bool process_async(const char* request_data, size_t request_len,
                     std::function<void(const char*, size_t)> on_complete);
    
    // Check if request needs CPU-bound processing
    static bool needs_cpu_processing(const char* request_data, size_t request_len);
    
private:
    std::unique_ptr<CPUPool> cpu_pool_;
};

// Template implementation for submit_with_result
template<typename T>
std::future<T> CPUPool::submit_with_result(CPUTaskResult<T> task) {
    auto promise = std::make_shared<std::promise<T>>();
    auto future = promise->get_future();
    
    submit([task, promise]() {
        try {
            promise->set_value(task());
        } catch (...) {
            promise->set_exception(std::current_exception());
        }
    });
    
    return future;
}
