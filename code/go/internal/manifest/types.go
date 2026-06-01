// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Package manifest defines the shared data model (the declarable subset of the
// SUSE Machinery system description) and is the single authority for manifest
// serialisation: JSON and YAML edges, the resolve-format rule, and the
// canonical-model identity hash. The data model is a single set of Go structs;
// JSON and YAML are strictly serialisations of it.
package manifest

// Scope attribute keys (Machinery / sitar convention).
const (
	PackageSystemRPM     = "rpm"
	RepositorySystemZypp = "zypp"
	InitSystemSystemd    = "systemd"
)

// ScopeWrapper is the {_attributes, _elements} idiom. Each scope is a concrete
// struct (Option B from the hints) so JSON tags are explicit on every field.

// PackageRecord is the Machinery package identity subset.
type PackageRecord struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Release string `json:"release"`
	Arch    string `json:"arch"`
}

// PackagesScope wraps PackageRecord elements.
type PackagesScope struct {
	Attributes map[string]interface{} `json:"_attributes"`
	Elements   []PackageRecord        `json:"_elements"`
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

// RepositoriesScope wraps RepositoryRecord elements.
type RepositoriesScope struct {
	Attributes map[string]interface{} `json:"_attributes"`
	Elements   []RepositoryRecord     `json:"_elements"`
}

// ServiceRecord is the Machinery service record, declarable states only.
type ServiceRecord struct {
	Name  string `json:"name"`
	State string `json:"state"` // enabled | disabled | masked
}

// ServicesScope wraps ServiceRecord elements.
type ServicesScope struct {
	Attributes map[string]interface{} `json:"_attributes"`
	Elements   []ServiceRecord        `json:"_elements"`
}

// ManagedFileRecord is a declared /etc file/link/dir record.
type ManagedFileRecord struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // file | link | dir
	Mode        string `json:"mode"`
	User        string `json:"user"`
	Group       string `json:"group"`
	SHA256      string `json:"sha256"`
	Target      string `json:"target"`
	ContentRef  string `json:"content_ref"`
	PackageName string `json:"package_name"`
}

// ConfigFilesScope wraps ManagedFileRecord elements.
type ConfigFilesScope struct {
	Attributes map[string]interface{} `json:"_attributes"`
	Elements   []ManagedFileRecord    `json:"_elements"`
}

// ManagedBaselineRecord is a changed packaged file outside /etc (full scan).
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

// ChangedManagedFilesScope wraps ManagedBaselineRecord elements.
type ChangedManagedFilesScope struct {
	Attributes map[string]interface{}  `json:"_attributes"`
	Elements   []ManagedBaselineRecord `json:"_elements"`
}

// UnmanagedFileRecord is an out-of-band file no package owns (full scan).
type UnmanagedFileRecord struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Mode   string `json:"mode"`
	User   string `json:"user"`
	Group  string `json:"group"`
	SHA256 string `json:"sha256"`
	Target string `json:"target"`
}

// UnmanagedFilesScope wraps UnmanagedFileRecord elements.
type UnmanagedFilesScope struct {
	Attributes map[string]interface{} `json:"_attributes"`
	Elements   []UnmanagedFileRecord  `json:"_elements"`
}

// ManifestMeta is the manifest header.
type ManifestMeta struct {
	FormatVersion int    `json:"format_version"`
	Generator     string `json:"generator"`
	CreatedAt     string `json:"created_at"`
	DesiredSHA256 string `json:"desired_sha256"`
}

// Manifest is the typed data model. Optional scopes are pointers so an absent
// scope (nil) is distinguished from a present-but-empty scope.
type Manifest struct {
	Meta         ManifestMeta       `json:"meta"`
	Packages     *PackagesScope     `json:"packages,omitempty"`
	Repositories *RepositoriesScope `json:"repositories,omitempty"`
	Services     *ServicesScope     `json:"services,omitempty"`
	ConfigFiles  *ConfigFilesScope  `json:"config_files,omitempty"`

	// Observational scopes, present only in describe/verify actual state read
	// with scope=full; never in a desired manifest or applied record.
	ChangedManagedFiles *ChangedManagedFilesScope `json:"changed_managed_files,omitempty"`
	UnmanagedFiles      *UnmanagedFilesScope      `json:"unmanaged_files,omitempty"`
}
