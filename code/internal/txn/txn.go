// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Package txn resolves the transaction binding (acquire-transaction-context).
// The binding is deliberately abstract: auto, external, or internal. The
// convergence code path is identical regardless of which binding is resolved.
package txn

import (
	"os"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/sysiface"
)

// Mode is the transaction mode.
type Mode string

const (
	ModeAuto     Mode = "auto"
	ModeExternal Mode = "external"
	ModeInternal Mode = "internal"
)

// Context is a resolved transaction context.
type Context struct {
	Mode       Mode
	Root       string // writable new-generation root
	OpenedHere bool   // true iff this tool opened the transaction
}

// Acquirer resolves transaction contexts.
type Acquirer struct {
	Runner sysiface.CommandRunner
	// transactionRootEnv names the environment indicator a transactional-update
	// run sets; presence means we are inside an externally-opened transaction.
	// Read only as a detection signal, never to control configurable behaviour.
	NewRootDetect func() (string, bool)
}

// NewAcquirer builds an Acquirer with the OS runner and the default detector.
func NewAcquirer(runner sysiface.CommandRunner) *Acquirer {
	return &Acquirer{Runner: runner, NewRootDetect: defaultDetect}
}

// defaultDetect reports whether the process runs inside a snapshot transaction
// opened by transactional-update, and the writable new-generation root.
func defaultDetect() (string, bool) {
	// transactional-update exposes the new root via TRANSACTIONAL_UPDATE and the
	// new snapshot path. We detect a writable new-generation root marker file.
	if root := os.Getenv("TRANSACTIONAL_UPDATE_NEW_ROOT"); root != "" {
		if fi, err := os.Stat(root); err == nil && fi.IsDir() {
			return root, true
		}
	}
	return "", false
}

// Acquire resolves mode and returns a Context (acquire-transaction-context
// STEPS 1–4). Failure returns a transaction Diagnostic.
func (a *Acquirer) Acquire(mode Mode) (Context, *diag.Diagnostic) {
	resolved := mode
	if mode == ModeAuto {
		if _, inside := a.NewRootDetect(); inside {
			resolved = ModeExternal
		} else {
			resolved = ModeInternal
		}
	}

	switch resolved {
	case ModeExternal:
		root, inside := a.NewRootDetect()
		if !inside {
			return Context{}, diag.New(diag.DomainTransaction,
				"external mode requires running inside a transaction (no new-generation root present)")
		}
		return Context{Mode: ModeExternal, Root: root, OpenedHere: false}, nil
	case ModeInternal:
		root, d := a.openInternal()
		if d != nil {
			return Context{}, d
		}
		return Context{Mode: ModeInternal, Root: root, OpenedHere: true}, nil
	default:
		return Context{}, diag.New(diag.DomainInvocation, "unknown transaction mode %q", mode)
	}
}

// openInternal opens a new snapshot transaction through the zypper-merged
// transactional machinery. On any failure it returns a transaction error.
func (a *Acquirer) openInternal() (string, *diag.Diagnostic) {
	// transactional-update --no-selfupdate ... or zypper's merged machinery.
	// We probe via transactional-update; absence/failure is a transaction error.
	stdout, stderr, err := a.Runner.Run("transactional-update", "--quiet", "shell", "--", "true")
	if err != nil {
		return "", diag.New(diag.DomainTransaction,
			"could not open snapshot transaction: %v (%s)", err, firstLine(stderr))
	}
	root := firstLine(stdout)
	if root == "" {
		return "", diag.New(diag.DomainTransaction, "transaction opened but no new-generation root reported")
	}
	return root, nil
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
