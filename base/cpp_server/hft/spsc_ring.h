#pragma once

#include <atomic>
#include <cstddef>
#include <cstdint>
#include <type_traits>

namespace hft {

// Single-producer single-consumer ring buffer.
// Requirements:
// - Capacity must be a power of two.
// - T should be trivially copyable for best performance.
template <typename T, size_t CapacityPow2>
class SpscRing {
    static_assert((CapacityPow2 & (CapacityPow2 - 1)) == 0, "Capacity must be power of two");
    static_assert(std::is_trivially_copyable_v<T>, "T should be trivially copyable");

public:
    SpscRing() : head_(0), tail_(0) {}

    bool try_push(const T& v) {
        const uint32_t head = head_.load(std::memory_order_relaxed);
        const uint32_t next = head + 1;
        if (next - tail_.load(std::memory_order_acquire) > CapacityPow2) {
            return false;  // full
        }
        buf_[head & mask_] = v;
        head_.store(next, std::memory_order_release);
        return true;
    }

    bool try_pop(T& out) {
        const uint32_t tail = tail_.load(std::memory_order_relaxed);
        if (tail == head_.load(std::memory_order_acquire)) {
            return false;  // empty
        }
        out = buf_[tail & mask_];
        tail_.store(tail + 1, std::memory_order_release);
        return true;
    }

    uint32_t approx_size() const {
        const uint32_t h = head_.load(std::memory_order_acquire);
        const uint32_t t = tail_.load(std::memory_order_acquire);
        return h - t;
    }

private:
    static constexpr uint32_t mask_ = (uint32_t)(CapacityPow2 - 1);
    alignas(64) std::atomic<uint32_t> head_;
    alignas(64) std::atomic<uint32_t> tail_;
    alignas(64) T buf_[CapacityPow2];
};

}  // namespace hft

