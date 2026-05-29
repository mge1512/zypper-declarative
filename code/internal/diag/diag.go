// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Package diag is the shared diagnostic type. Internal behaviours return
// Diagnostics to their caller rather than exiting; only the verb layer
// (internal/cli) maps a Diagnostic's domain to an exit code. A Diagnostic
// carries a severity, a domain, and a message, and is written to stderr one per
// line.
package diag

import "fmt"

// Severity is the level of a diagnostic.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Domain identifies the subsystem a diagnostic relates to. The exit-code mapping
// in the verb layer keys off the domain.
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

// Diagnostic is a single severity/domain/message triple.
type Diagnostic struct {
	Severity Severity
	Domain   Domain
	Message  string
}

func (d *Diagnostic) Error() string {
	return string(d.Severity) + " [" + string(d.Domain) + "] " + d.Message
}

// Line renders the diagnostic as a single stderr line.
func (d *Diagnostic) Line() string {
	return string(d.Severity) + ": " + string(d.Domain) + ": " + d.Message
}

// Errorf builds an error-severity Diagnostic in the given domain.
func Errorf(domain Domain, format string, args ...interface{}) *Diagnostic {
	return &Diagnostic{
		Severity: SeverityError,
		Domain:   domain,
		Message:  fmt.Sprintf(format, args...),
	}
}

// Warnf builds a warning-severity Diagnostic in the given domain.
func Warnf(domain Domain, format string, args ...interface{}) *Diagnostic {
	return &Diagnostic{
		Severity: SeverityWarning,
		Domain:   domain,
		Message:  fmt.Sprintf(format, args...),
	}
}
