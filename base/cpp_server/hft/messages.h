#pragma once

#include <cstdint>

namespace hft {

enum class Side : uint8_t { Buy = 0, Sell = 1 };
enum class OrdStatus : uint8_t { NewAck = 0, Filled = 1, Rejected = 2, Canceled = 3 };

// Strategy -> OrderManager/Exchange
struct Order {
    uint64_t ts_send_ns;     // when strategy sent the order
    uint64_t cl_id;          // client order id
    Side side;
    uint8_t _pad0[7];
    int64_t price_ticks;
    int32_t qty;
    int32_t _pad1;
};
static_assert(sizeof(Order) == 40, "Order size");

// Exchange/Sim -> Strategy
struct ExecReport {
    uint64_t ts_send_ns;     // when exchange sent this report
    uint64_t cl_id;          // echoes client order id
    OrdStatus status;
    Side side;
    uint8_t _pad0[6];
    int64_t fill_price_ticks;
    int32_t filled_qty;
    int32_t position_after;  // net position after applying this report
};
static_assert(sizeof(ExecReport) == 40, "ExecReport size");

}  // namespace hft

