package kernel

// Plugin extends the kernel at construction time (auth, tracing, metrics registration).
type Plugin interface {
	Install(k Kernel)
}

// ApplyPlugins runs Install on each plugin (call before Kernel.Start).
func ApplyPlugins(k Kernel, plugins ...Plugin) {
	for _, p := range plugins {
		if p != nil {
			p.Install(k)
		}
	}
}
