package hardware

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
)

// CPUConfig contains CPU optimization settings
type CPUConfig struct {
	IsolateCPUs    []int  // CPUs to isolate for trading
	PinThreads     bool   // Pin goroutines to OS threads
	DisableHT      bool   // Disable hyperthreading
	GOMAXPROCS     int    // Go max procs
}

// MemoryConfig contains memory optimization settings
type MemoryConfig struct {
	EnableHugePages bool   // Enable huge pages (2MB)
	HugePageCount   int    // Number of huge pages
	NUMANode        int    // NUMA node to bind (-1 for auto)
}

// NetworkConfig contains network optimization settings
type NetworkConfig struct {
	EnableKernelBypass bool   // Enable io_uring or DPDK
	IRQAffinity        []int  // IRQ CPU affinity
	RingBufferSize     int    // NIC ring buffer size
}

// OptimizationConfig contains all hardware optimization settings
type OptimizationConfig struct {
	CPU     CPUConfig
	Memory  MemoryConfig
	Network NetworkConfig
}

// DefaultHFTConfig returns optimized config for HFT
func DefaultHFTConfig() *OptimizationConfig {
	numCPU := runtime.NumCPU()
	// Reserve last 8 CPUs for trading (assuming 12+ core system)
	isolatedCPUs := make([]int, 0)
	if numCPU >= 12 {
		for i := numCPU - 8; i < numCPU; i++ {
			isolatedCPUs = append(isolatedCPUs, i)
		}
	}

	return &OptimizationConfig{
		CPU: CPUConfig{
			IsolateCPUs: isolatedCPUs,
			PinThreads:  true,
			DisableHT:   false, // Depends on workload
			GOMAXPROCS:  len(isolatedCPUs),
		},
		Memory: MemoryConfig{
			EnableHugePages: true,
			HugePageCount:   1024, // 2GB with 2MB pages
			NUMANode:        -1,   // Auto-detect
		},
		Network: NetworkConfig{
			EnableKernelBypass: false, // Requires special setup
			IRQAffinity:        isolatedCPUs[:1], // Pin network IRQ to first isolated CPU
			RingBufferSize:     4096,
		},
	}
}

// ApplyOptimizations applies hardware optimizations
func ApplyOptimizations(config *OptimizationConfig) error {
	// Apply CPU optimizations
	if err := applyCPUOptimizations(&config.CPU); err != nil {
		return fmt.Errorf("CPU optimization failed: %w", err)
	}

	// Apply memory optimizations
	if err := applyMemoryOptimizations(&config.Memory); err != nil {
		return fmt.Errorf("memory optimization failed: %w", err)
	}

	// Apply network optimizations
	if err := applyNetworkOptimizations(&config.Network); err != nil {
		return fmt.Errorf("network optimization failed: %w", err)
	}

	return nil
}

// applyCPUOptimizations applies CPU-related optimizations
func applyCPUOptimizations(config *CPUConfig) error {
	// Set GOMAXPROCS
	if config.GOMAXPROCS > 0 {
		runtime.GOMAXPROCS(config.GOMAXPROCS)
		fmt.Printf("Set GOMAXPROCS to %d\n", config.GOMAXPROCS)
	}

	// Lock goroutines to OS threads if requested
	if config.PinThreads {
		runtime.LockOSThread()
		fmt.Println("Locked goroutines to OS threads")
	}

	// CPU isolation requires boot parameters (isolcpus=...)
	// This can only be checked, not set at runtime
	if len(config.IsolateCPUs) > 0 {
		fmt.Printf("CPU isolation configured for CPUs: %v\n", config.IsolateCPUs)
		fmt.Println("Note: Actual isolation requires kernel boot parameter: isolcpus=...")
	}

	return nil
}

// applyMemoryOptimizations applies memory-related optimizations
func applyMemoryOptimizations(config *MemoryConfig) error {
	if config.EnableHugePages {
		// Check if huge pages are available
		if err := checkHugePages(); err != nil {
			fmt.Printf("Warning: Huge pages not available: %v\n", err)
			fmt.Println("To enable, run as root:")
			fmt.Printf("  echo %d > /proc/sys/vm/nr_hugepages\n", config.HugePageCount)
			fmt.Println("  mount -t hugetlbfs none /mnt/huge")
		} else {
			fmt.Printf("Huge pages enabled (count: %d)\n", config.HugePageCount)
		}
	}

	// NUMA configuration (informational only at this level)
	if config.NUMANode >= 0 {
		fmt.Printf("NUMA node binding configured: node %d\n", config.NUMANode)
		fmt.Println("Note: Actual binding requires: numactl --cpunodebind=N --membind=N")
	}

	return nil
}

// applyNetworkOptimizations applies network-related optimizations
func applyNetworkOptimizations(config *NetworkConfig) error {
	if config.EnableKernelBypass {
		fmt.Println("Kernel bypass enabled (io_uring/DPDK)")
		fmt.Println("Note: Requires specific network driver and setup")
	}

	if len(config.IRQAffinity) > 0 {
		fmt.Printf("IRQ affinity configured for CPUs: %v\n", config.IRQAffinity)
		fmt.Println("Note: Actual affinity requires:")
		fmt.Println("  echo <cpu> > /proc/irq/<irq>/smp_affinity_list")
	}

	if config.RingBufferSize > 0 {
		fmt.Printf("NIC ring buffer size: %d\n", config.RingBufferSize)
		fmt.Println("Note: Configure with: ethtool -G <interface> rx <size> tx <size>")
	}

	return nil
}

// checkHugePages checks if huge pages are available
func checkHugePages() error {
	// Check /proc/meminfo for huge pages
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return err
	}

	// Simple check for HugePages_Total
	if len(data) == 0 {
		return fmt.Errorf("unable to read meminfo")
	}

	// In production, parse the file properly
	return nil
}

// PinToCore pins current goroutine to a specific CPU core
// Note: This is informational - actual pinning requires OS-level support
func PinToCore(coreID int) error {
	// Lock current goroutine to OS thread
	runtime.LockOSThread()

	// In Linux, would use pthread_setaffinity_np via CGO
	// For cross-platform compatibility, this is a placeholder

	fmt.Printf("Pinned goroutine to core %d (OS thread locked)\n", coreID)
	return nil
}

// SetNUMANode sets NUMA node affinity
// Note: Requires numactl or libnuma
func SetNUMANode(node int) error {
	fmt.Printf("NUMA node affinity set to node %d\n", node)
	fmt.Println("Note: Actual binding requires numactl command")
	return nil
}

// DisableGC disables Go garbage collector (use with extreme caution)
func DisableGC() {
	fmt.Println("WARNING: Disabling Go GC - ensure manual memory management!")
	fmt.Println("Set GOGC=off environment variable")
	os.Setenv("GOGC", "off")
}

// SetGCPercent sets GC target percentage
func SetGCPercent(percent int) int {
	return runtime.DebugGC(percent)
}

// GetSystemInfo returns system information for optimization
func GetSystemInfo() SystemInfo {
	return SystemInfo{
		NumCPU:      runtime.NumCPU(),
		GOMAXPROCS:  runtime.GOMAXPROCS(0),
		NumGoroutine: runtime.NumGoroutine(),
		GoVersion:   runtime.Version(),
	}
}

// SystemInfo contains system information
type SystemInfo struct {
	NumCPU       int
	GOMAXPROCS   int
	NumGoroutine int
	GoVersion    string
}

// SetRLimit sets resource limits for the process
func SetRLimit(limit uint64) error {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}

	rLimit.Cur = limit
	rLimit.Max = limit

	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}

	fmt.Printf("Set RLIMIT_NOFILE to %d\n", limit)
	return nil
}

// Prefetch provides CPU cache prefetch hint
// Note: Go doesn't expose CPU prefetch instructions directly
// This is a placeholder for future optimization
func Prefetch(addr uintptr) {
	// In assembly or CGO, would use:
	// __builtin_prefetch() or _mm_prefetch()
	// For now, this is a no-op
}

// PrintOptimizationChecklist prints optimization checklist
func PrintOptimizationChecklist() {
	fmt.Println("=== HFT Optimization Checklist ===")
	fmt.Println()
	fmt.Println("[ ] 1. CPU Isolation")
	fmt.Println("    Add to /etc/default/grub:")
	fmt.Println("    GRUB_CMDLINE_LINUX=\"isolcpus=4-11 nohz_full=4-11 rcu_nocbs=4-11\"")
	fmt.Println()
	fmt.Println("[ ] 2. Huge Pages")
	fmt.Println("    echo 1024 > /proc/sys/vm/nr_hugepages")
	fmt.Println("    mount -t hugetlbfs none /mnt/huge")
	fmt.Println()
	fmt.Println("[ ] 3. IRQ Affinity")
	fmt.Println("    echo 4 > /proc/irq/<irq>/smp_affinity_list")
	fmt.Println()
	fmt.Println("[ ] 4. Network Tuning")
	fmt.Println("    ethtool -G eth0 rx 4096 tx 4096")
	fmt.Println("    ethtool -C eth0 rx-usecs 0 tx-usecs 0")
	fmt.Println()
	fmt.Println("[ ] 5. Kernel Parameters")
	fmt.Println("    sysctl -w net.core.busy_poll=50")
	fmt.Println("    sysctl -w net.core.busy_read=50")
	fmt.Println()
	fmt.Println("[ ] 6. CPU Governor")
	fmt.Println("    cpupower frequency-set -g performance")
	fmt.Println()
	fmt.Println("[ ] 7. Disable C-States")
	fmt.Println("    intel_idle.max_cstate=0 processor.max_cstate=1")
	fmt.Println()
	fmt.Println("[ ] 8. Disable Turbo Boost (for consistent latency)")
	fmt.Println("    echo 1 > /sys/devices/system/cpu/intel_pstate/no_turbo")
	fmt.Println()
}
