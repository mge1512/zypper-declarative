// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Package manifest is the single home of the typed data model — the declarable
// subset of the SUSE Machinery system description — together with its JSON and
// YAML serialisations, the shared format-resolution authority, the
// canonical-model identity hash, and schema validation. JSON and YAML are
// treated strictly as edges around one in-memory model (TYPES section).
package manifest

// ScopeWrapper is the shared Machinery/sitar idiom: scope-level metadata in
// _attributes and the scope records in _elements. Initialise both to non-nil
// zero values so JSON serialises as objects/arrays, never null.
type ScopeWrapper[T any] struct {
	Attributes map[string]interface{} `json:"_attributes" yaml:"_attributes"`
	Elements   []T                    `json:"_elements" yaml:"_elements"`
}

// ManifestMeta is the manifest meta block. format_version is always 1.
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

// ServiceRecord is the Machinery service record, declarable states only.
type ServiceRecord struct {
	Name  string `json:"name" yaml:"name"`
	State string `json:"state" yaml:"state"`
}

// ManagedFileRecord is aligned with the Machinery changed_config_files record,
// extended with sha256 and a content reference for declared content.
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

// Scope aliases for the four declarable scopes.
type (
	PackagesScope     = ScopeWrapper[PackageRecord]
	RepositoriesScope = ScopeWrapper[RepositoryRecord]
	ServicesScope     = ScopeWrapper[ServiceRecord]
	ConfigFilesScope  = ScopeWrapper[ManagedFileRecord]
)

// Manifest is the shared document. A scope absent from the document (nil
// pointer) means unmanaged; a present scope with empty _elements asserts the
// scope should be exactly empty.
type Manifest struct {
	Meta         ManifestMeta       `json:"meta" yaml:"meta"`
	Packages     *PackagesScope     `json:"packages,omitempty" yaml:"packages,omitempty"`
	Repositories *RepositoriesScope `json:"repositories,omitempty" yaml:"repositories,omitempty"`
	Services     *ServicesScope     `json:"services,omitempty" yaml:"services,omitempty"`
	ConfigFiles  *ConfigFilesScope  `json:"config_files,omitempty" yaml:"config_files,omitempty"`
}

// AppliedRecord is a Manifest whose packages scope is fully resolved (the lock)
// and whose meta.desired_sha256 is set. Same Go shape as Manifest.
type AppliedRecord = Manifest

// Format is the serialisation of the manifest data model.
type Format string

const (
	// FormatJSON is the canonical, default, Machinery-compatible serialisation.
	FormatJSON Format = "json"
	// FormatYAML is the opt-in serialisation of the identical data model.
	FormatYAML Format = "yaml"
)

// EmptyPackages returns an initialised empty packages scope (never nil).
func EmptyPackages() *PackagesScope {
	return &PackagesScope{Attributes: map[string]interface{}{"package_system": "rpm"}, Elements: []PackageRecord{}}
}

// EmptyRepositories returns an initialised empty repositories scope.
func EmptyRepositories() *RepositoriesScope {
	return &RepositoriesScope{Attributes: map[string]interface{}{"repository_system": "zypp"}, Elements: []RepositoryRecord{}}
}

// EmptyServices returns an initialised empty services scope.
func EmptyServices() *ServicesScope {
	return &ServicesScope{Attributes: map[string]interface{}{"init_system": "systemd"}, Elements: []ServiceRecord{}}
}

// EmptyConfigFiles returns an initialised empty config_files scope.
func EmptyConfigFiles() *ConfigFilesScope {
	return &ConfigFilesScope{Attributes: map[string]interface{}{}, Elements: []ManagedFileRecord{}}
}
