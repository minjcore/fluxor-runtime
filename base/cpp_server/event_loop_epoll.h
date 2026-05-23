// epoll Event Loop Implementation (Linux)
#pragma once

#include "event_loop.h"
#include <sys/epoll.h>
#include <unordered_map>

class EpollEventLoop : public EventLoop {
public:
    EpollEventLoop();
    ~EpollEventLoop() override;

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
    bool ctl_add_or_mod(int fd, uint32_t events, void* ptr);
    bool set_interest(int fd, uint32_t interest, void* ptr);
    void process_listener_events();
    void process_conn_event(TCPConnection* conn, uint32_t events);

    int epfd_ = -1;
    int listener_fd_ = -1;
    int eventfd_ = -1; // wakeup on shutdown
    bool accepting_ = true;

    // Fast fd->conn mapping for unregister (and safety)
    std::unordered_map<int, TCPConnection*> fd_to_conn_;
    std::unordered_map<int, uint32_t> fd_interest_;

    static constexpr int MAX_EVENTS = 1024;
    epoll_event events_[MAX_EVENTS];
};

