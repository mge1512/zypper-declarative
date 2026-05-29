// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Package manifest defines the shared data model (the declarable subset of the
// SUSE Machinery system description) and its serialisations. The same Manifest
// shape is produced by describe (actual state) and consumed by apply/diff/
// verify/status (desired state and applied record).
package manifest

// ScopeAttributes is the scope-level metadata object (_attributes). It may be
// null in the document; an absent/null value is represented as nil here.
type ScopeAttributes map[string]interface{}

// ManifestMeta is the meta block of a Manifest.
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

// PackagesScope is ScopeWrapper<PackageRecord>.
type PackagesScope struct {
	Attributes ScopeAttributes `json:"_attributes" yaml:"_attributes"`
	Elements   []PackageRecord `json:"_elements" yaml:"_elements"`
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

// RepositoriesScope is ScopeWrapper<RepositoryRecord>.
type RepositoriesScope struct {
	Attributes ScopeAttributes    `json:"_attributes" yaml:"_attributes"`
	Elements   []RepositoryRecord `json:"_elements" yaml:"_elements"`
}

// ServiceRecord is the Machinery service record, declarable states only.
type ServiceRecord struct {
	Name  string `json:"name" yaml:"name"`
	State string `json:"state" yaml:"state"`
}

// ServicesScope is ScopeWrapper<ServiceRecord>.
type ServicesScope struct {
	Attributes ScopeAttributes `json:"_attributes" yaml:"_attributes"`
	Elements   []ServiceRecord `json:"_elements" yaml:"_elements"`
}

// ManagedFileRecord aligns with the Machinery changed_config_files record,
// extended with sha256 and content_ref.
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

// ConfigFilesScope is ScopeWrapper<ManagedFileRecord>.
type ConfigFilesScope struct {
	Attributes ScopeAttributes     `json:"_attributes" yaml:"_attributes"`
	Elements   []ManagedFileRecord `json:"_elements" yaml:"_elements"`
}

// Manifest is the typed data model. A scope absent from the document means the
// converger makes no assertion about it (unmanaged); a present scope with empty
// _elements means the converger asserts the scope should be exactly empty.
// Pointer fields distinguish "absent" (nil) from "present-but-empty" (non-nil
// with empty Elements).
type Manifest struct {
	Meta         ManifestMeta       `json:"meta" yaml:"meta"`
	Packages     *PackagesScope     `json:"packages,omitempty" yaml:"packages,omitempty"`
	Repositories *RepositoriesScope `json:"repositories,omitempty" yaml:"repositories,omitempty"`
	Services     *ServicesScope     `json:"services,omitempty" yaml:"services,omitempty"`
	ConfigFiles  *ConfigFilesScope  `json:"config_files,omitempty" yaml:"config_files,omitempty"`
}

// AppliedRecord is a Manifest with the packages scope fully resolved (the lock)
// and meta.desired_sha256 set. It is the same Go type as Manifest.
type AppliedRecord = Manifest

// Empty returns a Manifest with all scopes absent (unmanaged) and a valid meta.
// This is the "first-ever apply" / "no applied record" representation.
func Empty() Manifest {
	return Manifest{
		Meta: ManifestMeta{FormatVersion: 1},
	}
}
