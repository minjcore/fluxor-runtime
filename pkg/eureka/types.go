package eureka

import (
	"sync"
	"time"
)

// InstanceStatus represents the status of a service instance
type InstanceStatus string

const (
	InstanceStatusUp      InstanceStatus = "UP"
	InstanceStatusDown   InstanceStatus = "DOWN"
	InstanceStatusStarting InstanceStatus = "STARTING"
	InstanceStatusOutOfService InstanceStatus = "OUT_OF_SERVICE"
)

// ServiceInstance represents a single instance of a service
type ServiceInstance struct {
	// Instance ID (unique identifier for this instance)
	InstanceID string `json:"instanceId"`

	// Service name (e.g., "user-service")
	ServiceName string `json:"serviceName"`

	// Host address
	Host string `json:"host"`

	// Port number
	Port int `json:"port"`

	// Secure port (HTTPS)
	SecurePort int `json:"securePort,omitempty"`

	// Status of the instance
	Status InstanceStatus `json:"status"`

	// Metadata (key-value pairs)
	Metadata map[string]string `json:"metadata,omitempty"`

	// Registration timestamp
	RegisteredAt time.Time `json:"registeredAt"`

	// Last renewal timestamp (heartbeat)
	LastRenewal time.Time `json:"lastRenewal"`

	// Lease expiration time
	LeaseExpiration time.Time `json:"leaseExpiration"`

	// Lease duration (how long the lease is valid)
	LeaseDuration time.Duration `json:"leaseDuration"`

	// Health check URL (optional)
	HealthCheckURL string `json:"healthCheckUrl,omitempty"`

	// Home page URL (optional)
	HomePageURL string `json:"homePageUrl,omitempty"`

	// Status page URL (optional)
	StatusPageURL string `json:"statusPageUrl,omitempty"`

	mu sync.RWMutex
}

// IsExpired returns true if the instance lease has expired
func (si *ServiceInstance) IsExpired() bool {
	si.mu.RLock()
	defer si.mu.RUnlock()
	return time.Now().After(si.LeaseExpiration)
}

// Renew updates the last renewal time and extends the lease
func (si *ServiceInstance) Renew(leaseDuration time.Duration) {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.LastRenewal = time.Now()
	si.LeaseExpiration = si.LastRenewal.Add(leaseDuration)
}

// UpdateStatus updates the instance status
func (si *ServiceInstance) UpdateStatus(status InstanceStatus) {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.Status = status
}

// GetStatus returns the current status
func (si *ServiceInstance) GetStatus() InstanceStatus {
	si.mu.RLock()
	defer si.mu.RUnlock()
	return si.Status
}

// GetMetadata returns a copy of metadata
func (si *ServiceInstance) GetMetadata() map[string]string {
	si.mu.RLock()
	defer si.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range si.Metadata {
		result[k] = v
	}
	return result
}

// SetMetadata sets metadata
func (si *ServiceInstance) SetMetadata(key, value string) {
	si.mu.Lock()
	defer si.mu.Unlock()
	if si.Metadata == nil {
		si.Metadata = make(map[string]string)
	}
	si.Metadata[key] = value
}

// ServiceInfo represents all instances of a service
type ServiceInfo struct {
	// Service name
	Name string `json:"name"`

	// All instances of this service
	Instances []*ServiceInstance `json:"instances"`

	mu sync.RWMutex
}

// GetInstances returns all healthy instances
func (si *ServiceInfo) GetInstances() []*ServiceInstance {
	si.mu.RLock()
	defer si.mu.RUnlock()
	result := make([]*ServiceInstance, 0, len(si.Instances))
	for _, inst := range si.Instances {
		if !inst.IsExpired() && inst.GetStatus() == InstanceStatusUp {
			result = append(result, inst)
		}
	}
	return result
}

// GetAllInstances returns all instances regardless of status
func (si *ServiceInfo) GetAllInstances() []*ServiceInstance {
	si.mu.RLock()
	defer si.mu.RUnlock()
	result := make([]*ServiceInstance, len(si.Instances))
	copy(result, si.Instances)
	return result
}

// AddInstance adds an instance to the service
func (si *ServiceInfo) AddInstance(instance *ServiceInstance) {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.Instances = append(si.Instances, instance)
}

// RemoveInstance removes an instance by ID
func (si *ServiceInfo) RemoveInstance(instanceID string) bool {
	si.mu.Lock()
	defer si.mu.Unlock()
	for i, inst := range si.Instances {
		if inst.InstanceID == instanceID {
			si.Instances = append(si.Instances[:i], si.Instances[i+1:]...)
			return true
		}
	}
	return false
}

// FindInstance finds an instance by ID
func (si *ServiceInfo) FindInstance(instanceID string) *ServiceInstance {
	si.mu.RLock()
	defer si.mu.RUnlock()
	for _, inst := range si.Instances {
		if inst.InstanceID == instanceID {
			return inst
		}
	}
	return nil
}

// RegistrationRequest represents a service registration request
type RegistrationRequest struct {
	Instance *ServiceInstance `json:"instance"`
}

// RenewalRequest represents a lease renewal request
type RenewalRequest struct {
	ServiceName string `json:"serviceName"`
	InstanceID  string `json:"instanceId"`
}

// DiscoveryResponse represents the response for service discovery
type DiscoveryResponse struct {
	ServiceName string             `json:"serviceName"`
	Instances   []*ServiceInstance `json:"instances"`
}

// RegistryStats represents statistics about the registry
type RegistryStats struct {
	TotalServices  int `json:"totalServices"`
	TotalInstances int `json:"totalInstances"`
	HealthyInstances int `json:"healthyInstances"`
	ExpiredInstances int `json:"expiredInstances"`
}
