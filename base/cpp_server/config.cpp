// Server Configuration Implementation

#include "config.h"
#include <cstdio>
#include <cstring>
#include <cstdlib>
#include <thread>

ServerConfig g_config;

bool ServerConfig::parse_args(int argc, char* argv[]) {
    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "-p") == 0 || strcmp(argv[i], "--port") == 0) {
            if (i + 1 < argc) {
                port = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: -p requires a port number\n");
                return false;
            }
        } else if (strcmp(argv[i], "-w") == 0 || strcmp(argv[i], "--workers") == 0) {
            if (i + 1 < argc) {
                num_workers = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: -w requires a number\n");
                return false;
            }
        } else if (strcmp(argv[i], "--max-connections") == 0) {
            if (i + 1 < argc) {
                max_connections = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --max-connections requires a number\n");
                return false;
            }
        } else if (strcmp(argv[i], "--pool-size") == 0) {
            if (i + 1 < argc) {
                initial_pool_size = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --pool-size requires a number\n");
                return false;
            }
        } else if (strcmp(argv[i], "--backlog") == 0) {
            if (i + 1 < argc) {
                listen_backlog = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --backlog requires a number\n");
                return false;
            }
        } else if (strcmp(argv[i], "--buffer-size") == 0) {
            if (i + 1 < argc) {
                buffer_size = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --buffer-size requires a number\n");
                return false;
            }
        }
        #if PLATFORM_LINUX
        else if (strcmp(argv[i], "--ring-size") == 0) {
            if (i + 1 < argc) {
                ring_size = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --ring-size requires a number\n");
                return false;
            }
        } else if (strcmp(argv[i], "--initial-accepts") == 0) {
            if (i + 1 < argc) {
                initial_accepts = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --initial-accepts requires a number\n");
                return false;
            }
        } else if (strcmp(argv[i], "--busy-poll-us") == 0) {
            if (i + 1 < argc) {
                busy_poll_us = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --busy-poll-us requires a number\n");
                return false;
            }
        }
        #elif PLATFORM_MACOS
        else if (strcmp(argv[i], "--kqueue-batch") == 0) {
            if (i + 1 < argc) {
                kqueue_batch_size = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --kqueue-batch requires a number\n");
                return false;
            }
        } else if (strcmp(argv[i], "--flush-threshold") == 0) {
            if (i + 1 < argc) {
                flush_threshold = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --flush-threshold requires a number\n");
                return false;
            }
        } else if (strcmp(argv[i], "--timeout") == 0) {
            if (i + 1 < argc) {
                adaptive_timeout_ms = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --timeout requires a number\n");
                return false;
            }
        } else if (strcmp(argv[i], "--max-accepts") == 0) {
            if (i + 1 < argc) {
                max_accepts_per_event = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --max-accepts requires a number\n");
                return false;
            }
        }
        #endif
        else if (strcmp(argv[i], "--cpu-threads") == 0) {
            if (i + 1 < argc) {
                cpu_pool_threads = atoi(argv[++i]);
            } else {
                fprintf(stderr, "Error: --cpu-threads requires a number\n");
                return false;
            }
        } else if (strcmp(argv[i], "-h") == 0 || strcmp(argv[i], "--help") == 0) {
            return false;  // Signal to print help
        } else if (strcmp(argv[i], "--config") == 0) {
            if (i + 1 < argc) {
                if (!load_from_file(argv[++i])) {
                    return false;
                }
            } else {
                fprintf(stderr, "Error: --config requires a filename\n");
                return false;
            }
        }
    }
    
    // Auto-detect workers if 0
    if (num_workers == 0) {
        num_workers = std::thread::hardware_concurrency();
        if (num_workers == 0) num_workers = 1;
    }
    
    // Auto-detect CPU pool threads if 0
    if (cpu_pool_threads == 0) {
        cpu_pool_threads = std::thread::hardware_concurrency();
        if (cpu_pool_threads == 0) cpu_pool_threads = 1;
    }
    
    return validate();
}

bool ServerConfig::load_from_file(const std::string& filename) {
    // TODO: Implement config file parsing (JSON, YAML, or simple key=value)
    // For now, just a placeholder
    fprintf(stderr, "Config file loading not yet implemented: %s\n", filename.c_str());
    return false;
}

void ServerConfig::print() const {
    printf("Server Configuration:\n");
    printf("  Port: %d\n", port);
    printf("  Workers: %d\n", num_workers);
    printf("  Max Connections: %d\n", max_connections);
    printf("  Initial Pool Size: %zu\n", initial_pool_size);
    printf("  Listen Backlog: %d\n", listen_backlog);
    printf("  Buffer Size: %zu\n", buffer_size);
    
    #if PLATFORM_LINUX
    printf("  Ring Size: %d\n", ring_size);
    printf("  Initial Accepts: %d\n", initial_accepts);
    printf("  Busy Poll: %d us\n", busy_poll_us);
    #elif PLATFORM_MACOS
    printf("  Kqueue Batch Size: %d\n", kqueue_batch_size);
    printf("  Flush Threshold: %d\n", flush_threshold);
    printf("  Adaptive Timeout: %d ms (max: %d ms)\n", adaptive_timeout_ms, max_adaptive_timeout_ms);
    printf("  Max Accepts Per Event: %d\n", max_accepts_per_event);
    #endif
    
    printf("  CPU Pool Threads: %zu\n", cpu_pool_threads);
    printf("  CPU Pool Timeout: %d ms\n", cpu_pool_timeout_ms);
    printf("  Memory Pool Size: %zu\n", memory_pool_size);
    printf("  Buffer Pool Size: %zu\n", buffer_pool_size);
}

bool ServerConfig::validate() const {
    if (port < 1 || port > 65535) {
        fprintf(stderr, "Error: Port must be between 1 and 65535\n");
        return false;
    }
    
    if (num_workers < 1) {
        fprintf(stderr, "Error: Number of workers must be >= 1\n");
        return false;
    }
    
    if (max_connections < 1) {
        fprintf(stderr, "Error: Max connections must be >= 1\n");
        return false;
    }
    
    if (initial_pool_size > static_cast<size_t>(max_connections)) {
        fprintf(stderr, "Warning: Initial pool size (%zu) > max connections (%d)\n", 
                initial_pool_size, max_connections);
    }
    
    if (buffer_size < 1024) {
        fprintf(stderr, "Error: Buffer size must be >= 1024\n");
        return false;
    }
    
    #if PLATFORM_LINUX
    if (ring_size < 64) {
        fprintf(stderr, "Error: Ring size must be >= 64\n");
        return false;
    }
    if (busy_poll_us < 0 || busy_poll_us > 1000000) {
        fprintf(stderr, "Error: Busy poll must be between 0 and 1000000 us\n");
        return false;
    }
    #elif PLATFORM_MACOS
    if (kqueue_batch_size < 1 || kqueue_batch_size > 1024) {
        fprintf(stderr, "Error: Kqueue batch size must be between 1 and 1024\n");
        return false;
    }
    if (adaptive_timeout_ms < 1 || adaptive_timeout_ms > 1000) {
        fprintf(stderr, "Error: Adaptive timeout must be between 1 and 1000 ms\n");
        return false;
    }
    #endif
    
    return true;
}
