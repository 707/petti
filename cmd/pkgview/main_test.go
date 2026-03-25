package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/nad/pkgview/internal/cli"
)

func TestRealMain(t *testing.T) {
	original := runCLI
	originalVersion := version
	t.Cleanup(func() {
		runCLI = original
		version = originalVersion
	})
	version = "0.6.3"

	var gotArgs []string
	runCLI = func(_ context.Context, args []string, deps cli.Deps) int {
		gotArgs = args
		if deps.Stdout == nil || deps.Stderr == nil {
			t.Fatal("stdout/stderr should be wired")
		}
		if deps.Version != "0.6.3" {
			t.Fatalf("Version = %q, want %q", deps.Version, "0.6.3")
		}
		return 7
	}

	code := realMain([]string{"--version"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 7 {
		t.Fatalf("realMain() = %d, want 7", code)
	}
	if len(gotArgs) != 1 || gotArgs[0] != "--version" {
		t.Fatalf("args = %#v", gotArgs)
	}
}

func TestMain(t *testing.T) {
	originalArgs := osArgs
	originalExit := exitProcess
	originalRun := runCLI
	t.Cleanup(func() {
		osArgs = originalArgs
		exitProcess = originalExit
		runCLI = originalRun
	})

	osArgs = []string{"pkgview", "--version"}
	runCLI = func(context.Context, []string, cli.Deps) int { return 3 }

	var gotExit int
	exitProcess = func(code int) {
		gotExit = code
	}

	main()
	if gotExit != 3 {
		t.Fatalf("exit code = %d, want 3", gotExit)
	}
}
