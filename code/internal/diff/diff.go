// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Package diff implements the two pure comparison behaviours:
// compute-intent-diff (desired versus applied, scope by scope; yields deletions)
// and compute-drift (actual versus reference declaration). Neither reads the
// filesystem, the rpmdb, or any process: they compare in-memory Manifest
// documents only.
package diff

import (
	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// Diff is the intent diff: desired_new versus applied_old.
type Diff struct {
	PackagesInstall []manifest.PackageRecord
	PackagesRemove  []manifest.PackageRecord
	ReposSet        []manifest.RepositoryRecord
	FilesWrite      []manifest.ManagedFileRecord
	FilesDelete     []string
	UnitsChange     []manifest.ServiceRecord
}

// Empty reports whether the intent diff has no changes in any scope.
func (d *Diff) Empty() bool {
	return len(d.PackagesInstall) == 0 &&
		len(d.PackagesRemove) == 0 &&
		len(d.ReposSet) == 0 &&
		len(d.FilesWrite) == 0 &&
		len(d.FilesDelete) == 0 &&
		len(d.UnitsChange) == 0
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
func (r *DriftReport) Empty() bool {
	return len(r.FilesModified) == 0 &&
		len(r.FilesExtra) == 0 &&
		len(r.UnitsDivergent) == 0 &&
		len(r.PackagesDivergent) == 0
}

// Count returns the total number of drift items.
func (r *DriftReport) Count() int {
	return len(r.FilesModified) + len(r.FilesExtra) + len(r.UnitsDivergent) + len(r.PackagesDivergent)
}

// ComputeIntentDiff implements BEHAVIOR/INTERNAL: compute-intent-diff. A scope
// absent in desired produces no change for that scope (unmanaged); a present
// scope is reconciled to exactly its elements. The filesystem is not read.
func ComputeIntentDiff(desired *manifest.Manifest, applied *manifest.AppliedRecord) Diff {
	var d Diff

	// Packages.
	if desired.Packages != nil {
		d.PackagesInstall = append([]manifest.PackageRecord(nil), desired.Packages.Elements...)
		desiredNames := nameSet(desired.Packages.Elements)
		for _, p := range appliedPackages(applied) {
			if !desiredNames[p.Name] {
				d.PackagesRemove = append(d.PackagesRemove, p)
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
		desiredPaths := fileNameSet(desired.ConfigFiles.Elements)
		for _, e := range appliedFiles(applied) {
			if !desiredPaths[e.Name] {
				d.FilesDelete = append(d.FilesDelete, e.Name)
			}
		}
	}

	// Services.
	if desired.Services != nil {
		appliedState := serviceStateMap(appliedServices(applied))
		for _, s := range desired.Services.Elements {
			prev, ok := appliedState[s.Name]
			if !ok || prev != s.State {
				d.UnitsChange = append(d.UnitsChange, s)
			}
		}
	}

	return d
}

// ComputeDrift implements BEHAVIOR/INTERNAL: compute-drift. It compares the
// actual-state Manifest against the reference declaration, scope by scope on
// identity fields, honouring the keep-list and the syncpoint. It performs no I/O.
func ComputeDrift(actual *manifest.Manifest, reference *manifest.AppliedRecord, keepList map[string]bool) DriftReport {
	var r DriftReport

	// files_modified: declared files whose actual sha256 differs.
	actualFiles := fileMap(actualFilesOf(actual))
	for _, e := range appliedFiles(reference) {
		if af, ok := actualFiles[e.Name]; ok {
			if af.SHA256 != e.SHA256 {
				r.FilesModified = append(r.FilesModified, e.Name)
			}
		}
		// A declared file absent from actual is treated as matching.
	}

	// files_extra: unpackaged, undeclared /etc files, not keep-listed, not syncpoint.
	declaredPaths := fileNameSet(appliedFiles(reference))
	for _, a := range actualFilesOf(actual) {
		if declaredPaths[a.Name] {
			continue
		}
		if a.PackageName != "" {
			continue
		}
		if a.Name == Syncpoint || (keepList != nil && keepList[a.Name]) {
			continue
		}
		r.FilesExtra = append(r.FilesExtra, a.Name)
	}

	// units_divergent: declared units reported with a different actual state.
	actualUnits := serviceStateMap(actualServicesOf(actual))
	for _, u := range appliedServices(reference) {
		if state, ok := actualUnits[u.Name]; !ok || state != u.State {
			r.UnitsDivergent = append(r.UnitsDivergent, u)
		}
	}

	// packages_divergent: identity-field symmetric difference.
	refPkgs := packageIdentitySet(appliedPackages(reference))
	actPkgs := packageIdentitySet(actualPackagesOf(actual))
	for _, p := range appliedPackages(reference) {
		if !actPkgs[packageIdentity(p)] {
			r.PackagesDivergent = append(r.PackagesDivergent, p)
		}
	}
	for _, p := range actualPackagesOf(actual) {
		if !refPkgs[packageIdentity(p)] {
			r.PackagesDivergent = append(r.PackagesDivergent, p)
		}
	}

	return r
}

// Syncpoint is the /etc reference path that is never written, deleted, or
// reported as drift.
const Syncpoint = "/etc/etc.syncpoint"
