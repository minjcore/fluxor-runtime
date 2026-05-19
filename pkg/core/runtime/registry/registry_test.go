package registry

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// testComponent is a simple test component implementation.
type testComponent struct {
	name string
}

func (c *testComponent) Name() string {
	return c.name
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.AllowOverwrite {
		t.Error("Expected AllowOverwrite to be false")
	}

	if config.MaxComponents != 0 {
		t.Errorf("Expected MaxComponents 0, got %d", config.MaxComponents)
	}
}

func TestNewManager(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	if manager == nil {
		t.Fatal("Expected manager to be non-nil")
	}

	if manager.Count() != 0 {
		t.Errorf("Expected initial count 0, got %d", manager.Count())
	}
}

func TestRegister(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component := &testComponent{name: "test"}
	err := manager.Register("test", component)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if manager.Count() != 1 {
		t.Errorf("Expected count 1, got %d", manager.Count())
	}

	if !manager.Exists("test") {
		t.Error("Expected component to exist")
	}
}

func TestRegister_WithOptions(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component := &testComponent{name: "test"}
	err := manager.Register("test", component,
		WithPriority(10),
		WithMetadataKey("type", "service"),
		WithMetadataKey("version", "1.0"),
	)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	info, err := manager.GetComponentInfo("test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if info.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", info.Priority)
	}

	if info.Metadata["type"] != "service" {
		t.Errorf("Expected metadata type 'service', got %v", info.Metadata["type"])
	}

	if info.Metadata["version"] != "1.0" {
		t.Errorf("Expected metadata version '1.0', got %v", info.Metadata["version"])
	}
}

func TestRegister_EmptyName(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component := &testComponent{name: "test"}
	err := manager.Register("", component)
	if err == nil {
		t.Fatal("Expected error for empty name")
	}

	if e, ok := err.(*Error); ok {
		if e.Code != ErrCodeEmptyName {
			t.Errorf("Expected error code %s, got %s", ErrCodeEmptyName, e.Code)
		}
	} else {
		t.Errorf("Expected *Error, got %T", err)
	}
}

func TestRegister_NilComponent(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Register("test", nil)
	if err == nil {
		t.Fatal("Expected error for nil component")
	}

	if e, ok := err.(*Error); ok {
		if e.Code != ErrCodeNilComponent {
			t.Errorf("Expected error code %s, got %s", ErrCodeNilComponent, e.Code)
		}
	} else {
		t.Errorf("Expected *Error, got %T", err)
	}
}

func TestRegister_AlreadyExists(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component1 := &testComponent{name: "test"}
	err := manager.Register("test", component1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	component2 := &testComponent{name: "test2"}
	err = manager.Register("test", component2)
	if err == nil {
		t.Fatal("Expected error for duplicate name")
	}

	if e, ok := err.(*Error); ok {
		if e.Code != ErrCodeAlreadyExists {
			t.Errorf("Expected error code %s, got %s", ErrCodeAlreadyExists, e.Code)
		}
	} else {
		t.Errorf("Expected *Error, got %T", err)
	}
}

func TestRegister_AllowOverwrite(t *testing.T) {
	config := DefaultConfig()
	config.AllowOverwrite = true
	manager := NewManager(config)

	component1 := &testComponent{name: "test1"}
	err := manager.Register("test", component1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	component2 := &testComponent{name: "test2"}
	err = manager.Register("test", component2)
	if err != nil {
		t.Fatalf("Expected no error on overwrite, got %v", err)
	}

	component, exists := manager.Get("test")
	if !exists {
		t.Fatal("Expected component to exist")
	}

	if component.Name() != "test2" {
		t.Errorf("Expected component name test2, got %s", component.Name())
	}
}

func TestRegister_MaxComponents(t *testing.T) {
	config := DefaultConfig()
	config.MaxComponents = 2
	manager := NewManager(config)

	component1 := &testComponent{name: "test1"}
	err := manager.Register("test1", component1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	component2 := &testComponent{name: "test2"}
	err = manager.Register("test2", component2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	component3 := &testComponent{name: "test3"}
	err = manager.Register("test3", component3)
	if err == nil {
		t.Fatal("Expected error for exceeding max components")
	}

	if e, ok := err.(*Error); ok {
		if e.Code != ErrCodeMaxComponents {
			t.Errorf("Expected error code %s, got %s", ErrCodeMaxComponents, e.Code)
		}
	} else {
		t.Errorf("Expected *Error, got %T", err)
	}
}

func TestRegister_Validator(t *testing.T) {
	config := DefaultConfig()
	config.Validator = func(name string, component Component) error {
		if name == "invalid" {
			return fmt.Errorf("invalid component name")
		}
		return nil
	}
	manager := NewManager(config)

	component := &testComponent{name: "test"}
	err := manager.Register("invalid", component)
	if err == nil {
		t.Fatal("Expected error for invalid component")
	}

	if e, ok := err.(*Error); ok {
		if e.Code != ErrCodeValidationFailed {
			t.Errorf("Expected error code %s, got %s", ErrCodeValidationFailed, e.Code)
		}
	} else {
		t.Errorf("Expected *Error, got %T", err)
	}

	// Valid component should register successfully
	err = manager.Register("valid", component)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestUnregister(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component := &testComponent{name: "test"}
	err := manager.Register("test", component)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = manager.Unregister("test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if manager.Count() != 0 {
		t.Errorf("Expected count 0, got %d", manager.Count())
	}

	if manager.Exists("test") {
		t.Error("Expected component to not exist")
	}
}

func TestUnregister_NotFound(t *testing.T) {
	manager := NewManager(DefaultConfig())

	err := manager.Unregister("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent component")
	}

	if e, ok := err.(*Error); ok {
		if e.Code != ErrCodeNotFound {
			t.Errorf("Expected error code %s, got %s", ErrCodeNotFound, e.Code)
		}
	} else {
		t.Errorf("Expected *Error, got %T", err)
	}
}

func TestGet(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component := &testComponent{name: "test"}
	err := manager.Register("test", component)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	retrieved, exists := manager.Get("test")
	if !exists {
		t.Fatal("Expected component to exist")
	}

	if retrieved != component {
		t.Error("Expected retrieved component to be the same instance")
	}

	// Check that access tracking was updated
	info, err := manager.GetComponentInfo("test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if info.AccessCount != 1 {
		t.Errorf("Expected AccessCount 1, got %d", info.AccessCount)
	}

	if info.LastAccessTime.IsZero() {
		t.Error("Expected LastAccessTime to be set")
	}
}

func TestGetComponentInfo(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component := &testComponent{name: "test"}
	manager.Register("test", component, WithPriority(5), WithMetadataKey("key", "value"))

	info, err := manager.GetComponentInfo("test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if info.Name != "test" {
		t.Errorf("Expected name test, got %s", info.Name)
	}

	if info.Priority != 5 {
		t.Errorf("Expected priority 5, got %d", info.Priority)
	}

	if info.Metadata["key"] != "value" {
		t.Errorf("Expected metadata key 'value', got %v", info.Metadata["key"])
	}

	if info.RegisteredAt.IsZero() {
		t.Error("Expected RegisteredAt to be set")
	}
}

func TestGetComponentInfo_NotFound(t *testing.T) {
	manager := NewManager(DefaultConfig())

	_, err := manager.GetComponentInfo("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent component")
	}

	if e, ok := err.(*Error); ok {
		if e.Code != ErrCodeNotFound {
			t.Errorf("Expected error code %s, got %s", ErrCodeNotFound, e.Code)
		}
	} else {
		t.Errorf("Expected *Error, got %T", err)
	}
}

func TestListSorted(t *testing.T) {
	manager := NewManager(DefaultConfig())

	// Register components with different priorities and times
	time.Sleep(1 * time.Millisecond)
	manager.Register("a", &testComponent{name: "a"}, WithPriority(3))
	time.Sleep(1 * time.Millisecond)
	manager.Register("b", &testComponent{name: "b"}, WithPriority(1))
	time.Sleep(1 * time.Millisecond)
	manager.Register("c", &testComponent{name: "c"}, WithPriority(2))

	// Test SortByName
	names := manager.ListSorted(SortByName)
	expected := []string{"a", "b", "c"}
	if len(names) != len(expected) {
		t.Fatalf("Expected %d names, got %d", len(expected), len(names))
	}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("Expected name %s at index %d, got %s", name, i, names[i])
		}
	}

	// Test SortByPriority
	names = manager.ListSorted(SortByPriority)
	expected = []string{"b", "c", "a"} // priorities: 1, 2, 3
	if len(names) != len(expected) {
		t.Fatalf("Expected %d names, got %d", len(expected), len(names))
	}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("Expected name %s at index %d, got %s", name, i, names[i])
		}
	}
}

func TestListComponents(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component1 := &testComponent{name: "test1"}
	component2 := &testComponent{name: "test2"}

	manager.Register("test1", component1, WithPriority(1))
	manager.Register("test2", component2, WithPriority(2))

	components := manager.ListComponents()
	if len(components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(components))
	}

	// Verify components are returned
	found1, found2 := false, false
	for _, comp := range components {
		if comp.Name == "test1" {
			found1 = true
			if comp.Priority != 1 {
				t.Errorf("Expected priority 1, got %d", comp.Priority)
			}
		}
		if comp.Name == "test2" {
			found2 = true
			if comp.Priority != 2 {
				t.Errorf("Expected priority 2, got %d", comp.Priority)
			}
		}
	}

	if !found1 || !found2 {
		t.Error("Expected both components to be in list")
	}
}

func TestFilter(t *testing.T) {
	manager := NewManager(DefaultConfig())

	manager.Register("service1", &testComponent{name: "service1"}, WithMetadataKey("type", "service"))
	manager.Register("service2", &testComponent{name: "service2"}, WithMetadataKey("type", "service"))
	manager.Register("worker1", &testComponent{name: "worker1"}, WithMetadataKey("type", "worker"))

	// Filter by metadata
	services := manager.Filter(func(info *ComponentInfo) bool {
		return info.Metadata["type"] == "service"
	})

	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}

	// Filter by priority
	highPriority := manager.Filter(func(info *ComponentInfo) bool {
		return info.Priority >= 0 // All have default priority 0
	})

	if len(highPriority) != 3 {
		t.Errorf("Expected 3 components, got %d", len(highPriority))
	}
}

func TestFind(t *testing.T) {
	manager := NewManager(DefaultConfig())

	manager.Register("service1", &testComponent{name: "service1"}, WithMetadataKey("type", "service"))
	manager.Register("worker1", &testComponent{name: "worker1"}, WithMetadataKey("type", "worker"))

	// Find by metadata
	info, found := manager.Find(func(comp *ComponentInfo) bool {
		return comp.Metadata["type"] == "service"
	})

	if !found {
		t.Fatal("Expected to find service component")
	}

	if info.Name != "service1" {
		t.Errorf("Expected name service1, got %s", info.Name)
	}

	// Find non-existent
	_, found = manager.Find(func(comp *ComponentInfo) bool {
		return comp.Metadata["type"] == "nonexistent"
	})

	if found {
		t.Error("Expected not to find component")
	}
}

func TestRegisterMany(t *testing.T) {
	manager := NewManager(DefaultConfig())

	components := map[string]Component{
		"test1": &testComponent{name: "test1"},
		"test2": &testComponent{name: "test2"},
		"test3": &testComponent{name: "test3"},
	}

	err := manager.RegisterMany(components)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if manager.Count() != 3 {
		t.Errorf("Expected count 3, got %d", manager.Count())
	}
}

func TestRegisterMany_WithError(t *testing.T) {
	config := DefaultConfig()
	config.MaxComponents = 2
	manager := NewManager(config)

	components := map[string]Component{
		"test1": &testComponent{name: "test1"},
		"test2": &testComponent{name: "test2"},
		"test3": &testComponent{name: "test3"},
	}

	err := manager.RegisterMany(components)
	if err == nil {
		t.Fatal("Expected error for exceeding max components")
	}

	// Should have rolled back
	if manager.Count() != 0 {
		t.Errorf("Expected count 0 after rollback, got %d", manager.Count())
	}
}

func TestUnregisterMany(t *testing.T) {
	manager := NewManager(DefaultConfig())

	manager.Register("test1", &testComponent{name: "test1"})
	manager.Register("test2", &testComponent{name: "test2"})
	manager.Register("test3", &testComponent{name: "test3"})

	err := manager.UnregisterMany([]string{"test1", "test2", "test3"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if manager.Count() != 0 {
		t.Errorf("Expected count 0, got %d", manager.Count())
	}
}

func TestUnregisterMany_WithError(t *testing.T) {
	manager := NewManager(DefaultConfig())

	manager.Register("test1", &testComponent{name: "test1"})

	err := manager.UnregisterMany([]string{"test1", "nonexistent"})
	if err == nil {
		t.Fatal("Expected error for nonexistent component")
	}

	// test1 should still be unregistered
	if manager.Exists("test1") {
		t.Error("Expected test1 to be unregistered")
	}
}

func TestStats_AccessTracking(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component := &testComponent{name: "test"}
	manager.Register("test", component)

	stats := manager.Stats()
	if stats.TotalAccesses != 0 {
		t.Errorf("Expected TotalAccesses 0, got %d", stats.TotalAccesses)
	}

	manager.Get("test")
	manager.Get("test")

	stats = manager.Stats()
	if stats.TotalAccesses != 2 {
		t.Errorf("Expected TotalAccesses 2, got %d", stats.TotalAccesses)
	}

	if stats.LastAccessTime.IsZero() {
		t.Error("Expected LastAccessTime to be set")
	}
}

func TestOnAccess_Callback(t *testing.T) {
	var accessedName string
	var accessedComponent Component

	config := DefaultConfig()
	config.OnAccess = func(name string, component Component) {
		accessedName = name
		accessedComponent = component
	}

	manager := NewManager(config)

	component := &testComponent{name: "test"}
	manager.Register("test", component)
	manager.Get("test")

	if accessedName != "test" {
		t.Errorf("Expected accessed name test, got %s", accessedName)
	}

	if accessedComponent != component {
		t.Error("Expected accessed component to be the same instance")
	}
}

func TestOnAccessAsync_Callback(t *testing.T) {
	var accessedName string
	var mu sync.Mutex

	config := DefaultConfig()
	config.OnAccessAsync = func(name string, component Component) {
		mu.Lock()
		defer mu.Unlock()
		accessedName = name
	}

	manager := NewManager(config)

	component := &testComponent{name: "test"}
	manager.Register("test", component)
	manager.Get("test")

	// Wait a bit for async callback
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if accessedName != "test" {
		t.Errorf("Expected accessed name test, got %s", accessedName)
	}
	mu.Unlock()
}

func TestConcurrentRegister(t *testing.T) {
	manager := NewManager(DefaultConfig())

	var wg sync.WaitGroup
	numGoroutines := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			component := &testComponent{name: fmt.Sprintf("test%d", id)}
			manager.Register(fmt.Sprintf("test%d", id), component)
		}(i)
	}

	wg.Wait()

	if manager.Count() != numGoroutines {
		t.Errorf("Expected count %d, got %d", numGoroutines, manager.Count())
	}
}

func TestConcurrentGet(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component := &testComponent{name: "test"}
	manager.Register("test", component)

	var wg sync.WaitGroup
	numGoroutines := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, exists := manager.Get("test")
			if !exists {
				t.Error("Expected component to exist")
			}
		}()
	}

	wg.Wait()

	info, _ := manager.GetComponentInfo("test")
	if info.AccessCount != int64(numGoroutines) {
		t.Errorf("Expected AccessCount %d, got %d", numGoroutines, info.AccessCount)
	}
}

func TestRegisterWithContext(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component := &testComponent{name: "test"}
	ctx := context.Background()

	err := RegisterWithContext(ctx, manager, "test", component)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !manager.Exists("test") {
		t.Error("Expected component to exist")
	}
}

func TestRegisterWithContext_WithOptions(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component := &testComponent{name: "test"}
	ctx := context.Background()

	err := RegisterWithContext(ctx, manager, "test", component, WithPriority(5))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	info, _ := manager.GetComponentInfo("test")
	if info.Priority != 5 {
		t.Errorf("Expected priority 5, got %d", info.Priority)
	}
}

// Keep existing tests that don't conflict
func TestList(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component1 := &testComponent{name: "test1"}
	component2 := &testComponent{name: "test2"}
	component3 := &testComponent{name: "test3"}

	manager.Register("test1", component1)
	manager.Register("test2", component2)
	manager.Register("test3", component3)

	list := manager.List()
	if len(list) != 3 {
		t.Errorf("Expected list length 3, got %d", len(list))
	}

	// Check that all names are present
	names := make(map[string]bool)
	for _, name := range list {
		names[name] = true
	}

	if !names["test1"] || !names["test2"] || !names["test3"] {
		t.Error("Expected all component names to be in list")
	}
}

func TestList_Empty(t *testing.T) {
	manager := NewManager(DefaultConfig())

	list := manager.List()
	if len(list) != 0 {
		t.Errorf("Expected empty list, got %d items", len(list))
	}
}

func TestCount(t *testing.T) {
	manager := NewManager(DefaultConfig())

	if manager.Count() != 0 {
		t.Errorf("Expected initial count 0, got %d", manager.Count())
	}

	component1 := &testComponent{name: "test1"}
	manager.Register("test1", component1)
	if manager.Count() != 1 {
		t.Errorf("Expected count 1, got %d", manager.Count())
	}

	component2 := &testComponent{name: "test2"}
	manager.Register("test2", component2)
	if manager.Count() != 2 {
		t.Errorf("Expected count 2, got %d", manager.Count())
	}

	manager.Unregister("test1")
	if manager.Count() != 1 {
		t.Errorf("Expected count 1, got %d", manager.Count())
	}
}

func TestExists(t *testing.T) {
	manager := NewManager(DefaultConfig())

	if manager.Exists("test") {
		t.Error("Expected component to not exist")
	}

	component := &testComponent{name: "test"}
	manager.Register("test", component)
	if !manager.Exists("test") {
		t.Error("Expected component to exist")
	}

	manager.Unregister("test")
	if manager.Exists("test") {
		t.Error("Expected component to not exist after unregister")
	}
}

func TestClear(t *testing.T) {
	manager := NewManager(DefaultConfig())

	component1 := &testComponent{name: "test1"}
	component2 := &testComponent{name: "test2"}
	component3 := &testComponent{name: "test3"}

	manager.Register("test1", component1)
	manager.Register("test2", component2)
	manager.Register("test3", component3)

	if manager.Count() != 3 {
		t.Errorf("Expected count 3, got %d", manager.Count())
	}

	manager.Clear()

	if manager.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", manager.Count())
	}

	if manager.Exists("test1") || manager.Exists("test2") || manager.Exists("test3") {
		t.Error("Expected all components to be removed")
	}
}

func TestStats(t *testing.T) {
	manager := NewManager(DefaultConfig())

	stats := manager.Stats()
	if stats.TotalRegistered != 0 {
		t.Errorf("Expected TotalRegistered 0, got %d", stats.TotalRegistered)
	}
	if stats.TotalUnregistered != 0 {
		t.Errorf("Expected TotalUnregistered 0, got %d", stats.TotalUnregistered)
	}
	if stats.CurrentCount != 0 {
		t.Errorf("Expected CurrentCount 0, got %d", stats.CurrentCount)
	}

	component1 := &testComponent{name: "test1"}
	manager.Register("test1", component1)

	stats = manager.Stats()
	if stats.TotalRegistered != 1 {
		t.Errorf("Expected TotalRegistered 1, got %d", stats.TotalRegistered)
	}
	if stats.CurrentCount != 1 {
		t.Errorf("Expected CurrentCount 1, got %d", stats.CurrentCount)
	}
	if stats.LastRegisterTime.IsZero() {
		t.Error("Expected LastRegisterTime to be set")
	}

	component2 := &testComponent{name: "test2"}
	manager.Register("test2", component2)

	stats = manager.Stats()
	if stats.TotalRegistered != 2 {
		t.Errorf("Expected TotalRegistered 2, got %d", stats.TotalRegistered)
	}
	if stats.CurrentCount != 2 {
		t.Errorf("Expected CurrentCount 2, got %d", stats.CurrentCount)
	}

	manager.Unregister("test1")

	stats = manager.Stats()
	if stats.TotalUnregistered != 1 {
		t.Errorf("Expected TotalUnregistered 1, got %d", stats.TotalUnregistered)
	}
	if stats.CurrentCount != 1 {
		t.Errorf("Expected CurrentCount 1, got %d", stats.CurrentCount)
	}
	if stats.LastUnregisterTime.IsZero() {
		t.Error("Expected LastUnregisterTime to be set")
	}
}

func TestOnRegister_Callback(t *testing.T) {
	var registeredName string
	var registeredComponent Component

	config := DefaultConfig()
	config.OnRegister = func(name string, component Component) {
		registeredName = name
		registeredComponent = component
	}

	manager := NewManager(config)

	component := &testComponent{name: "test"}
	err := manager.Register("test", component)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if registeredName != "test" {
		t.Errorf("Expected registered name test, got %s", registeredName)
	}

	if registeredComponent != component {
		t.Error("Expected registered component to be the same instance")
	}
}

func TestOnUnregister_Callback(t *testing.T) {
	var unregisteredName string
	var unregisteredComponent Component

	config := DefaultConfig()
	config.OnUnregister = func(name string, component Component) {
		unregisteredName = name
		unregisteredComponent = component
	}

	manager := NewManager(config)

	component := &testComponent{name: "test"}
	manager.Register("test", component)
	manager.Unregister("test")

	if unregisteredName != "test" {
		t.Errorf("Expected unregistered name test, got %s", unregisteredName)
	}

	if unregisteredComponent != component {
		t.Error("Expected unregistered component to be the same instance")
	}
}

func TestError_Error(t *testing.T) {
	err := NewError(ErrCodeEmptyName, "test message")
	expected := "EMPTY_NAME: test message"
	if err.Error() != expected {
		t.Errorf("Expected error message %s, got %s", expected, err.Error())
	}
}

func TestError_Error_NoCode(t *testing.T) {
	err := &Error{Message: "test message"}
	expected := "test message"
	if err.Error() != expected {
		t.Errorf("Expected error message %s, got %s", expected, err.Error())
	}
}
