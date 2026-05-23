// Server Configuration - Centralized settings management

#pragma once

#include <cstddef>
#include <cstdint>
#include <string>

#include "platform.h"

struct ServerConfig {
    // Server settings
    int port = 8083;
    int num_workers = 1;  // 0 = auto-detect
    
    // Connection settings
    int max_connections = 10000;
    size_t initial_pool_size = 500;
    int listen_backlog = 4096;
    
    // Buffer settings
    size_t buffer_size = 8192;
    
    // Event loop settings
    #if PLATFORM_LINUX
    int ring_size = 4096;
    int initial_accepts = 20;
    // Busy-polling (lower latency, higher CPU). Unit: microseconds.
    // Applied via SO_BUSY_POLL on accepted sockets (Linux only).
    int busy_poll_us = 0;
    #elif PLATFORM_MACOS
    int kqueue_batch_size = 256;
    int flush_threshold = 8;
    int adaptive_timeout_ms = 10;
    int max_adaptive_timeout_ms = 100;
    int max_accepts_per_event = 50;
    #endif
    
    // CPU layer settings
    size_t cpu_pool_threads = 0;  // 0 = auto-detect (hardware_concurrency)
    int cpu_pool_timeout_ms = 100;
    
    // Memory layer settings
    size_t memory_pool_size = 1024;
    size_t buffer_pool_size = 512;
    
    // Load from command line arguments
    bool parse_args(int argc, char* argv[]);
    
    // Load from config file (future)
    bool load_from_file(const std::string& filename);
    
    // Print current configuration
    void print() const;
    
    // Validate configuration
    bool validate() const;
};

// Global config instance
extern ServerConfig g_config;
