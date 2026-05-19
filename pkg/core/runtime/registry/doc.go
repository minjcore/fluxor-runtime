// Package registry provides component registration and management for runtime components.
//
// The registry allows you to register, unregister, and retrieve components by name.
// It provides thread-safe operations, statistics tracking, callback support for
// component lifecycle events, filtering, sorting, and batch operations.
//
// Features:
//   - Component metadata and priority support
//   - Access tracking (last access time, access count)
//   - Filtering and searching capabilities
//   - Multiple sorting options (by name, priority, registration time, access time, access count)
//   - Batch operations (register/unregister multiple components)
//   - Component validation
//   - Lifecycle callbacks (register, unregister, access)
//
// Example usage:
//
//	config := registry.DefaultConfig()
//	config.OnRegister = func(name string, component registry.Component) {
//		fmt.Printf("Component %s registered\n", name)
//	}
//
//	manager := registry.NewManager(config)
//
//	// Register a component with options
//	err := manager.Register("my-component", myComponent,
//		registry.WithPriority(10),
//		registry.WithMetadataKey("type", "service"),
//		registry.WithMetadataKey("version", "1.0"),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Retrieve a component
//	component, exists := manager.Get("my-component")
//	if exists {
//		// Use component
//	}
//
//	// Get detailed component information
//	info, err := manager.GetComponentInfo("my-component")
//	if err == nil {
//		fmt.Printf("Component accessed %d times\n", info.AccessCount)
//	}
//
//	// Filter components
//	services := manager.Filter(func(info *registry.ComponentInfo) bool {
//		return info.Metadata["type"] == "service"
//	})
//
//	// List components sorted by priority
//	names := manager.ListSorted(registry.SortByPriority)
//
//	// Unregister a component
//	err = manager.Unregister("my-component")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Package registry
package registry
