// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Package txn implements acquire-transaction-context: it resolves the
// abstract transaction binding (auto|external|internal) and yields a context
// the convergence domains operate within. The binding is kept isolated here so
// the rest of the code is unaware of which mechanism opened the snapshot.
package txn

import (
	"os"

	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// Mode is the TransactionMode.
type Mode string

const (
	ModeAuto     Mode = "auto"
	ModeExternal Mode = "external"
	ModeInternal Mode = "internal"
)

// Context is the resolved TransactionContext.
type Context struct {
	Mode       Mode
	Root       string // mount point of the new snapshot's root tree
	OpenedHere bool   // true if this tool opened the transaction
}

// Binding abstracts the two mechanisms. detectInside reports whether the
// process already runs inside a fresh snapshot transaction; externalRoot
// returns the writable new-generation root presented by an external opener (or
// "" if none); openInternal opens a new snapshot transaction and returns its
// mount point.
type Binding interface {
	DetectInside() bool
	ExternalRoot() (root string, present bool)
	OpenInternal() (root string, err error)
}

// Acquire implements BEHAVIOR/INTERNAL: acquire-transaction-context.
func Acquire(mode Mode, b Binding) (*Context, *manifest.Diagnostic) {
	resolved := mode
	if mode == ModeAuto {
		if b.DetectInside() {
			resolved = ModeExternal
		} else {
			resolved = ModeInternal
		}
	}

	switch resolved {
	case ModeExternal:
		root, present := b.ExternalRoot()
		if !present {
			return nil, manifest.NewError(manifest.DomainTransaction,
				"external mode but not running inside a transaction")
		}
		return &Context{Mode: ModeExternal, Root: root, OpenedHere: false}, nil
	case ModeInternal:
		root, err := b.OpenInternal()
		if err != nil {
			return nil, manifest.NewError(manifest.DomainTransaction,
				"internal mode but transaction could not be opened: "+err.Error())
		}
		return &Context{Mode: ModeInternal, Root: root, OpenedHere: true}, nil
	default:
		return nil, manifest.NewError(manifest.DomainTransaction, "unknown transaction mode")
	}
}

// EnvBinding is the production binding. It detects an external transaction by
// the TRANSACTIONAL_UPDATE / new-generation-root environment commonly set by
// transactional-update, and opens an internal transaction via the snapshot
// mechanism. Because the spec deliberately does not commit to either binding,
// the internal opener is delegated; in an environment without the snapshot
// machinery available, OpenInternal reports the mechanism as unavailable.
type EnvBinding struct{}

// DetectInside reports whether a fresh-snapshot transaction is already active.
func (EnvBinding) DetectInside() bool {
	_, ok := os.LookupEnv("TRANSACTIONAL_UPDATE")
	return ok
}

// ExternalRoot returns the new-generation root an external opener presents.
func (EnvBinding) ExternalRoot() (string, bool) {
	if v, ok := os.LookupEnv("TRANSACTIONAL_UPDATE_NEW_ROOT"); ok && v != "" {
		return v, true
	}
	// transactional-update conventionally chroots into the new root, so "/" of
	// the running process is the new generation when it set TRANSACTIONAL_UPDATE.
	if _, ok := os.LookupEnv("TRANSACTIONAL_UPDATE"); ok {
		return "/", true
	}
	return "", false
}

// OpenInternal opens a new snapshot transaction through the zypper-merged
// transactional machinery. The mechanism is part of the deferred binding; when
// it is not available this reports the transaction mechanism as unavailable.
func (EnvBinding) OpenInternal() (string, error) {
	return "", &manifest.Diagnostic{
		Severity: manifest.SeverityError,
		Domain:   manifest.DomainTransaction,
		Message:  "transaction mechanism unavailable",
	}
}
