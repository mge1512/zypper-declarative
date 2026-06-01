// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Package txn implements acquire-transaction-context: it resolves the abstract
// transaction binding (auto | external | internal) and yields a context the
// convergence domains operate within. The binding is isolated here so the rest
// of the code is unaware of which mechanism opened the snapshot.
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

// ParseMode validates a mode= value.
func ParseMode(s string) (Mode, *diag.Diagnostic) {
	switch Mode(s) {
	case ModeAuto, ModeExternal, ModeInternal:
		return Mode(s), nil
	default:
		return "", diag.New(diag.DomainInvocation, "unknown transaction mode %q", s)
	}
}

// Context is the resolved transaction context.
type Context struct {
	Mode       Mode
	Root       string
	OpenedHere bool
}

// envNewRoot is the environment-independent signal a separate mechanism uses to
// expose the new-generation root. The spec forbids behaviour control via env
// vars; this is detection of the surrounding transaction, not a behaviour knob.
const envNewRoot = "TRANSACTIONAL_UPDATE_NEWROOT"

// Acquire resolves the transaction binding for mode and returns a context.
func Acquire(mode Mode) (*Context, *diag.Diagnostic) {
	switch mode {
	case ModeAuto:
		if insideTransaction() {
			return acquireExternal()
		}
		return acquireInternal()
	case ModeExternal:
		return acquireExternal()
	case ModeInternal:
		return acquireInternal()
	default:
		return nil, diag.New(diag.DomainInvocation, "unknown transaction mode %q", mode)
	}
}

// insideTransaction detects whether the process already runs inside a fresh
// snapshot transaction (a new-generation root is exposed).
func insideTransaction() bool {
	r := os.Getenv(envNewRoot)
	if r == "" {
		return false
	}
	fi, err := os.Stat(r)
	return err == nil && fi.IsDir()
}

func acquireExternal() (*Context, *diag.Diagnostic) {
	r := os.Getenv(envNewRoot)
	if r == "" {
		return nil, diag.New(diag.DomainTransaction,
			"external mode but not running inside a snapshot transaction")
	}
	fi, err := os.Stat(r)
	if err != nil || !fi.IsDir() {
		return nil, diag.New(diag.DomainTransaction,
			"external mode but the new-generation root %q is not present", r)
	}
	return &Context{Mode: ModeExternal, Root: r, OpenedHere: false}, nil
}

// acquireInternal opens a new snapshot transaction through the zypper-merged
// transactional machinery. Opening a real snapshot is reserved for the
// apply-on-live-host milestone; here it reports that the internal mechanism is
// unavailable unless a transaction root is exposed, so callers fail cleanly
// rather than mutating the running system.
func acquireInternal() (*Context, *diag.Diagnostic) {
	if r := os.Getenv(envNewRoot); r != "" {
		if fi, err := os.Stat(r); err == nil && fi.IsDir() {
			return &Context{Mode: ModeInternal, Root: r, OpenedHere: true}, nil
		}
	}
	return nil, diag.New(diag.DomainTransaction,
		"internal transaction mechanism unavailable: no writable new-generation root")
}
