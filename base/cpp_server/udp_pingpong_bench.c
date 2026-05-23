// UDP ping-pong benchmark for the mini "FH" protocol.
// Goal: correct KPI (p50/p99 + cycles/op) on 1 core, fixed payload, ping-pong.
//
// Build (Linux/macOS):
//   clang -O3 -Wall -Wextra -std=c11 udp_pingpong_bench.c -o udp_pingpong_bench
//
// Run:
//   ./udp_pingpong_bench --host 127.0.0.1 --port 6380 -n 1000000 --pin 0
//
// Protocol:
//   req (8B):  'F''H' ver=1 op=1 token(u32, network order)
//   resp(12B): 'F''H' ver=1 op=op|0x80 status token ts_ms(u32)

#include <arpa/inet.h>
#include <errno.h>
#include <inttypes.h>
#include <netinet/in.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <time.h>
#include <unistd.h>

#ifdef __linux__
#include <sched.h>
#endif

enum { FH_VER = 1, FH_OP_PING = 1 };

struct __attribute__((packed)) fh_req {
    uint8_t magic[2];
    uint8_t ver;
    uint8_t op;
    uint32_t token; // network order
};

static inline uint64_t nsec_now(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (uint64_t)ts.tv_sec * 1000000000ULL + (uint64_t)ts.tv_nsec;
}

#if defined(__x86_64__) || defined(__i386__)
static inline uint64_t rdtsc_ordered(void) {
    unsigned hi, lo;
    __asm__ __volatile__("lfence\nrdtsc" : "=a"(lo), "=d"(hi) :: "memory");
    return ((uint64_t)hi << 32) | lo;
}
#else
static inline uint64_t rdtsc_ordered(void) { return 0; }
#endif

static int cmp_u64(const void *a, const void *b) {
    uint64_t x = *(const uint64_t *)a;
    uint64_t y = *(const uint64_t *)b;
    return (x > y) - (x < y);
}

static void pin_to_cpu(int cpu) {
#ifdef __linux__
    cpu_set_t set;
    CPU_ZERO(&set);
    CPU_SET(cpu, &set);
    if (sched_setaffinity(0, sizeof(set), &set) != 0) {
        perror("sched_setaffinity");
    }
#else
    (void)cpu;
#endif
}

static void usage(const char *p) {
    fprintf(stderr, "Usage: %s --host IP --port P -n N [--pin CPU]\n", p);
}

int main(int argc, char **argv) {
    const char *host = "127.0.0.1";
    int port = 6380;
    uint64_t n = 1000000;
    int pin = -1;

    for (int i = 1; i < argc; i++) {
        if (!strcmp(argv[i], "--host") && i + 1 < argc) host = argv[++i];
        else if (!strcmp(argv[i], "--port") && i + 1 < argc) port = atoi(argv[++i]);
        else if (!strcmp(argv[i], "-n") && i + 1 < argc) n = (uint64_t)strtoull(argv[++i], NULL, 10);
        else if (!strcmp(argv[i], "--pin") && i + 1 < argc) pin = atoi(argv[++i]);
        else { usage(argv[0]); return 2; }
    }

    if (pin >= 0) pin_to_cpu(pin);

    int fd = socket(AF_INET, SOCK_DGRAM, 0);
    if (fd < 0) { perror("socket"); return 1; }

    struct sockaddr_in dst;
    memset(&dst, 0, sizeof(dst));
    dst.sin_family = AF_INET;
    dst.sin_port = htons((uint16_t)port);
    if (inet_pton(AF_INET, host, &dst.sin_addr) != 1) {
        fprintf(stderr, "bad --host\n");
        return 2;
    }

    // Tight timeout to fail fast on misconfig.
    struct timeval tv = {.tv_sec = 1, .tv_usec = 0};
    setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv));

    uint64_t *lat_ns = (uint64_t *)malloc((size_t)n * sizeof(uint64_t));
    uint64_t *cyc = (uint64_t *)malloc((size_t)n * sizeof(uint64_t));
    if (!lat_ns || !cyc) { fprintf(stderr, "oom\n"); return 1; }

    struct fh_req req = {.magic = {'F','H'}, .ver = FH_VER, .op = FH_OP_PING, .token = 0};
    uint8_t resp[64];

    uint64_t ok = 0;
    for (uint64_t i = 0; i < n; i++) {
        req.token = htonl((uint32_t)i);
        uint64_t t0 = nsec_now();
        uint64_t c0 = rdtsc_ordered();
        ssize_t s = sendto(fd, &req, sizeof(req), 0, (struct sockaddr *)&dst, sizeof(dst));
        if (s != (ssize_t)sizeof(req)) { perror("sendto"); break; }

        socklen_t sl = sizeof(dst);
        ssize_t r = recvfrom(fd, resp, sizeof(resp), 0, (struct sockaddr *)&dst, &sl);
        uint64_t c1 = rdtsc_ordered();
        uint64_t t1 = nsec_now();
        if (r < 0) { perror("recvfrom"); break; }
        if (r < 12) continue;
        if (resp[0] != 'F' || resp[1] != 'H' || resp[2] != FH_VER) continue;
        // token check (best-effort)
        uint32_t tok = 0;
        memcpy(&tok, resp + 5, 4);
        if (tok != req.token) continue;

        lat_ns[ok] = t1 - t0;
        cyc[ok] = (c0 && c1 && c1 > c0) ? (c1 - c0) : 0;
        ok++;
    }

    if (ok < 100) {
        fprintf(stderr, "too few samples: %" PRIu64 "\n", ok);
        return 1;
    }

    qsort(lat_ns, (size_t)ok, sizeof(uint64_t), cmp_u64);
    qsort(cyc, (size_t)ok, sizeof(uint64_t), cmp_u64);

    uint64_t p50 = lat_ns[(size_t)(ok * 50 / 100)];
    uint64_t p99 = lat_ns[(size_t)(ok * 99 / 100)];

    uint64_t c50 = cyc[(size_t)(ok * 50 / 100)];
    uint64_t c99 = cyc[(size_t)(ok * 99 / 100)];

    printf("FH UDP ping-pong: samples=%" PRIu64 " host=%s port=%d pin=%d\n", ok, host, port, pin);
    printf("latency: p50=%" PRIu64 " ns (%.3f us)  p99=%" PRIu64 " ns (%.3f us)\n",
           p50, (double)p50 / 1000.0, p99, (double)p99 / 1000.0);
    if (c99 > 0) {
        printf("cycles/op: p50=%" PRIu64 "  p99=%" PRIu64 "\n", c50, c99);
    } else {
        printf("cycles/op: unavailable (non-x86 or rdtsc unsupported)\n");
    }

    free(lat_ns);
    free(cyc);
    close(fd);
    return 0;
}

