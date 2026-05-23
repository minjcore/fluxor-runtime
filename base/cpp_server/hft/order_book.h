#pragma once

#include <cstdint>

namespace hft {

// Minimal top-of-book (best bid/ask) book builder.
// This is intentionally tiny; real HFT needs full depth, order-by-order, etc.
class TopOfBook {
public:
    void on_bid(int64_t price_ticks, int32_t qty) {
        bid_px_ = price_ticks;
        bid_qty_ = qty;
        ++updates_;
    }
    void on_ask(int64_t price_ticks, int32_t qty) {
        ask_px_ = price_ticks;
        ask_qty_ = qty;
        ++updates_;
    }

    bool has_both() const { return bid_qty_ > 0 && ask_qty_ > 0; }
    int64_t bid_px() const { return bid_px_; }
    int64_t ask_px() const { return ask_px_; }
    int32_t bid_qty() const { return bid_qty_; }
    int32_t ask_qty() const { return ask_qty_; }
    uint64_t updates() const { return updates_; }

    int64_t mid_px_ticks() const { return (bid_px_ + ask_px_) / 2; }
    int64_t spread_ticks() const { return ask_px_ - bid_px_; }

private:
    int64_t bid_px_ = 0;
    int64_t ask_px_ = 0;
    int32_t bid_qty_ = 0;
    int32_t ask_qty_ = 0;
    uint64_t updates_ = 0;
};

}  // namespace hft

