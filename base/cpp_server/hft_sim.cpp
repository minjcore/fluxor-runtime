#include <atomic>
#include <cerrno>
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <csignal>
#include <thread>

#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>

#include "hft/protocol.h"
#include "hft/spsc_ring.h"
#include "hft/time.h"
#include "hft/latency_stats.h"
#include "hft/order_book.h"
#include "hft/threading.h"

// Minimal HFT-ish simulator:
// - UDP publisher (optional --selftest) sends MdUpdate packets to localhost
// - UDP receiver parses into SPSC ring
// - Strategy thread updates top-of-book and prints periodic stats + latency

namespace {

std::atomic<bool>* g_running_ptr = nullptr;

void on_signal(int /*sig*/) {
    if (g_running_ptr) {
        g_running_ptr->store(false, std::memory_order_relaxed);
    }
}

struct Args {
    int listen_port = 9000;
    bool selftest = false;
    int selftest_rate_hz = 200000;  // messages/sec
    int pin_base = -1;              // if >=0 pin rx/strategy/tx threads
};

Args parse_args(int argc, char** argv) {
    Args a;
    for (int i = 1; i < argc; ++i) {
        if (std::strcmp(argv[i], "--port") == 0 && i + 1 < argc) {
            a.listen_port = std::atoi(argv[++i]);
        } else if (std::strcmp(argv[i], "--selftest") == 0) {
            a.selftest = true;
        } else if (std::strcmp(argv[i], "--rate") == 0 && i + 1 < argc) {
            a.selftest_rate_hz = std::atoi(argv[++i]);
        } else if (std::strcmp(argv[i], "--pin-base") == 0 && i + 1 < argc) {
            a.pin_base = std::atoi(argv[++i]);
        } else if (std::strcmp(argv[i], "--help") == 0 || std::strcmp(argv[i], "-h") == 0) {
            std::printf("Usage: %s [--port 9000] [--selftest] [--rate 200000] [--pin-base -1]\n", argv[0]);
            std::exit(0);
        }
    }
    return a;
}

int make_udp_socket_bound(int port) {
    const int fd = ::socket(AF_INET, SOCK_DGRAM, 0);
    if (fd < 0) return -1;

    int opt = 1;
    (void)::setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

    sockaddr_in addr{};
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = htonl(INADDR_ANY);
    addr.sin_port = htons((uint16_t)port);
    if (::bind(fd, (sockaddr*)&addr, sizeof(addr)) < 0) {
        ::close(fd);
        return -1;
    }
    return fd;
}

int make_udp_socket_connected(const char* ip, int port) {
    const int fd = ::socket(AF_INET, SOCK_DGRAM, 0);
    if (fd < 0) return -1;

    sockaddr_in addr{};
    addr.sin_family = AF_INET;
    addr.sin_port = htons((uint16_t)port);
    if (::inet_pton(AF_INET, ip, &addr.sin_addr) != 1) {
        ::close(fd);
        return -1;
    }
    if (::connect(fd, (sockaddr*)&addr, sizeof(addr)) < 0) {
        ::close(fd);
        return -1;
    }
    return fd;
}

}  // namespace

int main(int argc, char** argv) {
    const Args args = parse_args(argc, argv);

    std::atomic<bool> running{true};
    g_running_ptr = &running;
    std::signal(SIGINT, on_signal);
    std::signal(SIGTERM, on_signal);
    // Ensure line-buffered output (useful when redirected/captured)
    setvbuf(stdout, nullptr, _IOLBF, 0);

    hft::SpscRing<hft::MdUpdate, 1u << 16> q;  // 65536 entries

    const int rx_fd = make_udp_socket_bound(args.listen_port);
    if (rx_fd < 0) {
        std::perror("bind udp");
        return 1;
    }

    std::thread publisher;
    if (args.selftest) {
        publisher = std::thread([&]() {
            if (args.pin_base >= 0) hft::pin_this_thread_to_cpu(args.pin_base + 2);
            const int tx_fd = make_udp_socket_connected("127.0.0.1", args.listen_port);
            if (tx_fd < 0) {
                std::perror("connect udp");
                return;
            }
            uint32_t seq = 0;
            int64_t px = 100000;  // ticks
            const int rate = (args.selftest_rate_hz > 0) ? args.selftest_rate_hz : 1;
            const uint64_t sleep_ns = 1000000000ull / (uint64_t)rate;

            while (running.load(std::memory_order_relaxed)) {
                hft::MdUpdate m{};
                m.ts_send_ns = hft::now_ns();
                m.seq = seq++;
                m.side = (seq & 1) ? 0 : 1;
                px += (m.side == 0) ? 0 : 1;
                m.price_ticks = px;
                m.qty = 1;
                (void)::send(tx_fd, &m, sizeof(m), 0);

                // crude rate control (good enough for selftest)
                const uint64_t start = hft::now_ns();
                while (hft::now_ns() - start < sleep_ns) {
                    // spin
                }
            }
            ::close(tx_fd);
        });
    }

    std::thread receiver([&]() {
        if (args.pin_base >= 0) hft::pin_this_thread_to_cpu(args.pin_base + 0);
        while (running.load(std::memory_order_relaxed)) {
            hft::MdUpdate m{};
            const ssize_t n = ::recv(rx_fd, &m, sizeof(m), 0);
            if (n == (ssize_t)sizeof(m)) {
                // drop if queue full
                (void)q.try_push(m);
            } else if (n < 0) {
                if (errno == EINTR) continue;
                // Best effort: keep running on transient errors
            }
        }
    });

    std::thread strategy([&]() {
        if (args.pin_base >= 0) hft::pin_this_thread_to_cpu(args.pin_base + 1);
        hft::TopOfBook book;
        hft::LatencyStats lat;
        uint64_t last_print_ns = hft::now_ns();
        uint64_t processed = 0;
        uint64_t dropped_est = 0;

        while (running.load(std::memory_order_relaxed)) {
            hft::MdUpdate m{};
            if (!q.try_pop(m)) {
                // light spin
                continue;
            }

            const uint64_t now = hft::now_ns();
            lat.add(now - m.ts_send_ns);
            ++processed;

            if (m.side == 0) book.on_bid(m.price_ticks, m.qty);
            else book.on_ask(m.price_ticks, m.qty);

            // Periodic print (1s)
            if (now - last_print_ns >= 1000000000ull) {
                const uint32_t qsz = q.approx_size();
                // we can’t know true drops without a counter; this is a heuristic
                if (qsz > (1u << 15)) dropped_est += 1;

                std::printf(
                    "pps=%llu q=%u book(bid=%lld ask=%lld spr=%lld) drops~=%llu ",
                    (unsigned long long)processed,
                    (unsigned)qsz,
                    (long long)book.bid_px(),
                    (long long)book.ask_px(),
                    (long long)book.spread_ticks(),
                    (unsigned long long)dropped_est
                );
                lat.print_summary("lat");
                processed = 0;
                lat.reset();
                last_print_ns = now;
            }
        }
    });

    std::printf("hft_sim listening UDP port %d%s\n", args.listen_port, args.selftest ? " (selftest ON)" : "");
    std::printf("Press Ctrl+C to stop.\n");

    // Wait until signal flips running=false
    while (running.load(std::memory_order_relaxed)) {
        std::this_thread::sleep_for(std::chrono::milliseconds(50));
    }

    receiver.join();
    strategy.join();
    if (publisher.joinable()) publisher.join();
    ::close(rx_fd);
    return 0;
}

