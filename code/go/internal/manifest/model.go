// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Package manifest is the single typed data model for the declarable subset of
// the SUSE Machinery system description (packages, repositories, services,
// config_files) plus the observational scopes (changed_managed_files,
// unmanaged_files). JSON is the canonical serialisation (Machinery
// format_version 1); YAML is an opt-in alternative of the identical model.
//
// This package owns the model, its (de)serialisation, resolve-format, and the
// canonical-model identity hash (desired_sha256).
package manifest

// ManifestMeta is the meta block of a Manifest.
type ManifestMeta struct {
	FormatVersion int    `json:"format_version" yaml:"format_version"`
	Generator     string `json:"generator" yaml:"generator"`
	CreatedAt     string `json:"created_at" yaml:"created_at"`
	DesiredSHA256 string `json:"desired_sha256" yaml:"desired_sha256"`
}

// PackageRecord is a Machinery PackageRecord (identity subset).
type PackageRecord struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
	Release string `json:"release" yaml:"release"`
	Arch    string `json:"arch" yaml:"arch"`
}

// RepositoryRecord is a Machinery zypp repository record.
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

// ServiceRecord is a Machinery service record, declarable states only.
type ServiceRecord struct {
	Name  string `json:"name" yaml:"name"`
	State string `json:"state" yaml:"state"` // enabled | disabled | masked
}

// ManagedFileRecord is a declarable config_files (/etc) record: a SUPERSET of
// the Machinery changed-config-files record.
type ManagedFileRecord struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"` // file | link | dir
	Mode        string `json:"mode" yaml:"mode"`
	User        string `json:"user" yaml:"user"`
	Group       string `json:"group" yaml:"group"`
	SHA256      string `json:"sha256" yaml:"sha256"`
	Target      string `json:"target" yaml:"target"`
	ContentRef  string `json:"content_ref" yaml:"content_ref"`
	PackageName string `json:"package_name" yaml:"package_name"`
}

// ManagedBaselineRecord is a Machinery changed_managed_files record (a packaged
// file outside /etc that differs from the package baseline).
type ManagedBaselineRecord struct {
	Name        string   `json:"name" yaml:"name"`
	Type        string   `json:"type" yaml:"type"`
	Mode        string   `json:"mode" yaml:"mode"`
	User        string   `json:"user" yaml:"user"`
	Group       string   `json:"group" yaml:"group"`
	SHA256      string   `json:"sha256" yaml:"sha256"`
	Target      string   `json:"target" yaml:"target"`
	PackageName string   `json:"package_name" yaml:"package_name"`
	Changes     []string `json:"changes" yaml:"changes"`
}

// UnmanagedFileRecord is a Machinery unmanaged_files record (a file no package
// owns).
type UnmanagedFileRecord struct {
	Name   string `json:"name" yaml:"name"`
	Type   string `json:"type" yaml:"type"`
	Mode   string `json:"mode" yaml:"mode"`
	User   string `json:"user" yaml:"user"`
	Group  string `json:"group" yaml:"group"`
	SHA256 string `json:"sha256" yaml:"sha256"`
	Target string `json:"target" yaml:"target"`
}

// PackagesScope is ScopeWrapper<PackageRecord>.
type PackagesScope struct {
	Attributes map[string]interface{} `json:"_attributes" yaml:"_attributes"`
	Elements   []PackageRecord        `json:"_elements" yaml:"_elements"`
}

// RepositoriesScope is ScopeWrapper<RepositoryRecord>.
type RepositoriesScope struct {
	Attributes map[string]interface{} `json:"_attributes" yaml:"_attributes"`
	Elements   []RepositoryRecord     `json:"_elements" yaml:"_elements"`
}

// ServicesScope is ScopeWrapper<ServiceRecord>.
type ServicesScope struct {
	Attributes map[string]interface{} `json:"_attributes" yaml:"_attributes"`
	Elements   []ServiceRecord        `json:"_elements" yaml:"_elements"`
}

// ConfigFilesScope is ScopeWrapper<ManagedFileRecord>.
type ConfigFilesScope struct {
	Attributes map[string]interface{} `json:"_attributes" yaml:"_attributes"`
	Elements   []ManagedFileRecord    `json:"_elements" yaml:"_elements"`
}

// ChangedManagedFilesScope is ScopeWrapper<ManagedBaselineRecord> (observational).
type ChangedManagedFilesScope struct {
	Attributes map[string]interface{}  `json:"_attributes" yaml:"_attributes"`
	Elements   []ManagedBaselineRecord `json:"_elements" yaml:"_elements"`
}

// UnmanagedFilesScope is ScopeWrapper<UnmanagedFileRecord> (observational).
type UnmanagedFilesScope struct {
	Attributes map[string]interface{} `json:"_attributes" yaml:"_attributes"`
	Elements   []UnmanagedFileRecord  `json:"_elements" yaml:"_elements"`
}

// Manifest is the shared schema. Declarable scopes are pointers: a nil pointer
// means the scope is ABSENT (unmanaged); a present scope with empty _elements
// asserts the scope should be exactly empty.
type Manifest struct {
	Meta         ManifestMeta       `json:"meta" yaml:"meta"`
	Packages     *PackagesScope     `json:"packages,omitempty" yaml:"packages,omitempty"`
	Repositories *RepositoriesScope `json:"repositories,omitempty" yaml:"repositories,omitempty"`
	Services     *ServicesScope     `json:"services,omitempty" yaml:"services,omitempty"`
	ConfigFiles  *ConfigFilesScope  `json:"config_files,omitempty" yaml:"config_files,omitempty"`

	// Observational, never declarable; present only in describe/verify actual
	// state read with scope=full.
	ChangedManagedFiles *ChangedManagedFilesScope `json:"changed_managed_files,omitempty" yaml:"changed_managed_files,omitempty"`
	UnmanagedFiles      *UnmanagedFilesScope      `json:"unmanaged_files,omitempty" yaml:"unmanaged_files,omitempty"`
}

// Diff is the intent diff (desired vs applied), computed scope by scope.
type Diff struct {
	PackagesInstall []PackageRecord
	PackagesRemove  []PackageRecord
	ReposSet        []RepositoryRecord
	FilesWrite      []ManagedFileRecord
	FilesDelete     []string
	UnitsChange     []ServiceRecord
}

// Empty reports whether the intent diff implies no change.
func (d Diff) Empty() bool {
	return len(d.PackagesInstall) == 0 && len(d.PackagesRemove) == 0 &&
		len(d.ReposSet) == 0 && len(d.FilesWrite) == 0 &&
		len(d.FilesDelete) == 0 && len(d.UnitsChange) == 0
}

// DriftReport is the drift diff (actual vs declared).
type DriftReport struct {
	FilesModified         []string
	FilesExtra            []string
	UnitsDivergent        []ServiceRecord
	PackagesDivergent     []PackageRecord
	ManagedFilesModified  []string
	UnmanagedFilesPresent []string
}

// Empty reports whether the drift report indicates the actual state equals the
// declaration (modulo the keep-list).
func (r DriftReport) Empty() bool {
	return len(r.FilesModified) == 0 && len(r.FilesExtra) == 0 &&
		len(r.UnitsDivergent) == 0 && len(r.PackagesDivergent) == 0 &&
		len(r.ManagedFilesModified) == 0 && len(r.UnmanagedFilesPresent) == 0
}

// Count returns the total number of drift items.
func (r DriftReport) Count() int {
	return len(r.FilesModified) + len(r.FilesExtra) + len(r.UnitsDivergent) +
		len(r.PackagesDivergent) + len(r.ManagedFilesModified) + len(r.UnmanagedFilesPresent)
}

// Format selects a manifest serialisation.
type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)
