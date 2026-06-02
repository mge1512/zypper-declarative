// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Package manifest is the single typed data model for the declarable subset of
// the SUSE Machinery system description (packages, repositories, services,
// config_files), plus the observational scopes used only in actual-state output.
// It provides JSON and YAML (de)serialisation, the resolve-format authority, and
// the canonical-model identity hash.
package manifest

// ScopeWrapper is the shared Machinery/sitar scope idiom: a scope-level
// attributes object and the records in the scope. _attributes is ALWAYS a JSON
// object (empty {} when there are no attributes), never null.
type ScopeWrapper[T any] struct {
	Attributes map[string]interface{} `json:"_attributes"`
	Elements   []T                    `json:"_elements"`
}

// NewScope builds an initialised, empty-but-valid scope wrapper.
func NewScope[T any](attrs map[string]interface{}) ScopeWrapper[T] {
	if attrs == nil {
		attrs = map[string]interface{}{}
	}
	return ScopeWrapper[T]{Attributes: attrs, Elements: []T{}}
}

// ManifestMeta is the manifest meta block (Machinery format_version 1).
type ManifestMeta struct {
	FormatVersion int    `json:"format_version"`
	Generator     string `json:"generator"`
	CreatedAt     string `json:"created_at"`
	DesiredSHA256 string `json:"desired_sha256"`
}

// PackageRecord is the Machinery package identity subset.
type PackageRecord struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Release string `json:"release"`
	Arch    string `json:"arch"`
}

// RepositoryRecord is the Machinery zypp repository record.
type RepositoryRecord struct {
	Alias       string `json:"alias"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Type        string `json:"type"`
	Enabled     bool   `json:"enabled"`
	GPGCheck    bool   `json:"gpgcheck"`
	Autorefresh bool   `json:"autorefresh"`
	Priority    int    `json:"priority"`
}

// ServiceRecord is the Machinery service record, declarable states only.
type ServiceRecord struct {
	Name  string `json:"name"`
	State string `json:"state"` // enabled | disabled | masked
}

// ManagedFileRecord is the declarable config_files record (a superset of the
// Machinery changed-config-files record). status and changes are informational
// fields populated by describe; they are tolerated on input.
type ManagedFileRecord struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // file | link | dir
	Mode        string   `json:"mode"`
	User        string   `json:"user"`
	Group       string   `json:"group"`
	SHA256      string   `json:"sha256"`
	Target      string   `json:"target"`
	ContentRef  string   `json:"content_ref"`
	PackageName string   `json:"package_name"`
	Status      string   `json:"status,omitempty"`
	Changes     []string `json:"changes,omitempty"`
}

// ManagedBaselineRecord is the Machinery changed_managed_files record (out of /etc).
type ManagedBaselineRecord struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Mode        string   `json:"mode"`
	User        string   `json:"user"`
	Group       string   `json:"group"`
	SHA256      string   `json:"sha256"`
	Target      string   `json:"target"`
	PackageName string   `json:"package_name"`
	Changes     []string `json:"changes"`
}

// UnmanagedFileRecord is the Machinery unmanaged_files record (out of /etc).
type UnmanagedFileRecord struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Mode   string `json:"mode"`
	User   string `json:"user"`
	Group  string `json:"group"`
	SHA256 string `json:"sha256"`
	Target string `json:"target"`
}

// Manifest is the shared document. Declarable scopes are optional (a nil pointer
// means the scope is absent/unmanaged). The observational scopes appear only in
// scope=full actual state.
type Manifest struct {
	Meta                ManifestMeta                         `json:"meta"`
	Packages            *ScopeWrapper[PackageRecord]         `json:"packages,omitempty"`
	Repositories        *ScopeWrapper[RepositoryRecord]      `json:"repositories,omitempty"`
	Services            *ScopeWrapper[ServiceRecord]         `json:"services,omitempty"`
	ConfigFiles         *ScopeWrapper[ManagedFileRecord]     `json:"config_files,omitempty"`
	ChangedManagedFiles *ScopeWrapper[ManagedBaselineRecord] `json:"changed_managed_files,omitempty"`
	UnmanagedFiles      *ScopeWrapper[UnmanagedFileRecord]   `json:"unmanaged_files,omitempty"`
}

// Format is the manifest serialisation format.
type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)
