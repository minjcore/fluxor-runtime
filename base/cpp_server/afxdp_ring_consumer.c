// AF_XDP user-space ring consumer (Linux-only).
// - mmaps RX/TX + FILL/COMP rings
// - poll() descriptors
// - parses L2/L3/L4 headers directly on packet bytes in UMEM
//
// Build (Linux):
//   clang -O2 -Wall -Wextra -std=c11 afxdp_ring_consumer.c -o afxdp_consumer
//
// Run (assumes XDP program redirects packets to XSKMAP for (ifname,queue)):
//   sudo ./afxdp_consumer --ifname eth0 --queue 0 --port 6380

#ifdef __linux__

#include <errno.h>
#include <fcntl.h>
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/if_xdp.h>
#include <linux/in.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <net/if.h>
#include <poll.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <sys/mman.h>
#include <sys/socket.h>
#include <sys/syscall.h>
#include <sys/types.h>
#include <unistd.h>

// ---------- Mini binary protocol over UDP ----------
// Request (8 bytes):  "FH" ver=1 op token(u32)
// Response (12 bytes): "FH" ver=1 op|0x80 status token(u32) ts_ms(u32 low)
enum { FH_VER = 1, FH_OP_PING = 1 };
struct __attribute__((packed)) fh_req {
    uint8_t magic[2];
    uint8_t ver;
    uint8_t op;
    uint32_t token; // network order
};
struct __attribute__((packed)) fh_resp {
    uint8_t magic[2];
    uint8_t ver;
    uint8_t op;
    uint8_t status;
    uint32_t token; // network order
    uint32_t ts_ms; // network order (low bits)
};

static inline uint32_t now_ms_u32(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return (uint32_t)((uint64_t)ts.tv_sec * 1000ULL + (uint64_t)ts.tv_nsec / 1000000ULL);
}

// ---------- Tiny ring helpers (subset of libbpf xsk_ring_* patterns) ----------

struct u32_ring {
    uint32_t *producer;
    uint32_t *consumer;
    uint32_t *flags;
    uint32_t size;
    uint32_t mask;
    uint32_t cached_prod;
    uint32_t cached_cons;
    void *ring; // pointer to ring entries (typed by user)
};

static inline uint32_t u32_load_acquire(const uint32_t *p) {
    return __atomic_load_n(p, __ATOMIC_ACQUIRE);
}
static inline void u32_store_release(uint32_t *p, uint32_t v) {
    __atomic_store_n(p, v, __ATOMIC_RELEASE);
}

static inline uint32_t ring_cons_peek(struct u32_ring *r, uint32_t n, uint32_t *idx) {
    uint32_t cons = r->cached_cons;
    uint32_t prod = r->cached_prod;
    if (cons == prod) {
        prod = u32_load_acquire(r->producer);
        r->cached_prod = prod;
    }
    uint32_t avail = prod - cons;
    if (avail == 0) return 0;
    if (n > avail) n = avail;
    *idx = cons & r->mask;
    return n;
}

static inline void ring_cons_release(struct u32_ring *r, uint32_t n) {
    r->cached_cons += n;
    u32_store_release(r->consumer, r->cached_cons);
}

static inline uint32_t ring_prod_reserve(struct u32_ring *r, uint32_t n, uint32_t *idx) {
    uint32_t prod = r->cached_prod;
    uint32_t cons = r->cached_cons;
    uint32_t free_entries = r->size - (prod - cons);
    if (free_entries < n) {
        cons = u32_load_acquire(r->consumer);
        r->cached_cons = cons;
        free_entries = r->size - (prod - cons);
        if (free_entries < n) return 0;
    }
    *idx = prod & r->mask;
    return n;
}

static inline void ring_prod_submit(struct u32_ring *r, uint32_t n) {
    r->cached_prod += n;
    u32_store_release(r->producer, r->cached_prod);
}

// ---------- Checksums ----------
static inline uint32_t csum_add_u16(uint32_t sum, uint16_t v) {
    return sum + (uint32_t)v;
}
static inline uint32_t csum_add_u32(uint32_t sum, uint32_t v) {
    sum += (v >> 16) & 0xFFFF;
    sum += v & 0xFFFF;
    return sum;
}
static inline uint16_t csum_fold(uint32_t sum) {
    while (sum >> 16) sum = (sum & 0xFFFF) + (sum >> 16);
    return (uint16_t)~sum;
}
static inline uint16_t ipv4_hdr_checksum(const struct iphdr *ip) {
    // ip->check must be 0 in caller.
    const uint16_t *p = (const uint16_t *)ip;
    uint32_t sum = 0;
    // ip header length is ihl*4 bytes
    uint32_t words = (uint32_t)ip->ihl * 2;
    for (uint32_t i = 0; i < words; i++) sum = csum_add_u16(sum, ntohs(p[i]));
    return htons(csum_fold(sum));
}
static inline uint16_t udp_checksum(const struct iphdr *ip, const struct udphdr *udp, const uint8_t *payload, uint32_t plen) {
    uint32_t sum = 0;
    // pseudo header
    sum = csum_add_u32(sum, ntohl(ip->saddr));
    sum = csum_add_u32(sum, ntohl(ip->daddr));
    sum = csum_add_u16(sum, ip->protocol);
    sum = csum_add_u16(sum, ntohs(udp->len));
    // udp header
    const uint16_t *p = (const uint16_t *)udp;
    for (uint32_t i = 0; i < sizeof(*udp) / 2; i++) sum = csum_add_u16(sum, ntohs(p[i]));
    // payload
    const uint8_t *d = payload;
    for (uint32_t i = 0; i + 1 < plen; i += 2) {
        sum = csum_add_u16(sum, (uint16_t)((d[i] << 8) | d[i + 1]));
    }
    if (plen & 1) {
        sum = csum_add_u16(sum, (uint16_t)(d[plen - 1] << 8));
    }
    uint16_t out = csum_fold(sum);
    // RFC768: checksum of zero means "no checksum"; if computed is 0, transmit as 0xFFFF.
    if (out == 0) out = 0xFFFF;
    return htons(out);
}

// ---------- Packet parsing (L2/L3/L4) ----------

static inline bool parse_l2_l3_l4(const uint8_t *pkt, uint32_t len, uint16_t *dport_out) {
    if (len < sizeof(struct ethhdr)) return false;
    const struct ethhdr *eth = (const struct ethhdr *)pkt;
    uint16_t proto = ntohs(eth->h_proto);
    uint32_t off = sizeof(*eth);

    // Skip single VLAN if present.
    if (proto == ETH_P_8021Q || proto == ETH_P_8021AD) {
        if (len < off + 4) return false;
        proto = (uint16_t)(pkt[off + 2] << 8 | pkt[off + 3]);
        off += 4;
    }

    if (proto != ETH_P_IP) return false;
    if (len < off + sizeof(struct iphdr)) return false;
    const struct iphdr *ip = (const struct iphdr *)(pkt + off);
    if (ip->ihl < 5) return false;
    uint32_t ip_hlen = (uint32_t)ip->ihl * 4;
    if (len < off + ip_hlen) return false;
    off += ip_hlen;

    if (ip->protocol == IPPROTO_UDP) {
        if (len < off + sizeof(struct udphdr)) return false;
        const struct udphdr *uh = (const struct udphdr *)(pkt + off);
        *dport_out = ntohs(uh->dest);
        return true;
    }
    if (ip->protocol == IPPROTO_TCP) {
        if (len < off + sizeof(struct tcphdr)) return false;
        const struct tcphdr *th = (const struct tcphdr *)(pkt + off);
        *dport_out = ntohs(th->dest);
        return true;
    }
    return false;
}

struct vlan_hdr { uint16_t tci; uint16_t encap; } __attribute__((packed));

// Parse UDP/IPv4 and locate payload. Supports 0 or 1 VLAN tag.
static inline bool parse_udp4(const uint8_t *pkt, uint32_t len,
                              bool *has_vlan, uint16_t *vlan_tci,
                              const struct ethhdr **eth_out,
                              const struct iphdr **ip_out,
                              const struct udphdr **udp_out,
                              const uint8_t **pl_out, uint32_t *pl_len_out) {
    if (len < sizeof(struct ethhdr)) return false;
    const struct ethhdr *eth = (const struct ethhdr *)pkt;
    uint16_t proto = ntohs(eth->h_proto);
    uint32_t off = sizeof(*eth);
    *has_vlan = false;
    *vlan_tci = 0;
    if (proto == ETH_P_8021Q || proto == ETH_P_8021AD) {
        if (len < off + sizeof(struct vlan_hdr)) return false;
        const struct vlan_hdr *vh = (const struct vlan_hdr *)(pkt + off);
        *has_vlan = true;
        *vlan_tci = ntohs(vh->tci);
        proto = ntohs(vh->encap);
        off += sizeof(*vh);
    }
    if (proto != ETH_P_IP) return false;
    if (len < off + sizeof(struct iphdr)) return false;
    const struct iphdr *ip = (const struct iphdr *)(pkt + off);
    if (ip->ihl < 5) return false;
    uint32_t ip_hlen = (uint32_t)ip->ihl * 4;
    if (len < off + ip_hlen) return false;
    if (ip->protocol != IPPROTO_UDP) return false;
    off += ip_hlen;
    if (len < off + sizeof(struct udphdr)) return false;
    const struct udphdr *udp = (const struct udphdr *)(pkt + off);
    off += sizeof(*udp);
    if (len < off) return false;
    *eth_out = eth;
    *ip_out = ip;
    *udp_out = udp;
    *pl_out = pkt + off;
    *pl_len_out = len - off;
    return true;
}

// Craft an in-place UDP response into 'buf' (UMEM frame at addr). Returns response length or 0 on fail.
static inline uint32_t craft_udp_response(uint8_t *buf, uint32_t frame_cap,
                                         bool has_vlan, uint16_t vlan_tci,
                                         const struct ethhdr *rx_eth,
                                         const struct iphdr *rx_ip,
                                         const struct udphdr *rx_udp,
                                         const struct fh_req *req) {
    (void)frame_cap;
    struct fh_resp resp = {
        .magic = {'F','H'},
        .ver = FH_VER,
        .op = (uint8_t)(req->op | 0x80),
        .status = 0,
        .token = req->token,
        .ts_ms = htonl(now_ms_u32()),
    };
    const uint32_t payload_len = (uint32_t)sizeof(resp);

    uint32_t off = 0;
    if (frame_cap < sizeof(struct ethhdr) + (has_vlan ? sizeof(struct vlan_hdr) : 0) + sizeof(struct iphdr) + sizeof(struct udphdr) + payload_len) {
        return 0;
    }

    // Ethernet (swap MACs)
    struct ethhdr *eth = (struct ethhdr *)(buf + off);
    memcpy(eth->h_dest, rx_eth->h_source, ETH_ALEN);
    memcpy(eth->h_source, rx_eth->h_dest, ETH_ALEN);
    eth->h_proto = has_vlan ? htons(ETH_P_8021Q) : htons(ETH_P_IP);
    off += sizeof(*eth);

    if (has_vlan) {
        struct vlan_hdr *vh = (struct vlan_hdr *)(buf + off);
        vh->tci = htons(vlan_tci);
        vh->encap = htons(ETH_P_IP);
        off += sizeof(*vh);
    }

    // IPv4
    struct iphdr *ip = (struct iphdr *)(buf + off);
    memset(ip, 0, sizeof(*ip));
    ip->version = 4;
    ip->ihl = 5;
    ip->ttl = 64;
    ip->protocol = IPPROTO_UDP;
    ip->saddr = rx_ip->daddr;
    ip->daddr = rx_ip->saddr;
    uint16_t tot_len = (uint16_t)(sizeof(struct iphdr) + sizeof(struct udphdr) + payload_len);
    ip->tot_len = htons(tot_len);
    ip->id = 0;
    ip->frag_off = htons(0);
    ip->check = 0;
    ip->check = ipv4_hdr_checksum(ip);
    off += sizeof(*ip);

    // UDP
    struct udphdr *udp = (struct udphdr *)(buf + off);
    memset(udp, 0, sizeof(*udp));
    udp->source = rx_udp->dest;
    udp->dest = rx_udp->source;
    udp->len = htons((uint16_t)(sizeof(struct udphdr) + payload_len));
    udp->check = 0;
    off += sizeof(*udp);

    // payload
    memcpy(buf + off, &resp, sizeof(resp));
    udp->check = udp_checksum(ip, udp, buf + off, payload_len);
    off += payload_len;

    return off;
}

// ---------- AF_XDP setup ----------

static int xdp_set_u32(int fd, int opt, uint32_t v) {
    return setsockopt(fd, SOL_XDP, opt, &v, sizeof(v));
}

static void die(const char *msg) {
    perror(msg);
    exit(1);
}

struct xdp_ctx {
    int xsk_fd;
    void *umem_area;
    size_t umem_len;

    struct u32_ring rx;
    struct u32_ring tx;
    struct u32_ring fill;
    struct u32_ring comp;

    struct xdp_mmap_offsets off;
};

static void *xdp_mmap_ring(int fd, off_t off, size_t len) {
    void *p = mmap(NULL, len, PROT_READ | PROT_WRITE, MAP_SHARED | MAP_POPULATE, fd, off);
    if (p == MAP_FAILED) die("mmap");
    return p;
}

static void ring_init_desc(struct u32_ring *r, void *base, const struct xdp_ring_offset *ro, uint32_t size) {
    r->producer = (uint32_t *)((uint8_t *)base + ro->producer);
    r->consumer = (uint32_t *)((uint8_t *)base + ro->consumer);
    r->flags = (uint32_t *)((uint8_t *)base + ro->flags);
    r->ring = (void *)((uint8_t *)base + ro->desc);
    r->size = size;
    r->mask = size - 1;
    r->cached_prod = u32_load_acquire(r->producer);
    r->cached_cons = u32_load_acquire(r->consumer);
}

static void ring_init_addr(struct u32_ring *r, void *base, const struct xdp_ring_offset *ro, uint32_t size) {
    r->producer = (uint32_t *)((uint8_t *)base + ro->producer);
    r->consumer = (uint32_t *)((uint8_t *)base + ro->consumer);
    r->flags = (uint32_t *)((uint8_t *)base + ro->flags);
    r->ring = (void *)((uint8_t *)base + ro->desc); // u64 addresses
    r->size = size;
    r->mask = size - 1;
    r->cached_prod = u32_load_acquire(r->producer);
    r->cached_cons = u32_load_acquire(r->consumer);
}

static void xdp_get_mmap_offsets(int fd, struct xdp_mmap_offsets *off) {
    socklen_t optlen = sizeof(*off);
    if (getsockopt(fd, SOL_XDP, XDP_MMAP_OFFSETS, off, &optlen) != 0) die("getsockopt(XDP_MMAP_OFFSETS)");
}

static void usage(const char *prog) {
    fprintf(stderr, "Usage: %s --ifname IFACE --queue Q --port P [--xskmap /sys/fs/bpf/...]\n", prog);
}

static int bpf_syscall(enum bpf_cmd cmd, union bpf_attr *attr) {
    return (int)syscall(__NR_bpf, cmd, attr, sizeof(*attr));
}

static int bpf_obj_get_path(const char *path) {
    union bpf_attr attr;
    memset(&attr, 0, sizeof(attr));
    attr.pathname = (uint64_t)(uintptr_t)path;
    return bpf_syscall(BPF_OBJ_GET, &attr);
}

static int bpf_map_update_u32_u32(int map_fd, uint32_t key, uint32_t value) {
    union bpf_attr attr;
    memset(&attr, 0, sizeof(attr));
    attr.map_fd = (uint32_t)map_fd;
    attr.key = (uint64_t)(uintptr_t)&key;
    attr.value = (uint64_t)(uintptr_t)&value;
    attr.flags = BPF_ANY;
    return bpf_syscall(BPF_MAP_UPDATE_ELEM, &attr);
}

int main(int argc, char **argv) {
    const char *ifname = NULL;
    uint32_t queue_id = 0;
    uint16_t port = 6380;
    const char *xskmap_path = NULL;

    for (int i = 1; i < argc; i++) {
        if (!strcmp(argv[i], "--ifname") && i + 1 < argc) ifname = argv[++i];
        else if (!strcmp(argv[i], "--queue") && i + 1 < argc) queue_id = (uint32_t)atoi(argv[++i]);
        else if (!strcmp(argv[i], "--port") && i + 1 < argc) port = (uint16_t)atoi(argv[++i]);
        else if (!strcmp(argv[i], "--xskmap") && i + 1 < argc) xskmap_path = argv[++i];
        else { usage(argv[0]); return 2; }
    }
    if (!ifname) { usage(argv[0]); return 2; }

    // Ring/UMEM sizing (keep small & simple).
    const uint32_t RX_SZ = 2048;
    const uint32_t TX_SZ = 2048;
    const uint32_t FILL_SZ = 2048;
    const uint32_t COMP_SZ = 2048;
    const uint32_t FRAME_SZ = 2048; // must be power-of-two aligned enough for packets
    const uint32_t NUM_FRAMES = 4096;

    struct xdp_ctx x = {0};
    x.xsk_fd = socket(AF_XDP, SOCK_RAW, 0);
    if (x.xsk_fd < 0) die("socket(AF_XDP)");

    xdp_get_mmap_offsets(x.xsk_fd, &x.off);

    // UMEM region.
    x.umem_len = (size_t)NUM_FRAMES * FRAME_SZ;
    x.umem_area = mmap(NULL, x.umem_len, PROT_READ | PROT_WRITE,
                       MAP_PRIVATE | MAP_ANONYMOUS | MAP_POPULATE, -1, 0);
    if (x.umem_area == MAP_FAILED) die("mmap(umem)");

    struct xdp_umem_reg mr = {
        .addr = (uint64_t)(uintptr_t)x.umem_area,
        .len = x.umem_len,
        .chunk_size = FRAME_SZ,
        .headroom = 0,
        .flags = 0,
    };
    if (setsockopt(x.xsk_fd, SOL_XDP, XDP_UMEM_REG, &mr, sizeof(mr)) != 0) die("setsockopt(XDP_UMEM_REG)");

    if (xdp_set_u32(x.xsk_fd, XDP_UMEM_FILL_RING, FILL_SZ) != 0) die("setsockopt(FILL_RING)");
    if (xdp_set_u32(x.xsk_fd, XDP_UMEM_COMPLETION_RING, COMP_SZ) != 0) die("setsockopt(COMP_RING)");
    if (xdp_set_u32(x.xsk_fd, XDP_RX_RING, RX_SZ) != 0) die("setsockopt(RX_RING)");
    if (xdp_set_u32(x.xsk_fd, XDP_TX_RING, TX_SZ) != 0) die("setsockopt(TX_RING)");

    // mmap rings
    const size_t rx_map_sz = x.off.rx.desc + RX_SZ * sizeof(struct xdp_desc);
    const size_t tx_map_sz = x.off.tx.desc + TX_SZ * sizeof(struct xdp_desc);
    const size_t fr_map_sz = x.off.fr.desc + FILL_SZ * sizeof(uint64_t);
    const size_t cr_map_sz = x.off.cr.desc + COMP_SZ * sizeof(uint64_t);

    void *rx_map = xdp_mmap_ring(x.xsk_fd, XDP_PGOFF_RX_RING, rx_map_sz);
    void *tx_map = xdp_mmap_ring(x.xsk_fd, XDP_PGOFF_TX_RING, tx_map_sz);
    void *fr_map = xdp_mmap_ring(x.xsk_fd, XDP_PGOFF_FILL_RING, fr_map_sz);
    void *cr_map = xdp_mmap_ring(x.xsk_fd, XDP_PGOFF_COMPLETION_RING, cr_map_sz);

    ring_init_desc(&x.rx, rx_map, &x.off.rx, RX_SZ);
    ring_init_desc(&x.tx, tx_map, &x.off.tx, TX_SZ);
    ring_init_addr(&x.fill, fr_map, &x.off.fr, FILL_SZ);
    ring_init_addr(&x.comp, cr_map, &x.off.cr, COMP_SZ);

    // Bind socket to (ifindex, queue).
    int ifindex = if_nametoindex(ifname);
    if (ifindex == 0) die("if_nametoindex");
    struct sockaddr_xdp sxdp = {
        .sxdp_family = AF_XDP,
        .sxdp_ifindex = ifindex,
        .sxdp_queue_id = queue_id,
        .sxdp_flags = XDP_USE_NEED_WAKEUP, // safe default; may reduce syscalls with busy poll
    };
    if (bind(x.xsk_fd, (struct sockaddr *)&sxdp, sizeof(sxdp)) != 0) die("bind(AF_XDP)");

    // Optional: update pinned XSKMAP so XDP program can redirect into this socket.
    if (xskmap_path) {
        int map_fd = bpf_obj_get_path(xskmap_path);
        if (map_fd < 0) die("bpf_obj_get(xskmap)");
        uint32_t v = (uint32_t)x.xsk_fd;
        if (bpf_map_update_u32_u32(map_fd, queue_id, v) != 0) die("bpf_map_update_elem(xskmap)");
        close(map_fd);
    }

    // Populate initial fill ring with all UMEM frames.
    {
        uint32_t idx = 0;
        uint32_t n = ring_prod_reserve(&x.fill, FILL_SZ, &idx);
        if (n == 0) die("fill ring reserve");
        uint64_t *addrs = (uint64_t *)x.fill.ring;
        for (uint32_t i = 0; i < n; i++) {
            addrs[(idx + i) & x.fill.mask] = (uint64_t)i * FRAME_SZ;
        }
        ring_prod_submit(&x.fill, n);
    }

    fprintf(stderr, "AF_XDP consumer on %s q=%u filter dport=%u\n", ifname, queue_id, (unsigned)port);

    struct pollfd pfd = {.fd = x.xsk_fd, .events = POLLIN};
    uint64_t pkts = 0, matched = 0, replied = 0;

    for (;;) {
        int pr = poll(&pfd, 1, 1000);
        if (pr < 0) {
            if (errno == EINTR) continue;
            die("poll");
        }
        if (pr == 0) {
            fprintf(stderr, "pkts=%llu matched=%llu replied=%llu\n",
                    (unsigned long long)pkts, (unsigned long long)matched, (unsigned long long)replied);
            continue;
        }

        // Reclaim completed TX buffers -> FILL ring.
        {
            uint32_t cidx = 0;
            uint32_t cn = ring_cons_peek(&x.comp, 64, &cidx);
            if (cn) {
                uint32_t fidx = 0;
                uint32_t fn = ring_prod_reserve(&x.fill, cn, &fidx);
                if (fn == cn) {
                    uint64_t *comp_addrs = (uint64_t *)x.comp.ring;
                    uint64_t *fill_addrs = (uint64_t *)x.fill.ring;
                    for (uint32_t i = 0; i < cn; i++) {
                        fill_addrs[(fidx + i) & x.fill.mask] = comp_addrs[(cidx + i) & x.comp.mask];
                    }
                    ring_cons_release(&x.comp, cn);
                    ring_prod_submit(&x.fill, cn);
                } else {
                    // If we can't recycle, just keep completions for next loop.
                }
            }
        }

        // Consume RX descriptors.
        uint32_t rx_idx = 0;
        uint32_t rcvd = ring_cons_peek(&x.rx, 64, &rx_idx);
        if (rcvd == 0) continue;

        struct xdp_desc *descs = (struct xdp_desc *)x.rx.ring;
        uint64_t *fill_addrs = (uint64_t *)x.fill.ring;
        struct xdp_desc *tx_descs = (struct xdp_desc *)x.tx.ring;

        // Reserve up to rcvd refill slots (we'll submit only what we actually recycle).
        uint32_t fill_idx = 0;
        uint32_t can_refill = ring_prod_reserve(&x.fill, rcvd, &fill_idx);
        if (can_refill == 0) {
            ring_cons_release(&x.rx, rcvd);
            continue;
        }
        uint32_t fill_used = 0;

        uint32_t tx_idx = 0;
        uint32_t tx_reserved = ring_prod_reserve(&x.tx, rcvd, &tx_idx); // worst-case 1 reply per rx
        uint32_t tx_used = 0;

        for (uint32_t i = 0; i < rcvd; i++) {
            struct xdp_desc d = descs[(rx_idx + i) & x.rx.mask];
            uint64_t addr = d.addr;
            uint32_t len = d.len;
            // Access packet bytes in UMEM.
            if (addr + len > x.umem_len) continue; // safety
            const uint8_t *pkt = (const uint8_t *)x.umem_area + addr;

            uint16_t dport = 0;
            if (parse_l2_l3_l4(pkt, len, &dport) && dport == port) {
                matched++;
            }
            pkts++;

            // If UDP + our mini protocol, craft response and TX it.
            bool has_vlan = false;
            uint16_t vlan_tci = 0;
            const struct ethhdr *eth = NULL;
            const struct iphdr *ip = NULL;
            const struct udphdr *udp = NULL;
            const uint8_t *pl = NULL;
            uint32_t pl_len = 0;
            if (tx_used < tx_reserved &&
                parse_udp4(pkt, len, &has_vlan, &vlan_tci, &eth, &ip, &udp, &pl, &pl_len) &&
                ntohs(udp->dest) == port && pl_len >= sizeof(struct fh_req)) {
                const struct fh_req *req = (const struct fh_req *)pl;
                if (req->magic[0] == 'F' && req->magic[1] == 'H' && req->ver == FH_VER && req->op == FH_OP_PING) {
                    uint8_t *out = (uint8_t *)x.umem_area + addr;
                    uint32_t out_len = craft_udp_response(out, 2048, has_vlan, vlan_tci, eth, ip, udp, req);
                    if (out_len > 0) {
                        struct xdp_desc *td = &tx_descs[(tx_idx + tx_used) & x.tx.mask];
                        td->addr = addr;
                        td->len = out_len;
                        tx_used++;
                        replied++;
                        continue; // do NOT recycle addr to fill; reclaimed via completion ring.
                    }
                }
            }

            // Default: recycle buffer back to fill ring.
            if (fill_used < can_refill) {
                fill_addrs[(fill_idx + fill_used) & x.fill.mask] = addr;
                fill_used++;
            }
        }

        ring_cons_release(&x.rx, rcvd);
        if (fill_used) ring_prod_submit(&x.fill, fill_used);

        if (tx_used) {
            ring_prod_submit(&x.tx, tx_used);
            // Kick TX if driver needs wakeup.
            if (x.tx.flags && (*x.tx.flags & XDP_RING_NEED_WAKEUP)) {
                (void)sendto(x.xsk_fd, NULL, 0, MSG_DONTWAIT, NULL, 0);
            }
        }
    }

    return 0;
}

#else
int main(void) { return 0; }
#endif

