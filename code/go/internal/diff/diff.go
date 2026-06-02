// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Package diff implements compute-intent-diff and compute-drift. Both are pure:
// they perform no filesystem, rpmdb, or process I/O.
package diff

import (
	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// Diff is the intent diff (desired_new versus applied_old).
type Diff struct {
	PackagesInstall []manifest.PackageRecord
	PackagesRemove  []manifest.PackageRecord
	ReposSet        []manifest.RepositoryRecord
	FilesWrite      []manifest.ManagedFileRecord
	FilesDelete     []string
	UnitsChange     []manifest.ServiceRecord
}

// Empty reports whether the intent diff requires no change.
func (d *Diff) Empty() bool {
	return len(d.PackagesInstall) == 0 && len(d.PackagesRemove) == 0 &&
		len(d.ReposSet) == 0 && len(d.FilesWrite) == 0 &&
		len(d.FilesDelete) == 0 && len(d.UnitsChange) == 0
}

// DriftReport is the drift diff (actual versus declared).
type DriftReport struct {
	FilesModified         []string
	FilesExtra            []string
	UnitsDivergent        []manifest.ServiceRecord
	PackagesDivergent     []manifest.PackageRecord
	ManagedFilesModified  []string
	UnmanagedFilesPresent []string
}

// Empty reports whether the drift report is empty (actual equals declaration).
func (r *DriftReport) Empty() bool {
	return len(r.FilesModified) == 0 && len(r.FilesExtra) == 0 &&
		len(r.UnitsDivergent) == 0 && len(r.PackagesDivergent) == 0 &&
		len(r.ManagedFilesModified) == 0 && len(r.UnmanagedFilesPresent) == 0
}

// ComputeIntentDiff implements BEHAVIOR/INTERNAL: compute-intent-diff.
func ComputeIntentDiff(desired, applied *manifest.Manifest) *Diff {
	d := &Diff{
		PackagesInstall: []manifest.PackageRecord{},
		PackagesRemove:  []manifest.PackageRecord{},
		ReposSet:        []manifest.RepositoryRecord{},
		FilesWrite:      []manifest.ManagedFileRecord{},
		FilesDelete:     []string{},
		UnitsChange:     []manifest.ServiceRecord{},
	}

	// 1. packages
	if desired.Packages != nil {
		d.PackagesInstall = append(d.PackagesInstall, desired.Packages.Elements...)
		desiredNames := map[string]bool{}
		for _, p := range desired.Packages.Elements {
			desiredNames[p.Name] = true
		}
		if applied.Packages != nil {
			for _, p := range applied.Packages.Elements {
				if !desiredNames[p.Name] {
					d.PackagesRemove = append(d.PackagesRemove, p)
				}
			}
		}
	}

	// 2. repositories
	if desired.Repositories != nil {
		d.ReposSet = append(d.ReposSet, desired.Repositories.Elements...)
	}

	// 3. config_files
	if desired.ConfigFiles != nil {
		d.FilesWrite = append(d.FilesWrite, desired.ConfigFiles.Elements...)
		desiredPaths := map[string]bool{}
		for _, f := range desired.ConfigFiles.Elements {
			desiredPaths[f.Name] = true
		}
		if applied.ConfigFiles != nil {
			for _, f := range applied.ConfigFiles.Elements {
				if !desiredPaths[f.Name] {
					d.FilesDelete = append(d.FilesDelete, f.Name)
				}
			}
		}
	}

	// 4. services
	if desired.Services != nil {
		appliedState := map[string]string{}
		if applied.Services != nil {
			for _, s := range applied.Services.Elements {
				appliedState[s.Name] = s.State
			}
		}
		for _, s := range desired.Services.Elements {
			if prev, ok := appliedState[s.Name]; !ok || prev != s.State {
				d.UnitsChange = append(d.UnitsChange, s)
			}
		}
	}

	return d
}

// ComputeDrift implements BEHAVIOR/INTERNAL: compute-drift.
//
//	keepList is the set of paths that must never appear in files_extra (or the
//	integrity categories). It is matched by exact path.
func ComputeDrift(actual, reference *manifest.Manifest, keepList map[string]bool) *DriftReport {
	r := &DriftReport{
		FilesModified:         []string{},
		FilesExtra:            []string{},
		UnitsDivergent:        []manifest.ServiceRecord{},
		PackagesDivergent:     []manifest.PackageRecord{},
		ManagedFilesModified:  []string{},
		UnmanagedFilesPresent: []string{},
	}
	if keepList == nil {
		keepList = map[string]bool{}
	}

	// Build actual config_files index by name.
	actualFiles := map[string]manifest.ManagedFileRecord{}
	if actual.ConfigFiles != nil {
		for _, a := range actual.ConfigFiles.Elements {
			actualFiles[a.Name] = a
		}
	}
	referenceNames := map[string]bool{}

	// 1. files_modified
	if reference.ConfigFiles != nil {
		for _, e := range reference.ConfigFiles.Elements {
			referenceNames[e.Name] = true
			a, ok := actualFiles[e.Name]
			if !ok {
				// A declared entry absent from actual matches the declaration.
				continue
			}
			modified := false
			if a.Type != e.Type {
				modified = true // type transition
			} else if e.Type == "file" && a.SHA256 != e.SHA256 {
				modified = true
			} else if e.Type == "link" && a.Target != e.Target {
				modified = true
			}
			if modified {
				r.FilesModified = append(r.FilesModified, e.Name)
			}
		}
	}

	// 2. files_extra
	if actual.ConfigFiles != nil {
		for _, a := range actual.ConfigFiles.Elements {
			if referenceNames[a.Name] {
				continue
			}
			if a.PackageName != "" {
				continue // package managed, not "extra"
			}
			if keepList[a.Name] || a.Name == "/etc/etc.syncpoint" {
				continue
			}
			r.FilesExtra = append(r.FilesExtra, a.Name)
		}
	}

	// 3. units_divergent
	if reference.Services != nil {
		actualUnits := map[string]string{}
		if actual.Services != nil {
			for _, s := range actual.Services.Elements {
				actualUnits[s.Name] = s.State
			}
		}
		for _, u := range reference.Services.Elements {
			if st, ok := actualUnits[u.Name]; ok && st != u.State {
				r.UnitsDivergent = append(r.UnitsDivergent, u)
			}
		}
	}

	// 4. packages_divergent: present in one but not the other (identity fields).
	if reference.Packages != nil || actual.Packages != nil {
		refSet := map[string]manifest.PackageRecord{}
		actSet := map[string]manifest.PackageRecord{}
		if reference.Packages != nil {
			for _, p := range reference.Packages.Elements {
				refSet[pkgKey(p)] = p
			}
		}
		if actual.Packages != nil {
			for _, p := range actual.Packages.Elements {
				actSet[pkgKey(p)] = p
			}
		}
		for k, p := range refSet {
			if _, ok := actSet[k]; !ok {
				r.PackagesDivergent = append(r.PackagesDivergent, p)
			}
		}
		for k, p := range actSet {
			if _, ok := refSet[k]; !ok {
				r.PackagesDivergent = append(r.PackagesDivergent, p)
			}
		}
	}

	// 5. integrity categories (scope=full): presence is itself drift.
	if actual.ChangedManagedFiles != nil {
		for _, e := range actual.ChangedManagedFiles.Elements {
			if keepList[e.Name] {
				continue
			}
			r.ManagedFilesModified = append(r.ManagedFilesModified, e.Name)
		}
	}
	if actual.UnmanagedFiles != nil {
		for _, e := range actual.UnmanagedFiles.Elements {
			if keepList[e.Name] {
				continue
			}
			r.UnmanagedFilesPresent = append(r.UnmanagedFilesPresent, e.Name)
		}
	}

	return r
}

// pkgKey identifies a package for drift comparison by its declarable identity.
// When a reference package carries no version (a desired-style entry), identity
// reduces to name+arch so a resolved actual record still matches by name.
func pkgKey(p manifest.PackageRecord) string {
	if p.Version == "" && p.Release == "" {
		return p.Name + "\x00" + p.Arch
	}
	return p.Name + "\x00" + p.Version + "\x00" + p.Release + "\x00" + p.Arch
}
