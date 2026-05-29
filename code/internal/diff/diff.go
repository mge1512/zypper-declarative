// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Package diff computes the intent diff (desired vs applied) and the drift report
// (actual vs declaration). Both are pure comparisons of in-memory Manifest
// documents; this package performs no filesystem, rpmdb, or process I/O.
package diff

import (
	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// Diff is the intent diff: desired_new versus applied_old, scope by scope.
type Diff struct {
	PackagesInstall []manifest.PackageRecord
	PackagesRemove  []manifest.PackageRecord
	ReposSet        []manifest.RepositoryRecord
	FilesWrite      []manifest.ManagedFileRecord
	FilesDelete     []string
	UnitsChange     []manifest.ServiceRecord
}

// Empty reports whether the diff contains no changes.
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

// Empty reports whether the drift report is empty (actual equals reference
// modulo the keep-list).
func (r DriftReport) Empty() bool {
	return len(r.FilesModified) == 0 && len(r.FilesExtra) == 0 &&
		len(r.UnitsDivergent) == 0 && len(r.PackagesDivergent) == 0
}

// Count returns the total number of drift items.
func (r DriftReport) Count() int {
	return len(r.FilesModified) + len(r.FilesExtra) + len(r.UnitsDivergent) + len(r.PackagesDivergent)
}

// ComputeIntentDiff computes the changes from the previously applied declaration
// to the desired declaration, scope by scope. A scope absent in desired produces
// no change for that scope; the filesystem is not consulted.
func ComputeIntentDiff(desired manifest.Manifest, applied manifest.AppliedRecord) Diff {
	d := Diff{}

	// Packages.
	if desired.Packages != nil {
		d.PackagesInstall = append([]manifest.PackageRecord(nil), desired.Packages.Elements...)
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

	// Repositories.
	if desired.Repositories != nil {
		d.ReposSet = append([]manifest.RepositoryRecord(nil), desired.Repositories.Elements...)
	}

	// Config files.
	if desired.ConfigFiles != nil {
		d.FilesWrite = append([]manifest.ManagedFileRecord(nil), desired.ConfigFiles.Elements...)
		desiredFileNames := map[string]bool{}
		for _, f := range desired.ConfigFiles.Elements {
			desiredFileNames[f.Name] = true
		}
		if applied.ConfigFiles != nil {
			for _, f := range applied.ConfigFiles.Elements {
				if !desiredFileNames[f.Name] {
					d.FilesDelete = append(d.FilesDelete, f.Name)
				}
			}
		}
	}

	// Services.
	if desired.Services != nil {
		appliedState := map[string]string{}
		if applied.Services != nil {
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

// DriftOptions carry the exclusion sets for compute-drift.
type DriftOptions struct {
	KeepList map[string]bool
}

const syncpoint = "/etc/etc.syncpoint"

// ComputeDrift compares an actual-state Manifest against a declaration, scope by
// scope on identity fields, and reports divergence. It performs no I/O.
func ComputeDrift(actual manifest.Manifest, reference manifest.AppliedRecord, opts DriftOptions) DriftReport {
	r := DriftReport{}
	keep := opts.KeepList
	if keep == nil {
		keep = map[string]bool{}
	}

	// 1. files_modified: declared files whose actual sha256 differs. A declared
	// file absent from actual is treated as matching.
	actualFiles := map[string]manifest.ManagedFileRecord{}
	if actual.ConfigFiles != nil {
		for _, a := range actual.ConfigFiles.Elements {
			actualFiles[a.Name] = a
		}
	}
	referenceFiles := map[string]bool{}
	if reference.ConfigFiles != nil {
		for _, e := range reference.ConfigFiles.Elements {
			referenceFiles[e.Name] = true
			if a, ok := actualFiles[e.Name]; ok && a.SHA256 != e.SHA256 {
				r.FilesModified = append(r.FilesModified, e.Name)
			}
		}
	}

	// 2. files_extra: actual unpackaged, undeclared /etc files, not keep-listed,
	// not the syncpoint.
	if actual.ConfigFiles != nil {
		for _, a := range actual.ConfigFiles.Elements {
			if referenceFiles[a.Name] {
				continue
			}
			if a.PackageName != "" {
				continue // package-owned files are not "extra"
			}
			if keep[a.Name] || a.Name == syncpoint {
				continue
			}
			r.FilesExtra = append(r.FilesExtra, a.Name)
		}
	}

	// 3. units_divergent: declared units whose actual state differs.
	actualUnits := map[string]string{}
	if actual.Services != nil {
		for _, s := range actual.Services.Elements {
			actualUnits[s.Name] = s.State
		}
	}
	if reference.Services != nil {
		for _, u := range reference.Services.Elements {
			if st, ok := actualUnits[u.Name]; ok && st != u.State {
				r.UnitsDivergent = append(r.UnitsDivergent, u)
			}
		}
	}

	// 4. packages_divergent: packages present in one but not the other (identity
	// fields).
	actualPkgs := map[string]manifest.PackageRecord{}
	if actual.Packages != nil {
		for _, p := range actual.Packages.Elements {
			actualPkgs[pkgKey(p)] = p
		}
	}
	refPkgs := map[string]manifest.PackageRecord{}
	if reference.Packages != nil {
		for _, p := range reference.Packages.Elements {
			refPkgs[pkgKey(p)] = p
		}
	}
	for k, p := range refPkgs {
		if _, ok := actualPkgs[k]; !ok {
			r.PackagesDivergent = append(r.PackagesDivergent, p)
		}
	}
	for k, p := range actualPkgs {
		if _, ok := refPkgs[k]; !ok {
			r.PackagesDivergent = append(r.PackagesDivergent, p)
		}
	}

	return r
}

func pkgKey(p manifest.PackageRecord) string {
	return p.Name + "\x00" + p.Version + "\x00" + p.Release + "\x00" + p.Arch
}
