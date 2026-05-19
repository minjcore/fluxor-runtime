package eureka

import (
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// RegistryConfig configures the service registry
type RegistryConfig struct {
	// DefaultLeaseDuration is the default lease duration for new registrations
	// Default: 90 seconds
	DefaultLeaseDuration time.Duration

	// RenewalThreshold is how often instances should renew (typically 1/3 of lease duration)
	// Default: 30 seconds
	RenewalThreshold time.Duration

	// EvictionInterval is how often to check for expired instances
	// Default: 60 seconds
	EvictionInterval time.Duration
}

// DefaultRegistryConfig returns default configuration
func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		DefaultLeaseDuration: 90 * time.Second,
		RenewalThreshold:     30 * time.Second,
		EvictionInterval:     60 * time.Second,
	}
}

// Registry is the core service registry implementation
type Registry struct {
	// Map of service name -> ServiceInfo
	services map[string]*ServiceInfo

	// Map of instance ID -> ServiceInstance (for quick lookup)
	instances map[string]*ServiceInstance

	// Configuration
	config *RegistryConfig

	mu sync.RWMutex
}

// NewRegistry creates a new service registry
func NewRegistry(config *RegistryConfig) *Registry {
	failfast.NotNil(config, "config")

	if config.DefaultLeaseDuration <= 0 {
		config.DefaultLeaseDuration = 90 * time.Second
	}
	if config.RenewalThreshold <= 0 {
		config.RenewalThreshold = 30 * time.Second
	}
	if config.EvictionInterval <= 0 {
		config.EvictionInterval = 60 * time.Second
	}

	return &Registry{
		services:  make(map[string]*ServiceInfo),
		instances: make(map[string]*ServiceInstance),
		config:    config,
	}
}

// Register registers a new service instance
func (r *Registry) Register(instance *ServiceInstance) error {
	failfast.NotNil(instance, "instance")
	failfast.If(instance.InstanceID != "", "instanceId cannot be empty")
	failfast.If(instance.ServiceName != "", "serviceName cannot be empty")

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if instance already exists
	if existing, exists := r.instances[instance.InstanceID]; exists {
		// Update existing instance
		existing.mu.Lock()
		existing.Host = instance.Host
		existing.Port = instance.Port
		existing.SecurePort = instance.SecurePort
		existing.Status = instance.Status
		existing.HealthCheckURL = instance.HealthCheckURL
		existing.HomePageURL = instance.HomePageURL
		existing.StatusPageURL = instance.StatusPageURL
		if instance.Metadata != nil {
			if existing.Metadata == nil {
				existing.Metadata = make(map[string]string)
			}
			for k, v := range instance.Metadata {
				existing.Metadata[k] = v
			}
		}
		existing.Renew(r.config.DefaultLeaseDuration)
		existing.mu.Unlock()
		return nil
	}

	// Set registration time and lease
	now := time.Now()
	instance.RegisteredAt = now
	instance.LastRenewal = now
	instance.LeaseExpiration = now.Add(r.config.DefaultLeaseDuration)
	instance.LeaseDuration = r.config.DefaultLeaseDuration

	// Add to service info
	serviceInfo, exists := r.services[instance.ServiceName]
	if !exists {
		serviceInfo = &ServiceInfo{
			Name:      instance.ServiceName,
			Instances: make([]*ServiceInstance, 0),
		}
		r.services[instance.ServiceName] = serviceInfo
	}
	serviceInfo.AddInstance(instance)

	// Add to instances map
	r.instances[instance.InstanceID] = instance

	return nil
}

// Renew renews the lease for an instance
func (r *Registry) Renew(serviceName, instanceID string) error {
	failfast.If(serviceName != "", "serviceName cannot be empty")
	failfast.If(instanceID != "", "instanceId cannot be empty")

	r.mu.RLock()
	instance, exists := r.instances[instanceID]
	r.mu.RUnlock()

	if !exists {
		return &RegistryError{
			Code:    "INSTANCE_NOT_FOUND",
			Message: "instance not found: " + instanceID,
		}
	}

	// Verify service name matches
	if instance.ServiceName != serviceName {
		return &RegistryError{
			Code:    "SERVICE_NAME_MISMATCH",
			Message: "service name mismatch",
		}
	}

	// Renew lease
	instance.Renew(r.config.DefaultLeaseDuration)

	return nil
}

// Unregister removes an instance from the registry
func (r *Registry) Unregister(serviceName, instanceID string) error {
	failfast.If(serviceName != "", "serviceName cannot be empty")
	failfast.If(instanceID != "", "instanceId cannot be empty")

	r.mu.Lock()
	defer r.mu.Unlock()

	instance, exists := r.instances[instanceID]
	if !exists {
		return &RegistryError{
			Code:    "INSTANCE_NOT_FOUND",
			Message: "instance not found: " + instanceID,
		}
	}

	// Verify service name matches
	if instance.ServiceName != serviceName {
		return &RegistryError{
			Code:    "SERVICE_NAME_MISMATCH",
			Message: "service name mismatch",
		}
	}

	// Remove from service info
	if serviceInfo, exists := r.services[serviceName]; exists {
		serviceInfo.RemoveInstance(instanceID)
		// Remove service if no instances left
		if len(serviceInfo.Instances) == 0 {
			delete(r.services, serviceName)
		}
	}

	// Remove from instances map
	delete(r.instances, instanceID)

	return nil
}

// GetInstances returns all healthy instances for a service
func (r *Registry) GetInstances(serviceName string) []*ServiceInstance {
	failfast.If(serviceName != "", "serviceName cannot be empty")

	r.mu.RLock()
	serviceInfo, exists := r.services[serviceName]
	r.mu.RUnlock()

	if !exists {
		return []*ServiceInstance{}
	}

	return serviceInfo.GetInstances()
}

// GetAllInstances returns all instances for a service (including unhealthy/expired)
func (r *Registry) GetAllInstances(serviceName string) []*ServiceInstance {
	failfast.If(serviceName != "", "serviceName cannot be empty")

	r.mu.RLock()
	serviceInfo, exists := r.services[serviceName]
	r.mu.RUnlock()

	if !exists {
		return []*ServiceInstance{}
	}

	return serviceInfo.GetAllInstances()
}

// GetInstance returns a specific instance by ID
func (r *Registry) GetInstance(instanceID string) (*ServiceInstance, bool) {
	failfast.If(instanceID != "", "instanceId cannot be empty")

	r.mu.RLock()
	defer r.mu.RUnlock()
	instance, exists := r.instances[instanceID]
	return instance, exists
}

// ListServices returns all registered service names
func (r *Registry) ListServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]string, 0, len(r.services))
	for name := range r.services {
		services = append(services, name)
	}
	return services
}

// UpdateStatus updates the status of an instance
func (r *Registry) UpdateStatus(serviceName, instanceID string, status InstanceStatus) error {
	failfast.If(serviceName != "", "serviceName cannot be empty")
	failfast.If(instanceID != "", "instanceId cannot be empty")

	r.mu.RLock()
	instance, exists := r.instances[instanceID]
	r.mu.RUnlock()

	if !exists {
		return &RegistryError{
			Code:    "INSTANCE_NOT_FOUND",
			Message: "instance not found: " + instanceID,
		}
	}

	if instance.ServiceName != serviceName {
		return &RegistryError{
			Code:    "SERVICE_NAME_MISMATCH",
			Message: "service name mismatch",
		}
	}

	instance.UpdateStatus(status)
	return nil
}

// EvictExpired removes all expired instances
func (r *Registry) EvictExpired() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	evicted := 0
	now := time.Now()

	for instanceID, instance := range r.instances {
		if now.After(instance.LeaseExpiration) {
			// Remove from service info
			if serviceInfo, exists := r.services[instance.ServiceName]; exists {
				serviceInfo.RemoveInstance(instanceID)
				if len(serviceInfo.Instances) == 0 {
					delete(r.services, instance.ServiceName)
				}
			}

			// Remove from instances map
			delete(r.instances, instanceID)
			evicted++
		}
	}

	return evicted
}

// GetStats returns registry statistics
func (r *Registry) GetStats() *RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := &RegistryStats{
		TotalServices: len(r.services),
	}

	now := time.Now()
	for _, instance := range r.instances {
		stats.TotalInstances++
		if instance.GetStatus() == InstanceStatusUp && !instance.IsExpired() {
			stats.HealthyInstances++
		}
		if now.After(instance.LeaseExpiration) {
			stats.ExpiredInstances++
		}
	}

	return stats
}

// RegistryError represents a registry operation error
type RegistryError struct {
	Code    string
	Message string
}

func (e *RegistryError) Error() string {
	return e.Code + ": " + e.Message
}
