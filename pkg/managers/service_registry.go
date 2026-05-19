package managers

import (
	"sync"
	"time"
)

// ServiceInfo represents information about a registered service
type ServiceInfo struct {
	Name            string                 // Service name
	Type            string                 // Service type (verticle, component, etc.)
	Status          ServiceStatus          // Current status
	RegisteredAt    time.Time              // When service was registered
	LastHealthCheck time.Time              // Last health check timestamp
	Metadata        map[string]interface{} // Additional service metadata
	Dependencies    []string               // Service dependencies
	mu              sync.RWMutex
}

// ServiceStatus represents the health status of a service
type ServiceStatus string

const (
	ServiceStatusStarting  ServiceStatus = "starting"
	ServiceStatusHealthy   ServiceStatus = "healthy"
	ServiceStatusUnhealthy ServiceStatus = "unhealthy"
	ServiceStatusStopping  ServiceStatus = "stopping"
	ServiceStatusStopped   ServiceStatus = "stopped"
	ServiceStatusUnknown   ServiceStatus = "unknown"
)

// ServiceRegistry manages all registered services
type ServiceRegistry struct {
	services map[string]*ServiceInfo
	mu       sync.RWMutex
}

// newServiceRegistry creates a new service registry
func newServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]*ServiceInfo),
	}
}

// Register registers a new service
func (sr *ServiceRegistry) Register(name, serviceType string, dependencies []string) *ServiceInfo {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	info := &ServiceInfo{
		Name:         name,
		Type:         serviceType,
		Status:       ServiceStatusStarting,
		RegisteredAt: time.Now(),
		Metadata:     make(map[string]interface{}),
		Dependencies: dependencies,
	}

	sr.services[name] = info
	return info
}

// Unregister removes a service from the registry
func (sr *ServiceRegistry) Unregister(name string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	delete(sr.services, name)
}

// Get retrieves service information
func (sr *ServiceRegistry) Get(name string) (*ServiceInfo, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	info, exists := sr.services[name]
	return info, exists
}

// List returns all registered services
func (sr *ServiceRegistry) List() []*ServiceInfo {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	list := make([]*ServiceInfo, 0, len(sr.services))
	for _, info := range sr.services {
		list = append(list, info)
	}
	return list
}

// UpdateStatus updates the status of a service
func (sr *ServiceRegistry) UpdateStatus(name string, status ServiceStatus) {
	sr.mu.RLock()
	info, exists := sr.services[name]
	sr.mu.RUnlock()

	if !exists {
		return
	}

	info.mu.Lock()
	info.Status = status
	info.LastHealthCheck = time.Now()
	info.mu.Unlock()
}

// GetStatus returns the status of a service
func (sr *ServiceRegistry) GetStatus(name string) ServiceStatus {
	sr.mu.RLock()
	info, exists := sr.services[name]
	sr.mu.RUnlock()

	if !exists {
		return ServiceStatusUnknown
	}

	info.mu.RLock()
	defer info.mu.RUnlock()
	return info.Status
}

// SetMetadata sets metadata for a service
func (sr *ServiceRegistry) SetMetadata(name string, key string, value interface{}) {
	sr.mu.RLock()
	info, exists := sr.services[name]
	sr.mu.RUnlock()

	if !exists {
		return
	}

	info.mu.Lock()
	info.Metadata[key] = value
	info.mu.Unlock()
}

// GetMetadata retrieves metadata for a service
func (sr *ServiceRegistry) GetMetadata(name string, key string) (interface{}, bool) {
	sr.mu.RLock()
	info, exists := sr.services[name]
	sr.mu.RUnlock()

	if !exists {
		return nil, false
	}

	info.mu.RLock()
	defer info.mu.RUnlock()
	value, exists := info.Metadata[key]
	return value, exists
}

// HealthySummary returns count of services by status
func (sr *ServiceRegistry) HealthySummary() map[ServiceStatus]int {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	summary := make(map[ServiceStatus]int)
	for _, info := range sr.services {
		info.mu.RLock()
		status := info.Status
		info.mu.RUnlock()
		summary[status]++
	}
	return summary
}

// IsHealthy returns true if all services are healthy
func (sr *ServiceRegistry) IsHealthy() bool {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	for _, info := range sr.services {
		info.mu.RLock()
		status := info.Status
		info.mu.RUnlock()

		if status != ServiceStatusHealthy {
			return false
		}
	}
	return true
}

// Managers methods for service registry

// RegisterService registers a service with the Managers
func (m *Managers) RegisterService(name, serviceType string, dependencies []string) *ServiceInfo {
	if m.serviceRegistry == nil {
		m.mu.Lock()
		if m.serviceRegistry == nil {
			m.serviceRegistry = newServiceRegistry()
		}
		m.mu.Unlock()
	}
	return m.serviceRegistry.Register(name, serviceType, dependencies)
}

// UnregisterService removes a service from Managers registry
func (m *Managers) UnregisterService(name string) {
	if m.serviceRegistry != nil {
		m.serviceRegistry.Unregister(name)
	}
}

// GetService retrieves service information
func (m *Managers) GetService(name string) (*ServiceInfo, bool) {
	if m.serviceRegistry == nil {
		return nil, false
	}
	return m.serviceRegistry.Get(name)
}

// ListServices returns all registered services
func (m *Managers) ListServices() []*ServiceInfo {
	if m.serviceRegistry == nil {
		return nil
	}
	return m.serviceRegistry.List()
}

// UpdateServiceStatus updates the status of a service
func (m *Managers) UpdateServiceStatus(name string, status ServiceStatus) {
	if m.serviceRegistry != nil {
		m.serviceRegistry.UpdateStatus(name, status)
	}
}

// ServiceHealthSummary returns health summary of all services
func (m *Managers) ServiceHealthSummary() map[ServiceStatus]int {
	if m.serviceRegistry == nil {
		return make(map[ServiceStatus]int)
	}
	return m.serviceRegistry.HealthySummary()
}

// IsAllServicesHealthy returns true if all services are healthy
func (m *Managers) IsAllServicesHealthy() bool {
	if m.serviceRegistry == nil {
		return true
	}
	return m.serviceRegistry.IsHealthy()
}
