// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Package diff implements compute-intent-diff and compute-drift. Both are pure:
// they perform no filesystem, rpmdb, or process I/O; they compare in-memory
// Manifest documents only.
package diff

import "github.com/mge1512/zypper-declarative/internal/manifest"

// Diff is the intent diff: desired_new versus applied_old, scope by scope.
type Diff struct {
	PackagesInstall []manifest.PackageRecord
	PackagesRemove  []manifest.PackageRecord
	ReposSet        []manifest.RepositoryRecord
	FilesWrite      []manifest.ManagedFileRecord
	FilesDelete     []string
	UnitsChange     []manifest.ServiceRecord
}

// Empty reports whether the intent diff would change nothing.
func (d Diff) Empty() bool {
	return len(d.PackagesInstall) == 0 && len(d.PackagesRemove) == 0 &&
		len(d.ReposSet) == 0 && len(d.FilesWrite) == 0 &&
		len(d.FilesDelete) == 0 && len(d.UnitsChange) == 0
}

// DriftReport is the drift diff: actual versus declared.
type DriftReport struct {
	FilesModified     []string
	FilesExtra        []string
	UnitsDivergent    []manifest.ServiceRecord
	PackagesDivergent []manifest.PackageRecord
}

// Empty reports whether the drift report indicates no divergence.
func (r DriftReport) Empty() bool {
	return len(r.FilesModified) == 0 && len(r.FilesExtra) == 0 &&
		len(r.UnitsDivergent) == 0 && len(r.PackagesDivergent) == 0
}

// Count returns the total number of drift items.
func (r DriftReport) Count() int {
	return len(r.FilesModified) + len(r.FilesExtra) + len(r.UnitsDivergent) + len(r.PackagesDivergent)
}

const syncpoint = "/etc/etc.syncpoint"

// ComputeIntentDiff computes the change from applied to desired, scope by scope
// (compute-intent-diff STEPS 1–5). A scope absent in desired contributes
// nothing. The filesystem is not read.
func ComputeIntentDiff(desired *manifest.Manifest, applied *manifest.AppliedRecord) Diff {
	var d Diff
	d.PackagesInstall = []manifest.PackageRecord{}
	d.PackagesRemove = []manifest.PackageRecord{}
	d.ReposSet = []manifest.RepositoryRecord{}
	d.FilesWrite = []manifest.ManagedFileRecord{}
	d.FilesDelete = []string{}
	d.UnitsChange = []manifest.ServiceRecord{}

	// STEP 1 — packages
	if desired.Packages != nil {
		d.PackagesInstall = append(d.PackagesInstall, desired.Packages.Elements...)
		desiredNames := nameSet(desired.Packages.Elements)
		if applied != nil && applied.Packages != nil {
			for _, p := range applied.Packages.Elements {
				if !desiredNames[p.Name] {
					d.PackagesRemove = append(d.PackagesRemove, p)
				}
			}
		}
	}

	// STEP 2 — repositories
	if desired.Repositories != nil {
		d.ReposSet = append(d.ReposSet, desired.Repositories.Elements...)
	}

	// STEP 3 — config_files
	if desired.ConfigFiles != nil {
		d.FilesWrite = append(d.FilesWrite, desired.ConfigFiles.Elements...)
		desiredPaths := filePathSet(desired.ConfigFiles.Elements)
		if applied != nil && applied.ConfigFiles != nil {
			for _, f := range applied.ConfigFiles.Elements {
				if !desiredPaths[f.Name] {
					d.FilesDelete = append(d.FilesDelete, f.Name)
				}
			}
		}
	}

	// STEP 4 — services
	if desired.Services != nil {
		appliedState := map[string]string{}
		if applied != nil && applied.Services != nil {
			for _, s := range applied.Services.Elements {
				appliedState[s.Name] = s.State
			}
		}
		for _, s := range desired.Services.Elements {
			if appliedState[s.Name] != s.State {
				d.UnitsChange = append(d.UnitsChange, s)
			}
		}
	}

	return d
}

// ComputeDrift compares an actual-state Manifest against a reference declaration
// on identity fields (compute-drift STEPS 1–5). keepList excludes paths from
// files_extra. No I/O is performed.
func ComputeDrift(actual *manifest.Manifest, reference *manifest.AppliedRecord, keepList map[string]bool) DriftReport {
	var r DriftReport
	r.FilesModified = []string{}
	r.FilesExtra = []string{}
	r.UnitsDivergent = []manifest.ServiceRecord{}
	r.PackagesDivergent = []manifest.PackageRecord{}

	// STEP 1 — files_modified
	actualFileSha := map[string]string{}
	actualFilePresent := map[string]bool{}
	if actual.ConfigFiles != nil {
		for _, f := range actual.ConfigFiles.Elements {
			actualFileSha[f.Name] = f.SHA256
			actualFilePresent[f.Name] = true
		}
	}
	declaredPaths := map[string]bool{}
	if reference != nil && reference.ConfigFiles != nil {
		for _, e := range reference.ConfigFiles.Elements {
			declaredPaths[e.Name] = true
			if actualFilePresent[e.Name] && actualFileSha[e.Name] != e.SHA256 {
				r.FilesModified = append(r.FilesModified, e.Name)
			}
			// absent from actual => treated as matching (no entry)
		}
	}

	// STEP 2 — files_extra
	if actual.ConfigFiles != nil {
		for _, a := range actual.ConfigFiles.Elements {
			if declaredPaths[a.Name] {
				continue
			}
			if a.PackageName != "" {
				continue // package-owned, not extra
			}
			if a.Name == syncpoint || keepList[a.Name] {
				continue
			}
			r.FilesExtra = append(r.FilesExtra, a.Name)
		}
	}

	// STEP 3 — units_divergent
	actualUnitState := map[string]string{}
	if actual.Services != nil {
		for _, s := range actual.Services.Elements {
			actualUnitState[s.Name] = s.State
		}
	}
	if reference != nil && reference.Services != nil {
		for _, u := range reference.Services.Elements {
			if st, ok := actualUnitState[u.Name]; ok && st != u.State {
				r.UnitsDivergent = append(r.UnitsDivergent, u)
			} else if !ok {
				// reference declares it, actual does not report it -> divergent
				r.UnitsDivergent = append(r.UnitsDivergent, u)
			}
		}
	}

	// STEP 4 — packages_divergent: present in one but not the other (identity).
	refPkgs := map[string]manifest.PackageRecord{}
	if reference != nil && reference.Packages != nil {
		for _, p := range reference.Packages.Elements {
			refPkgs[pkgKey(p)] = p
		}
	}
	actPkgs := map[string]manifest.PackageRecord{}
	if actual.Packages != nil {
		for _, p := range actual.Packages.Elements {
			actPkgs[pkgKey(p)] = p
		}
	}
	for k, p := range refPkgs {
		if _, ok := actPkgs[k]; !ok {
			r.PackagesDivergent = append(r.PackagesDivergent, p)
		}
	}
	for k, p := range actPkgs {
		if _, ok := refPkgs[k]; !ok {
			r.PackagesDivergent = append(r.PackagesDivergent, p)
		}
	}

	return r
}

func nameSet(pkgs []manifest.PackageRecord) map[string]bool {
	s := map[string]bool{}
	for _, p := range pkgs {
		s[p.Name] = true
	}
	return s
}

func filePathSet(files []manifest.ManagedFileRecord) map[string]bool {
	s := map[string]bool{}
	for _, f := range files {
		s[f.Name] = true
	}
	return s
}

func pkgKey(p manifest.PackageRecord) string {
	return p.Name + "\x00" + p.Version + "\x00" + p.Release + "\x00" + p.Arch
}
