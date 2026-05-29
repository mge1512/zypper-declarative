// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Package txn resolves the transaction binding (external vs zypper-internal) and
// yields a context the convergence domains operate within. The binding is kept
// isolated here so the rest of the code is unaware of which mechanism opened the
// snapshot; the convergence code path is identical regardless.
package txn

import (
	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/system"
)

// Mode is the transaction mode.
type Mode string

const (
	ModeAuto     Mode = "auto"
	ModeExternal Mode = "external"
	ModeInternal Mode = "internal"
)

// Context is the resolved transaction context.
type Context struct {
	Mode       Mode
	Root       string // mount point of the new snapshot's root tree
	OpenedHere bool   // true if this tool opened the transaction
}

// Acquirer resolves a transaction context for a mode.
type Acquirer interface {
	Acquire(mode Mode) (Context, *diag.Diagnostic)
}

// SystemAcquirer is the production Acquirer driving snapper / transactional
// machinery through a CommandRunner.
type SystemAcquirer struct {
	Runner system.CommandRunner
	// InsideTransaction reports whether the process already runs inside a fresh
	// snapshot transaction (used by mode=auto and mode=external). Injected so the
	// detection mechanism stays abstract.
	InsideTransaction func() bool
	// ExternalRoot returns the writable new-generation root mount point when an
	// external mechanism opened the transaction, or "" if none is present.
	ExternalRoot func() string
	// OpenInternal opens a new snapshot transaction and returns its mount point.
	OpenInternal func() (string, error)
}

// Acquire implements the spec acquire-transaction-context behaviour.
func (a *SystemAcquirer) Acquire(mode Mode) (Context, *diag.Diagnostic) {
	resolved := mode
	if mode == ModeAuto {
		if a.InsideTransaction != nil && a.InsideTransaction() {
			resolved = ModeExternal
		} else {
			resolved = ModeInternal
		}
	}

	switch resolved {
	case ModeExternal:
		root := ""
		if a.ExternalRoot != nil {
			root = a.ExternalRoot()
		}
		if root == "" {
			return Context{}, diag.Errorf(diag.DomainTransaction,
				"external mode but not running inside a transaction")
		}
		return Context{Mode: ModeExternal, Root: root, OpenedHere: false}, nil
	case ModeInternal:
		if a.OpenInternal == nil {
			return Context{}, diag.Errorf(diag.DomainTransaction,
				"internal mode but transaction machinery unavailable")
		}
		root, err := a.OpenInternal()
		if err != nil {
			return Context{}, diag.Errorf(diag.DomainTransaction,
				"internal mode but transaction could not be opened: %v", err)
		}
		return Context{Mode: ModeInternal, Root: root, OpenedHere: true}, nil
	default:
		return Context{}, diag.Errorf(diag.DomainTransaction, "unknown transaction mode: %s", resolved)
	}
}
