// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Package txn implements BEHAVIOR/INTERNAL: acquire-transaction-context. The
// binding between this tool and the snapshot transaction is deliberately
// abstract: auto detects, external asserts a writable new-generation root,
// internal opens a snapshot through the zypper-merged machinery. The convergence
// path is identical regardless of which binding resolves.
package txn

import (
	"os"

	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// Mode is auto | external | internal.
type Mode string

const (
	ModeAuto     Mode = "auto"
	ModeExternal Mode = "external"
	ModeInternal Mode = "internal"
)

// Context is the resolved transaction context the convergence domains operate in.
type Context struct {
	Mode       Mode
	Root       string
	OpenedHere bool
}

// Acquirer resolves a transaction context. The default Acquirer detects an
// external transaction via the TRANSACTIONAL_UPDATE marker environment that the
// external opener sets, and otherwise reports the internal mechanism as
// unavailable in this build (the internal binding is host-specific; see
// INTERFACES). It is injectable so callers/tests can supply a binding.
type Acquirer interface {
	Acquire(mode Mode) (*Context, *manifest.Diagnostic)
}

// DefaultAcquirer is the production Acquirer.
type DefaultAcquirer struct {
	// ExternalRoot, when non-empty, is the writable new-generation root an
	// external opener provided. The external opener (transactional-update)
	// communicates the new root out of band; this is the seam to read it.
	ExternalRoot string
}

// Acquire implements acquire-transaction-context.
func (a *DefaultAcquirer) Acquire(mode Mode) (*Context, *manifest.Diagnostic) {
	if mode == "" {
		mode = ModeAuto
	}
	resolved := mode
	if mode == ModeAuto {
		if a.externalRoot() != "" {
			resolved = ModeExternal
		} else {
			resolved = ModeInternal
		}
	}
	switch resolved {
	case ModeExternal:
		root := a.externalRoot()
		if root == "" {
			d := manifest.NewError(manifest.DomainTransaction,
				"external transaction mode requested but not running inside a snapshot transaction")
			return nil, &d
		}
		return &Context{Mode: ModeExternal, Root: root, OpenedHere: false}, nil
	case ModeInternal:
		// Opening a snapshot through the zypper-merged transactional machinery is
		// host-specific (SLES 16.1) and not available in this build environment.
		d := manifest.NewError(manifest.DomainTransaction,
			"internal transaction mechanism unavailable in this environment")
		return nil, &d
	default:
		d := manifest.NewError(manifest.DomainTransaction, "unknown transaction mode: "+string(mode))
		return nil, &d
	}
}

func (a *DefaultAcquirer) externalRoot() string {
	if a.ExternalRoot != "" {
		return a.ExternalRoot
	}
	// transactional-update exposes the new snapshot root via this variable when
	// it has opened a transaction around the invocation. Reading it here is not
	// configuration-via-env-var (forbidden by the template): it is the contract
	// by which the external opener hands the tool the writable root.
	return os.Getenv("TRANSACTIONAL_UPDATE_NEW_ROOT")
}
