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
#include "hft/seq.h"
#include "hft/spsc_ring.h"
#include "hft/time.h"
#include "hft/latency_stats.h"
#include "hft/order_book.h"
#include "hft/messages.h"
#include "hft/risk.h"
#include "hft/order_manager.h"
#include "hft/execution_sim.h"
#include "hft/threading.h"

// hft_engine: end-to-end toy HFT pipeline
// - UDP market data (optional selftest publisher)
// - Feed thread -> SPSC -> Strategy thread
// - Strategy updates top-of-book, runs trivial strategy, sends orders via SPSC
// - Execution sim thread fills orders and returns ExecReports via SPSC
// - Prints throughput + latency + position + gaps

namespace {

std::atomic<bool>* g_running_ptr = nullptr;
void on_signal(int /*sig*/) {
    if (g_running_ptr) g_running_ptr->store(false, std::memory_order_relaxed);
}

struct Args {
    int md_port = 9000;
    bool selftest = false;
    int md_rate_hz = 200000;
    int duration_s = 0;          // 0 = until Ctrl+C

    // Strategy knobs
    int64_t base_px = 100000;
    int64_t max_spread_ticks = 2;
    int32_t order_qty = 1;
    int32_t max_abs_pos = 100;
    int32_t max_order_qty = 10;

    // Exec sim
    uint32_t fill_delay_ns = 0;

    // Linux-first knobs
    int pin_base = -1;  // if >=0, pin threads to pin_base..pin_base+3
};

Args parse_args(int argc, char** argv) {
    Args a;
    for (int i = 1; i < argc; ++i) {
        if (std::strcmp(argv[i], "--md-port") == 0 && i + 1 < argc) a.md_port = std::atoi(argv[++i]);
        else if (std::strcmp(argv[i], "--selftest") == 0) a.selftest = true;
        else if (std::strcmp(argv[i], "--md-rate") == 0 && i + 1 < argc) a.md_rate_hz = std::atoi(argv[++i]);
        else if (std::strcmp(argv[i], "--duration") == 0 && i + 1 < argc) a.duration_s = std::atoi(argv[++i]);
        else if (std::strcmp(argv[i], "--max-spread") == 0 && i + 1 < argc) a.max_spread_ticks = std::atoll(argv[++i]);
        else if (std::strcmp(argv[i], "--qty") == 0 && i + 1 < argc) a.order_qty = std::atoi(argv[++i]);
        else if (std::strcmp(argv[i], "--max-pos") == 0 && i + 1 < argc) a.max_abs_pos = std::atoi(argv[++i]);
        else if (std::strcmp(argv[i], "--fill-delay-ns") == 0 && i + 1 < argc) a.fill_delay_ns = (uint32_t)std::strtoul(argv[++i], nullptr, 10);
        else if (std::strcmp(argv[i], "--pin-base") == 0 && i + 1 < argc) a.pin_base = std::atoi(argv[++i]);
        else if (std::strcmp(argv[i], "--help") == 0 || std::strcmp(argv[i], "-h") == 0) {
            std::printf(
                "Usage: %s [--md-port 9000] [--selftest] [--md-rate 200000] [--duration 0]\n"
                "          [--max-spread 2] [--qty 1] [--max-pos 100] [--fill-delay-ns 0] [--pin-base -1]\n",
                argv[0]
            );
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
    setvbuf(stdout, nullptr, _IOLBF, 0);

    const Args args = parse_args(argc, argv);

    std::atomic<bool> running{true};
    g_running_ptr = &running;
    std::signal(SIGINT, on_signal);
    std::signal(SIGTERM, on_signal);

    // Optional: pin main thread too (keeps prints/timers stable-ish)
    if (args.pin_base >= 0) {
        hft::pin_this_thread_to_cpu(args.pin_base + 4);
    }

    // Queues
    hft::SpscRing<hft::MdUpdate, 1u << 16> md_q;
    hft::SpscRing<hft::Order, 1u << 14> ord_q;
    hft::SpscRing<hft::ExecReport, 1u << 14> er_q;

    const int rx_fd = make_udp_socket_bound(args.md_port);
    if (rx_fd < 0) {
        std::perror("bind md udp");
        return 1;
    }

    // Optional selftest MD publisher (localhost)
    std::thread md_pub;
    if (args.selftest) {
        md_pub = std::thread([&]() {
            if (args.pin_base >= 0) hft::pin_this_thread_to_cpu(args.pin_base + 3);
            const int tx_fd = make_udp_socket_connected("127.0.0.1", args.md_port);
            if (tx_fd < 0) {
                std::perror("connect md udp");
                return;
            }
            uint32_t seq = 1;
            int64_t bid = args.base_px;
            int64_t ask = args.base_px + 1;
            const int rate = (args.md_rate_hz > 0) ? args.md_rate_hz : 1;
            const uint64_t sleep_ns = 1000000000ull / (uint64_t)rate;

            while (running.load(std::memory_order_relaxed)) {
                const uint64_t t = hft::now_ns();
                // alternate bid/ask updates
                hft::MdUpdate m{};
                m.ts_send_ns = t;
                m.seq = seq++;
                const bool is_bid = (seq & 1u) != 0;
                m.side = is_bid ? 0 : 1;
                if (is_bid) bid += 0;
                else ask = bid + 1;
                m.price_ticks = is_bid ? bid : ask;
                m.qty = 1;
                (void)::send(tx_fd, &m, sizeof(m), 0);

                const uint64_t start = hft::now_ns();
                while (hft::now_ns() - start < sleep_ns) {
                    // spin
                }
            }
            ::close(tx_fd);
        });
    }

    std::thread feed([&]() {
        if (args.pin_base >= 0) hft::pin_this_thread_to_cpu(args.pin_base + 0);
        while (running.load(std::memory_order_relaxed)) {
            hft::MdUpdate m{};
            const ssize_t n = ::recv(rx_fd, &m, sizeof(m), 0);
            if (n == (ssize_t)sizeof(m)) {
                (void)md_q.try_push(m);  // drop if full
            } else if (n < 0 && errno == EINTR) {
                continue;
            }
        }
    });

    std::thread exec_sim([&]() {
        if (args.pin_base >= 0) hft::pin_this_thread_to_cpu(args.pin_base + 2);
        hft::ExecSimConfig cfg{};
        cfg.fill_delay_ns = args.fill_delay_ns;
        hft::ExecutionSim sim(cfg);
        int32_t position = 0;
        while (running.load(std::memory_order_relaxed)) {
            hft::Order o{};
            if (!ord_q.try_pop(o)) continue;
            const hft::ExecReport er = sim.on_order(o, position);
            (void)er_q.try_push(er);
        }
    });

    std::thread strategy([&]() {
        if (args.pin_base >= 0) hft::pin_this_thread_to_cpu(args.pin_base + 1);
        hft::SeqTracker seq;
        hft::TopOfBook book;
        hft::LatencyStats md_lat;
        hft::LatencyStats rt_lat;

        hft::RiskConfig rcfg{};
        rcfg.max_abs_position = args.max_abs_pos;
        rcfg.max_order_qty = args.max_order_qty;
        hft::RiskState rstate{};
        hft::OrderManager om(rcfg);

        uint64_t md_count = 0, ord_count = 0, er_count = 0;
        uint64_t last_print = hft::now_ns();
        const uint64_t start = last_print;

        while (running.load(std::memory_order_relaxed)) {
            // Consume MD
            hft::MdUpdate m{};
            if (md_q.try_pop(m)) {
                const uint64_t now = hft::now_ns();
                md_lat.add(now - m.ts_send_ns);
                seq.on_seq(m.seq);
                ++md_count;

                if (m.side == 0) book.on_bid(m.price_ticks, m.qty);
                else book.on_ask(m.price_ticks, m.qty);

                // Trivial strategy: if spread small, cross to buy at ask / sell at bid to mean-revert
                if (book.has_both() && book.spread_ticks() <= args.max_spread_ticks) {
                    // Simple “ping-pong”: if position <=0 buy 1 at ask, else sell 1 at bid
                    hft::Order o{};
                    if (rstate.position <= 0) {
                        if (om.prepare_and_check(o, rstate, hft::Side::Buy, book.ask_px(), args.order_qty)) {
                            (void)ord_q.try_push(o);
                            ++ord_count;
                        }
                    } else {
                        if (om.prepare_and_check(o, rstate, hft::Side::Sell, book.bid_px(), args.order_qty)) {
                            (void)ord_q.try_push(o);
                            ++ord_count;
                        }
                    }
                }
            }

            // Consume exec reports
            hft::ExecReport er{};
            while (er_q.try_pop(er)) {
                const uint64_t now = hft::now_ns();
                rt_lat.add(now - er.ts_send_ns);
                om.apply_exec(rstate, er);
                ++er_count;
            }

            const uint64_t now = hft::now_ns();
            if (now - last_print >= 1000000000ull) {
                std::printf(
                    "md/s=%llu ord/s=%llu er/s=%llu q(md=%u ord=%u er=%u) pos=%d rejects=%llu gaps=%llu dup=%llu ",
                    (unsigned long long)md_count,
                    (unsigned long long)ord_count,
                    (unsigned long long)er_count,
                    (unsigned)md_q.approx_size(),
                    (unsigned)ord_q.approx_size(),
                    (unsigned)er_q.approx_size(),
                    (int)rstate.position,
                    (unsigned long long)rstate.rejects,
                    (unsigned long long)seq.gaps,
                    (unsigned long long)seq.dup_or_old
                );
                md_lat.print_summary("md_lat");
                rt_lat.print_summary("rt_lat");
                md_count = ord_count = er_count = 0;
                md_lat.reset();
                rt_lat.reset();
                last_print = now;
            }

            if (args.duration_s > 0 && (now - start) >= (uint64_t)args.duration_s * 1000000000ull) {
                running.store(false, std::memory_order_relaxed);
            }
        }
    });

    std::printf("hft_engine md_port=%d selftest=%s\n", args.md_port, args.selftest ? "on" : "off");
    std::printf("Press Ctrl+C to stop.\n");

    strategy.join();
    running.store(false, std::memory_order_relaxed);
    feed.join();
    exec_sim.join();
    if (md_pub.joinable()) md_pub.join();
    ::close(rx_fd);
    return 0;
}

