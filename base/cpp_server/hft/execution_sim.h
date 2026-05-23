#pragma once

#include <cstdint>

#include "messages.h"
#include "time.h"

namespace hft {

struct ExecSimConfig {
    int64_t tick_size = 1;           // ticks
    int32_t fill_qty = 1;            // immediate fill quantity
    uint32_t fill_delay_ns = 0;      // artificial delay (spin)
};

// Deterministic "exchange" simulator:
// - Acks immediately (optional)
// - Fills immediately with fixed qty at the order price
class ExecutionSim {
public:
    explicit ExecutionSim(ExecSimConfig cfg) : cfg_(cfg) {}

    // Apply order to position and produce a fill report.
    ExecReport on_order(const Order& o, int32_t& position) const {
        if (cfg_.fill_delay_ns) spin_delay(cfg_.fill_delay_ns);

        ExecReport er{};
        er.ts_send_ns = now_ns();
        er.cl_id = o.cl_id;
        er.side = o.side;

        // Always fill fixed qty (capped by order qty)
        const int32_t fill_qty = (o.qty < cfg_.fill_qty) ? o.qty : cfg_.fill_qty;
        er.status = (fill_qty > 0) ? OrdStatus::Filled : OrdStatus::Rejected;
        er.filled_qty = fill_qty;
        er.fill_price_ticks = o.price_ticks;

        if (er.status == OrdStatus::Filled) {
            position += (o.side == Side::Buy) ? fill_qty : -fill_qty;
        }
        er.position_after = position;
        return er;
    }

private:
    static inline void spin_delay(uint32_t ns) {
        const uint64_t start = now_ns();
        while ((now_ns() - start) < (uint64_t)ns) {
            // spin
        }
    }

    ExecSimConfig cfg_;
};

}  // namespace hft

