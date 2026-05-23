// io_uring Event Loop Implementation (Linux)

#pragma once

#include "event_loop.h"
#include <liburing.h>
#include <cstdint>
#include <memory>

class IOUringEventLoop : public EventLoop {
public:
    IOUringEventLoop();
    ~IOUringEventLoop() override;
    
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
    void process_completions();
    void submit_initial_accepts();
    void handle_completion(struct io_uring_cqe* cqe);
    void flush_submit();
    void submit_deferred_accepts();
    io_uring_sqe* get_sqe_or_flush();
    
    io_uring ring_;
    int listener_fd_ = -1;
    bool accepting_ = true;
    unsigned pending_submit_ = 0; // number of SQEs queued but not submitted
    unsigned deferred_accepts_ = 0; // accepts to (re)post when SQE available

    // Wakeup mechanism to break io_uring_submit_and_wait on shutdown (like epoll/kqueue).
    int wakeup_fd_ = -1;
    bool wakeup_poll_armed_ = false;
    static constexpr uint64_t kUserDataAccept = 0;
    static constexpr uint64_t kUserDataWakeup = 1; // sentinel (conn pointers are 64B-aligned)

    static constexpr int RING_SIZE = 4096;
    static constexpr int INITIAL_ACCEPTS = 20;
};
