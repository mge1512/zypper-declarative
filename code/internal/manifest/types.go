// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Package manifest holds the single shared data model: the declarable subset
// of the SUSE Machinery system description, using the ScopeWrapper idiom and
// underscore_style JSON keys. JSON and YAML are serialisations of this one
// model. The applied record is the same shape with the packages scope fully
// resolved and meta.desired_sha256 set.
package manifest

// Severity of a diagnostic.
type Severity string

const (
	SeverityError   Severity = "Error"
	SeverityWarning Severity = "Warning"
)

// Diagnostic domains (spec TYPES Diagnostic.domain).
const (
	DomainPackages     = "packages"
	DomainRepositories = "repositories"
	DomainFiles        = "files"
	DomainUnits        = "units"
	DomainManifest     = "manifest"
	DomainTransaction  = "transaction"
	DomainInvocation   = "invocation"
)

// Diagnostic is a single advisory or error message, carrying its severity and
// domain. Internal behaviours return these to their caller; the verb layer
// maps them to exit codes and writes them to stderr.
type Diagnostic struct {
	Severity Severity
	Domain   string
	Message  string
}

// Error implements the error interface so a Diagnostic can be returned as an
// error from internal behaviours.
func (d *Diagnostic) Error() string {
	return d.Domain + ": " + d.Message
}

// NewError constructs an Error-severity diagnostic.
func NewError(domain, message string) *Diagnostic {
	return &Diagnostic{Severity: SeverityError, Domain: domain, Message: message}
}

// NewWarning constructs a Warning-severity diagnostic.
func NewWarning(domain, message string) *Diagnostic {
	return &Diagnostic{Severity: SeverityWarning, Domain: domain, Message: message}
}

// ScopeWrapper is the Machinery/sitar {_attributes, _elements} idiom.
type ScopeWrapper[T any] struct {
	Attributes map[string]interface{} `json:"_attributes"`
	Elements   []T                    `json:"_elements"`
}

// PackageRecord is a Machinery PackageRecord (identity subset).
type PackageRecord struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Release string `json:"release"`
	Arch    string `json:"arch"`
}

// RepositoryRecord is a Machinery zypp repository record.
type RepositoryRecord struct {
	Alias       string `json:"alias"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
	GPGCheck    bool   `json:"gpgcheck"`
	AutoRefresh bool   `json:"autorefresh"`
	Priority    int    `json:"priority"`
}

// ServiceRecord is a Machinery service record, declarable states only.
type ServiceRecord struct {
	Name  string `json:"name"`
	State string `json:"state"` // enabled | disabled | masked
}

// ManagedFileRecord is the declarable /etc file record.
type ManagedFileRecord struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // file | link | dir
	Mode        string `json:"mode"`
	User        string `json:"user"`
	Group       string `json:"group"`
	SHA256      string `json:"sha256"`
	ContentRef  string `json:"content_ref"`
	PackageName string `json:"package_name"`
}

// ManagedBaselineRecord is a changed_managed_files element (outside /etc).
type ManagedBaselineRecord struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Mode        string   `json:"mode"`
	User        string   `json:"user"`
	Group       string   `json:"group"`
	SHA256      string   `json:"sha256"`
	PackageName string   `json:"package_name"`
	Changes     []string `json:"changes"`
}

// UnmanagedFileRecord is an unmanaged_files element (outside /etc).
type UnmanagedFileRecord struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Mode   string `json:"mode"`
	User   string `json:"user"`
	Group  string `json:"group"`
	SHA256 string `json:"sha256"`
}

// Scope type aliases.
type PackagesScope = ScopeWrapper[PackageRecord]
type RepositoriesScope = ScopeWrapper[RepositoryRecord]
type ServicesScope = ScopeWrapper[ServiceRecord]
type ConfigFilesScope = ScopeWrapper[ManagedFileRecord]
type ChangedManagedFilesScope = ScopeWrapper[ManagedBaselineRecord]
type UnmanagedFilesScope = ScopeWrapper[UnmanagedFileRecord]

// Meta is the ManifestMeta block.
type Meta struct {
	FormatVersion int    `json:"format_version"`
	Generator     string `json:"generator"`
	CreatedAt     string `json:"created_at"`
	DesiredSHA256 string `json:"desired_sha256"`
}

// Manifest is the shared data model. Declarable scopes are pointers so an
// absent scope (nil) is distinguishable from a present-but-empty scope.
// Observational scopes are present only under scope=full.
type Manifest struct {
	Meta                Meta                      `json:"meta"`
	Packages            *PackagesScope            `json:"packages,omitempty"`
	Repositories        *RepositoriesScope        `json:"repositories,omitempty"`
	Services            *ServicesScope            `json:"services,omitempty"`
	ConfigFiles         *ConfigFilesScope         `json:"config_files,omitempty"`
	ChangedManagedFiles *ChangedManagedFilesScope `json:"changed_managed_files,omitempty"`
	UnmanagedFiles      *UnmanagedFilesScope      `json:"unmanaged_files,omitempty"`
}

// AppliedRecord is a Manifest with the packages scope fully resolved and
// meta.desired_sha256 set. Represented by the same Go type; the validity
// predicate is enforced where it is constructed/loaded.
type AppliedRecord = Manifest

// Diff is the intent diff: desired_new versus applied_old, scope by scope.
type Diff struct {
	PackagesInstall []PackageRecord
	PackagesRemove  []PackageRecord
	ReposSet        []RepositoryRecord
	FilesWrite      []ManagedFileRecord
	FilesDelete     []string
	UnitsChange     []ServiceRecord
}

// Empty reports whether the intent diff carries no change.
func (d Diff) Empty() bool {
	return len(d.PackagesInstall) == 0 && len(d.PackagesRemove) == 0 &&
		len(d.ReposSet) == 0 && len(d.FilesWrite) == 0 &&
		len(d.FilesDelete) == 0 && len(d.UnitsChange) == 0
}

// DriftReport is the drift diff: actual versus declared.
type DriftReport struct {
	FilesModified         []string
	FilesExtra            []string
	UnitsDivergent        []ServiceRecord
	PackagesDivergent     []PackageRecord
	ManagedFilesModified  []string
	UnmanagedFilesPresent []string
}

// Empty reports whether the drift report carries no divergence.
func (r DriftReport) Empty() bool {
	return len(r.FilesModified) == 0 && len(r.FilesExtra) == 0 &&
		len(r.UnitsDivergent) == 0 && len(r.PackagesDivergent) == 0 &&
		len(r.ManagedFilesModified) == 0 && len(r.UnmanagedFilesPresent) == 0
}

// Format is a manifest serialisation.
type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)
