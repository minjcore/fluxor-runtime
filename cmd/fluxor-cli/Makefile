# Makefile for fluxor-cli (C++ bridge lives in base/private/cpp_bridge)

GO = go
TARGET = fluxor-cli

all: $(TARGET)

# Build: CGO compiles base/private/cpp_bridge when building
$(TARGET):
	$(GO) build -o $(TARGET) .

example: $(TARGET)
	./$(TARGET) cpp-example

clean:
	rm -f $(TARGET)

install: $(TARGET)
	$(GO) install .

test: $(TARGET)
	@echo "Testing Go-C++ integration..."
	./$(TARGET) cpp-example

.PHONY: all example clean install test
