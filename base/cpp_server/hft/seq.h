#pragma once

#include <cstdint>

namespace hft {

// Tracks sequence gaps in a stream.
struct SeqTracker {
    uint32_t next = 0;
    uint64_t gaps = 0;
    uint64_t dup_or_old = 0;

    void reset() { next = 0; gaps = 0; dup_or_old = 0; }

    // Returns true if seq is the expected next sequence.
    bool on_seq(uint32_t seq) {
        if (next == 0) {
            next = seq + 1;
            return true;
        }
        if (seq == next) {
            ++next;
            return true;
        }
        if (seq > next) {
            gaps += (uint64_t)(seq - next);
            next = seq + 1;
            return false;
        }
        // seq < next
        ++dup_or_old;
        return false;
    }
};

}  // namespace hft

