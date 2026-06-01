// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Canonicalisation helpers: deterministic scope ordering and a deep clone used
// before serialisation and hashing.
package manifest

import "sort"

// cloneSorted returns a copy of the manifest with each present scope's
// _elements sorted by its identity key and _attributes normalised to a non-nil
// object. This makes serialisation deterministic and the identity hash stable.
func (m *Manifest) cloneSorted() *Manifest {
	c := &Manifest{Meta: m.Meta}

	if m.Packages != nil {
		els := append([]PackageRecord(nil), m.Packages.Elements...)
		sort.SliceStable(els, func(i, j int) bool {
			if els[i].Name != els[j].Name {
				return els[i].Name < els[j].Name
			}
			return els[i].Arch < els[j].Arch
		})
		c.Packages = &PackagesScope{Attributes: nonNil(m.Packages.Attributes), Elements: els}
	}
	if m.Repositories != nil {
		els := append([]RepositoryRecord(nil), m.Repositories.Elements...)
		sort.SliceStable(els, func(i, j int) bool { return els[i].Alias < els[j].Alias })
		c.Repositories = &RepositoriesScope{Attributes: nonNil(m.Repositories.Attributes), Elements: els}
	}
	if m.Services != nil {
		els := append([]ServiceRecord(nil), m.Services.Elements...)
		sort.SliceStable(els, func(i, j int) bool { return els[i].Name < els[j].Name })
		c.Services = &ServicesScope{Attributes: nonNil(m.Services.Attributes), Elements: els}
	}
	if m.ConfigFiles != nil {
		els := append([]ManagedFileRecord(nil), m.ConfigFiles.Elements...)
		sort.SliceStable(els, func(i, j int) bool { return els[i].Name < els[j].Name })
		c.ConfigFiles = &ConfigFilesScope{Attributes: nonNil(m.ConfigFiles.Attributes), Elements: els}
	}
	if m.ChangedManagedFiles != nil {
		els := append([]ManagedBaselineRecord(nil), m.ChangedManagedFiles.Elements...)
		sort.SliceStable(els, func(i, j int) bool { return els[i].Name < els[j].Name })
		c.ChangedManagedFiles = &ChangedManagedFilesScope{Attributes: nonNil(m.ChangedManagedFiles.Attributes), Elements: els}
	}
	if m.UnmanagedFiles != nil {
		els := append([]UnmanagedFileRecord(nil), m.UnmanagedFiles.Elements...)
		sort.SliceStable(els, func(i, j int) bool { return els[i].Name < els[j].Name })
		c.UnmanagedFiles = &UnmanagedFilesScope{Attributes: nonNil(m.UnmanagedFiles.Attributes), Elements: els}
	}
	return c
}

// nonNil ensures _attributes serialises as an object, never null.
func nonNil(a map[string]interface{}) map[string]interface{} {
	if a == nil {
		return map[string]interface{}{}
	}
	return a
}
