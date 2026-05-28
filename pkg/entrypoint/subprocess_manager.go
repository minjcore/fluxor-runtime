package entrypoint

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

const (
	subprocDrainTimeout = 30 * time.Second
	subprocStartWindow  = 500 * time.Millisecond
)

// SubprocessManager spawns, monitors, and drains verticle subprocess binaries.
// Each subprocess receives FLUXOR_NATS_URL and FLUXOR_DEPLOYMENT_ID via env.
type SubprocessManager struct {
	natsURL string

	mu    sync.Mutex
	procs map[string]*managedProc
}

type managedProc struct {
	id      string
	binary  string
	cmd     *exec.Cmd
	started time.Time
	exited  chan struct{}
}

// NewSubprocessManager creates a manager that passes natsURL to every subprocess.
func NewSubprocessManager(natsURL string) *SubprocessManager {
	return &SubprocessManager{
		natsURL: natsURL,
		procs:   make(map[string]*managedProc),
	}
}

// Spawn starts binary as a subprocess verticle and returns its deployment ID.
// The binary must call entrypoint.RunSubprocess() in its main().
func (m *SubprocessManager) Spawn(binaryPath string) (string, error) {
	id := "subprocess." + uuid.New().String()

	cmd := exec.Command(binaryPath)
	cmd.Env = append(os.Environ(),
		"FLUXOR_NATS_URL="+m.natsURL,
		"FLUXOR_DEPLOYMENT_ID="+id,
	)
	cmd.Stdout = &linePrefix{prefix: fmt.Sprintf("[%s] ", id[:24]), w: os.Stdout}
	cmd.Stderr = &linePrefix{prefix: fmt.Sprintf("[%s] ", id[:24]), w: os.Stderr}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("spawn %s: %w", binaryPath, err)
	}

	proc := &managedProc{
		id:      id,
		binary:  binaryPath,
		cmd:     cmd,
		started: time.Now(),
		exited:  make(chan struct{}),
	}

	m.mu.Lock()
	m.procs[id] = proc
	m.mu.Unlock()

	go func() {
		_ = cmd.Wait()
		close(proc.exited)
		m.mu.Lock()
		delete(m.procs, id)
		m.mu.Unlock()
		log.Printf("[subprocess] %s exited", id)
	}()

	// Fail-fast: if process dies immediately, report the error.
	select {
	case <-proc.exited:
		return "", fmt.Errorf("subprocess %s exited immediately after spawn", binaryPath)
	case <-time.After(subprocStartWindow):
	}

	log.Printf("[subprocess] spawned %s → %s (pid=%d)", binaryPath, id, cmd.Process.Pid)
	return id, nil
}

// Kill sends SIGTERM, waits for graceful exit, then SIGKILL on timeout.
func (m *SubprocessManager) Kill(id string) error {
	m.mu.Lock()
	proc, ok := m.procs[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("subprocess %q not found", id)
	}

	log.Printf("[subprocess] draining %s (pid=%d)...", id, proc.cmd.Process.Pid)
	_ = proc.cmd.Process.Signal(syscall.SIGTERM)

	select {
	case <-proc.exited:
		log.Printf("[subprocess] %s drained cleanly", id)
	case <-time.After(subprocDrainTimeout):
		log.Printf("[subprocess] drain timeout for %s, SIGKILL", id)
		_ = proc.cmd.Process.Kill()
		<-proc.exited
	}
	return nil
}

// List returns IDs of all running subprocesses.
func (m *SubprocessManager) List() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, 0, len(m.procs))
	for id := range m.procs {
		ids = append(ids, id)
	}
	return ids
}

// StopAll drains all subprocesses (called during parent shutdown).
func (m *SubprocessManager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.procs))
	for id := range m.procs {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.Kill(id)
	}
}

// linePrefix writes each log line with a prefix for subprocess identification.
type linePrefix struct {
	prefix string
	w      io.Writer
	buf    []byte
}

func (p *linePrefix) Write(b []byte) (int, error) {
	p.buf = append(p.buf, b...)
	for {
		idx := bytes.IndexByte(p.buf, '\n')
		if idx < 0 {
			break
		}
		line := p.buf[:idx+1]
		fmt.Fprintf(p.w, "%s%s", p.prefix, line)
		p.buf = p.buf[idx+1:]
	}
	return len(b), nil
}
