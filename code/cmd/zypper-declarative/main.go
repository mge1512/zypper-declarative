// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Entry point for the zypper-declarative CLI. Contains only CLI dispatch:
// signal handling, argument forwarding, and calling into the implementation
// package (per SOURCE-PARTITIONING: one-entry-one-implementation).
package main

import (
	"os"
	"os/signal"
	"syscall"

	zd "github.com/mge1512/zypper-declarative/internal/zypperdeclarative"
)

func main() {
	// Clean exit on SIGTERM/SIGINT. apply must not leave a partially converged
	// snapshot as the default boot target; an interrupted converge discards the
	// transaction (the snapshot is only sealed at the final step, so an
	// interrupt before that leaves the running system unchanged).
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		// No partial output; exit cleanly. The transaction (if any) is not
		// sealed and is therefore not the default boot target.
		os.Exit(ExitInterrupted)
	}()

	code := zd.Run(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}

// ExitInterrupted is the exit code used when the process is interrupted by a
// signal before completion. It is non-zero so callers see the run did not
// complete; per the spec this leaves the system unchanged.
const ExitInterrupted = 130
