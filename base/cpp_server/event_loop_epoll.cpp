// epoll Event Loop Implementation (Linux)

#include "event_loop_epoll.h"
#include "tcp_layer.h"
#include "config.h"

#include <sys/eventfd.h>
#include <sys/socket.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>
#include <stdio.h>

EpollEventLoop::EpollEventLoop() {}

EpollEventLoop::~EpollEventLoop() {
    cleanup();
}

namespace {
// Store tags in epoll_event.data.u64
constexpr uint64_t kEpollTagListener = 1;
constexpr uint64_t kEpollTagEventFd  = 2;
} // namespace

bool EpollEventLoop::setup() {
    epfd_ = epoll_create1(EPOLL_CLOEXEC);
    if (epfd_ < 0) {
        perror("epoll_create1");
        return false;
    }

    // eventfd for wakeup on shutdown
    eventfd_ = eventfd(0, EFD_NONBLOCK | EFD_CLOEXEC);
    if (eventfd_ < 0) {
        perror("eventfd");
        ::close(epfd_);
        epfd_ = -1;
        return false;
    }

    // Register eventfd for read
    epoll_event ev{};
    ev.events = EPOLLIN;
    ev.data.u64 = kEpollTagEventFd;
    if (epoll_ctl(epfd_, EPOLL_CTL_ADD, eventfd_, &ev) < 0) {
        perror("epoll_ctl(eventfd)");
        ::close(eventfd_);
        ::close(epfd_);
        eventfd_ = -1;
        epfd_ = -1;
        return false;
    }

    running_ = true;
    return true;
}

void EpollEventLoop::cleanup() {
    fd_to_conn_.clear();
    if (eventfd_ >= 0) {
        ::close(eventfd_);
        eventfd_ = -1;
    }
    if (epfd_ >= 0) {
        ::close(epfd_);
        epfd_ = -1;
    }
    listener_fd_ = -1;
}

bool EpollEventLoop::register_listener(int listener_fd) {
    listener_fd_ = listener_fd;
    accepting_ = true;
    epoll_event ev{};
    ev.events = EPOLLIN | EPOLLET; // edge-triggered accept loop
    ev.data.u64 = kEpollTagListener;
    if (epoll_ctl(epfd_, EPOLL_CTL_ADD, listener_fd_, &ev) < 0) {
        perror("epoll_ctl(listener)");
        return false;
    }
    return true;
}

bool EpollEventLoop::set_accepting(bool enable) {
    if (listener_fd_ < 0) return false;
    if (accepting_ == enable) return true;

    if (!enable) {
        // Stop receiving accept events.
        epoll_ctl(epfd_, EPOLL_CTL_DEL, listener_fd_, nullptr);
        accepting_ = false;
        return true;
    }

    // Re-register listener for accepts.
    epoll_event ev{};
    ev.events = EPOLLIN | EPOLLET;
    ev.data.u64 = kEpollTagListener;
    if (epoll_ctl(epfd_, EPOLL_CTL_ADD, listener_fd_, &ev) < 0) {
        return false;
    }
    accepting_ = true;
    return true;
}

bool EpollEventLoop::ctl_add_or_mod(int fd, uint32_t events, void* ptr) {
    epoll_event ev{};
    ev.events = events;
    ev.data.u64 = static_cast<uint64_t>(reinterpret_cast<uintptr_t>(ptr));
    if (epoll_ctl(epfd_, EPOLL_CTL_MOD, fd, &ev) == 0) return true;
    if (errno == ENOENT) {
        return epoll_ctl(epfd_, EPOLL_CTL_ADD, fd, &ev) == 0;
    }
    return false;
}

bool EpollEventLoop::set_interest(int fd, uint32_t interest, void* ptr) {
    auto it = fd_interest_.find(fd);
    if (it != fd_interest_.end() && it->second == interest) {
        return true; // no change
    }
    if (!ctl_add_or_mod(fd, interest, ptr)) {
        return false;
    }
    fd_interest_[fd] = interest;
    return true;
}

bool EpollEventLoop::register_read(TCPConnection* conn) {
    // Backpressure: don't read while a write is in-flight.
    if (!conn || !conn->is_active() || conn->is_reading() || conn->is_writing()) return false;
    conn->set_reading(true);
    fd_to_conn_[conn->fd()] = conn;

    // Correctness-first: only enable READ interest when we're actually reading.
    // Also use level-triggered for connections to avoid ET edge-loss when
    // backpressure temporarily disables reading.
    uint32_t ev = EPOLLIN;
    if (!set_interest(conn->fd(), ev, conn)) {
        perror("epoll_ctl(register_read)");
        return false;
    }
    return true;
}

bool EpollEventLoop::register_write(TCPConnection* conn) {
    if (!conn || !conn->is_active() || !conn->is_writing()) return false;
    fd_to_conn_[conn->fd()] = conn;

    // Fast path: try to write immediately (like kqueue backend)
    const int fd = conn->fd();
    const size_t start_pos = conn->write_pos();
    while (conn->write_pos() < conn->write_len()) {
        size_t remaining = conn->write_len() - conn->write_pos();
        ssize_t n = send(fd, conn->write_buffer() + conn->write_pos(), remaining, 0);
        if (n > 0) {
            conn->advance_write_pos((size_t)n);
            continue;
        }
        if (n < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
            break; // need EPOLLOUT
        }
        // fatal error
        conn->set_writing(false);
        if (server_ && event_callback_) {
            EventData ev{};
            ev.conn = conn;
            ev.type = EventType::ERROR;
            ev.result = -1;
            ev.error_code = errno;
            event_callback_(server_, ev);
        }
        return false;
    }

    // Completed without needing EPOLLOUT
    if (conn->write_pos() >= conn->write_len()) {
        // Write complete: report completion and let TCPServer reset + re-arm read.
        if (server_ && event_callback_) {
            EventData ev{};
            ev.conn = conn;
            ev.type = EventType::WRITE;
            ev.result = (ssize_t)(conn->write_pos() - start_pos);
            ev.error_code = 0;
            event_callback_(server_, ev);
        }
        return true;
    }

    // Correctness/backpressure: do not keep EPOLLIN enabled while we are not reading.
    uint32_t ev = EPOLLOUT;
    if (!set_interest(conn->fd(), ev, conn)) {
        perror("epoll_ctl(register_write)");
        return false;
    }
    return true;
}

void EpollEventLoop::unregister_connection(TCPConnection* conn) {
    if (!conn) return;
    fd_to_conn_.erase(conn->fd());
    fd_interest_.erase(conn->fd());
    epoll_ctl(epfd_, EPOLL_CTL_DEL, conn->fd(), nullptr);
}

void EpollEventLoop::process_listener_events() {
    // Accept as many as possible (edge-triggered)
    while (true) {
        int fd = accept4(listener_fd_, nullptr, nullptr, SOCK_NONBLOCK | SOCK_CLOEXEC);
        if (fd < 0) {
            if (errno == EAGAIN || errno == EWOULDBLOCK) break;
            // transient accept errors: ignore
            break;
        }

        if (!server_ || !event_callback_) {
            ::close(fd);
            continue;
        }

        EventData ev{};
        ev.conn = nullptr;
        ev.type = EventType::ACCEPT;
        ev.result = fd;
        ev.error_code = 0;
        event_callback_(server_, ev);
    }
}

void EpollEventLoop::process_conn_event(TCPConnection* conn, uint32_t events) {
    if (!conn || !conn->is_active() || !server_ || !event_callback_) return;

    const int fd = conn->fd();

    if (events & (EPOLLERR | EPOLLHUP)) {
        int err = 0;
        socklen_t len = sizeof(err);
        getsockopt(fd, SOL_SOCKET, SO_ERROR, &err, &len);
        EventData ev{};
        ev.conn = conn;
        ev.type = EventType::ERROR;
        ev.result = -1;
        ev.error_code = err ? err : EIO;
        // clear flags to allow close path
        conn->set_reading(false);
        conn->set_writing(false);
        event_callback_(server_, ev);
        return;
    }

    // Prefer handling WRITE first if both ready (reduces head-of-line blocking)
    if ((events & EPOLLOUT) && conn->is_writing()) {
        size_t remaining = conn->write_len() - conn->write_pos();
        if (remaining > 0) {
        // Drain socket writable state.
            ssize_t total = 0;
            while (conn->write_pos() + (size_t)total < conn->write_len()) {
                remaining = conn->write_len() - (conn->write_pos() + (size_t)total);
                ssize_t n = send(fd, conn->write_buffer() + conn->write_pos() + (size_t)total, remaining, 0);
                if (n > 0) {
                    total += n;
                    continue;
                }
                if (n < 0 && (errno == EAGAIN || errno == EWOULDBLOCK)) {
                    break; // wait for next EPOLLOUT
                }
                // fatal error
                EventData ev{};
                ev.conn = conn;
                ev.type = EventType::ERROR;
                ev.result = -1;
                ev.error_code = errno;
                conn->set_writing(false);
                conn->set_reading(false);
                event_callback_(server_, ev);
                return;
            }

            if (total > 0) {
                conn->advance_write_pos((size_t)total);
                if (conn->write_pos() >= conn->write_len()) {
                    // Write complete: report completion and let TCPServer reset + re-arm read.
                    EventData ev{};
                    ev.conn = conn;
                    ev.type = EventType::WRITE;
                    ev.result = total;
                    ev.error_code = 0;
                    event_callback_(server_, ev);
                    return;
                }
                EventData ev{};
                ev.conn = conn;
                ev.type = EventType::WRITE;
                ev.result = total;
                ev.error_code = 0;
                event_callback_(server_, ev);
            }
            return;
        }
    }

    if ((events & EPOLLIN) && conn->is_reading()) {
        size_t start = conn->read_pos();
        size_t remaining = conn->buffer_capacity() - start;
        if (remaining == 0) {
            conn->set_reading(false);
            EventData ev{};
            ev.conn = conn;
            ev.type = EventType::READ;
            ev.result = 0;
            ev.error_code = 0;
            event_callback_(server_, ev);
            return;
        }

        // Drain readable state. Accumulate into buffer without touching conn->read_pos();
        // TCPServer will advance it once by event.result.
        ssize_t total = 0;
        while (remaining > (size_t)total) {
            ssize_t n = recv(fd, conn->read_buffer() + start + (size_t)total, remaining - (size_t)total, 0);
            if (n > 0) {
                total += n;
                continue;
            }
            if (n == 0) {
                // EOF
                break;
            }
            if (errno == EAGAIN || errno == EWOULDBLOCK) {
                break;
            }
            // fatal error
            conn->set_reading(false);
            EventData ev{};
            ev.conn = conn;
            ev.type = EventType::ERROR;
            ev.result = -1;
            ev.error_code = errno;
            conn->set_writing(false);
            event_callback_(server_, ev);
            return;
        }

        conn->set_reading(false);
        EventData ev{};
        ev.conn = conn;
        ev.type = EventType::READ;
        ev.result = total; // may be 0 on EOF
        ev.error_code = 0;
        event_callback_(server_, ev);
        return;
    }
}

void EpollEventLoop::run() {
    running_ = true;
    while (running_ && server_ && TCPServer::global_running_.load()) {
        int timeout_ms = 10; // keep responsive; can make adaptive later
        int n = epoll_wait(epfd_, events_, MAX_EVENTS, timeout_ms);
        if (n < 0) {
            if (errno == EINTR) {
                if (!running_ || !TCPServer::global_running_.load()) break;
                continue;
            }
            perror("epoll_wait");
            break;
        }

        for (int i = 0; i < n; i++) {
            epoll_event& ev = events_[i];
            const uint64_t tag = ev.data.u64;

            // wakeup
            if (tag == kEpollTagEventFd) {
                uint64_t v;
                // eventfd read: best-effort; we just need to drain the wakeup
                const ssize_t rc = ::read(eventfd_, &v, sizeof(v));
                if (rc < 0) {
                    // Ignore errors here; worst case we spin until next wakeup.
                }
                continue;
            }

            // listener
            if (tag == kEpollTagListener) {
                process_listener_events();
                continue;
            }

            // connections store pointer value in u64
            TCPConnection* conn = reinterpret_cast<TCPConnection*>(static_cast<uintptr_t>(tag));
            process_conn_event(conn, ev.events);
        }
    }
}

void EpollEventLoop::shutdown() {
    running_ = false;
    if (eventfd_ >= 0) {
        uint64_t one = 1;
        // eventfd write: best-effort wakeup to break epoll_wait
        const ssize_t rc = ::write(eventfd_, &one, sizeof(one));
        if (rc < 0) {
            // Ignore errors on shutdown.
        }
    }
}

