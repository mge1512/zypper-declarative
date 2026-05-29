// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Entry point for zypper-declarative. Thin CLI dispatch only: signal handling,
// argument hand-off to internal/cli, and exit-code propagation. No behaviour is
// implemented here (SOURCE-PARTITIONING: one-entry-one-implementation).
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mge1512/zypper-declarative/internal/cli"
	"github.com/mge1512/zypper-declarative/internal/sysiface"
)

func main() {
	// Clean exit on SIGTERM and SIGINT with no partial output: an interrupted
	// apply discards its transaction (the snapshot is never sealed/activated),
	// so exiting non-zero leaves the running system unchanged.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		os.Exit(cli.ExitInvocation)
	}()

	app := &cli.App{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Runner: &sysiface.OSCommandRunner{},
	}
	os.Exit(app.Run(os.Args[1:]))
}
