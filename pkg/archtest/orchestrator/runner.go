package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

const (
	checkTimeout = 10 * time.Minute
	trustTimeout = 30 * time.Second
)

// CheckResult represents the result of running arch:check on a service
type CheckResult struct {
	ServiceName string
	ExitCode    int
	Output      string
}

// RunServiceCheck executes mise run arch:check for a service
func RunServiceCheck(servicePath, serviceName string) CheckResult {
	// First, trust the mise.toml file in this directory
	trustCtx, trustCancel := context.WithTimeout(context.Background(), trustTimeout)
	defer trustCancel()

	trustCmd := exec.CommandContext(trustCtx, "mise", "trust")
	trustCmd.Dir = servicePath
	if err := trustCmd.Run(); err != nil {
		return CheckResult{
			ServiceName: serviceName,
			ExitCode:    1,
			Output:      fmt.Sprintf("mise trust failed: %v", err),
		}
	}

	// Now run the arch:check task
	ctx, cancel := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "mise", "run", "arch:check")
	cmd.Dir = servicePath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	// Combine stdout and stderr for output
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	return CheckResult{
		ServiceName: serviceName,
		ExitCode:    exitCode,
		Output:      output,
	}
}
