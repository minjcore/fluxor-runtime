package vpn

import (
	"fmt"
	"net"
	"sync"
)

// IPPool manages IP address allocation for VPN clients
type IPPool struct {
	network   *net.IPNet
	mu        sync.RWMutex
	allocated map[string]bool
	available []net.IP
}

// NewIPPool creates a new IP pool from a network CIDR
func NewIPPool(network *net.IPNet) (*IPPool, error) {
	if network == nil {
		return nil, fmt.Errorf("network cannot be nil")
	}

	pool := &IPPool{
		network:   network,
		allocated: make(map[string]bool),
		available: []net.IP{},
	}

	// Generate available IPs from network
	ip := make(net.IP, len(network.IP))
	copy(ip, network.IP)
	
	// Increment to first usable IP (skip network address)
	incIP(ip)

	// Calculate network size for safety check
	maxIPs := getNetworkSize(network)
	if maxIPs > 10000 {
		maxIPs = 10000 // Limit for safety
	}

	// Generate all usable IPs in the network
	for network.Contains(ip) && len(pool.available) < maxIPs {
		// Skip network address (already skipped) and broadcast address
		if !isBroadcast(ip, network) {
			allocatedIP := make(net.IP, len(ip))
			copy(allocatedIP, ip)
			pool.available = append(pool.available, allocatedIP)
		}
		incIP(ip)
	}

	return pool, nil
}

// Allocate allocates an IP address from the pool
func (p *IPPool) Allocate() (net.IP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, ip := range p.available {
		ipStr := ip.String()
		if !p.allocated[ipStr] {
			p.allocated[ipStr] = true
			allocated := make(net.IP, len(ip))
			copy(allocated, ip)
			return allocated, nil
		}
	}

	return nil, fmt.Errorf("no available IP addresses in pool")
}

// Network returns the pool's network (for Contains check)
func (p *IPPool) Network() *net.IPNet {
	return p.network
}

// Release releases an IP address back to the pool
func (p *IPPool) Release(ip net.IP) {
	if ip == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	ipStr := ip.String()
	if p.allocated[ipStr] {
		delete(p.allocated, ipStr)
	}
}

// IsAllocated checks if an IP is allocated
func (p *IPPool) IsAllocated(ip net.IP) bool {
	if ip == nil {
		return false
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.allocated[ip.String()]
}

// AvailableCount returns the number of available IPs
func (p *IPPool) AvailableCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return len(p.available) - len(p.allocated)
}

// Helper functions
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// isBroadcast checks if an IP is the broadcast address for the network
func isBroadcast(ip net.IP, network *net.IPNet) bool {
	if ip == nil || network == nil {
		return false
	}

	// Calculate broadcast address
	mask := network.Mask
	broadcast := make(net.IP, len(network.IP))
	copy(broadcast, network.IP)

	for i := range broadcast {
		broadcast[i] = network.IP[i] | ^mask[i]
	}

	// Compare with given IP
	return ip.Equal(broadcast)
}

// getNetworkSize calculates the number of usable IPs in a network
func getNetworkSize(network *net.IPNet) int {
	ones, bits := network.Mask.Size()
	if bits == 0 {
		return 0
	}
	// Subtract 2 for network and broadcast addresses
	size := 1 << (bits - ones)
	if size > 2 {
		return size - 2
	}
	return 0
}
