// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Package manifest is the single owner of the manifest data model and its
// serialisations. The manifest is a typed data model (the declarable Machinery
// subset: packages, repositories, services, config_files, using the
// ScopeWrapper {_attributes,_elements} idiom and underscore_style field names).
// JSON is the canonical serialisation (format_version 1); YAML is an opt-in
// serialisation of the identical model. All serialisation choices are routed
// through ResolveFormat so input and output behave symmetrically.
package manifest

import (
	"encoding/json"
)

// Format is the serialisation of the manifest data model.
type Format string

const (
	// FormatJSON is the canonical, Machinery-compatible serialisation.
	FormatJSON Format = "json"
	// FormatYAML is the opt-in YAML serialisation of the same model.
	FormatYAML Format = "yaml"
)

// ScopeAttributes is scope-level metadata (_attributes); it may be nil.
type ScopeAttributes map[string]interface{}

// Meta is the manifest meta block.
type Meta struct {
	FormatVersion int    `json:"format_version"`
	Generator     string `json:"generator"`
	CreatedAt     string `json:"created_at"`
	DesiredSHA256 string `json:"desired_sha256"`
}

// PackageRecord is the Machinery PackageRecord identity subset.
type PackageRecord struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Release string `json:"release"`
	Arch    string `json:"arch"`
}

// PackagesScope is ScopeWrapper<PackageRecord>.
type PackagesScope struct {
	Attributes ScopeAttributes `json:"_attributes"`
	Elements   []PackageRecord `json:"_elements"`
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

// RepositoriesScope is ScopeWrapper<RepositoryRecord>.
type RepositoriesScope struct {
	Attributes ScopeAttributes    `json:"_attributes"`
	Elements   []RepositoryRecord `json:"_elements"`
}

// ServiceRecord is the Machinery service record, declarable states only.
type ServiceRecord struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

// ServicesScope is ScopeWrapper<ServiceRecord>.
type ServicesScope struct {
	Attributes ScopeAttributes `json:"_attributes"`
	Elements   []ServiceRecord `json:"_elements"`
}

// ManagedFileRecord is aligned with the Machinery changed_config_files record
// and extended with sha256 and a content reference.
type ManagedFileRecord struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Mode        string `json:"mode"`
	User        string `json:"user"`
	Group       string `json:"group"`
	SHA256      string `json:"sha256"`
	ContentRef  string `json:"content_ref"`
	PackageName string `json:"package_name"`
}

// ConfigFilesScope is ScopeWrapper<ManagedFileRecord>.
type ConfigFilesScope struct {
	Attributes ScopeAttributes     `json:"_attributes"`
	Elements   []ManagedFileRecord `json:"_elements"`
}

// Manifest is the shared data model. Scopes are pointers: a nil pointer means
// the scope is ABSENT (unmanaged); a non-nil scope with empty Elements means the
// scope is PRESENT and asserted empty.
type Manifest struct {
	Meta         Meta               `json:"meta"`
	Packages     *PackagesScope     `json:"packages,omitempty"`
	Repositories *RepositoriesScope `json:"repositories,omitempty"`
	Services     *ServicesScope     `json:"services,omitempty"`
	ConfigFiles  *ConfigFilesScope  `json:"config_files,omitempty"`
}

// AppliedRecord is a Manifest with the packages scope fully resolved (the lock)
// and meta.desired_sha256 set. It is a structural alias of Manifest.
type AppliedRecord = Manifest

// Empty returns a Manifest with all scopes absent (the first-ever-apply state).
func Empty() Manifest {
	return Manifest{Meta: Meta{FormatVersion: 1}}
}

// MarshalCanonicalJSON serialises the manifest as pretty-printed canonical JSON
// (Machinery format_version 1). The byte output is suitable for on-disk storage
// and stdout. The identity hash is computed separately by CanonicalHash, which
// uses a compact, key-sorted form.
func (m *Manifest) MarshalCanonicalJSON() ([]byte, error) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
