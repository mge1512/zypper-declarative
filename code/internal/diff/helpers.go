// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Pure helpers for the diff package: scope accessors that treat a nil scope as
// empty, identity-key sets, and lookup maps. None perform I/O.

package diff

import "github.com/mge1512/zypper-declarative/internal/manifest"

func appliedPackages(m *manifest.Manifest) []manifest.PackageRecord {
	if m == nil || m.Packages == nil {
		return nil
	}
	return m.Packages.Elements
}

func appliedFiles(m *manifest.Manifest) []manifest.ManagedFileRecord {
	if m == nil || m.ConfigFiles == nil {
		return nil
	}
	return m.ConfigFiles.Elements
}

func appliedServices(m *manifest.Manifest) []manifest.ServiceRecord {
	if m == nil || m.Services == nil {
		return nil
	}
	return m.Services.Elements
}

func actualFilesOf(m *manifest.Manifest) []manifest.ManagedFileRecord {
	if m == nil || m.ConfigFiles == nil {
		return nil
	}
	return m.ConfigFiles.Elements
}

func actualServicesOf(m *manifest.Manifest) []manifest.ServiceRecord {
	if m == nil || m.Services == nil {
		return nil
	}
	return m.Services.Elements
}

func actualPackagesOf(m *manifest.Manifest) []manifest.PackageRecord {
	if m == nil || m.Packages == nil {
		return nil
	}
	return m.Packages.Elements
}

func nameSet(pkgs []manifest.PackageRecord) map[string]bool {
	s := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		s[p.Name] = true
	}
	return s
}

func fileNameSet(files []manifest.ManagedFileRecord) map[string]bool {
	s := make(map[string]bool, len(files))
	for _, f := range files {
		s[f.Name] = true
	}
	return s
}

func fileMap(files []manifest.ManagedFileRecord) map[string]manifest.ManagedFileRecord {
	m := make(map[string]manifest.ManagedFileRecord, len(files))
	for _, f := range files {
		m[f.Name] = f
	}
	return m
}

func serviceStateMap(svcs []manifest.ServiceRecord) map[string]string {
	m := make(map[string]string, len(svcs))
	for _, s := range svcs {
		m[s.Name] = s.State
	}
	return m
}

func packageIdentity(p manifest.PackageRecord) string {
	return p.Name + "\x00" + p.Version + "\x00" + p.Release + "\x00" + p.Arch
}

func packageIdentitySet(pkgs []manifest.PackageRecord) map[string]bool {
	s := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		s[packageIdentity(p)] = true
	}
	return s
}
