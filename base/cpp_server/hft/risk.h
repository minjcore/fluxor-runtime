#pragma once

#include <cstdint>

#include "messages.h"

namespace hft {

struct RiskConfig {
    int32_t max_abs_position = 100;  // |pos| <= this
    int32_t max_order_qty = 10;      // per order
};

struct RiskState {
    int32_t position = 0;  // net position (+long, -short)
    int32_t reserved = 0;  // pending delta from outstanding orders
    uint64_t rejects = 0;
};

inline bool risk_check(const RiskConfig& cfg, const RiskState& st, const Order& o) {
    if (o.qty <= 0 || o.qty > cfg.max_order_qty) return false;
    const int32_t eff_pos = st.position + st.reserved;
    const int32_t new_pos = eff_pos + ((o.side == Side::Buy) ? o.qty : -o.qty);
    if (new_pos > cfg.max_abs_position || new_pos < -cfg.max_abs_position) return false;
    return true;
}

}  // namespace hft

