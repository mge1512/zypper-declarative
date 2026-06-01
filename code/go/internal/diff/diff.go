// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Package diff implements the two pure comparisons: compute-intent-diff
// (desired vs applied) and compute-drift (actual vs reference). Neither reads
// the filesystem, the rpmdb, or any process: both are pure functions of two
// in-memory Manifest documents.
package diff

import (
	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// KeepList is the set of persistent-but-undeclared paths drift must never
// report and converge-files must never delete.
type KeepList map[string]bool

// Has reports membership.
func (k KeepList) Has(p string) bool { return k != nil && k[p] }

const syncpoint = "/etc/etc.syncpoint"

// ComputeIntentDiff implements BEHAVIOR/INTERNAL: compute-intent-diff. A scope
// absent in desired produces no change for that scope (unmanaged).
func ComputeIntentDiff(desired *manifest.Manifest, applied *manifest.Manifest) manifest.Diff {
	var d manifest.Diff

	// 1. packages
	if desired.Packages != nil {
		d.PackagesInstall = append([]manifest.PackageRecord(nil), desired.Packages.Elements...)
		desiredNames := map[string]bool{}
		for _, p := range desired.Packages.Elements {
			desiredNames[p.Name] = true
		}
		if applied != nil && applied.Packages != nil {
			for _, p := range applied.Packages.Elements {
				if !desiredNames[p.Name] {
					d.PackagesRemove = append(d.PackagesRemove, p)
				}
			}
		}
	}

	// 2. repositories
	if desired.Repositories != nil {
		d.ReposSet = append([]manifest.RepositoryRecord(nil), desired.Repositories.Elements...)
	}

	// 3. config_files: files_write = desired; files_delete = declared_old - declared_new
	if desired.ConfigFiles != nil {
		d.FilesWrite = append([]manifest.ManagedFileRecord(nil), desired.ConfigFiles.Elements...)
		desiredPaths := map[string]bool{}
		for _, e := range desired.ConfigFiles.Elements {
			desiredPaths[e.Name] = true
		}
		if applied != nil && applied.ConfigFiles != nil {
			for _, e := range applied.ConfigFiles.Elements {
				if !desiredPaths[e.Name] {
					d.FilesDelete = append(d.FilesDelete, e.Name)
				}
			}
		}
	}

	// 4. services: units_change = desired records whose state differs from applied
	if desired.Services != nil {
		appliedState := map[string]string{}
		if applied != nil && applied.Services != nil {
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

// ComputeDrift implements BEHAVIOR/INTERNAL: compute-drift. actual is an
// actual-state Manifest; reference is the declaration to compare against. keep
// is the keep-list. Performs no I/O.
func ComputeDrift(actual *manifest.Manifest, reference *manifest.Manifest, keep KeepList) manifest.DriftReport {
	var r manifest.DriftReport

	actualFiles := map[string]manifest.ManagedFileRecord{}
	if actual.ConfigFiles != nil {
		for _, a := range actual.ConfigFiles.Elements {
			actualFiles[a.Name] = a
		}
	}
	refFiles := map[string]bool{}
	if reference.ConfigFiles != nil {
		for _, e := range reference.ConfigFiles.Elements {
			refFiles[e.Name] = true
		}
	}

	// 1. files_modified
	if reference.ConfigFiles != nil {
		for _, e := range reference.ConfigFiles.Elements {
			a, present := actualFiles[e.Name]
			if !present {
				// A declared entry absent from actual is treated as matching.
				continue
			}
			if fileDiffers(e, a) {
				r.FilesModified = append(r.FilesModified, e.Name)
			}
		}
	}

	// 2. files_extra: unpackaged, undeclared /etc files, not keep-listed/syncpoint
	if actual.ConfigFiles != nil {
		for _, a := range actual.ConfigFiles.Elements {
			if refFiles[a.Name] {
				continue
			}
			if a.PackageName != "" {
				continue // package-owned but undeclared is not "extra"
			}
			if a.Name == syncpoint || keep.Has(a.Name) {
				continue
			}
			r.FilesExtra = append(r.FilesExtra, a.Name)
		}
	}

	// 3. units_divergent
	if reference.Services != nil {
		actualState := map[string]string{}
		if actual.Services != nil {
			for _, s := range actual.Services.Elements {
				actualState[s.Name] = s.State
			}
		}
		for _, u := range reference.Services.Elements {
			if st, ok := actualState[u.Name]; ok && st != u.State {
				r.UnitsDivergent = append(r.UnitsDivergent, u)
			} else if !ok {
				// actual does not report the declared unit's state: divergent.
				r.UnitsDivergent = append(r.UnitsDivergent, u)
			}
		}
	}

	// 4. packages_divergent: any package present in one but not the other (identity)
	if reference.Packages != nil || actual.Packages != nil {
		refSet := map[string]manifest.PackageRecord{}
		if reference.Packages != nil {
			for _, p := range reference.Packages.Elements {
				refSet[pkgKey(p)] = p
			}
		}
		actSet := map[string]manifest.PackageRecord{}
		if actual.Packages != nil {
			for _, p := range actual.Packages.Elements {
				actSet[pkgKey(p)] = p
			}
		}
		// Only meaningful when the reference declares packages: a bare actual
		// scope with no reference is not drift against a declaration. Guard on
		// reference presence so an actual-only state does not flag everything.
		if reference.Packages != nil {
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
	}

	// 5. integrity categories (full scan): presence is itself drift.
	if actual.ChangedManagedFiles != nil {
		for _, e := range actual.ChangedManagedFiles.Elements {
			if keep.Has(e.Name) {
				continue
			}
			r.ManagedFilesModified = append(r.ManagedFilesModified, e.Name)
		}
	}
	if actual.UnmanagedFiles != nil {
		for _, e := range actual.UnmanagedFiles.Elements {
			if keep.Has(e.Name) {
				continue
			}
			r.UnmanagedFilesPresent = append(r.UnmanagedFilesPresent, e.Name)
		}
	}

	return r
}

// fileDiffers reports whether actual record a diverges from declared record e,
// with type as part of identity.
func fileDiffers(e, a manifest.ManagedFileRecord) bool {
	if a.Type != e.Type {
		return true // type transition
	}
	switch e.Type {
	case "file":
		return a.SHA256 != e.SHA256
	case "link":
		return a.Target != e.Target
	default:
		return false
	}
}

func pkgKey(p manifest.PackageRecord) string {
	return p.Name + "\x00" + p.Version + "\x00" + p.Release + "\x00" + p.Arch
}
