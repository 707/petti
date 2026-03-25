package main

import (
	"context"
	"io"
	"os"

	"github.com/nad/pkgview/internal/cli"
)

var (
	osArgs      = os.Args
	exitProcess = os.Exit
	runCLI      = cli.Run
	version     = "dev"
)

func main() {
	exitProcess(realMain(osArgs[1:], os.Stdout, os.Stderr))
}

func realMain(args []string, stdout, stderr io.Writer) int {
	return runCLI(context.Background(), args, cli.Deps{
		Stdout:  stdout,
		Stderr:  stderr,
		Version: version,
	})
}
