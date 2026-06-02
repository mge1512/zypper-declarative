// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Package txn implements acquire-transaction-context: resolving the (deliberately
// deferred) transaction binding between an external opener and the zypper-internal
// machinery, yielding a context the convergence domains operate within.
package txn

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
	Root       string
	OpenedHere bool
}

// Diagnostic is a domain-tagged transaction error.
type Diagnostic struct {
	Severity string
	Domain   string
	Message  string
}

func (d *Diagnostic) Error() string { return d.Message }

// EnvProbe abstracts the environment checks acquire-transaction-context makes,
// so the binding can be resolved without committing to a concrete mechanism.
type EnvProbe interface {
	// InsideTransaction reports whether the process already runs inside a fresh
	// snapshot transaction (mode=auto detection).
	InsideTransaction() bool
	// ExternalRoot returns the writable new-generation root provided by an
	// external opener, and whether one is present.
	ExternalRoot() (string, bool)
	// OpenInternal opens a new snapshot transaction and returns its root.
	OpenInternal() (string, error)
}

// Acquire implements BEHAVIOR/INTERNAL: acquire-transaction-context.
func Acquire(mode Mode, probe EnvProbe) (*Context, error) {
	resolved := mode
	if mode == ModeAuto {
		if probe.InsideTransaction() {
			resolved = ModeExternal
		} else {
			resolved = ModeInternal
		}
	}
	switch resolved {
	case ModeExternal:
		root, ok := probe.ExternalRoot()
		if !ok {
			return nil, &Diagnostic{Severity: "Error", Domain: "transaction",
				Message: "transaction mechanism unavailable: not running inside a snapshot transaction (mode=external)"}
		}
		return &Context{Mode: ModeExternal, Root: root, OpenedHere: false}, nil
	case ModeInternal:
		root, err := probe.OpenInternal()
		if err != nil {
			return nil, &Diagnostic{Severity: "Error", Domain: "transaction",
				Message: "transaction mechanism unavailable: could not open a snapshot transaction (mode=internal): " + err.Error()}
		}
		return &Context{Mode: ModeInternal, Root: root, OpenedHere: true}, nil
	default:
		return nil, &Diagnostic{Severity: "Error", Domain: "transaction", Message: "unknown transaction mode"}
	}
}
