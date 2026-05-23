#pragma once

#include <cstdint>

#include "messages.h"
#include "risk.h"
#include "time.h"

namespace hft {

// Very small order manager: assigns ids and applies exec reports to risk state.
// In real HFT you'd track per-order state, partial fills, cancels, replaces, etc.
class OrderManager {
public:
    explicit OrderManager(RiskConfig cfg) : cfg_(cfg) {}

    uint64_t next_id() { return ++last_id_; }

    bool prepare_and_check(Order& o, RiskState& st, Side side, int64_t px_ticks, int32_t qty) {
        o.ts_send_ns = now_ns();
        o.cl_id = next_id();
        o.side = side;
        o.price_ticks = px_ticks;
        o.qty = qty;
        if (!risk_check(cfg_, st, o)) {
            ++st.rejects;
            return false;
        }
        // Reserve position for this in-flight order (pre-trade risk).
        st.reserved += (side == Side::Buy) ? qty : -qty;
        return true;
    }

    void apply_exec(RiskState& st, const ExecReport& er) {
        if (er.status == OrdStatus::Filled) {
            // Release reserved based on filled qty + side.
            st.reserved -= (er.side == Side::Buy) ? er.filled_qty : -er.filled_qty;
            st.position = er.position_after;
        }
    }

private:
    RiskConfig cfg_;
    uint64_t last_id_ = 0;
};

}  // namespace hft

