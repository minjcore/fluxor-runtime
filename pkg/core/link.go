package core

import (
	"sync"
)

// ExitSignal is sent to linked or monitoring verticles when a deployment exits.
type ExitSignal struct {
	From  string         // Deployment ID that exited
	State DeploymentState
	Err   error
}

// MonitorRef identifies a monitor so it can be removed with Demonitor.
type MonitorRef struct {
	monitorerID string // deployment ID of the verticle that called Monitor
	targetID    string
	ref         uint64
}

// LinkManager manages bidirectional links and one-way monitors between deployments.
// When a deployment exits (FAILED or STOPPED), exit signals are delivered via EventBus
// to linked peers and to monitoring verticles.
type LinkManager struct {
	eventBus EventBus
	mu       sync.RWMutex
	// links: for each deployment ID, the set of deployment IDs it is linked to (bidirectional)
	links map[string]map[string]struct{}
	// monitors: for each target deployment ID, the set of monitor refs (monitorer gets exit when target exits)
	monitorsByTarget map[string]map[MonitorRef]struct{}
	// monitorRefs: ref -> target ID (for Demonitor)
	monitorRefs map[MonitorRef]string
	nextRef     uint64
}

// NewLinkManager creates a LinkManager that will use the given EventBus to deliver exit signals.
func NewLinkManager(eventBus EventBus) *LinkManager {
	return &LinkManager{
		eventBus:        eventBus,
		links:           make(map[string]map[string]struct{}),
		monitorsByTarget: make(map[string]map[MonitorRef]struct{}),
		monitorRefs:     make(map[MonitorRef]string),
		nextRef:         1,
	}
}

// exitAddress returns the EventBus address for receiving exit signals for the given deployment ID.
const exitAddressPrefix = "core.exit."

func exitAddress(deploymentID string) string {
	return exitAddressPrefix + deploymentID
}

// Link creates a bidirectional link between the caller (selfID) and targetID.
// When either exits, the other will receive an ExitSignal on its core.exit.<id> address.
func (m *LinkManager) Link(selfID, targetID string) error {
	if selfID == "" || targetID == "" {
		return &EventBusError{Code: "INVALID_ID", Message: "Link: deployment ID cannot be empty"}
	}
	if selfID == targetID {
		return &EventBusError{Code: "INVALID_LINK", Message: "cannot link to self"}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	addLink(m.links, selfID, targetID)
	addLink(m.links, targetID, selfID)
	return nil
}

func addLink(links map[string]map[string]struct{}, a, b string) {
	if links[a] == nil {
		links[a] = make(map[string]struct{})
	}
	links[a][b] = struct{}{}
}

// Unlink removes the bidirectional link between selfID and targetID.
func (m *LinkManager) Unlink(selfID, targetID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	removeLink(m.links, selfID, targetID)
	removeLink(m.links, targetID, selfID)
	return nil
}

func removeLink(links map[string]map[string]struct{}, a, b string) {
	if links[a] != nil {
		delete(links[a], b)
		if len(links[a]) == 0 {
			delete(links, a)
		}
	}
}

// Monitor creates a one-way monitor: when targetID exits, the monitorer (selfID) will receive
// an ExitSignal on its core.exit.<selfID> address. Returns a MonitorRef for Demonitor.
func (m *LinkManager) Monitor(selfID, targetID string) (MonitorRef, error) {
	if selfID == "" || targetID == "" {
		return MonitorRef{}, &EventBusError{Code: "INVALID_ID", Message: "Monitor: deployment ID cannot be empty"}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	ref := MonitorRef{monitorerID: selfID, targetID: targetID, ref: m.nextRef}
	m.nextRef++
	if m.monitorsByTarget[targetID] == nil {
		m.monitorsByTarget[targetID] = make(map[MonitorRef]struct{})
	}
	m.monitorsByTarget[targetID][ref] = struct{}{}
	m.monitorRefs[ref] = targetID
	return ref, nil
}

// Demonitor removes the monitor identified by ref.
func (m *LinkManager) Demonitor(ref MonitorRef) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	targetID, ok := m.monitorRefs[ref]
	if !ok {
		return nil // already removed or invalid
	}
	delete(m.monitorRefs, ref)
	if m.monitorsByTarget[targetID] != nil {
		delete(m.monitorsByTarget[targetID], ref)
		if len(m.monitorsByTarget[targetID]) == 0 {
			delete(m.monitorsByTarget, targetID)
		}
	}
	return nil
}

// NotifyExit is called when a deployment exits. It delivers ExitSignal to all linked peers
// and all verticles monitoring this deployment.
func (m *LinkManager) NotifyExit(deploymentID string, state DeploymentState, err error) {
	signal := &ExitSignal{From: deploymentID, State: state, Err: err}
	m.mu.RLock()
	linked := make([]string, 0, len(m.links[deploymentID]))
	for id := range m.links[deploymentID] {
		linked = append(linked, id)
	}
	monitorers := make([]string, 0)
	if refs := m.monitorsByTarget[deploymentID]; refs != nil {
		for ref := range refs {
			monitorers = append(monitorers, ref.monitorerID)
		}
	}
	m.mu.RUnlock()

	// Deliver to linked peers
	for _, id := range linked {
		_ = m.eventBus.Publish(exitAddress(id), signal)
	}
	// Deliver to monitors
	for _, id := range monitorers {
		_ = m.eventBus.Publish(exitAddress(id), signal)
	}
}

// ExitSignalAddress returns the EventBus address on which a deployment receives exit signals
// (when linked or monitored deployments exit). Use with eventBus.Consumer(ExitSignalAddress(ctx.DeploymentID())).
func ExitSignalAddress(deploymentID string) string {
	return exitAddress(deploymentID)
}
