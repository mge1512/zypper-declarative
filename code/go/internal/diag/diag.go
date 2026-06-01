// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Package diag defines the Diagnostic value carried by internal behaviours.
// Internal behaviours return Diagnostics (as errors) to their caller; only the
// CLI verb layer maps a Diagnostic's domain/severity to an exit code.
package diag

import "fmt"

// Severity is Error or Warning.
type Severity string

const (
	SeverityError   Severity = "Error"
	SeverityWarning Severity = "Warning"
)

// Domain identifies the subsystem a diagnostic concerns.
type Domain string

const (
	DomainPackages     Domain = "packages"
	DomainRepositories Domain = "repositories"
	DomainServices     Domain = "units"
	DomainFiles        Domain = "files"
	DomainManifest     Domain = "manifest"
	DomainTransaction  Domain = "transaction"
	DomainInvocation   Domain = "invocation"
)

// Diagnostic is a structured diagnostic. It implements error so it can travel
// through Go's error idiom while preserving domain and severity.
type Diagnostic struct {
	Severity Severity
	Domain   Domain
	Message  string
}

// Error renders the diagnostic as a single line: "<severity> [<domain>] <message>".
func (d *Diagnostic) Error() string {
	return fmt.Sprintf("%s [%s] %s", d.Severity, d.Domain, d.Message)
}

// New constructs an error-severity Diagnostic.
func New(domain Domain, format string, args ...interface{}) *Diagnostic {
	return &Diagnostic{Severity: SeverityError, Domain: domain, Message: fmt.Sprintf(format, args...)}
}

// Warn constructs a warning-severity Diagnostic.
func Warn(domain Domain, format string, args ...interface{}) *Diagnostic {
	return &Diagnostic{Severity: SeverityWarning, Domain: domain, Message: fmt.Sprintf(format, args...)}
}
