// Event Loop Factory Implementation

#include "event_loop.h"
#include <memory>

#include "platform.h"

#if PLATFORM_LINUX
#include "event_loop_epoll.h"
#include "event_loop_uring.h"
#elif PLATFORM_MACOS
#include "event_loop_kqueue.h"
#else
#error "Unsupported platform"
#endif

std::unique_ptr<EventLoop> create_event_loop() {
#if PLATFORM_LINUX
    // Default to epoll on Linux for best HTTP socket throughput.
    // io_uring network can be enabled later via a config flag if needed.
    return std::make_unique<EpollEventLoop>();
#elif PLATFORM_MACOS
    return std::make_unique<KqueueEventLoop>();
#else
#error "Unsupported platform"
#endif
}
