// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Command zypper-declarative is the entry point: it sets up signal handling for
// a clean exit, then dispatches to the CLI verb layer. It contains only CLI
// dispatch wiring; all behaviour lives in the internal packages.
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mge1512/zypper-declarative/internal/cli"
)

func main() {
	// Clean exit on SIGTERM and SIGINT, with no partial output: an interrupted
	// run exits without leaving a partially converged snapshot as the default
	// boot target (the transaction is discarded by not committing it).
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		os.Exit(cli.ExitOK)
	}()

	io := cli.IO{Stdout: os.Stdout, Stderr: os.Stderr}
	os.Exit(cli.Run(os.Args[1:], io))
}
