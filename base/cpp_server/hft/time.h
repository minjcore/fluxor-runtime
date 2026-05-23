#pragma once

#include <cstdint>
#include <time.h>

namespace hft {

// Monotonic time in nanoseconds (for latency measurement).
inline uint64_t now_ns() {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (uint64_t)ts.tv_sec * 1000000000ull + (uint64_t)ts.tv_nsec;
}

}  // namespace hft

