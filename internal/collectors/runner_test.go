package collectors

import (
	"context"
	"runtime"
	"testing"
)

func TestExecRunnerLookPath(t *testing.T) {
	runner := ExecRunner{}
	path, err := runner.LookPath("sh")
	if runtime.GOOS == "windows" {
		t.Skip("petti v1 does not target windows")
	}
	if err != nil {
		t.Fatalf("LookPath() error = %v", err)
	}
	if path == "" {
		t.Fatal("LookPath() returned empty path")
	}
}

func TestExecRunnerRunSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("petti v1 does not target windows")
	}
	result, err := ExecRunner{}.Run(context.Background(), "sh", "-c", "printf 'ok\\n'")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Stdout != "ok" {
		t.Fatalf("result.Stdout = %q, want %q", result.Stdout, "ok")
	}
}

func TestExecRunnerRunExitError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("petti v1 does not target windows")
	}
	result, err := ExecRunner{}.Run(context.Background(), "sh", "-c", "printf 'boom' >&2; exit 7")
	if err != nil {
		t.Fatalf("Run() error = %v, want nil on exit error", err)
	}
	if result.ExitCode != 7 {
		t.Fatalf("result.ExitCode = %d, want 7", result.ExitCode)
	}
}
