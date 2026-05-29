// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Package zypperdeclarative implements the declarable-subset data model of
// the SUSE Machinery system description and the convergence behaviours of
// the zypper-declarative tool. This file defines the TYPES from the spec.
package zypperdeclarative

// SpecSHA256 is the SHA256 of the specification this code was produced from.
const SpecSHA256 = "714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014"

// Version is the tool version (matches spec META Version).
const Version = "0.4.0"

// Generator is the generator string embedded in produced manifests.
const Generator = "zypper-declarative " + Version

// ExitCode values per the spec.
const (
	ExitOK         = 0 // success
	ExitLogical    = 1 // logical failure
	ExitInvocation = 2 // invocation error
)

// Severity of a Diagnostic.
type Severity string

const (
	SeverityError   Severity = "Error"
	SeverityWarning Severity = "Warning"
)

// Diagnostic is a single error or warning with its domain.
type Diagnostic struct {
	Severity Severity
	Domain   string // packages | repositories | files | units | manifest | transaction | invocation
	Message  string
}

// Diagnostic domains.
const (
	DomainPackages     = "packages"
	DomainRepositories = "repositories"
	DomainFiles        = "files"
	DomainUnits        = "units"
	DomainManifest     = "manifest"
	DomainTransaction  = "transaction"
	DomainInvocation   = "invocation"
)

// Error implements the error interface so a Diagnostic can be returned as an
// error from internal behaviours.
func (d *Diagnostic) Error() string {
	return d.Severity.String() + " [" + d.Domain + "] " + d.Message
}

func (s Severity) String() string { return string(s) }

// newError builds an Error-severity Diagnostic.
func newError(domain, msg string) *Diagnostic {
	return &Diagnostic{Severity: SeverityError, Domain: domain, Message: msg}
}

// ManifestFormat is the serialisation of the manifest data model.
type ManifestFormat string

const (
	FormatJSON ManifestFormat = "json"
	FormatYAML ManifestFormat = "yaml"
)

// TransactionMode selects the transaction binding.
type TransactionMode string

const (
	ModeAuto     TransactionMode = "auto"
	ModeExternal TransactionMode = "external"
	ModeInternal TransactionMode = "internal"
)

// ManifestMeta is the Machinery meta block.
type ManifestMeta struct {
	FormatVersion int    `json:"format_version" yaml:"format_version"`
	Generator     string `json:"generator" yaml:"generator"`
	CreatedAt     string `json:"created_at" yaml:"created_at"`
	DesiredSHA256 string `json:"desired_sha256" yaml:"desired_sha256"`
}

// PackageRecord is the Machinery package identity subset.
type PackageRecord struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
	Release string `json:"release" yaml:"release"`
	Arch    string `json:"arch" yaml:"arch"`
}

// PackagesScope wraps PackageRecords with scope-level attributes.
type PackagesScope struct {
	Attributes map[string]interface{} `json:"_attributes" yaml:"_attributes"`
	Elements   []PackageRecord        `json:"_elements" yaml:"_elements"`
}

// RepositoryRecord is the Machinery zypp repository record.
type RepositoryRecord struct {
	Alias       string `json:"alias" yaml:"alias"`
	Name        string `json:"name" yaml:"name"`
	URL         string `json:"url" yaml:"url"`
	Type        string `json:"type" yaml:"type"`
	Enabled     bool   `json:"enabled" yaml:"enabled"`
	GPGCheck    bool   `json:"gpgcheck" yaml:"gpgcheck"`
	Autorefresh bool   `json:"autorefresh" yaml:"autorefresh"`
	Priority    int    `json:"priority" yaml:"priority"`
}

// RepositoriesScope wraps RepositoryRecords.
type RepositoriesScope struct {
	Attributes map[string]interface{} `json:"_attributes" yaml:"_attributes"`
	Elements   []RepositoryRecord     `json:"_elements" yaml:"_elements"`
}

// ServiceRecord is the Machinery service record (declarable states only).
type ServiceRecord struct {
	Name  string `json:"name" yaml:"name"`
	State string `json:"state" yaml:"state"` // enabled | disabled | masked
}

// ServicesScope wraps ServiceRecords.
type ServicesScope struct {
	Attributes map[string]interface{} `json:"_attributes" yaml:"_attributes"`
	Elements   []ServiceRecord        `json:"_elements" yaml:"_elements"`
}

// ManagedFileRecord is the declarable /etc file record.
type ManagedFileRecord struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Mode        string `json:"mode" yaml:"mode"`
	User        string `json:"user" yaml:"user"`
	Group       string `json:"group" yaml:"group"`
	SHA256      string `json:"sha256" yaml:"sha256"`
	ContentRef  string `json:"content_ref" yaml:"content_ref"`
	PackageName string `json:"package_name" yaml:"package_name"`
}

// ConfigFilesScope wraps ManagedFileRecords.
type ConfigFilesScope struct {
	Attributes map[string]interface{} `json:"_attributes" yaml:"_attributes"`
	Elements   []ManagedFileRecord    `json:"_elements" yaml:"_elements"`
}

// Manifest is the declarable subset of the Machinery system description.
// A nil scope pointer means the scope is absent (unmanaged); a present scope
// with empty Elements asserts the scope should be exactly empty.
type Manifest struct {
	Meta         ManifestMeta       `json:"meta" yaml:"meta"`
	Packages     *PackagesScope     `json:"packages,omitempty" yaml:"packages,omitempty"`
	Repositories *RepositoriesScope `json:"repositories,omitempty" yaml:"repositories,omitempty"`
	Services     *ServicesScope     `json:"services,omitempty" yaml:"services,omitempty"`
	ConfigFiles  *ConfigFilesScope  `json:"config_files,omitempty" yaml:"config_files,omitempty"`
}

// AppliedRecord is a Manifest with the packages scope fully resolved (the
// lock) and meta.desired_sha256 recorded.
type AppliedRecord = Manifest

// Diff is the intent diff: desired_new versus applied_old.
type Diff struct {
	PackagesInstall []PackageRecord
	PackagesRemove  []PackageRecord
	ReposSet        []RepositoryRecord
	FilesWrite      []ManagedFileRecord
	FilesDelete     []string
	UnitsChange     []ServiceRecord
}

// IsEmpty reports whether the intent diff makes no changes.
func (d *Diff) IsEmpty() bool {
	return len(d.PackagesInstall) == 0 &&
		len(d.PackagesRemove) == 0 &&
		len(d.ReposSet) == 0 &&
		len(d.FilesWrite) == 0 &&
		len(d.FilesDelete) == 0 &&
		len(d.UnitsChange) == 0
}

// DriftReport is the drift diff: actual versus declared.
type DriftReport struct {
	FilesModified     []string
	FilesExtra        []string
	UnitsDivergent    []ServiceRecord
	PackagesDivergent []PackageRecord
}

// IsEmpty reports whether actual equals reference (modulo the keep-list).
func (r *DriftReport) IsEmpty() bool {
	return len(r.FilesModified) == 0 &&
		len(r.FilesExtra) == 0 &&
		len(r.UnitsDivergent) == 0 &&
		len(r.PackagesDivergent) == 0
}

// TransactionContext is the binding the convergence domains operate within.
type TransactionContext struct {
	Mode       TransactionMode
	Root       string
	OpenedHere bool
}

// Syncpoint is the /etc reference that is never written or deleted.
const Syncpoint = "/etc/etc.syncpoint"

// AppliedRecordRelPath is the path of the applied record within a generation
// root.
const AppliedRecordRelPath = "usr/lib/zypper-declarative/applied.json"
