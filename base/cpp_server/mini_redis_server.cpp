// Mini Redis-like server built on cpp-server architecture (TCPServer + EventLoop)
// Implements a small subset of Redis over RESP (RESP2):
//   PING [msg], ECHO msg, GET key, SET key value [EX seconds|PX ms], DEL key..., EXISTS key...
//   INCR key, EXPIRE key seconds, TTL key, QUIT
//
// Notes:
// - Designed for interactive testing (redis-cli) and simple benchmarks.
// - Supports persistent connections and basic pipelining.
//   IMPORTANT: To preserve correctness with the current TCP layer, we only flush responses
//   when the read buffer contains a fully-parseable sequence of RESP commands.
// - Storage is shared across worker threads via sharded mutex map.

#include "tcp_layer.h"
#include "config.h"

#include <array>
#include <atomic>
#include <chrono>
#include <cctype>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <memory>
#include <mutex>
#include <signal.h>
#include <string>
#include <thread>
#include <unordered_map>
#include <utility>
#include <vector>

static inline uint64_t now_ms() {
    using namespace std::chrono;
    return duration_cast<milliseconds>(steady_clock::now().time_since_epoch()).count();
}

static inline std::string to_upper_ascii(std::string s) {
    for (char& c : s) {
        if (c >= 'a' && c <= 'z') c = (char)(c - 'a' + 'A');
    }
    return s;
}

static inline uint64_t fnv1a_64(const char* data, size_t len) {
    uint64_t h = 1469598103934665603ULL;
    for (size_t i = 0; i < len; i++) {
        h ^= (uint8_t)data[i];
        h *= 1099511628211ULL;
    }
    return h;
}

struct Entry {
    std::string value;
    uint64_t expire_at_ms = 0; // 0 = no expiry
};

class Store {
public:
    static constexpr size_t kShards = 256;

    Store() {
        for (auto& s : shards_) {
            s.map.reserve(4096);
        }
    }

    bool get(const std::string& key, std::string& out) {
        Shard& sh = shard_for(key);
        std::lock_guard<std::mutex> lock(sh.mu);
        auto it = sh.map.find(key);
        if (it == sh.map.end()) return false;
        if (is_expired_locked(it->second)) {
            sh.map.erase(it);
            return false;
        }
        out = it->second.value;
        return true;
    }

    void set(const std::string& key, std::string value, uint64_t expire_at) {
        Shard& sh = shard_for(key);
        std::lock_guard<std::mutex> lock(sh.mu);
        Entry& e = sh.map[key];
        e.value = std::move(value);
        e.expire_at_ms = expire_at;
    }

    int del(const std::vector<std::string>& keys, size_t start_idx) {
        int removed = 0;
        for (size_t i = start_idx; i < keys.size(); i++) {
            removed += del_one(keys[i]) ? 1 : 0;
        }
        return removed;
    }

    bool exists(const std::string& key) {
        Shard& sh = shard_for(key);
        std::lock_guard<std::mutex> lock(sh.mu);
        auto it = sh.map.find(key);
        if (it == sh.map.end()) return false;
        if (is_expired_locked(it->second)) {
            sh.map.erase(it);
            return false;
        }
        return true;
    }

    // TTL semantics (Redis):
    // -2 if key does not exist
    // -1 if key exists but has no associated expire
    // >=0 seconds to expire (rounded down)
    long long ttl_seconds(const std::string& key) {
        Shard& sh = shard_for(key);
        std::lock_guard<std::mutex> lock(sh.mu);
        auto it = sh.map.find(key);
        if (it == sh.map.end()) return -2;
        if (is_expired_locked(it->second)) {
            sh.map.erase(it);
            return -2;
        }
        if (it->second.expire_at_ms == 0) return -1;
        uint64_t now = now_ms();
        if (it->second.expire_at_ms <= now) {
            sh.map.erase(it);
            return -2;
        }
        return (long long)((it->second.expire_at_ms - now) / 1000ULL);
    }

    // EXPIRE key seconds:
    // 1 if the timeout was set
    // 0 if key does not exist
    int expire_seconds(const std::string& key, uint64_t seconds) {
        Shard& sh = shard_for(key);
        std::lock_guard<std::mutex> lock(sh.mu);
        auto it = sh.map.find(key);
        if (it == sh.map.end()) return 0;
        if (is_expired_locked(it->second)) {
            sh.map.erase(it);
            return 0;
        }
        it->second.expire_at_ms = now_ms() + seconds * 1000ULL;
        return 1;
    }

    // INCR key:
    // Treat missing as 0, set to 1. Error if not integer.
    // Returns (ok, value) in pair; ok=false indicates type error.
    std::pair<bool, long long> incr(const std::string& key) {
        Shard& sh = shard_for(key);
        std::lock_guard<std::mutex> lock(sh.mu);
        auto it = sh.map.find(key);
        long long cur = 0;
        uint64_t existing_expire_at = 0;
        if (it != sh.map.end()) {
            if (is_expired_locked(it->second)) {
                sh.map.erase(it);
                it = sh.map.end();
            } else {
                existing_expire_at = it->second.expire_at_ms;
                if (!parse_ll(it->second.value, cur)) {
                    return {false, 0};
                }
            }
        }
        long long next = cur + 1;
        Entry& e = sh.map[key];
        e.value = std::to_string(next);
        // Keep existing expiry if present, otherwise none.
        e.expire_at_ms = existing_expire_at;
        return {true, next};
    }

private:
    struct Shard {
        std::mutex mu;
        std::unordered_map<std::string, Entry> map;
    };

    std::array<Shard, kShards> shards_{};

    static bool parse_ll(const std::string& s, long long& out) {
        if (s.empty()) return false;
        char* end = nullptr;
        errno = 0;
        long long v = std::strtoll(s.c_str(), &end, 10);
        if (errno != 0) return false;
        if (!end || *end != '\0') return false;
        out = v;
        return true;
    }

    bool is_expired_locked(const Entry& e) const {
        return e.expire_at_ms != 0 && e.expire_at_ms <= now_ms();
    }

    Shard& shard_for(const std::string& key) {
        uint64_t h = fnv1a_64(key.data(), key.size());
        return shards_[(size_t)(h & (kShards - 1))];
    }

    bool del_one(const std::string& key) {
        Shard& sh = shard_for(key);
        std::lock_guard<std::mutex> lock(sh.mu);
        auto it = sh.map.find(key);
        if (it == sh.map.end()) return false;
        if (is_expired_locked(it->second)) {
            sh.map.erase(it);
            return false;
        }
        sh.map.erase(it);
        return true;
    }
};

// RESP helpers
static inline std::string resp_simple(const char* s) {
    std::string out;
    out.reserve(std::strlen(s) + 3);
    out.push_back('+');
    out.append(s);
    out.append("\r\n");
    return out;
}

static inline std::string resp_error(const char* s) {
    std::string out;
    out.reserve(std::strlen(s) + 3);
    out.push_back('-');
    out.append(s);
    out.append("\r\n");
    return out;
}

static inline std::string resp_integer(long long v) {
    std::string out;
    out.reserve(32);
    out.push_back(':');
    out.append(std::to_string(v));
    out.append("\r\n");
    return out;
}

static inline std::string resp_bulk(const std::string& s) {
    std::string out;
    out.reserve(s.size() + 64);
    out.push_back('$');
    out.append(std::to_string((long long)s.size()));
    out.append("\r\n");
    out.append(s);
    out.append("\r\n");
    return out;
}

static inline std::string resp_null_bulk() {
    return "$-1\r\n";
}

struct RespParseResult {
    bool ok = false;
    bool incomplete = false;
    size_t consumed = 0;
    std::vector<std::string> args;
    std::string error;
};

static inline bool scan_crlf(const char* buf, size_t len, size_t start, size_t& line_end) {
    // Find "\r\n" starting at 'start'. Returns index of '\r' in line_end.
    for (size_t i = start; i + 1 < len; i++) {
        if (buf[i] == '\r' && buf[i + 1] == '\n') {
            line_end = i;
            return true;
        }
    }
    return false;
}

static inline bool parse_int64_ascii(const char* buf, size_t start, size_t end, long long& out) {
    // parse buf[start:end) as int64 (no CRLF)
    if (end <= start) return false;
    bool neg = false;
    size_t i = start;
    if (buf[i] == '-') { neg = true; i++; }
    if (i >= end) return false;
    long long v = 0;
    for (; i < end; i++) {
        char c = buf[i];
        if (c < '0' || c > '9') return false;
        v = v * 10 + (c - '0');
    }
    out = neg ? -v : v;
    return true;
}

static RespParseResult parse_resp_command(const char* buf, size_t len) {
    RespParseResult r;
    if (len == 0) {
        r.incomplete = true;
        return r;
    }

    if (buf[0] == '*') {
        // Array of bulk strings
        size_t line_end = 0;
        if (!scan_crlf(buf, len, 0, line_end)) {
            r.incomplete = true;
            return r;
        }
        long long n = 0;
        if (!parse_int64_ascii(buf, 1, line_end, n) || n < 0) {
            r.ok = false;
            r.error = "ERR Protocol error: invalid multibulk length";
            return r;
        }
        size_t pos = line_end + 2;
        r.args.reserve((size_t)n);
        for (long long i = 0; i < n; i++) {
            if (pos >= len) { r.incomplete = true; return r; }
            if (buf[pos] != '$') {
                r.ok = false;
                r.error = "ERR Protocol error: expected '$'";
                return r;
            }
            size_t bulk_line_end = 0;
            if (!scan_crlf(buf, len, pos, bulk_line_end)) {
                r.incomplete = true;
                return r;
            }
            long long bulk_len = 0;
            if (!parse_int64_ascii(buf, pos + 1, bulk_line_end, bulk_len) || bulk_len < -1) {
                r.ok = false;
                r.error = "ERR Protocol error: invalid bulk length";
                return r;
            }
            pos = bulk_line_end + 2;
            if (bulk_len == -1) {
                // Null bulk string; represent as empty string (rare for commands)
                r.args.emplace_back();
                continue;
            }
            if (pos + (size_t)bulk_len + 2 > len) {
                r.incomplete = true;
                return r;
            }
            r.args.emplace_back(buf + pos, (size_t)bulk_len);
            pos += (size_t)bulk_len;
            if (buf[pos] != '\r' || buf[pos + 1] != '\n') {
                r.ok = false;
                r.error = "ERR Protocol error: missing CRLF after bulk string";
                return r;
            }
            pos += 2;
        }
        r.ok = true;
        r.consumed = pos;
        return r;
    }

    // Inline command: e.g. "PING\r\n" or "GET key\r\n"
    size_t line_end = 0;
    if (!scan_crlf(buf, len, 0, line_end)) {
        r.incomplete = true;
        return r;
    }
    // Split by spaces/tabs
    size_t i = 0;
    while (i < line_end) {
        while (i < line_end && (buf[i] == ' ' || buf[i] == '\t')) i++;
        if (i >= line_end) break;
        size_t j = i;
        while (j < line_end && buf[j] != ' ' && buf[j] != '\t') j++;
        r.args.emplace_back(buf + i, j - i);
        i = j;
    }
    r.ok = true;
    r.consumed = line_end + 2;
    return r;
}

class MiniRedisServer {
public:
    explicit MiniRedisServer(std::shared_ptr<Store> store) : store_(std::move(store)) {}

    void on_accept(TCPServer* server, TCPConnection* conn) {
        // Reset per-connection state (TCPConnection objects can be reused from pool).
        ConnState& st = states_[conn];
        st.pending_out.clear();
        st.close_after_write = false;
        server->submit_read(conn);
    }

    void on_read(TCPServer* server, TCPConnection* conn, ssize_t bytes_read) {
        if (bytes_read <= 0) {
            close_conn(server, conn);
            return;
        }

        const char* data = conn->read_buffer();
        size_t n = conn->read_pos();

        // Buffer invariant guard (fail-closed).
        if (n > conn->buffer_capacity()) {
            close_conn(server, conn);
            return;
        }

        ConnState& st = states_[conn];
        if (st.close_after_write) {
            // Client sent QUIT earlier; ignore further input.
            close_conn(server, conn);
            return;
        }

        // Parse as many complete commands as available.
        // To keep correctness with TCPServer's "reset read_pos after write complete",
        // we only flush responses when we can fully parse the current read buffer.
        size_t pos = 0;
        std::string batch_out;
        batch_out.reserve(256);
        bool saw_quit = false;

        while (pos < n) {
            RespParseResult pr = parse_resp_command(data + pos, n - pos);
            if (pr.incomplete) {
                break;
            }
            if (!pr.ok || pr.args.empty()) {
                // Protocol error: fail-closed to avoid desync.
                close_conn(server, conn);
                return;
            }

            std::string cmd = to_upper_ascii(pr.args[0]);
            batch_out.append(dispatch(cmd, pr.args));
            if (cmd == "QUIT") saw_quit = true;

            pos += pr.consumed;

            // Prevent unbounded response growth (write buffer is bounded).
            if (st.pending_out.size() + batch_out.size() > conn->buffer_capacity()) {
                close_conn(server, conn);
                return;
            }
        }

        if (pos < n) {
            // Incomplete trailing command: do NOT flush responses yet.
            // Keep responses accumulated so far in per-connection state.
            st.pending_out.append(batch_out);

            // Preserve the unconsumed bytes by shifting them to the front.
            // Buffer invariant: remaining <= capacity.
            const size_t remaining = n - pos;
            if (remaining > 0 && pos > 0) {
                std::memmove(conn->read_buffer(), conn->read_buffer() + pos, remaining);
            }
            conn->set_read_pos(remaining);

            if (conn->read_pos() >= conn->buffer_capacity()) {
                close_conn(server, conn);
                return;
            }
            server->submit_read(conn);
            return;
        }

        // Fully parsed current buffer: safe to flush all responses now.
        std::string out;
        out.reserve(st.pending_out.size() + batch_out.size());
        out.append(st.pending_out);
        out.append(batch_out);
        st.pending_out.clear();

        // Keep invariant: we've consumed all input.
        conn->set_read_pos(0);

        if (out.empty()) {
            // No command parsed (should not happen unless n==0), just keep reading.
            server->submit_read(conn);
            return;
        }

        if (saw_quit) {
            // Close after write completion (handled in on_write()).
            st.close_after_write = true;
        }
        safe_write(server, conn, out);
    }

    void on_write(TCPServer* server, TCPConnection* conn, ssize_t bytes_written) {
        if (bytes_written < 0) {
            close_conn(server, conn);
            return;
        }
        // NOTE: With the TCP layer change, on_write is invoked for both partial and complete writes.
        auto it = states_.find(conn);
        if (it != states_.end() && it->second.close_after_write) {
            // Only close once the write is complete.
            if (conn->write_pos() >= conn->write_len()) {
                close_conn(server, conn);
                return;
            }
        }
        if (conn->write_pos() < conn->write_len()) {
            server->submit_write_continue(conn);
        }
    }

    void on_error(TCPServer* server, TCPConnection* conn, int /* error */) {
        if (conn) close_conn(server, conn);
    }

private:
    struct ConnState {
        std::string pending_out;
        bool close_after_write = false;
    };

    std::shared_ptr<Store> store_;
    std::unordered_map<TCPConnection*, ConnState> states_;

    void close_conn(TCPServer* server, TCPConnection* conn) {
        if (!conn) return;
        states_.erase(conn);
        server->close_connection(conn);
    }

    static void safe_write(TCPServer* server, TCPConnection* conn, const std::string& resp) {
        // Avoid silent truncation inside submit_write().
        size_t cap = conn->buffer_capacity();
        if (resp.size() > cap) {
            const char* msg = "-ERR response too large\r\n";
            server->submit_write(conn, msg, std::strlen(msg));
            return;
        }
        server->submit_write(conn, resp.data(), resp.size());
    }

    std::string dispatch(const std::string& cmd, const std::vector<std::string>& args) {
        // Commands are case-insensitive; cmd already uppercase.
        if (cmd == "PING") {
            if (args.size() == 1) return resp_simple("PONG");
            if (args.size() == 2) return resp_bulk(args[1]);
            return resp_error("ERR wrong number of arguments for 'ping' command");
        }
        if (cmd == "ECHO") {
            if (args.size() != 2) return resp_error("ERR wrong number of arguments for 'echo' command");
            return resp_bulk(args[1]);
        }
        if (cmd == "GET") {
            if (args.size() != 2) return resp_error("ERR wrong number of arguments for 'get' command");
            std::string v;
            if (!store_->get(args[1], v)) return resp_null_bulk();
            return resp_bulk(v);
        }
        if (cmd == "SET") {
            if (args.size() < 3) return resp_error("ERR wrong number of arguments for 'set' command");
            if (args.size() != 3 && args.size() != 5) {
                return resp_error("ERR syntax error");
            }
            const std::string& key = args[1];
            const std::string& val = args[2];

            // Keep response within buffer; also keep values reasonably small by default.
            if (val.size() > 4096) {
                return resp_error("ERR value too large (max 4096 bytes)");
            }

            uint64_t expire_at = 0;
            if (args.size() == 5) {
                std::string opt = to_upper_ascii(args[3]);
                long long t = 0;
                if (args[4].empty()) return resp_error("ERR value is not an integer or out of range");
                char* end = nullptr;
                errno = 0;
                t = std::strtoll(args[4].c_str(), &end, 10);
                if (errno != 0 || !end || *end != '\0' || t <= 0) {
                    return resp_error("ERR value is not an integer or out of range");
                }
                uint64_t now = now_ms();
                if (opt == "EX") {
                    expire_at = now + (uint64_t)t * 1000ULL;
                } else if (opt == "PX") {
                    expire_at = now + (uint64_t)t;
                } else {
                    return resp_error("ERR syntax error");
                }
            }
            store_->set(key, val, expire_at);
            return resp_simple("OK");
        }
        if (cmd == "DEL") {
            if (args.size() < 2) return resp_error("ERR wrong number of arguments for 'del' command");
            int removed = store_->del(args, 1);
            return resp_integer(removed);
        }
        if (cmd == "EXISTS") {
            if (args.size() < 2) return resp_error("ERR wrong number of arguments for 'exists' command");
            long long count = 0;
            for (size_t i = 1; i < args.size(); i++) {
                if (store_->exists(args[i])) count++;
            }
            return resp_integer(count);
        }
        if (cmd == "INCR") {
            if (args.size() != 2) return resp_error("ERR wrong number of arguments for 'incr' command");
            auto res = store_->incr(args[1]);
            if (!res.first) return resp_error("ERR value is not an integer or out of range");
            return resp_integer(res.second);
        }
        if (cmd == "EXPIRE") {
            if (args.size() != 3) return resp_error("ERR wrong number of arguments for 'expire' command");
            char* end = nullptr;
            errno = 0;
            long long sec = std::strtoll(args[2].c_str(), &end, 10);
            if (errno != 0 || !end || *end != '\0' || sec < 0) {
                return resp_error("ERR value is not an integer or out of range");
            }
            int ok = store_->expire_seconds(args[1], (uint64_t)sec);
            return resp_integer(ok);
        }
        if (cmd == "TTL") {
            if (args.size() != 2) return resp_error("ERR wrong number of arguments for 'ttl' command");
            return resp_integer(store_->ttl_seconds(args[1]));
        }
        if (cmd == "QUIT") {
            if (args.size() != 1) return resp_error("ERR wrong number of arguments for 'quit' command");
            return resp_simple("OK");
        }

        return resp_error("ERR unknown command");
    }
};

static void signal_handler(int /*sig*/) {
    TCPServer::global_running_ = false;
}

static void print_usage(const char* prog) {
    printf("Usage: %s [options]\n", prog);
    printf("\nBasic Options:\n");
    printf("  -p, --port PORT          Server port (default: 6380)\n");
    printf("  -w, --workers N          Number of worker threads (default: 1, 0 = auto-detect)\n");
    printf("  -h, --help               Show this help message\n");
    printf("\nConnection Options:\n");
    printf("  --max-connections N      Maximum concurrent connections (default: 10000)\n");
    printf("  --pool-size N            Initial connection pool size (default: 500)\n");
    printf("  --backlog N              Listen backlog size (default: 4096)\n");
    printf("  --buffer-size N          Buffer size per connection (default: 8192)\n");
    #if PLATFORM_LINUX
    printf("\nLinux (io_uring) Options:\n");
    printf("  --ring-size N            io_uring ring size (default: 4096)\n");
    printf("  --initial-accepts N      Initial accept operations (default: 20)\n");
    printf("  --busy-poll-us N         SO_BUSY_POLL busy polling (default: 0)\n");
    #elif PLATFORM_MACOS
    printf("\nmacOS (kqueue) Options:\n");
    printf("  --kqueue-batch N         Kqueue batch size (default: 256)\n");
    printf("  --flush-threshold N      Event flush threshold (default: 8)\n");
    printf("  --timeout N              Adaptive timeout in ms (default: 10)\n");
    printf("  --max-accepts N          Max accepts per event (default: 50)\n");
    #endif
    printf("\nSupported Commands:\n");
    printf("  PING [msg], ECHO msg, GET key, SET key value [EX s|PX ms], DEL key..., EXISTS key...\n");
    printf("  INCR key, EXPIRE key seconds, TTL key, QUIT\n");
    printf("\n");
}

static void worker_thread(int thread_id, int port, const std::shared_ptr<Store>& store) {
    MiniRedisServer redis(store);
    TCPServer tcp_server(thread_id);
    if (!tcp_server.setup(port)) {
        fprintf(stderr, "Thread %d: Failed to setup TCP server\n", thread_id);
        return;
    }

    tcp_server.set_accept_callback([&redis](TCPServer* s, TCPConnection* c) {
        redis.on_accept(s, c);
    });
    tcp_server.set_read_callback([&redis](TCPServer* s, TCPConnection* c, ssize_t n) {
        redis.on_read(s, c, n);
    });
    tcp_server.set_write_callback([&redis](TCPServer* s, TCPConnection* c, ssize_t n) {
        redis.on_write(s, c, n);
    });
    tcp_server.set_error_callback([&redis](TCPServer* s, TCPConnection* c, int e) {
        redis.on_error(s, c, e);
    });

    tcp_server.run();
}

int main(int argc, char* argv[]) {
    // Set a Redis-like default port before parsing args.
    g_config.port = 6380;

    if (!g_config.parse_args(argc, argv)) {
        print_usage(argv[0]);
        return 1;
    }

    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    #if PLATFORM_LINUX
    printf("Starting mini-redis server (io_uring)\n");
    #elif PLATFORM_MACOS
    printf("Starting mini-redis server (kqueue)\n");
    #endif
    g_config.print();
    printf("\n");

    auto store = std::make_shared<Store>();

    std::vector<std::thread> workers;
    workers.reserve((size_t)g_config.num_workers);
    for (int i = 0; i < g_config.num_workers; i++) {
        workers.emplace_back(worker_thread, i, g_config.port, store);
    }

    for (auto& t : workers) t.join();

    printf("Server shutdown complete\n");
    return 0;
}
