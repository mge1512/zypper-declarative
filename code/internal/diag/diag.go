// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Package diag defines the Diagnostic type and the error-carrying convention
// used across internal behaviours. Internal behaviours return errors to their
// caller; only the verb layer (internal/cli) maps a Diagnostic's domain to an
// exit code. A Diagnostic carries severity, domain, and message per the spec.
package diag

import "fmt"

// Severity of a diagnostic.
type Severity string

const (
	SeverityError   Severity = "Error"
	SeverityWarning Severity = "Warning"
)

// Domain of a diagnostic, per the spec TYPES section.
type Domain string

const (
	DomainPackages     Domain = "packages"
	DomainRepositories Domain = "repositories"
	DomainFiles        Domain = "files"
	DomainUnits        Domain = "units"
	DomainManifest     Domain = "manifest"
	DomainTransaction  Domain = "transaction"
	DomainInvocation   Domain = "invocation"
)

// Diagnostic carries severity, domain, and a human-readable message. It
// implements the error interface so internal behaviours can return it directly.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Domain   Domain   `json:"domain"`
	Message  string   `json:"message"`
}

// Error renders the diagnostic for stderr: one line carrying its domain so a
// reader (and the test harness) can identify the failing domain.
func (d *Diagnostic) Error() string {
	return fmt.Sprintf("%s: [%s] %s", d.Severity, d.Domain, d.Message)
}

// Errorf constructs an error-severity Diagnostic with a formatted message.
func Errorf(domain Domain, format string, args ...interface{}) *Diagnostic {
	return &Diagnostic{
		Severity: SeverityError,
		Domain:   domain,
		Message:  fmt.Sprintf(format, args...),
	}
}

// Warnf constructs a warning-severity Diagnostic with a formatted message.
func Warnf(domain Domain, format string, args ...interface{}) *Diagnostic {
	return &Diagnostic{
		Severity: SeverityWarning,
		Domain:   domain,
		Message:  fmt.Sprintf(format, args...),
	}
}

// Line renders a single diagnostic line for stderr.
func (d *Diagnostic) Line() string {
	return d.Error()
}
