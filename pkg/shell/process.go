package shell

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

// ProcessSpec describes a process to run (command + args; terminal/interface).
type ProcessSpec struct {
	Name    string            // Unique name for the process (e.g. "api", "worker")
	Command string            // Executable (e.g. "python", "go", "node")
	Args    []string          // Arguments (e.g. ["app.py"], ["run", "main.go"])
	Cwd     string            // Working directory (empty = current)
	Env     map[string]string // Extra env vars (merged with os.Environ())
}

// ProcessState is the current state of a managed process.
type ProcessState struct {
	Name       string    `json:"name"`
	PID        int       `json:"pid"`
	Status     string    `json:"status"` // "running", "stopped", "exited"
	StartedAt  time.Time `json:"started_at,omitempty"`
	ExitedAt   time.Time `json:"exited_at,omitempty"`
	ExitCode   int       `json:"exit_code,omitempty"`
	Command    string    `json:"command"`
	Args       []string  `json:"args,omitempty"`
}

// ProcessRuntime runs and manages local processes (PM2-style: start, stop, restart, status, logs).
// Use NewDefaultProcessRuntime() for the default implementation.
type ProcessRuntime interface {
	Start(spec ProcessSpec) error
	Stop(name string) error
	Restart(name string) error
	Status(name string) (ProcessState, error)
	Logs(name string) ([]string, error)
	List() []ProcessState
}

// DefaultProcessRuntime is the default in-process implementation of ProcessRuntime.
// Tracks processes by name; Logs() returns in-memory tail (or empty until agent/buffer is added).
type DefaultProcessRuntime struct {
	mu       sync.RWMutex
	procs    map[string]*managedProcess
	logBuf   map[string][]string
	maxLogs  int
}

type managedProcess struct {
	spec   ProcessSpec
	cmd    *exec.Cmd
	start  time.Time
	exitAt time.Time
	code   int
}

// NewDefaultProcessRuntime returns a ProcessRuntime that runs processes locally (no daemon yet).
func NewDefaultProcessRuntime() ProcessRuntime {
	return &DefaultProcessRuntime{
		procs:   make(map[string]*managedProcess),
		logBuf:  make(map[string][]string),
		maxLogs: 1000,
	}
}

// Start starts a process from the given spec. Returns error if name already running or start fails.
func (r *DefaultProcessRuntime) Start(spec ProcessSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("process name is required")
	}
	if spec.Command == "" {
		return fmt.Errorf("process command is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.procs[spec.Name]; ok {
		return fmt.Errorf("process %q already running", spec.Name)
	}

	cmd := exec.Command(spec.Command, spec.Args...)
	cmd.Dir = spec.Cwd
	if cmd.Dir == "" {
		cmd.Dir, _ = os.Getwd()
	}
	cmd.Env = os.Environ()
	for k, v := range spec.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", spec.Name, err)
	}

	r.procs[spec.Name] = &managedProcess{spec: spec, cmd: cmd, start: time.Now()}
	r.logBuf[spec.Name] = nil

	go r.waitProcess(spec.Name)
	return nil
}

func (r *DefaultProcessRuntime) waitProcess(name string) {
	// cmd.Wait() must be called; we do it in a goroutine so Start() returns immediately.
	r.mu.RLock()
	mp, ok := r.procs[name]
	r.mu.RUnlock()
	if !ok {
		return
	}
	err := mp.cmd.Wait()
	r.mu.Lock()
	defer r.mu.Unlock()
	if mp2, ok := r.procs[name]; ok {
		mp2.exitAt = time.Now()
		mp2.code = 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				mp2.code = exitErr.ExitCode()
			}
		}
		// Mark as exited but keep in map so Status() can report exit code
		// Optionally delete: delete(r.procs, name)
	}
}

// Stop stops the process by name (SIGTERM then SIGKILL if needed). Idempotent.
func (r *DefaultProcessRuntime) Stop(name string) error {
	r.mu.Lock()
	mp, ok := r.procs[name]
	if !ok {
		r.mu.Unlock()
		return nil
	}
	cmd := mp.cmd
	r.mu.Unlock()

	if cmd.Process == nil {
		return nil
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		_ = cmd.Process.Kill()
	}
	// waitProcess will run and update state
	_ = cmd.Wait()
	r.mu.Lock()
	delete(r.procs, name)
	r.mu.Unlock()
	return nil
}

// Restart stops then starts the process again. The spec must still be known; we keep it in managedProcess.
func (r *DefaultProcessRuntime) Restart(name string) error {
	r.mu.RLock()
	mp, ok := r.procs[name]
	if !ok {
		r.mu.RUnlock()
		return fmt.Errorf("process %q not found", name)
	}
	spec := mp.spec
	r.mu.RUnlock()

	_ = r.Stop(name)
	return r.Start(spec)
}

// Status returns the current state of the process by name.
func (r *DefaultProcessRuntime) Status(name string) (ProcessState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mp, ok := r.procs[name]
	if !ok {
		return ProcessState{Name: name, Status: "stopped"}, nil
	}

	st := ProcessState{
		Name:      name,
		Command:   mp.spec.Command,
		Args:      mp.spec.Args,
		StartedAt: mp.start,
	}
	if mp.cmd.Process != nil {
		st.PID = mp.cmd.Process.Pid
		st.Status = "running"
	} else {
		st.Status = "exited"
		st.ExitedAt = mp.exitAt
		st.ExitCode = mp.code
	}
	return st, nil
}

// Logs returns recent log lines for the process. Default implementation returns empty (attach stdout/stderr later).
func (r *DefaultProcessRuntime) Logs(name string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lines, _ := r.logBuf[name]
	out := make([]string, len(lines))
	copy(out, lines)
	return out, nil
}

// List returns all known process states.
func (r *DefaultProcessRuntime) List() []ProcessState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []ProcessState
	for name := range r.procs {
		st, _ := r.statusLocked(name)
		list = append(list, st)
	}
	return list
}

func (r *DefaultProcessRuntime) statusLocked(name string) (ProcessState, bool) {
	mp, ok := r.procs[name]
	if !ok {
		return ProcessState{Name: name, Status: "stopped"}, false
	}
	st := ProcessState{
		Name:      name,
		Command:   mp.spec.Command,
		Args:      mp.spec.Args,
		StartedAt: mp.start,
	}
	if mp.cmd.Process != nil {
		st.PID = mp.cmd.Process.Pid
		st.Status = "running"
	} else {
		st.Status = "exited"
		st.ExitedAt = mp.exitAt
		st.ExitCode = mp.code
	}
	return st, true
}
