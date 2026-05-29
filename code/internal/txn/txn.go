// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Package txn implements BEHAVIOR/INTERNAL: acquire-transaction-context. The
// binding between this tool and the snapshot transaction is abstract: under
// external a separate mechanism opened it; under internal the zypper-merged
// machinery opens it; auto detects which applies. The convergence code path is
// identical regardless of the resolved binding.
package txn

import (
	"os"

	"github.com/mge1512/zypper-declarative/internal/diag"
)

// Mode is the transaction binding mode.
type Mode string

const (
	ModeAuto     Mode = "auto"
	ModeExternal Mode = "external"
	ModeInternal Mode = "internal"
)

// Context is the resolved transaction context the convergence domains operate
// within.
type Context struct {
	Mode       Mode
	Root       string
	OpenedHere bool
}

// envTransactionRoot is the conventional environment marker an external opener
// (e.g. transactional-update run) sets to advertise the new-generation root.
// Reading it here is detection of an externally-opened transaction, not
// behaviour control: the tool's own knobs are key=value options and presets.
const envTransactionRoot = "TRANSACTIONAL_UPDATE"

// Acquire resolves the binding for mode and yields a Context, or a transaction
// error to the caller.
func Acquire(mode Mode) (*Context, *diag.Diagnostic) {
	resolved := mode
	if mode == ModeAuto {
		if insideTransaction() {
			resolved = ModeExternal
		} else {
			resolved = ModeInternal
		}
	}

	switch resolved {
	case ModeExternal:
		root := externalRoot()
		if root == "" {
			return nil, diag.Errorf(diag.DomainTransaction,
				"external mode requires running inside a transaction (no new-generation root present)")
		}
		return &Context{Mode: ModeExternal, Root: root, OpenedHere: false}, nil
	case ModeInternal:
		root, err := openInternal()
		if err != nil {
			return nil, diag.Errorf(diag.DomainTransaction,
				"internal mode could not open a snapshot transaction: %v", err)
		}
		return &Context{Mode: ModeInternal, Root: root, OpenedHere: true}, nil
	default:
		return nil, diag.Errorf(diag.DomainInvocation, "unknown transaction mode %q", string(mode))
	}
}

// insideTransaction reports whether the process already runs inside a fresh
// snapshot transaction opened by an external mechanism.
func insideTransaction() bool {
	return externalRoot() != ""
}

// externalRoot returns the new-generation root advertised by an external opener,
// or "" if none.
func externalRoot() string {
	if v := os.Getenv(envTransactionRoot); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	return ""
}

// openInternal opens a new snapshot transaction through the zypper-merged
// transactional machinery and returns the new mount point. The concrete
// transactional-machinery binding (SLES 16.1) is environment-provided and out of
// scope for the language-neutral spec; without it the open fails, which the
// caller surfaces as a transaction error.
func openInternal() (string, error) {
	return "", errTransactionMachineryUnavailable
}

type transactionErr string

func (e transactionErr) Error() string { return string(e) }

const errTransactionMachineryUnavailable transactionErr = "zypper-merged transactional machinery is not available in this environment"
