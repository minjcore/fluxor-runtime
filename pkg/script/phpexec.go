package script

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PHPExecutor executes PHP scripts with proper context cancellation and output capture.
type PHPExecutor struct {
	PHPPath     string        // Path to PHP executable (default: "php")
	Timeout     time.Duration // Execution timeout (0 = no timeout)
	WorkingDir  string        // Working directory for script execution
	Environment []string      // Environment variables (key=value format)
}

// ExecutionResult contains the result of a PHP script execution.
type ExecutionResult struct {
	ExitCode   int           `json:"exitCode"`
	Stdout     string        `json:"stdout"`
	Stderr     string        `json:"stderr"`
	Duration   time.Duration `json:"duration"`
	ScriptPath string        `json:"scriptPath"`
	Args       []string      `json:"args"`
	Error      error         `json:"error,omitempty"`
}

// NewPHPExecutor creates a new PHP executor with default settings.
func NewPHPExecutor() *PHPExecutor {
	return &PHPExecutor{
		PHPPath:     "php",
		Timeout:     30 * time.Second,
		WorkingDir:  "",
		Environment: nil,
	}
}

// ExecuteScript executes a PHP script with the given arguments and context.
// The script path is validated to prevent directory traversal attacks.
func (e *PHPExecutor) ExecuteScript(ctx context.Context, scriptPath string, args []string) (*ExecutionResult, error) {
	startTime := time.Now()

	// Validate script path to prevent directory traversal
	if err := e.validateScriptPath(scriptPath); err != nil {
		return &ExecutionResult{
			ExitCode:   -1,
			ScriptPath: scriptPath,
			Args:       args,
			Error:      err,
			Duration:   time.Since(startTime),
		}, err
	}

	// Check if script file exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return &ExecutionResult{
			ExitCode:   -1,
			ScriptPath: scriptPath,
			Args:       args,
			Error:      fmt.Errorf("script file not found: %s", scriptPath),
			Duration:   time.Since(startTime),
		}, fmt.Errorf("script file not found: %s", scriptPath)
	}

	// Create command context with timeout if specified
	cmdCtx := ctx
	var cancel context.CancelFunc
	if e.Timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}

	// Build command: php script.php arg1 arg2 ...
	phpPath := e.PHPPath
	if phpPath == "" {
		phpPath = "php"
	}

	cmd := exec.CommandContext(cmdCtx, phpPath, append([]string{scriptPath}, args...)...)

	// Set working directory if specified
	if e.WorkingDir != "" {
		cmd.Dir = e.WorkingDir
	}

	// Set environment variables if specified
	if len(e.Environment) > 0 {
		cmd.Env = append(os.Environ(), e.Environment...)
	}

	// Capture stdout and stderr
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()
	duration := time.Since(startTime)

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			// Command failed to start or was killed
			exitCode = -1
		}
	}

	result := &ExecutionResult{
		ExitCode:   exitCode,
		Stdout:     strings.TrimSpace(stdout.String()),
		Stderr:     strings.TrimSpace(stderr.String()),
		Duration:   duration,
		ScriptPath: scriptPath,
		Args:       args,
	}

	// Set error if execution failed (non-zero exit code or execution error)
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			result.Error = fmt.Errorf("execution timeout after %v", e.Timeout)
		} else if cmdCtx.Err() == context.Canceled {
			result.Error = fmt.Errorf("execution cancelled")
		} else {
			result.Error = err
		}
	}

	return result, nil
}

// validateScriptPath validates the script path to prevent directory traversal attacks.
func (e *PHPExecutor) validateScriptPath(scriptPath string) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return fmt.Errorf("invalid script path: %w", err)
	}

	// Check for directory traversal attempts
	cleanPath := filepath.Clean(absPath)
	if cleanPath != absPath {
		return fmt.Errorf("invalid script path: contains directory traversal")
	}

	// Additional check: ensure path doesn't contain ".."
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid script path: contains '..'")
	}

	return nil
}

// SetPHPPath sets the path to the PHP executable.
func (e *PHPExecutor) SetPHPPath(path string) {
	e.PHPPath = path
}

// SetTimeout sets the execution timeout.
func (e *PHPExecutor) SetTimeout(timeout time.Duration) {
	e.Timeout = timeout
}

// SetWorkingDir sets the working directory for script execution.
func (e *PHPExecutor) SetWorkingDir(dir string) {
	e.WorkingDir = dir
}

// SetEnvironment sets environment variables for script execution.
// Variables should be in "KEY=VALUE" format.
func (e *PHPExecutor) SetEnvironment(env []string) {
	e.Environment = env
}
