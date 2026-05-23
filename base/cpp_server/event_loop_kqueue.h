// kqueue Event Loop Implementation (macOS)

#pragma once

#include "event_loop.h"
#include <sys/event.h>
#include <vector>
#include <unordered_map>

class KqueueEventLoop : public EventLoop {
public:
    KqueueEventLoop();
    ~KqueueEventLoop() override;
    
    bool setup() override;
    void cleanup() override;
    bool register_listener(int listener_fd) override;
    bool set_accepting(bool enable) override;
    bool register_read(TCPConnection* conn) override;
    bool register_write(TCPConnection* conn) override;
    void unregister_connection(TCPConnection* conn) override;
    void run() override;
    void shutdown() override;
    
private:
    void flush_pending_events();
    void process_event(const struct kevent& kev);
    
    int kq_ = -1;
    int listener_fd_ = -1;
    bool accepting_ = true;
    int wakeup_pipe_[2] = {-1, -1};  // Self-pipe for wakeup on shutdown
    std::unordered_map<int, TCPConnection*> fd_to_conn_;  // Faster than std::map for lookups
    
    // Adaptive timeout for CPU optimization
    int adaptive_timeout_ms_ = 10;
    int consecutive_idle_loops_ = 0;
    
    // Batch kevent operations to reduce syscall overhead
    std::vector<struct kevent> pending_events_;
    static constexpr int BATCH_SIZE = 256;
    static constexpr int FLUSH_THRESHOLD = 8;
};
