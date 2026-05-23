// Platform macros (single source of truth)
#pragma once

#if defined(__linux__)
#  define PLATFORM_LINUX 1
#  define PLATFORM_MACOS 0
#elif defined(__APPLE__)
#  define PLATFORM_LINUX 0
#  define PLATFORM_MACOS 1
#else
#  error "Unsupported platform (need Linux or macOS)"
#endif

