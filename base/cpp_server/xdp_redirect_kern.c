// XDP program: filter TCP/UDP dst port and redirect to AF_XDP (XSKMAP).
// - <200 lines, minimal + safe bounds checks.
// - Configure port at load time via 'target_port' (libbpf CO-RE style).
//
// Usage (loader side, not included):
// - Create XSKMAP, populate key=queue_id -> fd(xsk).
// - Load this program, set target_port, attach XDP to iface.
//
// Build example:
//   clang -O2 -g -target bpf -c xdp_redirect_kern.c -o xdp_redirect_kern.o

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/in.h>
#include <linux/udp.h>
#include <linux/tcp.h>

#include <bpf/bpf_helpers.h>

// AF_XDP socket map: key = RX queue id, value = xsk fd.
struct {
    __uint(type, BPF_MAP_TYPE_XSKMAP);
    __uint(max_entries, 64);
    __type(key, __u32);
    __type(value, __u32);
} xsks_map SEC(".maps");

// Destination port to redirect (network byte order comparison done in code).
// Can be overridden by user space loader (libbpf sets .rodata vars).
volatile const __u16 target_port = 6380;

static __always_inline int parse_eth(void **data, void *data_end, __u16 *eth_proto) {
    struct ethhdr *eth = *data;
    if ((void *)(eth + 1) > data_end) return -1;
    *eth_proto = eth->h_proto;
    *data = eth + 1;
    // Skip single VLAN tag if present.
    if (*eth_proto == __constant_htons(ETH_P_8021Q) || *eth_proto == __constant_htons(ETH_P_8021AD)) {
        struct vlan_hdr { __be16 tci; __be16 encap; };
        struct vlan_hdr *vh = *data;
        if ((void *)(vh + 1) > data_end) return -1;
        *eth_proto = vh->encap;
        *data = vh + 1;
    }
    return 0;
}

SEC("xdp")
int xdp_redirect_port(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    __u16 eth_proto = 0;
    if (parse_eth(&data, data_end, &eth_proto) < 0) return XDP_PASS;
    if (eth_proto != __constant_htons(ETH_P_IP)) return XDP_PASS;

    struct iphdr *iph = data;
    if ((void *)(iph + 1) > data_end) return XDP_PASS;
    if (iph->ihl < 5) return XDP_PASS;

    // Advance to L4 header (ihl is in 32-bit words).
    void *l4 = (void *)iph + (iph->ihl * 4);
    if (l4 > data_end) return XDP_PASS;

    __be16 dport = 0;
    if (iph->protocol == IPPROTO_UDP) {
        struct udphdr *uh = l4;
        if ((void *)(uh + 1) > data_end) return XDP_PASS;
        dport = uh->dest;
    } else if (iph->protocol == IPPROTO_TCP) {
        struct tcphdr *th = l4;
        if ((void *)(th + 1) > data_end) return XDP_PASS;
        dport = th->dest;
    } else {
        return XDP_PASS;
    }

    if (dport != __constant_htons(target_port)) return XDP_PASS;

    // Redirect to AF_XDP socket bound to this RX queue.
    return bpf_redirect_map(&xsks_map, ctx->rx_queue_index, 0);
}

char LICENSE[] SEC("license") = "GPL";

