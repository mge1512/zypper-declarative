// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Package diag defines the Diagnostic type and the error-carrying convention.
// Internal behaviours return errors to their caller; only the verb layer maps
// them to exit codes (decisions hints: error/exit-code convention).
package diag

import "fmt"

// Severity of a Diagnostic.
type Severity string

const (
	Error   Severity = "Error"
	Warning Severity = "Warning"
)

// Domain identifies which subsystem a Diagnostic concerns.
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

// Diagnostic carries severity, domain, and message. It implements error so it
// can flow through Go's error chain while preserving its domain.
type Diagnostic struct {
	Severity Severity
	Domain   Domain
	Message  string
}

// Error renders the diagnostic as a single stderr line:
//
//	<severity> [<domain>] <message>
func (d *Diagnostic) Error() string {
	return fmt.Sprintf("%s [%s] %s", d.Severity, d.Domain, d.Message)
}

// New builds an error-severity Diagnostic.
func New(domain Domain, format string, args ...interface{}) *Diagnostic {
	return &Diagnostic{Severity: Error, Domain: domain, Message: fmt.Sprintf(format, args...)}
}

// Warn builds a warning-severity Diagnostic.
func Warn(domain Domain, format string, args ...interface{}) *Diagnostic {
	return &Diagnostic{Severity: Warning, Domain: domain, Message: fmt.Sprintf(format, args...)}
}
