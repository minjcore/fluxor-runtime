package entrypoint

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

// AdminRequest is the wire format for admin socket commands.
type AdminRequest struct {
	Cmd  string `json:"cmd"`            // "deploy" | "undeploy" | "list"
	Path string `json:"path,omitempty"` // deploy: plugin .so path
	ID   string `json:"id,omitempty"`   // undeploy: deployment ID
}

// AdminResponse is the wire format for admin socket responses.
type AdminResponse struct {
	OK  bool     `json:"ok"`
	ID  string   `json:"id,omitempty"`
	IDs []string `json:"ids,omitempty"`
	Err string   `json:"error,omitempty"`
}

// startAdminSocket starts a Unix domain socket server on socketPath.
// Returns an io.Closer that shuts the listener down (causing the goroutine to exit).
// The socket file is removed when the listener closes.
func startAdminSocket(m *MainVerticle, socketPath string) io.Closer {
	_ = os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Printf("[admin] failed to listen on %s: %v", socketPath, err)
		return io.NopCloser(nil)
	}

	go func() {
		defer os.Remove(socketPath)
		defer ln.Close()
		log.Printf("[admin] socket ready — %s", socketPath)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // closed on shutdown
			}
			go handleAdminConn(m, conn)
		}
	}()

	return ln
}

func handleAdminConn(m *MainVerticle, conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	var req AdminRequest
	if err := dec.Decode(&req); err != nil {
		_ = enc.Encode(AdminResponse{OK: false, Err: err.Error()})
		return
	}

	switch req.Cmd {
	case "deploy":
		v, err := LoadVerticleFromPlugin(req.Path)
		if err != nil {
			_ = enc.Encode(AdminResponse{OK: false, Err: fmt.Sprintf("load plugin: %v", err)})
			return
		}
		id, err := m.DeployVerticle(v)
		if err != nil {
			_ = enc.Encode(AdminResponse{OK: false, Err: fmt.Sprintf("deploy: %v", err)})
			return
		}
		log.Printf("[admin] deployed %s → %s", req.Path, id)
		_ = enc.Encode(AdminResponse{OK: true, ID: id})

	case "undeploy":
		if req.ID == "" {
			_ = enc.Encode(AdminResponse{OK: false, Err: "id required"})
			return
		}
		if err := m.gocmd.UndeployVerticle(req.ID); err != nil {
			_ = enc.Encode(AdminResponse{OK: false, Err: err.Error()})
			return
		}
		log.Printf("[admin] undeployed %s", req.ID)
		_ = enc.Encode(AdminResponse{OK: true})

	case "list":
		m.mu.Lock()
		ids := make([]string, len(m.deploymentIDs))
		copy(ids, m.deploymentIDs)
		m.mu.Unlock()
		_ = enc.Encode(AdminResponse{OK: true, IDs: ids})

	default:
		_ = enc.Encode(AdminResponse{OK: false, Err: fmt.Sprintf("unknown command %q", req.Cmd)})
	}
}

// AdminDial connects to a running process's admin socket and sends a command.
// Returns the parsed response.
func AdminDial(socketPath string, req AdminRequest) (*AdminResponse, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", socketPath, err)
	}
	defer conn.Close()

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}

	var resp AdminResponse
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, fmt.Errorf("recv: %w", err)
	}
	return &resp, nil
}
