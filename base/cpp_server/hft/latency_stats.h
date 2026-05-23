#pragma once

#include <cstdint>
#include <cstdio>

namespace hft {

// Tiny fixed-bucket latency histogram (nanoseconds).
// Buckets: [0..2^0), [2^0..2^1), ..., [2^62..2^63)
struct LatencyStats {
    uint64_t count = 0;
    uint64_t min_ns = ~0ull;
    uint64_t max_ns = 0;
    uint64_t buckets[64] = {};

    void add(uint64_t ns) {
        ++count;
        if (ns < min_ns) min_ns = ns;
        if (ns > max_ns) max_ns = ns;
        const uint32_t b = bucket_index(ns);
        ++buckets[b];
    }

    void reset() {
        count = 0;
        min_ns = ~0ull;
        max_ns = 0;
        for (auto &v : buckets) v = 0;
    }

    void print_summary(const char* label) const {
        if (count == 0) {
            std::printf("%s: no samples\n", label);
            return;
        }
        std::printf(
            "%s: n=%llu min=%.2fus max=%.2fus\n",
            label,
            (unsigned long long)count,
            (double)min_ns / 1000.0,
            (double)max_ns / 1000.0
        );
    }

private:
    static uint32_t bucket_index(uint64_t ns) {
        if (ns == 0) return 0;
        // floor(log2(ns))
        return 63u - (uint32_t)__builtin_clzll(ns);
    }
};

}  // namespace hft

