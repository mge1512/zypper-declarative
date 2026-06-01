// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Package diff implements compute-intent-diff and compute-drift. Both are pure
// comparisons of in-memory Manifest documents; this package performs no
// filesystem, rpmdb, or process I/O.
package diff

import (
	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// SyncpointPath is never written or deleted by convergence and never reported
// as extra.
const SyncpointPath = "/etc/etc.syncpoint"

// Diff is the intent diff: desired_new versus applied_old.
type Diff struct {
	PackagesInstall []manifest.PackageRecord
	PackagesRemove  []manifest.PackageRecord
	ReposSet        []manifest.RepositoryRecord
	FilesWrite      []manifest.ManagedFileRecord
	FilesDelete     []string
	UnitsChange     []manifest.ServiceRecord
}

// Empty reports whether the intent diff makes no change.
func (d *Diff) Empty() bool {
	return len(d.PackagesInstall) == 0 && len(d.PackagesRemove) == 0 &&
		len(d.ReposSet) == 0 && len(d.FilesWrite) == 0 &&
		len(d.FilesDelete) == 0 && len(d.UnitsChange) == 0
}

// DriftReport is the drift diff: actual versus declared.
type DriftReport struct {
	FilesModified         []string
	FilesExtra            []string
	UnitsDivergent        []manifest.ServiceRecord
	PackagesDivergent     []manifest.PackageRecord
	ManagedFilesModified  []string
	UnmanagedFilesPresent []string
}

// Empty reports whether the drift report is empty.
func (r *DriftReport) Empty() bool {
	return len(r.FilesModified) == 0 && len(r.FilesExtra) == 0 &&
		len(r.UnitsDivergent) == 0 && len(r.PackagesDivergent) == 0 &&
		len(r.ManagedFilesModified) == 0 && len(r.UnmanagedFilesPresent) == 0
}

// ComputeIntentDiff computes the changes from the previously applied
// declaration to the desired declaration, scope by scope. A scope absent in
// desired produces no change for that scope.
func ComputeIntentDiff(desired *manifest.Manifest, applied *manifest.Manifest) *Diff {
	d := &Diff{
		PackagesInstall: []manifest.PackageRecord{},
		PackagesRemove:  []manifest.PackageRecord{},
		ReposSet:        []manifest.RepositoryRecord{},
		FilesWrite:      []manifest.ManagedFileRecord{},
		FilesDelete:     []string{},
		UnitsChange:     []manifest.ServiceRecord{},
	}

	// Packages.
	if desired.Packages != nil {
		d.PackagesInstall = append(d.PackagesInstall, desired.Packages.Elements...)
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

	// Repositories.
	if desired.Repositories != nil {
		d.ReposSet = append(d.ReposSet, desired.Repositories.Elements...)
	}

	// Config files.
	if desired.ConfigFiles != nil {
		d.FilesWrite = append(d.FilesWrite, desired.ConfigFiles.Elements...)
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

	// Services.
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

// ComputeDrift compares an actual-state Manifest against a reference
// declaration, scope by scope on identity fields. keepList is the set of paths
// that must never be reported.
func ComputeDrift(actual *manifest.Manifest, reference *manifest.Manifest, keepList map[string]bool) *DriftReport {
	r := &DriftReport{
		FilesModified:         []string{},
		FilesExtra:            []string{},
		UnitsDivergent:        []manifest.ServiceRecord{},
		PackagesDivergent:     []manifest.PackageRecord{},
		ManagedFilesModified:  []string{},
		UnmanagedFilesPresent: []string{},
	}

	actualFiles := map[string]manifest.ManagedFileRecord{}
	if actual.ConfigFiles != nil {
		for _, a := range actual.ConfigFiles.Elements {
			actualFiles[a.Name] = a
		}
	}
	refFiles := map[string]bool{}

	// files_modified: declared entries whose actual content/type differs.
	if reference.ConfigFiles != nil {
		for _, e := range reference.ConfigFiles.Elements {
			refFiles[e.Name] = true
			a, present := actualFiles[e.Name]
			if !present {
				// A declared entry absent from actual is treated as matching.
				continue
			}
			if a.Type != e.Type {
				r.FilesModified = append(r.FilesModified, e.Name)
				continue
			}
			switch e.Type {
			case "file":
				if a.SHA256 != e.SHA256 {
					r.FilesModified = append(r.FilesModified, e.Name)
				}
			case "link":
				if a.Target != e.Target {
					r.FilesModified = append(r.FilesModified, e.Name)
				}
			}
		}
	}

	// files_extra: actual unpackaged files not declared, not keep-listed, not syncpoint.
	if actual.ConfigFiles != nil {
		for _, a := range actual.ConfigFiles.Elements {
			if refFiles[a.Name] {
				continue
			}
			if a.PackageName != "" {
				continue // package-managed, not "extra"
			}
			if a.Name == SyncpointPath || keepList[a.Name] {
				continue
			}
			r.FilesExtra = append(r.FilesExtra, a.Name)
		}
	}

	// units_divergent: declared units whose actual state differs.
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
			}
		}
	}

	// packages_divergent: identity present in one but not the other.
	if reference.Packages != nil {
		refSet := map[string]manifest.PackageRecord{}
		for _, p := range reference.Packages.Elements {
			refSet[pkgKey(p)] = p
		}
		actSet := map[string]manifest.PackageRecord{}
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

	// Integrity categories (full scan): presence is itself drift.
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

// pkgKey is a package identity key on the declarable identity fields.
func pkgKey(p manifest.PackageRecord) string {
	return p.Name + "\x00" + p.Version + "\x00" + p.Release + "\x00" + p.Arch
}
