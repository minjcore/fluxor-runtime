#include <cstdio>
#include <cstring>

#include "../http_layer.h"

static int failures = 0;

static void expect_true(const char* name, bool v) {
    if (!v) {
        std::printf("FAIL: %s expected true\n", name);
        failures++;
    }
}

static void expect_false(const char* name, bool v) {
    if (v) {
        std::printf("FAIL: %s expected false\n", name);
        failures++;
    }
}

int main() {
    // Too short
    expect_false("len<4 empty", HTTPParser::is_complete_request("", 0));
    expect_false("len<4 short", HTTPParser::is_complete_request("GET", 3));

    // Basic complete request
    const char* req1 = "GET /test HTTP/1.1\r\nHost: x\r\n\r\n";
    expect_true("basic complete", HTTPParser::is_complete_request(req1, std::strlen(req1)));

    // Missing terminator
    const char* req2 = "GET /test HTTP/1.1\r\nHost: x\r\n";
    expect_false("missing crlfcrlf", HTTPParser::is_complete_request(req2, std::strlen(req2)));

    // Terminator in middle with extra bytes
    const char* req3 = "GET / HTTP/1.1\r\nA: b\r\n\r\nGARBAGE";
    expect_true("terminator + extra", HTTPParser::is_complete_request(req3, std::strlen(req3)));

    // Pattern split at end boundary
    const char* req4 = "X\r\n\r\n";
    expect_true("pattern at end", HTTPParser::is_complete_request(req4, std::strlen(req4)));

    // Ensure false when only partial terminator present
    const char* req5 = "GET / HTTP/1.1\r\n\r\n\r";
    expect_true("has full terminator even with trailing", HTTPParser::is_complete_request(req5, std::strlen(req5)));
    const char* req6 = "GET / HTTP/1.1\r\n\r";
    expect_false("partial terminator", HTTPParser::is_complete_request(req6, std::strlen(req6)));

    if (failures == 0) {
        std::printf("OK: test_http_parser\n");
        return 0;
    }
    std::printf("FAILURES: %d\n", failures);
    return 1;
}

