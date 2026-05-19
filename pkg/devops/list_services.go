package devops

import "fmt"

// ListSystemdServices runs systemctl list-units --type=service on the target VPS and returns the output.
func ListSystemdServices(target DeployTarget) (string, error) {
	client, err := NewSSHClient(target)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	// --no-pager so output is not piped to less
	out, err := client.ExecuteCommandWithOutput("systemctl list-units --type=service --all --no-pager")
	if err != nil {
		return "", fmt.Errorf("systemctl failed: %w", err)
	}
	return out, nil
}
