package agent

import (
	"bytes"
	"context"
	"log/slog"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Result holds command execution result
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Error    string
}

// Executor executes commands
type Executor struct{}

// NewExecutor creates a new Executor
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute runs a command and returns the result
func (e *Executor) Execute(command string, timeout int) (*Result, error) {
	result := &Result{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Determine shell based on OS
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result.Stdout = strings.TrimSpace(stdout.String())
	result.Stderr = strings.TrimSpace(stderr.String())

	if ctx.Err() == context.DeadlineExceeded {
		result.Error = "command timed out"
		slog.Warn("command timeout", "command", command, "timeout", timeout)
		return result, ctx.Err()
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = err.Error()
			return result, err
		}
	}

	return result, nil
}
