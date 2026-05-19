package devops

import "fmt"

// ServiceLogsOptions holds options for fetching service logs from VPS.
type ServiceLogsOptions struct {
	Lines int // number of log lines (default 80)
}

// ServiceLogs runs journalctl -u <service> on the target VPS and returns the output.
func ServiceLogs(target DeployTarget, serviceName string, opts ServiceLogsOptions) (string, error) {
	if serviceName == "" {
		return "", fmt.Errorf("service name is required")
	}
	lines := opts.Lines
	if lines <= 0 {
		lines = 80
	}

	client, err := NewSSHClient(target)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	cmd := fmt.Sprintf("journalctl -u %s -n %d --no-pager", serviceName, lines)
	out, err := client.ExecuteCommandWithOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("journalctl failed: %w", err)
	}
	return out, nil
}
