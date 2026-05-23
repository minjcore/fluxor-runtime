// io_uring Event Loop Implementation (Linux)

#include "event_loop_uring.h"
#include "tcp_layer.h"
#include "config.h"
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <string.h>
#include <stdio.h>
#include <sys/socket.h>
#include <sys/eventfd.h>
#include <poll.h>

IOUringEventLoop::IOUringEventLoop() {
    ring_ = {};
    ring_.ring_fd = -1;
}

IOUringEventLoop::~IOUringEventLoop() {
    cleanup();
}

bool IOUringEventLoop::setup() {
    // SYSCALL: io_uring_setup() - creates shared memory rings between userspace and kernel
    // NOTE: We intentionally do NOT enable SQPOLL by default.
    // It may require privileges / cause unstable latency on some systems.
    unsigned int flags = 0;
    
    int ring_size = g_config.ring_size;
    const int rc = io_uring_queue_init(ring_size, &ring_, flags);
    if (rc < 0) {
        errno = -rc;
        perror("io_uring_queue_init");
        return false;
    }

    // Wakeup fd so shutdown() can interrupt io_uring_submit_and_wait().
    wakeup_fd_ = eventfd(0, EFD_NONBLOCK | EFD_CLOEXEC);
    if (wakeup_fd_ < 0) {
        perror("eventfd");
        io_uring_queue_exit(&ring_);
        ring_.ring_fd = -1;
        return false;
    }

    // Arm a POLL_ADD so an eventfd write generates a CQE.
    io_uring_sqe* sqe = io_uring_get_sqe(&ring_);
    if (!sqe) {
        fprintf(stderr, "io_uring_get_sqe failed during setup\n");
        ::close(wakeup_fd_);
        wakeup_fd_ = -1;
        io_uring_queue_exit(&ring_);
        ring_.ring_fd = -1;
        return false;
    }
    io_uring_prep_poll_add(sqe, wakeup_fd_, POLLIN);
    io_uring_sqe_set_data64(sqe, kUserDataWakeup);
    pending_submit_++;
    wakeup_poll_armed_ = true;
    flush_submit();
    return true;
}

void IOUringEventLoop::cleanup() {
    if (wakeup_fd_ >= 0) {
        ::close(wakeup_fd_);
        wakeup_fd_ = -1;
    }
    wakeup_poll_armed_ = false;

    if (ring_.ring_fd >= 0) {
        io_uring_queue_exit(&ring_);
        ring_.ring_fd = -1;
    }
}

bool IOUringEventLoop::register_listener(int listener_fd) {
    listener_fd_ = listener_fd;
    accepting_ = true;
    
    // Submit initial accepts - use special marker to indicate accept operations
    // TCPServer will handle creating connections when accepts complete
    int initial_accepts = g_config.initial_accepts;
    deferred_accepts_ += (unsigned)initial_accepts;
    submit_deferred_accepts();
    flush_submit();
    return true;
}

bool IOUringEventLoop::set_accepting(bool enable) {
    if (listener_fd_ < 0) return false;
    if (accepting_ == enable) return true;
    accepting_ = enable;

    if (accepting_) {
        int initial_accepts = g_config.initial_accepts;
        deferred_accepts_ += (unsigned)initial_accepts;
        submit_deferred_accepts();
        flush_submit();
    }
    return true;
}

bool IOUringEventLoop::register_read(TCPConnection* conn) {
    // Backpressure: don't read while a write is in-flight.
    if (!conn || !conn->is_active() || conn->is_reading() || conn->is_writing()) return false;
    
    size_t remaining = conn->buffer_capacity() - conn->read_pos();
    if (remaining == 0) return false;
    
    io_uring_sqe* sqe = get_sqe_or_flush();
    if (!sqe) {
        // Fallback: do a best-effort synchronous recv so we don't deadlock the connection.
        // This matches the epoll/kqueue backends' "try immediate IO first" behavior.
        conn->set_reading(true);
        ssize_t n = ::recv(conn->fd(),
                           conn->read_buffer() + conn->read_pos(),
                           remaining, 0);
        conn->set_reading(false);
        if (!server_ || !event_callback_) return false;
        if (n >= 0) {
            EventData ev{};
            ev.conn = conn;
            ev.type = EventType::READ;
            ev.result = n;
            ev.error_code = 0;
            event_callback_(server_, ev);
            return true;
        }
        if (errno == EAGAIN || errno == EWOULDBLOCK || errno == EINTR) {
            // Try again later; caller can re-arm read.
            return false;
        }
        EventData ev{};
        ev.conn = conn;
        ev.type = EventType::ERROR;
        ev.result = -1;
        ev.error_code = errno;
        event_callback_(server_, ev);
        return false;
    }
    
    conn->set_reading(true);
    // For sockets, prefer RECV over READ (more correct / often faster).
    io_uring_prep_recv(sqe, conn->fd(),
                       conn->read_buffer() + conn->read_pos(),
                       remaining, 0);
    io_uring_sqe_set_data64(sqe, reinterpret_cast<uint64_t>(conn));
    pending_submit_++;
    return true;
}

bool IOUringEventLoop::register_write(TCPConnection* conn) {
    if (!conn || !conn->is_active() || !conn->is_writing()) return false;
    
    size_t remaining = conn->write_len() - conn->write_pos();
    if (remaining == 0) return false;
    
    io_uring_sqe* sqe = get_sqe_or_flush();
    if (!sqe) {
        // Fallback: best-effort synchronous send to avoid deadlock under SQE starvation.
        ssize_t n = ::send(conn->fd(),
                           conn->write_buffer() + conn->write_pos(),
                           remaining, 0);
        if (n > 0) {
            conn->advance_write_pos((size_t)n);
            if (!server_ || !event_callback_) return true;
            EventData ev{};
            ev.conn = conn;
            ev.type = EventType::WRITE;
            ev.result = n;
            ev.error_code = 0;
            event_callback_(server_, ev);
            return true;
        }
        if (n < 0 && (errno == EAGAIN || errno == EWOULDBLOCK || errno == EINTR)) {
            // We'll try again later.
            return false;
        }
        // fatal
        conn->set_writing(false);
        if (server_ && event_callback_) {
            EventData ev{};
            ev.conn = conn;
            ev.type = EventType::ERROR;
            ev.result = -1;
            ev.error_code = (n < 0) ? errno : EIO;
            event_callback_(server_, ev);
        }
        return false;
    }
    
    // For sockets, prefer SEND over WRITE.
    io_uring_prep_send(sqe, conn->fd(),
                       conn->write_buffer() + conn->write_pos(),
                       remaining, 0);
    io_uring_sqe_set_data64(sqe, reinterpret_cast<uint64_t>(conn));
    pending_submit_++;
    return true;
}

void IOUringEventLoop::unregister_connection(TCPConnection* conn) {
    // io_uring doesn't need explicit unregistration
    // Connection will be closed by TCPServer
    (void)conn;
}

void IOUringEventLoop::run() {
    running_ = true;
    
    while (running_ && server_ && TCPServer::global_running_.load()) {
        // Keep accept SQEs topped up (best-effort).
        submit_deferred_accepts();

        // Drain all currently available completions.
        process_completions();

        if (!running_ || !TCPServer::global_running_.load()) break;

        // Submit any queued SQEs (best-effort).
        if (pending_submit_ > 0) {
            io_uring_submit(&ring_);
            pending_submit_ = 0;
        }

        // Wait with timeout so we can observe global shutdown (like epoll/kqueue).
        // This prevents hanging if SIGINT is delivered to a different thread.
        struct io_uring_cqe* cqe = nullptr;
        __kernel_timespec ts;
        ts.tv_sec = 0;
        ts.tv_nsec = 10 * 1000 * 1000; // 10ms
        const int ret = io_uring_wait_cqe_timeout(&ring_, &cqe, &ts);
        if (ret == 0 && cqe) {
            handle_completion(cqe);
            continue;
        }
        if (ret == -ETIME) {
            continue; // periodic wake to check shutdown flags
        }
        if (ret == -EINTR) {
            if (!running_ || !TCPServer::global_running_.load()) break;
            continue;
        }
        // Other errors: exit loop.
        break;
    }
}

void IOUringEventLoop::shutdown() {
    running_ = false;
    // Best-effort wakeup so io_uring_submit_and_wait returns promptly.
    if (wakeup_fd_ >= 0) {
        uint64_t one = 1;
        (void)::write(wakeup_fd_, &one, sizeof(one));
    }
}

void IOUringEventLoop::process_completions() {
    // Process all available completions from kernel
    struct io_uring_cqe* cqe;
    while (io_uring_peek_cqe(&ring_, &cqe) == 0) {
        handle_completion(cqe);
    }
}

void IOUringEventLoop::handle_completion(struct io_uring_cqe* cqe) {
    const uint64_t udata = io_uring_cqe_get_data64(cqe);
    TCPConnection* conn = nullptr;
    if (udata != kUserDataAccept && udata != kUserDataWakeup) {
        conn = reinterpret_cast<TCPConnection*>(static_cast<uintptr_t>(udata));
    }
    int res = cqe->res;
    io_uring_cqe_seen(&ring_, cqe);
    
    if (!server_ || !event_callback_) return;

    if (udata == kUserDataWakeup) {
        // Drain eventfd and exit loop soon.
        if (wakeup_fd_ >= 0) {
            uint64_t v;
            (void)::read(wakeup_fd_, &v, sizeof(v));
        }
        return;
    }
    
    EventData event;
    event.result = res;
    event.error_code = (res < 0) ? -res : 0;
    
    if (res < 0) {
        // Error (negative errno)
        const int err = -res;
        if (udata == kUserDataAccept) {
            // Failed accept - re-submit accept
            if (accepting_) deferred_accepts_++;
        } else {
            // Transient non-blocking conditions are NORMAL for sockets.
            // Do NOT bubble these up as fatal errors.
            if (err == EAGAIN || err == EWOULDBLOCK || err == EINTR) {
                if (conn->is_reading()) {
                    conn->set_reading(false);
                    register_read(conn);
                } else if (conn->is_writing()) {
                    // Keep writing flag; just try again
                    register_write(conn);
                }
                return;
            }

            // Fatal connection error
            if (conn->is_reading()) conn->set_reading(false);
            if (conn->is_writing()) conn->set_writing(false);
            event.conn = conn;
            event.type = EventType::ERROR;
            event_callback_(server_, event);
        }
    } else if (udata == kUserDataAccept) {
        // Accept completion - fd is in res
        if (!accepting_) {
            if (res >= 0) ::close(res);
            return;
        }
        event.conn = nullptr;  // TCPServer will create connection
        event.type = EventType::ACCEPT;
        event.result = res;  // New file descriptor
        event_callback_(server_, event);
        
        // Re-submit accept to keep accepting
        if (accepting_) deferred_accepts_++;
    } else if (conn->is_reading()) {
        // Read completion
        conn->set_reading(false);
        event.conn = conn;
        event.type = EventType::READ;
        event_callback_(server_, event);
    } else if (conn->is_writing()) {
        // Write completion
        if (res > 0) {
            conn->advance_write_pos(res);
            if (conn->write_pos() >= conn->write_len()) {
                conn->set_writing(false);
            }
        } else {
            conn->set_writing(false);
        }
        event.conn = conn;
        event.type = EventType::WRITE;
        event_callback_(server_, event);
    }
}

void IOUringEventLoop::submit_initial_accepts() {
    // Already done in register_listener
}

void IOUringEventLoop::flush_submit() {
    if (pending_submit_ == 0) return;
    // Ignore return value; errors will be handled by subsequent CQEs/errno.
    io_uring_submit(&ring_);
    pending_submit_ = 0;
}

io_uring_sqe* IOUringEventLoop::get_sqe_or_flush() {
    io_uring_sqe* sqe = io_uring_get_sqe(&ring_);
    if (sqe) return sqe;
    flush_submit();
    return io_uring_get_sqe(&ring_);
}

void IOUringEventLoop::submit_deferred_accepts() {
    if (!accepting_ || listener_fd_ < 0 || deferred_accepts_ == 0) return;
    while (deferred_accepts_ > 0) {
        io_uring_sqe* sqe = get_sqe_or_flush();
        if (!sqe) break;
        io_uring_prep_accept(sqe, listener_fd_, nullptr, nullptr, 0);
        io_uring_sqe_set_data64(sqe, kUserDataAccept);
        pending_submit_++;
        deferred_accepts_--;
    }
}
