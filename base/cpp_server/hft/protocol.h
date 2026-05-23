#pragma once

#include <cstdint>

namespace hft {

// Minimal market-data message for a local simulator.
// All fields are little-endian on x86/arm64 macOS, but treat as host-endian for now.
// For a real venue, you'd define strict endian + alignment rules.
struct MdUpdate {
    uint64_t ts_send_ns;   // timestamp at publisher (monotonic ns)
    uint32_t seq;          // sequence number
    uint8_t  side;         // 0=bid, 1=ask
    uint8_t  _pad0[3];
    int64_t  price_ticks;  // integer ticks
    int32_t  qty;          // size
    int32_t  _pad1;
};

static_assert(sizeof(MdUpdate) == 32, "MdUpdate must be 32 bytes");

}  // namespace hft

