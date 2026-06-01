// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Diagnostic and Severity carry the spec's structured error reporting. They are
// returned by behaviours to their caller; only the CLI layer maps them to exit
// codes and writes them to stderr.
package manifest

// Severity is Error or Warning.
type Severity string

const (
	SeverityError   Severity = "Error"
	SeverityWarning Severity = "Warning"
)

// Domain values per the spec.
const (
	DomainPackages     = "packages"
	DomainRepositories = "repositories"
	DomainFiles        = "files"
	DomainUnits        = "units"
	DomainManifest     = "manifest"
	DomainTransaction  = "transaction"
	DomainInvocation   = "invocation"
)

// Diagnostic is a single structured error or warning.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Domain   string   `json:"domain"`
	Message  string   `json:"message"`
}

// Error implements the error interface so a Diagnostic can be returned as error.
func (d Diagnostic) Error() string { return d.Message }

// NewError builds an Error-severity diagnostic in the given domain.
func NewError(domain, message string) Diagnostic {
	return Diagnostic{Severity: SeverityError, Domain: domain, Message: message}
}

// NewWarning builds a Warning-severity diagnostic in the given domain.
func NewWarning(domain, message string) Diagnostic {
	return Diagnostic{Severity: SeverityWarning, Domain: domain, Message: message}
}
