package proxy

import (
	"fmt"
	"math/rand"
	"sync/atomic"
)

// loadBalancer implements LoadBalancer interface
type loadBalancer struct {
	strategy string
	roundRobinIndex int64 // atomic counter for round-robin
}

// newLoadBalancer creates a new load balancer with the specified strategy
func newLoadBalancer(strategy string) LoadBalancer {
	return &loadBalancer{
		strategy: strategy,
	}
}

// SelectBackend selects a backend using the configured strategy
func (lb *loadBalancer) SelectBackend(backends []BackendStatus) (*Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	switch lb.strategy {
	case "round-robin":
		return lb.roundRobin(backends)
	case "least-connections":
		return lb.leastConnections(backends)
	case "weighted":
		return lb.weighted(backends)
	case "random":
		return lb.random(backends)
	default:
		return lb.roundRobin(backends) // Default to round-robin
	}
}

// roundRobin selects backends in round-robin fashion
func (lb *loadBalancer) roundRobin(backends []BackendStatus) (*Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	index := atomic.AddInt64(&lb.roundRobinIndex, 1) - 1
	selected := backends[index%int64(len(backends))]
	return &selected.Backend, nil
}

// leastConnections selects the backend with the fewest active connections
func (lb *loadBalancer) leastConnections(backends []BackendStatus) (*Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	minConnections := backends[0].Connections
	selected := &backends[0].Backend

	for i := 1; i < len(backends); i++ {
		if backends[i].Connections < minConnections {
			minConnections = backends[i].Connections
			selected = &backends[i].Backend
		}
	}

	return selected, nil
}

// weighted selects a backend based on weight (higher weight = more traffic)
func (lb *loadBalancer) weighted(backends []BackendStatus) (*Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	// Calculate total weight
	totalWeight := 0
	for _, backend := range backends {
		totalWeight += backend.Backend.Weight
	}

	if totalWeight == 0 {
		// Fallback to round-robin if all weights are 0
		return lb.roundRobin(backends)
	}

	// Select random number in range [0, totalWeight)
	random := rand.Intn(totalWeight)

	// Find which backend this random number falls into
	currentWeight := 0
	for _, backend := range backends {
		currentWeight += backend.Backend.Weight
		if random < currentWeight {
			return &backend.Backend, nil
		}
	}

	// Fallback (shouldn't reach here)
	return &backends[0].Backend, nil
}

// random selects a random backend
func (lb *loadBalancer) random(backends []BackendStatus) (*Backend, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("no backends available")
	}

	index := rand.Intn(len(backends))
	return &backends[index].Backend, nil
}
