// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Canonical-model identity hashing. desired_sha256 is the SHA256 of a canonical
// serialisation of the parsed data model, format-independent: object keys are in
// a fixed order, scope _elements are sorted by their identity key, the
// serialisation is compact, and the volatile meta fields (generator, created_at,
// desired_sha256) are excluded so the same intent in JSON or YAML hashes
// identically and idempotence holds across a format switch.

package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// CanonicalHash returns the canonical-model SHA256 of the manifest. It does not
// mutate the receiver.
func (m *Manifest) CanonicalHash() (string, error) {
	c := m.canonicalCopy()
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// canonicalModel is the identity-only projection used for hashing. The meta
// block is reduced to its format_version; volatile fields are dropped. Scopes
// retain only their declarable identity, with elements sorted deterministically.
type canonicalModel struct {
	FormatVersion int                `json:"format_version"`
	Packages      *PackagesScope     `json:"packages,omitempty"`
	Repositories  *RepositoriesScope `json:"repositories,omitempty"`
	Services      *ServicesScope     `json:"services,omitempty"`
	ConfigFiles   *ConfigFilesScope  `json:"config_files,omitempty"`
}

func (m *Manifest) canonicalCopy() canonicalModel {
	c := canonicalModel{FormatVersion: m.Meta.FormatVersion}
	if m.Packages != nil {
		s := *m.Packages
		s.Elements = sortedPackages(s.Elements)
		c.Packages = &s
	}
	if m.Repositories != nil {
		s := *m.Repositories
		s.Elements = sortedRepositories(s.Elements)
		c.Repositories = &s
	}
	if m.Services != nil {
		s := *m.Services
		s.Elements = sortedServices(s.Elements)
		c.Services = &s
	}
	if m.ConfigFiles != nil {
		s := *m.ConfigFiles
		s.Elements = sortedFiles(s.Elements)
		c.ConfigFiles = &s
	}
	return c
}

func sortedPackages(in []PackageRecord) []PackageRecord {
	out := append([]PackageRecord(nil), in...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Arch < out[j].Arch
	})
	return out
}

func sortedRepositories(in []RepositoryRecord) []RepositoryRecord {
	out := append([]RepositoryRecord(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Alias < out[j].Alias })
	return out
}

func sortedServices(in []ServiceRecord) []ServiceRecord {
	out := append([]ServiceRecord(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func sortedFiles(in []ManagedFileRecord) []ManagedFileRecord {
	out := append([]ManagedFileRecord(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
