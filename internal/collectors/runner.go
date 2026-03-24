package collectors

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 15 * time.Second

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, command string, args ...string) (Result, error)
}

type ExecRunner struct{}

func (ExecRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (ExecRunner) Run(ctx context.Context, command string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	out, err := cmd.CombinedOutput()
	result := Result{
		Stdout: strings.TrimRight(string(out), "\n"),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.Stderr = strings.TrimRight(string(exitErr.Stderr), "\n")
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}
	return result, err
}
