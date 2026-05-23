#pragma once

#include <cstdint>
#include <cstdio>

#include "../platform.h"

#if PLATFORM_LINUX
#include <pthread.h>
#include <sched.h>
#endif

namespace hft {

// Best-effort CPU pinning for the *current thread*.
// On Linux this uses pthread_setaffinity_np; on macOS it's a no-op.
inline void pin_this_thread_to_cpu(int cpu) {
#if PLATFORM_LINUX
    if (cpu < 0) return;
    cpu_set_t set;
    CPU_ZERO(&set);
    CPU_SET(cpu, &set);
    const int rc = pthread_setaffinity_np(pthread_self(), sizeof(set), &set);
    if (rc != 0) {
        // Non-fatal; just log once-ish at caller discretion.
        std::fprintf(stderr, "pin_thread: failed to pin to cpu=%d (rc=%d)\n", cpu, rc);
    }
#else
    (void)cpu;
#endif
}

}  // namespace hft

