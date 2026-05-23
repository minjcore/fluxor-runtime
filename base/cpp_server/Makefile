# Makefile for C++ HTTP server
# Linux: Requires liburing-dev (sudo apt-get install liburing-dev)
# macOS: Uses kqueue (built-in)

UNAME_S := $(shell uname -s)
TARGET = cpp_server
MINI_REDIS_TARGET = mini_redis_server
HFT_TARGET = hft_sim
HFT_ENGINE_TARGET = hft_engine
TEST_HTTP_PARSER_TARGET = test_http_parser
SRC = main.cpp tcp_layer.cpp http_layer.cpp cpu_layer.cpp memory_layer.cpp config.cpp cache_layer.cpp event_loop.cpp
HFT_SRC = hft_sim.cpp
HFT_ENGINE_SRC = hft_engine.cpp
TEST_HTTP_PARSER_SRC = tests/test_http_parser.cpp
MINI_REDIS_SRC = mini_redis_server.cpp

# Platform-specific event loop sources
ifeq ($(UNAME_S),Linux)
    SRC += event_loop_epoll.cpp event_loop_uring.cpp
else ifeq ($(UNAME_S),Darwin)
    SRC += event_loop_kqueue.cpp
endif

OBJ = $(SRC:.cpp=.o)
HFT_OBJ = $(HFT_SRC:.cpp=.o)
HFT_ENGINE_OBJ = $(HFT_ENGINE_SRC:.cpp=.o)
TEST_HTTP_PARSER_OBJ = $(TEST_HTTP_PARSER_SRC:.cpp=.o)
MINI_REDIS_OBJ = $(MINI_REDIS_SRC:.cpp=.o)

# Core objects for non-server binaries (exclude main.cpp to avoid duplicate main)
CORE_SRC = tcp_layer.cpp http_layer.cpp cpu_layer.cpp memory_layer.cpp config.cpp cache_layer.cpp event_loop.cpp
ifeq ($(UNAME_S),Linux)
    CORE_SRC += event_loop_epoll.cpp event_loop_uring.cpp
else ifeq ($(UNAME_S),Darwin)
    CORE_SRC += event_loop_kqueue.cpp
endif
CORE_OBJ = $(CORE_SRC:.cpp=.o)

# Platform-specific settings
ifeq ($(UNAME_S),Linux)
    CXX = g++
    CXXFLAGS = -O3 -std=c++20 -march=native -Wall -Wextra -D_GNU_SOURCE -msse4.2 -flto -funroll-loops -finline-functions
    LDFLAGS = -luring -pthread -flto
    CHECK_LIBURING := $(shell pkg-config --exists liburing && echo "yes" || echo "no")
else ifeq ($(UNAME_S),Darwin)
    CXX = clang++
    # SSE4.2 is x86-only, not available on ARM macOS - only enable on x86_64
    UNAME_M := $(shell uname -m)
    ifeq ($(UNAME_M),x86_64)
        CXXFLAGS = -O3 -std=c++20 -march=native -Wall -Wextra -msse4.2 -flto -funroll-loops -finline-functions
    else
        CXXFLAGS = -O3 -std=c++20 -march=native -Wall -Wextra -flto -funroll-loops -finline-functions
    endif
    LDFLAGS = -pthread -flto
    CHECK_LIBURING := "not_needed"
else
    $(error Unsupported platform: $(UNAME_S))
endif

.PHONY: all clean check install-deps pgo pgo-clean hft hft-engine test mini-redis

all: check $(TARGET)

hft: check $(HFT_TARGET)

hft-engine: check $(HFT_ENGINE_TARGET)

test: check $(TEST_HTTP_PARSER_TARGET)
	./$(TEST_HTTP_PARSER_TARGET)

mini-redis: check $(MINI_REDIS_TARGET)

check:
	@if [ "$(UNAME_S)" = "Linux" ]; then \
		if [ "$(CHECK_LIBURING)" != "yes" ]; then \
			echo "ERROR: liburing not found!"; \
			echo "Install with: sudo apt-get install liburing-dev"; \
			exit 1; \
		fi; \
		echo "liburing found ✓"; \
	elif [ "$(UNAME_S)" = "Darwin" ]; then \
		echo "Building for macOS (kqueue) ✓"; \
	fi

$(TARGET): $(OBJ)
	$(CXX) $(CXXFLAGS) -o $(TARGET) $(OBJ) $(LDFLAGS)
	@echo "Build complete: $(TARGET)"

$(HFT_TARGET): $(HFT_OBJ)
	$(CXX) $(CXXFLAGS) -o $(HFT_TARGET) $(HFT_OBJ) $(LDFLAGS)
	@echo "Build complete: $(HFT_TARGET)"

$(HFT_ENGINE_TARGET): $(HFT_ENGINE_OBJ)
	$(CXX) $(CXXFLAGS) -o $(HFT_ENGINE_TARGET) $(HFT_ENGINE_OBJ) $(LDFLAGS)
	@echo "Build complete: $(HFT_ENGINE_TARGET)"

$(TEST_HTTP_PARSER_TARGET): $(TEST_HTTP_PARSER_OBJ) $(CORE_OBJ)
	$(CXX) $(CXXFLAGS) -o $(TEST_HTTP_PARSER_TARGET) $(TEST_HTTP_PARSER_OBJ) $(CORE_OBJ) $(LDFLAGS)
	@echo "Build complete: $(TEST_HTTP_PARSER_TARGET)"

$(MINI_REDIS_TARGET): $(MINI_REDIS_OBJ) $(CORE_OBJ)
	$(CXX) $(CXXFLAGS) -o $(MINI_REDIS_TARGET) $(MINI_REDIS_OBJ) $(CORE_OBJ) $(LDFLAGS)
	@echo "Build complete: $(MINI_REDIS_TARGET)"

%.o: %.cpp
	$(CXX) $(CXXFLAGS) -c $< -o $@

clean:
	rm -f $(TARGET) $(OBJ) $(HFT_TARGET) $(HFT_OBJ) $(HFT_ENGINE_TARGET) $(HFT_ENGINE_OBJ) $(TEST_HTTP_PARSER_TARGET) $(TEST_HTTP_PARSER_OBJ) $(MINI_REDIS_TARGET) $(MINI_REDIS_OBJ)

install-deps:
	@echo "Installing liburing-dev..."
	@echo "Note: Skipping apt-get update to avoid PPA issues"
	@sudo apt-get install -y liburing-dev || (echo "Install failed. Try manually: sudo apt-get install -y liburing-dev" && exit 1)

run: $(TARGET)
	./$(TARGET)

benchmark: $(TARGET)
	@echo "Starting server in background..."
	@./$(TARGET) &
	@sleep 2
	@echo "Running benchmark..."
	wrk -c10 -t2 -d10 http://localhost:8083/test
	@pkill $(TARGET) || true

# Profile-Guided Optimization (PGO) build targets
pgo: pgo-clean
	@echo "=== PGO Build: Step 1 - Generate Profile ==="
	@echo "Building instrumented binary..."
	$(CXX) $(CXXFLAGS) -fprofile-generate -o $(TARGET).pgo $(SRC) $(LDFLAGS)
	@echo "Running benchmark to generate profile data..."
	@./$(TARGET).pgo &
	@sleep 2
	@wrk -c10 -t2 -d30s http://localhost:8083/test > /dev/null 2>&1 || true
	@pkill $(TARGET).pgo || true
	@sleep 1
	@echo "=== PGO Build: Step 2 - Optimized Build ==="
	@echo "Building optimized binary with profile data..."
	$(CXX) $(CXXFLAGS) -fprofile-use -fprofile-correction -o $(TARGET) $(SRC) $(LDFLAGS)
	@echo "PGO build complete! Binary: $(TARGET)"
	@rm -f $(TARGET).pgo *.gcda *.gcno 2>/dev/null || true

pgo-clean:
	@echo "Cleaning PGO files..."
	@rm -f $(TARGET).pgo *.gcda *.gcno 2>/dev/null || true
