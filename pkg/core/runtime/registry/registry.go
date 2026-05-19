package registry

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Component represents a registered component.
type Component interface {
	// Name returns the name of the component.
	Name() string
}

// ComponentInfo contains metadata about a registered component.
type ComponentInfo struct {
	// Name is the unique identifier for the component.
	Name string

	// Component is the registered component.
	Component Component

	// Priority determines ordering when listing components. Lower values appear first.
	// Components with the same priority are ordered by registration time.
	Priority int

	// Metadata contains additional metadata about the component.
	Metadata map[string]interface{}

	// RegisteredAt is when the component was registered.
	RegisteredAt time.Time

	// LastAccessTime is when the component was last accessed via Get.
	LastAccessTime time.Time

	// AccessCount is the number of times this component has been accessed.
	AccessCount int64
}

// Manager provides component registration and management.
type Manager interface {
	// Register registers a component with the given name.
	// Returns an error if the name is empty, component is nil, or name already exists.
	Register(name string, component Component, opts ...Option) error

	// Unregister removes a component from the registry.
	// Returns an error if the component is not found.
	Unregister(name string) error

	// Get retrieves a component by name.
	// Returns the component and true if found, nil and false otherwise.
	Get(name string) (Component, bool)

	// GetComponentInfo returns detailed information about a component.
	GetComponentInfo(name string) (*ComponentInfo, error)

	// List returns all registered component names.
	List() []string

	// ListSorted returns all registered component names sorted by the given criteria.
	ListSorted(sortBy SortBy) []string

	// ListComponents returns all registered components with their info.
	ListComponents() []ComponentInfo

	// Count returns the number of registered components.
	Count() int

	// Exists checks if a component with the given name exists.
	Exists(name string) bool

	// Filter returns components that match the given filter function.
	Filter(filter func(*ComponentInfo) bool) []ComponentInfo

	// Find returns the first component that matches the given filter function.
	Find(filter func(*ComponentInfo) bool) (*ComponentInfo, bool)

	// Clear removes all components from the registry.
	Clear()

	// Stats returns statistics about the registry.
	Stats() Stats

	// RegisterMany registers multiple components at once.
	RegisterMany(components map[string]Component, opts ...Option) error

	// UnregisterMany unregisters multiple components at once.
	UnregisterMany(names []string) error
}

// Option configures component registration.
type Option func(*ComponentInfo)

// WithPriority sets the component priority.
func WithPriority(priority int) Option {
	return func(info *ComponentInfo) {
		info.Priority = priority
	}
}

// WithMetadata sets component metadata.
func WithMetadata(metadata map[string]interface{}) Option {
	return func(info *ComponentInfo) {
		if info.Metadata == nil {
			info.Metadata = make(map[string]interface{})
		}
		for k, v := range metadata {
			info.Metadata[k] = v
		}
	}
}

// WithMetadataKey sets a single metadata key-value pair.
func WithMetadataKey(key string, value interface{}) Option {
	return func(info *ComponentInfo) {
		if info.Metadata == nil {
			info.Metadata = make(map[string]interface{})
		}
		info.Metadata[key] = value
	}
}

// SortBy determines how components are sorted.
type SortBy int

const (
	// SortByName sorts by component name (alphabetical).
	SortByName SortBy = iota

	// SortByPriority sorts by priority (ascending), then by registration time.
	SortByPriority

	// SortByRegistrationTime sorts by registration time (oldest first).
	SortByRegistrationTime

	// SortByAccessTime sorts by last access time (most recent first).
	SortByAccessTime

	// SortByAccessCount sorts by access count (most accessed first).
	SortByAccessCount
)

// Config configures registry behavior.
type Config struct {
	// OnRegister is called when a component is registered.
	OnRegister func(name string, component Component)

	// OnRegisterAsync is called asynchronously when a component is registered.
	OnRegisterAsync func(name string, component Component)

	// OnUnregister is called when a component is unregistered.
	OnUnregister func(name string, component Component)

	// OnUnregisterAsync is called asynchronously when a component is unregistered.
	OnUnregisterAsync func(name string, component Component)

	// OnAccess is called when a component is accessed via Get.
	OnAccess func(name string, component Component)

	// OnAccessAsync is called asynchronously when a component is accessed.
	OnAccessAsync func(name string, component Component)

	// AllowOverwrite determines if registering a component with an existing name
	// should overwrite the existing component. Defaults to false.
	AllowOverwrite bool

	// MaxComponents is the maximum number of components that can be registered.
	// Zero means no limit.
	MaxComponents int

	// Validator is a function that validates components before registration.
	// If it returns an error, registration fails.
	Validator func(name string, component Component) error
}

// DefaultConfig returns the default registry configuration.
func DefaultConfig() Config {
	return Config{
		AllowOverwrite: false,
		MaxComponents:  0, // no limit
	}
}

// Stats contains statistics about the registry.
type Stats struct {
	// TotalRegistered is the total number of components that have been registered.
	TotalRegistered int64

	// TotalUnregistered is the total number of components that have been unregistered.
	TotalUnregistered int64

	// TotalAccesses is the total number of component accesses.
	TotalAccesses int64

	// CurrentCount is the current number of registered components.
	CurrentCount int

	// LastRegisterTime is when the last component was registered.
	LastRegisterTime time.Time

	// LastUnregisterTime is when the last component was unregistered.
	LastUnregisterTime time.Time

	// LastAccessTime is when a component was last accessed.
	LastAccessTime time.Time
}

// registryManager implements the Manager interface.
type registryManager struct {
	config Config

	// Component storage with metadata
	components map[string]*ComponentInfo
	mu         sync.RWMutex

	// Statistics
	totalRegistered   int64
	totalUnregistered int64
	totalAccesses      int64
	lastRegisterTime   atomic.Value // stores time.Time
	lastUnregisterTime atomic.Value // stores time.Time
	lastAccessTime     atomic.Value // stores time.Time
}

// NewManager creates a new registry manager with the given configuration.
func NewManager(config Config) Manager {
	return &registryManager{
		config:     config,
		components: make(map[string]*ComponentInfo),
	}
}

// Register registers a component with the given name and options.
func (r *registryManager) Register(name string, component Component, opts ...Option) error {
	if name == "" {
		return NewError(ErrCodeEmptyName, "component name cannot be empty")
	}
	if component == nil {
		return NewError(ErrCodeNilComponent, "component cannot be nil")
	}

	// Validate component if validator is configured
	if r.config.Validator != nil {
		if err := r.config.Validator(name, component); err != nil {
			return NewError(ErrCodeValidationFailed, fmt.Sprintf("validation failed: %v", err))
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already exists
	if _, exists := r.components[name]; exists {
		if !r.config.AllowOverwrite {
			return NewError(ErrCodeAlreadyExists, fmt.Sprintf("component %s already exists", name))
		}
	}

	// Check max components limit
	if r.config.MaxComponents > 0 && len(r.components) >= r.config.MaxComponents {
		if _, exists := r.components[name]; !exists {
			return NewError(ErrCodeMaxComponents, fmt.Sprintf("maximum number of components (%d) reached", r.config.MaxComponents))
		}
	}

	// Create component info
	info := &ComponentInfo{
		Name:         name,
		Component:    component,
		Priority:     0, // Default priority
		Metadata:     make(map[string]interface{}),
		RegisteredAt: time.Now(),
	}

	// Apply options
	for _, opt := range opts {
		opt(info)
	}

	// Store the component
	r.components[name] = info
	atomic.AddInt64(&r.totalRegistered, 1)
	r.lastRegisterTime.Store(time.Now())

	// Notify callbacks
	r.notifyRegister(name, component)

	return nil
}

// Unregister removes a component from the registry.
func (r *registryManager) Unregister(name string) error {
	if name == "" {
		return NewError(ErrCodeEmptyName, "component name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.components[name]
	if !exists {
		return NewError(ErrCodeNotFound, fmt.Sprintf("component %s not found", name))
	}

	delete(r.components, name)
	atomic.AddInt64(&r.totalUnregistered, 1)
	r.lastUnregisterTime.Store(time.Now())

	// Notify callbacks
	r.notifyUnregister(name, info.Component)

	return nil
}

// Get retrieves a component by name and updates access tracking.
func (r *registryManager) Get(name string) (Component, bool) {
	if name == "" {
		return nil, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.components[name]
	if !exists {
		return nil, false
	}

	// Update access tracking
	now := time.Now()
	info.LastAccessTime = now
	atomic.AddInt64(&info.AccessCount, 1)
	atomic.AddInt64(&r.totalAccesses, 1)
	r.lastAccessTime.Store(now)

	// Notify callbacks
	r.notifyAccess(name, info.Component)

	return info.Component, true
}

// GetComponentInfo returns detailed information about a component.
func (r *registryManager) GetComponentInfo(name string) (*ComponentInfo, error) {
	if name == "" {
		return nil, NewError(ErrCodeEmptyName, "component name cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.components[name]
	if !exists {
		return nil, NewError(ErrCodeNotFound, fmt.Sprintf("component %s not found", name))
	}

	// Return a copy to prevent external modification
	copy := *info
	// Deep copy metadata
	if info.Metadata != nil {
		copy.Metadata = make(map[string]interface{})
		for k, v := range info.Metadata {
			copy.Metadata[k] = v
		}
	}
	return &copy, nil
}

// List returns all registered component names.
func (r *registryManager) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.components))
	for name := range r.components {
		names = append(names, name)
	}

	return names
}

// ListSorted returns all registered component names sorted by the given criteria.
func (r *registryManager) ListSorted(sortBy SortBy) []string {
	infos := r.ListComponents()

	// Sort based on criteria
	switch sortBy {
	case SortByName:
		sort.Slice(infos, func(i, j int) bool {
			return infos[i].Name < infos[j].Name
		})
	case SortByPriority:
		sort.Slice(infos, func(i, j int) bool {
			if infos[i].Priority != infos[j].Priority {
				return infos[i].Priority < infos[j].Priority
			}
			return infos[i].RegisteredAt.Before(infos[j].RegisteredAt)
		})
	case SortByRegistrationTime:
		sort.Slice(infos, func(i, j int) bool {
			return infos[i].RegisteredAt.Before(infos[j].RegisteredAt)
		})
	case SortByAccessTime:
		sort.Slice(infos, func(i, j int) bool {
			return infos[i].LastAccessTime.After(infos[j].LastAccessTime)
		})
	case SortByAccessCount:
		sort.Slice(infos, func(i, j int) bool {
			return atomic.LoadInt64(&infos[i].AccessCount) > atomic.LoadInt64(&infos[j].AccessCount)
		})
	}

	names := make([]string, len(infos))
	for i, info := range infos {
		names[i] = info.Name
	}

	return names
}

// ListComponents returns all registered components with their info.
func (r *registryManager) ListComponents() []ComponentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ComponentInfo, 0, len(r.components))
	for _, info := range r.components {
		// Return copies to prevent external modification
		copy := *info
		// Deep copy metadata
		if info.Metadata != nil {
			copy.Metadata = make(map[string]interface{})
			for k, v := range info.Metadata {
				copy.Metadata[k] = v
			}
		}
		infos = append(infos, copy)
	}

	return infos
}

// Count returns the number of registered components.
func (r *registryManager) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.components)
}

// Exists checks if a component with the given name exists.
func (r *registryManager) Exists(name string) bool {
	if name == "" {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.components[name]
	return exists
}

// Filter returns components that match the given filter function.
func (r *registryManager) Filter(filter func(*ComponentInfo) bool) []ComponentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []ComponentInfo
	for _, info := range r.components {
		if filter(info) {
			// Return a copy
			copy := *info
			if info.Metadata != nil {
				copy.Metadata = make(map[string]interface{})
				for k, v := range info.Metadata {
					copy.Metadata[k] = v
				}
			}
			result = append(result, copy)
		}
	}

	return result
}

// Find returns the first component that matches the given filter function.
func (r *registryManager) Find(filter func(*ComponentInfo) bool) (*ComponentInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, info := range r.components {
		if filter(info) {
			// Return a copy
			copy := *info
			if info.Metadata != nil {
				copy.Metadata = make(map[string]interface{})
				for k, v := range info.Metadata {
					copy.Metadata[k] = v
				}
			}
			return &copy, true
		}
	}

	return nil, false
}

// Clear removes all components from the registry.
func (r *registryManager) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Notify about unregistering all components
	for name, info := range r.components {
		r.notifyUnregister(name, info.Component)
	}

	r.components = make(map[string]*ComponentInfo)
}

// Stats returns statistics about the registry.
func (r *registryManager) Stats() Stats {
	r.mu.RLock()
	currentCount := len(r.components)
	r.mu.RUnlock()

	var lastRegisterTime time.Time
	if v := r.lastRegisterTime.Load(); v != nil {
		lastRegisterTime = v.(time.Time)
	}

	var lastUnregisterTime time.Time
	if v := r.lastUnregisterTime.Load(); v != nil {
		lastUnregisterTime = v.(time.Time)
	}

	var lastAccessTime time.Time
	if v := r.lastAccessTime.Load(); v != nil {
		lastAccessTime = v.(time.Time)
	}

	return Stats{
		TotalRegistered:   atomic.LoadInt64(&r.totalRegistered),
		TotalUnregistered: atomic.LoadInt64(&r.totalUnregistered),
		TotalAccesses:     atomic.LoadInt64(&r.totalAccesses),
		CurrentCount:      currentCount,
		LastRegisterTime:  lastRegisterTime,
		LastUnregisterTime: lastUnregisterTime,
		LastAccessTime:    lastAccessTime,
	}
}

// RegisterMany registers multiple components at once.
func (r *registryManager) RegisterMany(components map[string]Component, opts ...Option) error {
	registered := make([]string, 0, len(components))

	for name, component := range components {
		wasExisting := r.Exists(name)

		if err := r.Register(name, component, opts...); err != nil {
			// Rollback only components we registered in this call
			for _, regName := range registered {
				r.Unregister(regName)
			}
			return fmt.Errorf("failed to register %s: %w", name, err)
		}

		// Only track if we actually registered it (not if it already existed without overwrite)
		if !wasExisting || r.config.AllowOverwrite {
			registered = append(registered, name)
		}
	}
	return nil
}

// UnregisterMany unregisters multiple components at once.
func (r *registryManager) UnregisterMany(names []string) error {
	var errors []error
	for _, name := range names {
		if err := r.Unregister(name); err != nil {
			errors = append(errors, fmt.Errorf("failed to unregister %s: %w", name, err))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("unregister errors: %v", errors)
	}
	return nil
}

// notifyRegister notifies callbacks about a component registration.
func (r *registryManager) notifyRegister(name string, component Component) {
	// Call synchronous callback
	if r.config.OnRegister != nil {
		r.config.OnRegister(name, component)
	}

	// Call asynchronous callback
	if r.config.OnRegisterAsync != nil {
		go r.config.OnRegisterAsync(name, component)
	}
}

// notifyUnregister notifies callbacks about a component unregistration.
func (r *registryManager) notifyUnregister(name string, component Component) {
	// Call synchronous callback
	if r.config.OnUnregister != nil {
		r.config.OnUnregister(name, component)
	}

	// Call asynchronous callback
	if r.config.OnUnregisterAsync != nil {
		go r.config.OnUnregisterAsync(name, component)
	}
}

// notifyAccess notifies callbacks about a component access.
func (r *registryManager) notifyAccess(name string, component Component) {
	// Call synchronous callback
	if r.config.OnAccess != nil {
		r.config.OnAccess(name, component)
	}

	// Call asynchronous callback
	if r.config.OnAccessAsync != nil {
		go r.config.OnAccessAsync(name, component)
	}
}

// RegisterWithContext registers a component with a context for timeout support.
// This is a convenience method that wraps Register.
func RegisterWithContext(ctx context.Context, manager Manager, name string, component Component, opts ...Option) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	done := make(chan error, 1)
	go func() {
		done <- manager.Register(name, component, opts...)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return NewError(ErrCodeContextCanceled, fmt.Sprintf("registration canceled: %v", ctx.Err()))
	}
}

// UnregisterWithContext unregisters a component with a context for timeout support.
// This is a convenience method that wraps Unregister.
func UnregisterWithContext(ctx context.Context, manager Manager, name string) error {
	if ctx == nil {
		return NewError(ErrCodeNilContext, "context cannot be nil")
	}

	done := make(chan error, 1)
	go func() {
		done <- manager.Unregister(name)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return NewError(ErrCodeContextCanceled, fmt.Sprintf("unregistration canceled: %v", ctx.Err()))
	}
}
