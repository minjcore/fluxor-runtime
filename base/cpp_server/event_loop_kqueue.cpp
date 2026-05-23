// kqueue Event Loop Implementation (macOS)

#include "event_loop_kqueue.h"
#include "tcp_layer.h"
#include "config.h"
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <string.h>
#include <stdio.h>
#include <sys/time.h>
#include <sys/socket.h>
#include <algorithm>

KqueueEventLoop::KqueueEventLoop() {
}

KqueueEventLoop::~KqueueEventLoop() {
    cleanup();
}

bool KqueueEventLoop::setup() {
    // SYSCALL: kqueue() - creates kernel event queue
    kq_ = kqueue();
    if (kq_ < 0) {
        perror("kqueue");
        return false;
    }
    
    // Create self-pipe for wakeup on shutdown
    if (pipe(wakeup_pipe_) < 0) {
        perror("pipe");
        ::close(kq_);
        kq_ = -1;
        return false;
    }
    
    // Set pipe to non-blocking
    fcntl(wakeup_pipe_[0], F_SETFL, O_NONBLOCK);
    fcntl(wakeup_pipe_[1], F_SETFL, O_NONBLOCK);
    
    // Register wakeup pipe read end with kqueue
    struct kevent kev;
    EV_SET(&kev, wakeup_pipe_[0], EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0, nullptr);
    if (kevent(kq_, &kev, 1, nullptr, 0, nullptr) < 0) {
        perror("kevent: wakeup_pipe");
        ::close(wakeup_pipe_[0]);
        ::close(wakeup_pipe_[1]);
        wakeup_pipe_[0] = -1;
        wakeup_pipe_[1] = -1;
        ::close(kq_);
        kq_ = -1;
        return false;
    }
    
    adaptive_timeout_ms_ = g_config.adaptive_timeout_ms;
    consecutive_idle_loops_ = 0;
    return true;
}

void KqueueEventLoop::cleanup() {
    // Close wakeup pipe
    if (wakeup_pipe_[0] >= 0) {
        ::close(wakeup_pipe_[0]);
        wakeup_pipe_[0] = -1;
    }
    if (wakeup_pipe_[1] >= 0) {
        ::close(wakeup_pipe_[1]);
        wakeup_pipe_[1] = -1;
    }
    
    if (kq_ >= 0) {
        ::close(kq_);
        kq_ = -1;
    }
    fd_to_conn_.clear();
    pending_events_.clear();
}

bool KqueueEventLoop::register_listener(int listener_fd) {
    listener_fd_ = listener_fd;
    accepting_ = true;
    
    // SYSCALL: kevent() - register event filter with kernel
    struct kevent kev;
    EV_SET(&kev, listener_fd_, EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0, nullptr);
    if (kevent(kq_, &kev, 1, nullptr, 0, nullptr) < 0) {
        perror("kevent: register_listener");
        return false;
    }
    return true;
}

bool KqueueEventLoop::set_accepting(bool enable) {
    if (listener_fd_ < 0) return false;
    if (accepting_ == enable) return true;

    struct kevent kev;
    EV_SET(&kev, listener_fd_, EVFILT_READ, EV_ADD | (enable ? EV_ENABLE : EV_DISABLE), 0, 0, nullptr);
    pending_events_.push_back(kev);
    flush_pending_events(); // apply immediately
    accepting_ = enable;
    return true;
}

bool KqueueEventLoop::register_read(TCPConnection* conn) {
    if (!conn || !conn->is_active() || conn->is_reading()) return false;
    
    conn->set_reading(true);
    fd_to_conn_[conn->fd()] = conn;
    
    // Use EV_ADD | EV_ENABLE | EV_ONESHOT for read events
    struct kevent kev;
    EV_SET(&kev, conn->fd(), EVFILT_READ, EV_ADD | EV_ENABLE | EV_ONESHOT, 0, 0, conn);
    pending_events_.push_back(kev);
    
    // Flush immediately for reads to avoid delays
    if (pending_events_.size() >= static_cast<size_t>(g_config.flush_threshold)) {
        flush_pending_events();
    }
    return true;
}

bool KqueueEventLoop::register_write(TCPConnection* conn) {
    if (!conn || !conn->is_active() || !conn->is_writing()) return false;
    
    fd_to_conn_[conn->fd()] = conn;
    
    // Cache values to avoid repeated function calls
    int fd = conn->fd();
    size_t write_pos = conn->write_pos();
    size_t write_len = conn->write_len();
    char* write_buf = conn->write_buffer();
    
    // Try writing immediately first
    ssize_t n = write(fd, write_buf + write_pos, write_len - write_pos);
    if (n > 0) {
        conn->advance_write_pos(n);
        size_t new_pos = write_pos + n;
        if (new_pos >= write_len) {
            // Write complete: report completion and let TCPServer reset + re-arm read.
            if (server_ && event_callback_) {
                EventData event;
                event.conn = conn;
                event.type = EventType::WRITE;
                event.result = n;
                event.error_code = 0;
                event_callback_(server_, event);
            }
            return true;
        }
        // Partial write - register for write event
    } else if (n < 0 && errno != EAGAIN && errno != EWOULDBLOCK) {
        // Error
        conn->set_writing(false);
        if (server_ && event_callback_) {
            EventData event;
            event.conn = conn;
            event.type = EventType::ERROR;
            event.result = -1;
            event.error_code = errno;
            event_callback_(server_, event);
        }
        return false;
    }
    
    // Register for write event (would block or partial write)
    struct kevent kev;
    EV_SET(&kev, fd, EVFILT_WRITE, EV_ADD | EV_ENABLE | EV_ONESHOT, 0, 0, conn);
    pending_events_.push_back(kev);
    
    // Flush immediately for writes to minimize latency (flush on every write)
    flush_pending_events();
    return true;
}

void KqueueEventLoop::unregister_connection(TCPConnection* conn) {
    if (!conn) return;
    
    int fd = conn->fd();
    fd_to_conn_.erase(fd);
    
    // Remove from kqueue
    struct kevent kev1, kev2;
    EV_SET(&kev1, fd, EVFILT_READ, EV_DELETE, 0, 0, nullptr);
    EV_SET(&kev2, fd, EVFILT_WRITE, EV_DELETE, 0, 0, nullptr);
    pending_events_.push_back(kev1);
    pending_events_.push_back(kev2);
    
    if (pending_events_.size() >= static_cast<size_t>(g_config.flush_threshold)) {
        flush_pending_events();
    }
}

void KqueueEventLoop::run() {
    running_ = true;
    
    // Increased batch size for better throughput
    struct kevent events[BATCH_SIZE];
    
    while (running_ && server_ && TCPServer::global_running_.load()) {
        // Flush any pending events before waiting
        flush_pending_events();
        
        // Check shutdown flag before blocking
        if (!running_ || !TCPServer::global_running_.load()) {
            break;
        }
        
        // Adaptive timeout: increase when idle, decrease when active
        struct timespec timeout;
        timeout.tv_sec = 0;
        timeout.tv_nsec = adaptive_timeout_ms_ * 1000000;  // Convert ms to nanoseconds
        
        // SYSCALL: kevent() - wait for kernel events
        int nev = kevent(kq_, nullptr, 0, events, BATCH_SIZE, &timeout);
        if (nev < 0) {
            if (errno == EINTR) {
                if (!running_ || !TCPServer::global_running_.load()) break;
                continue;
            }
            perror("kevent");
            break;
        }
        
        if (nev == 0) {
            // Timeout - no events, check shutdown flag
            if (!running_ || !TCPServer::global_running_.load()) {
                break;
            }
            // Increase timeout for next iteration
            consecutive_idle_loops_++;
            if (consecutive_idle_loops_ > 10) {
                adaptive_timeout_ms_ = std::min(g_config.max_adaptive_timeout_ms, 
                                                adaptive_timeout_ms_ + 10);
            }
            continue;
        }
        
        // Got events - reset adaptive timeout and idle counter
        consecutive_idle_loops_ = 0;
        // Reduce timeout immediately when active to minimize latency
        adaptive_timeout_ms_ = 1;  // Minimal timeout when active (1ms = 1,000,000ns)
        
        // Process events
        for (int i = 0; i < nev; i++) {
            // Check if this is wakeup pipe event (shutdown signal)
            if (events[i].ident == static_cast<uintptr_t>(wakeup_pipe_[0])) {
                // Wakeup pipe triggered - read to clear it
                char buf[256];
                read(wakeup_pipe_[0], buf, sizeof(buf));
                // Break out of loop (shutdown requested)
                break;
            }
            process_event(events[i]);
        }
        
        // Check shutdown flag after processing events
        if (!running_ || !TCPServer::global_running_.load()) {
            break;
        }
        
        // Flush pending events
        if (pending_events_.size() >= static_cast<size_t>(g_config.flush_threshold)) {
            flush_pending_events();
        }
    }
    
    // Final flush
    flush_pending_events();
}

void KqueueEventLoop::shutdown() {
    running_ = false;
    // Wake up kevent() by writing to wakeup pipe
    if (wakeup_pipe_[1] >= 0) {
        char c = 1;
        write(wakeup_pipe_[1], &c, 1);
    }
}

void KqueueEventLoop::flush_pending_events() {
    if (pending_events_.empty()) return;
    
    // Batch submit all pending events
    kevent(kq_, pending_events_.data(), pending_events_.size(), nullptr, 0, nullptr);
    pending_events_.clear();
}

void KqueueEventLoop::process_event(const struct kevent& kev) {
    int fd = (int)kev.ident;
    int filter = kev.filter;
    int flags = kev.flags;
    
    if (flags & EV_EOF) {
        if (fd == listener_fd_) return;
        auto it = fd_to_conn_.find(fd);
        if (it != fd_to_conn_.end()) {
            if (server_ && event_callback_) {
                EventData event;
                event.conn = it->second;
                event.type = EventType::ERROR;
                event.result = -1;
                event.error_code = ECONNRESET;
                event_callback_(server_, event);
            }
        }
        return;
    }
    
    if (fd == listener_fd_) {
        if (!accepting_) return;
        // Accept connections - handle bursts aggressively
        int accept_count = 0;
        int max_accepts = g_config.max_accepts_per_event;
        
        while (accept_count < max_accepts) {
            int new_fd = accept(listener_fd_, nullptr, nullptr);
            if (new_fd < 0) {
                if (errno == EAGAIN || errno == EWOULDBLOCK) {
                    break;  // No more connections
                }
                if (errno == ECONNABORTED) {
                    continue;  // Client aborted, try next
                }
                break;  // Other error
            }
            
            // Create connection (will be handled by TCPServer)
            if (server_ && event_callback_) {
                EventData event;
                event.conn = nullptr;  // TCPServer will create connection
                event.type = EventType::ACCEPT;
                event.result = new_fd;  // New file descriptor
                event.error_code = 0;
                event_callback_(server_, event);
            }
            accept_count++;
        }
    } else {
        // Cache connection pointer after lookup to avoid repeated finds
        auto it = fd_to_conn_.find(fd);
        if (it == fd_to_conn_.end()) return;
        TCPConnection* conn = it->second;  // Cache pointer
        
        if (filter == EVFILT_READ) {
            if (!conn->is_reading()) {
                // Stale event
                return;
            }
            
            // SYSCALL: read() - kernel copies data from socket buffer to userspace
            ssize_t n = read(conn->fd(), conn->read_buffer() + conn->read_pos(),
                            conn->buffer_capacity() - conn->read_pos());
            if (n > 0) {
                conn->set_reading(false);
                if (server_ && event_callback_) {
                    EventData event;
                    event.conn = conn;
                    event.type = EventType::READ;
                    event.result = n;
                    event.error_code = 0;
                    event_callback_(server_, event);
                }
            } else if (n == 0) {
                conn->set_reading(false);
                if (server_ && event_callback_) {
                    EventData event;
                    event.conn = conn;
                    event.type = EventType::ERROR;
                    event.result = -1;
                    event.error_code = ECONNRESET;
                    event_callback_(server_, event);
                }
            } else {
                // Error or EAGAIN/EWOULDBLOCK
                if (errno == EAGAIN || errno == EWOULDBLOCK) {
                    // Would block - re-enable read event
                    conn->set_reading(true);
                    struct kevent kev;
                    EV_SET(&kev, fd, EVFILT_READ, EV_ADD | EV_ENABLE | EV_ONESHOT, 0, 0, conn);
                    pending_events_.push_back(kev);
                } else {
                    // Real error
                    conn->set_reading(false);
                    if (server_ && event_callback_) {
                        EventData event;
                        event.conn = conn;
                        event.type = EventType::ERROR;
                        event.result = -1;
                        event.error_code = errno;
                        event_callback_(server_, event);
                    }
                }
            }
        } else if (filter == EVFILT_WRITE) {
            if (!conn->is_writing()) {
                // Stale event
                return;
            }
            
            // SYSCALL: write() - kernel copies data from userspace to socket buffer
            ssize_t n = write(conn->fd(), conn->write_buffer() + conn->write_pos(),
                             conn->write_len() - conn->write_pos());
            if (n > 0) {
                conn->advance_write_pos(n);
                
                // Check if write is complete
                if (conn->write_pos() >= conn->write_len()) {
                    conn->set_writing(false);
                    if (server_ && event_callback_) {
                        EventData event;
                        event.conn = conn;
                        event.type = EventType::WRITE;
                        event.result = n;
                        event.error_code = 0;
                        event_callback_(server_, event);
                    }
                } else {
                    // Partial write - re-enable write event
                    struct kevent kev;
                    EV_SET(&kev, fd, EVFILT_WRITE, EV_ADD | EV_ENABLE | EV_ONESHOT, 0, 0, conn);
                    pending_events_.push_back(kev);
                }
            } else if (n < 0) {
                if (errno == EAGAIN || errno == EWOULDBLOCK) {
                    // Would block - re-enable write event
                    struct kevent kev;
                    EV_SET(&kev, fd, EVFILT_WRITE, EV_ADD | EV_ENABLE | EV_ONESHOT, 0, 0, conn);
                    pending_events_.push_back(kev);
                } else {
                    // Real error
                    conn->set_writing(false);
                    if (server_ && event_callback_) {
                        EventData event;
                        event.conn = conn;
                        event.type = EventType::ERROR;
                        event.result = -1;
                        event.error_code = errno;
                        event_callback_(server_, event);
                    }
                }
            }
        }
    }
}
