// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Package diff implements compute-intent-diff and compute-drift. Both are pure
// comparisons over in-memory Manifest documents and perform no I/O.
package diff

import (
	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// SyncpointPath and the keep-list paths are never written or deleted by
// converge-files and never reported as files_extra by compute-drift.
const SyncpointPath = "/etc/etc.syncpoint"

// ComputeIntentDiff implements BEHAVIOR/INTERNAL: compute-intent-diff. It
// reconciles the previously applied declaration to the desired declaration,
// scope by scope. A scope absent in desired produces no change for that scope.
func ComputeIntentDiff(desired *manifest.Manifest, applied *manifest.AppliedRecord) manifest.Diff {
	var d manifest.Diff

	// 1. packages.
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

	// 2. repositories.
	if desired.Repositories != nil {
		d.ReposSet = append([]manifest.RepositoryRecord(nil), desired.Repositories.Elements...)
	}

	// 3. config_files.
	if desired.ConfigFiles != nil {
		d.FilesWrite = append([]manifest.ManagedFileRecord(nil), desired.ConfigFiles.Elements...)
		desiredPaths := map[string]bool{}
		for _, c := range desired.ConfigFiles.Elements {
			desiredPaths[c.Name] = true
		}
		if applied != nil && applied.ConfigFiles != nil {
			for _, c := range applied.ConfigFiles.Elements {
				if !desiredPaths[c.Name] {
					d.FilesDelete = append(d.FilesDelete, c.Name)
				}
			}
		}
	}

	// 4. services.
	if desired.Services != nil {
		appliedState := map[string]string{}
		if applied != nil && applied.Services != nil {
			for _, s := range applied.Services.Elements {
				appliedState[s.Name] = s.State
			}
		}
		for _, s := range desired.Services.Elements {
			prev, ok := appliedState[s.Name]
			if !ok || prev != s.State {
				d.UnitsChange = append(d.UnitsChange, s)
			}
		}
	}

	return d
}

// ComputeDrift implements BEHAVIOR/INTERNAL: compute-drift. It compares an
// actual-state Manifest against a declaration, on identity fields. keepList is
// the set of paths the keep-list suppresses.
func ComputeDrift(actual *manifest.Manifest, reference *manifest.AppliedRecord, keepList map[string]bool) manifest.DriftReport {
	var r manifest.DriftReport
	if keepList == nil {
		keepList = map[string]bool{}
	}

	// Index actual config_files by name.
	actualFiles := map[string]manifest.ManagedFileRecord{}
	if actual.ConfigFiles != nil {
		for _, a := range actual.ConfigFiles.Elements {
			actualFiles[a.Name] = a
		}
	}

	// 1. files_modified: declared file whose actual sha256 differs. A declared
	//    file absent from actual is treated as matching.
	declared := map[string]bool{}
	if reference != nil && reference.ConfigFiles != nil {
		for _, e := range reference.ConfigFiles.Elements {
			declared[e.Name] = true
			if a, ok := actualFiles[e.Name]; ok && a.SHA256 != e.SHA256 {
				r.FilesModified = append(r.FilesModified, e.Name)
			}
		}
	}

	// 2. files_extra: actual file undeclared, unpackaged, not keep-listed,
	//    not the syncpoint.
	if actual.ConfigFiles != nil {
		for _, a := range actual.ConfigFiles.Elements {
			if declared[a.Name] {
				continue
			}
			if a.PackageName != "" {
				continue // package-owned, not "extra"
			}
			if keepList[a.Name] || a.Name == SyncpointPath {
				continue
			}
			r.FilesExtra = append(r.FilesExtra, a.Name)
		}
	}

	// 3. units_divergent.
	actualUnits := map[string]string{}
	if actual.Services != nil {
		for _, s := range actual.Services.Elements {
			actualUnits[s.Name] = s.State
		}
	}
	if reference != nil && reference.Services != nil {
		for _, u := range reference.Services.Elements {
			if st, ok := actualUnits[u.Name]; ok && st != u.State {
				r.UnitsDivergent = append(r.UnitsDivergent, u)
			}
		}
	}

	// 4. packages_divergent: present in one but not the other (identity).
	refPkgs := map[string]bool{}
	if reference != nil && reference.Packages != nil {
		for _, p := range reference.Packages.Elements {
			refPkgs[pkgKey(p)] = true
		}
	}
	actPkgs := map[string]bool{}
	if actual.Packages != nil {
		for _, p := range actual.Packages.Elements {
			actPkgs[pkgKey(p)] = true
		}
	}
	if (reference != nil && reference.Packages != nil) || actual.Packages != nil {
		if reference != nil && reference.Packages != nil {
			for _, p := range reference.Packages.Elements {
				if !actPkgs[pkgKey(p)] {
					r.PackagesDivergent = append(r.PackagesDivergent, p)
				}
			}
		}
		if actual.Packages != nil {
			for _, p := range actual.Packages.Elements {
				if !refPkgs[pkgKey(p)] {
					r.PackagesDivergent = append(r.PackagesDivergent, p)
				}
			}
		}
	}

	// 5. Integrity categories (full scan): presence is itself drift.
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

func pkgKey(p manifest.PackageRecord) string {
	return p.Name + "\x00" + p.Version + "\x00" + p.Release + "\x00" + p.Arch
}
